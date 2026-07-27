using System.Text;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Profinet.Omron;
using HslCommunication.Profinet.Omron.Helper;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslOmronFinsClient : IHslOmronFinsClient
{
    private readonly HslOmronFinsClientOptions _options;
    private readonly IReadWriteNet _device;
    private readonly IOmronFins _fins;
    private readonly OmronFinsNet? _tcpClient;
    private readonly IDisposable _disposable;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslOmronFinsClient(HslOmronFinsClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _options = options;

        if (options.Transport == HslOmronFinsTransport.Tcp)
        {
            var client = new OmronFinsNet(options.Host, options.Port)
            {
                ConnectTimeOut = options.ConnectTimeoutMilliseconds,
                ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
                ReceiveUntilEmpty = options.ReceiveUntilEmpty,
            };
            Configure(client, client.ByteTransform, options);
            _device = client;
            _fins = client;
            _tcpClient = client;
            _disposable = client;
            return;
        }

        var udpClient = new OmronFinsUdp(options.Host, options.Port)
        {
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
        };
        Configure(udpClient, udpClient.ByteTransform, options);
        _device = udpClient;
        _fins = udpClient;
        _disposable = udpClient;
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();

            OperateResult result;
            if (_tcpClient is not null)
            {
                result = await _tcpClient.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            }
            else
            {
                // FINS UDP 没有连接握手，使用 CPU 状态命令确认目标 PLC 真正可达。
                result = await Task.Run(() => _fins.ReadCpuUnitStatus(), CancellationToken.None).ConfigureAwait(false);
            }

            _connected = result.IsSuccess;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure(ConnectErrorCode(), FormatError(result), retryable: true);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure(ConnectErrorCode(), exception.Message, retryable: true);
        }
        finally
        {
            _operationGate.Release();
        }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected)
            {
                _connected = false;
                return HslOperationResult.Success();
            }

            if (_tcpClient is null)
            {
                _connected = false;
                return HslOperationResult.Success();
            }

            var result = await _tcpClient.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = false;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_OMRON_FINS_CLOSE_FAILED", FormatError(result), retryable: false);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_OMRON_FINS_CLOSE_FAILED", exception.Message, retryable: false);
        }
        finally
        {
            _operationGate.Release();
        }
    }

    public async Task<HslReadResult> ReadAsync(HslOmronFinsReadRequest request, CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(request);
        if (request.ElementCount < 1)
        {
            return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", retryable: false);
        }

        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected)
            {
                return HslReadResult.Failure("HSL_NOT_CONNECTED", "HSL Omron FINS 通信会话尚未打开", retryable: true);
            }
            return await ReadCoreAsync(request, cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            return HslReadResult.Failure("HSL_OMRON_FINS_READ_FAILED", exception.Message, retryable: true);
        }
        finally
        {
            _operationGate.Release();
        }
    }

    public async ValueTask DisposeAsync()
    {
        if (_disposed) return;
        await CloseAsync(CancellationToken.None).ConfigureAwait(false);
        _disposed = true;
        _disposable.Dispose();
        _operationGate.Dispose();
    }

    private async Task<HslReadResult> ReadCoreAsync(HslOmronFinsReadRequest request, CancellationToken cancellationToken)
    {
        var count = checked((ushort)request.ElementCount);
        return request.DataType switch
        {
            HslValueType.Boolean => request.ElementCount == 1
                ? Map(await _device.ReadBoolAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed8 => await ReadRawAsync(request, bytes => request.ElementCount == 1
                ? unchecked((sbyte)bytes[0])
                : bytes.Select(value => unchecked((sbyte)value)).ToArray(), cancellationToken).ConfigureAwait(false),
            HslValueType.Unsigned8 => await ReadRawAsync(request, bytes => request.ElementCount == 1 ? bytes[0] : bytes, cancellationToken).ConfigureAwait(false),
            HslValueType.Signed16 => request.ElementCount == 1
                ? Map(await _device.ReadInt16Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned16 => request.ElementCount == 1
                ? Map(await _device.ReadUInt16Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed32 => request.ElementCount == 1
                ? Map(await _device.ReadInt32Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned32 => request.ElementCount == 1
                ? Map(await _device.ReadUInt32Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed64 => request.ElementCount == 1
                ? Map(await _device.ReadInt64Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned64 => request.ElementCount == 1
                ? Map(await _device.ReadUInt64Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.SinglePrecision => request.ElementCount == 1
                ? Map(await _device.ReadFloatAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.DoublePrecision => request.ElementCount == 1
                ? Map(await _device.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Text => Map(await _device.ReadStringAsync(request.Address, count, Encoding.UTF8).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Binary => await ReadRawAsync(request, bytes => bytes, cancellationToken).ConfigureAwait(false),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "HSL Omron FINS 读取数据类型不受支持", retryable: false),
        };
    }

    private async Task<HslReadResult> ReadRawAsync(
        HslOmronFinsReadRequest request,
        Func<byte[], object> convert,
        CancellationToken cancellationToken)
    {
        var wordCount = checked((ushort)Math.Ceiling(request.ElementCount / 2d));
        var result = await _device.ReadAsync(request.Address, wordCount).WaitAsync(cancellationToken).ConfigureAwait(false);
        return result.IsSuccess
            ? HslReadResult.Success(convert(result.Content.Take(request.ElementCount).ToArray()))
            : HslReadResult.Failure("HSL_OMRON_FINS_READ_FAILED", FormatError(result), retryable: true);
    }

    private static void Configure(IOmronFins fins, IByteTransform byteTransform, HslOmronFinsClientOptions options)
    {
        fins.PlcType = options.PlcType == HslOmronPlcType.CV ? OmronPlcType.CV : OmronPlcType.CSCJ;
        fins.ReadSplits = options.ReadSplits;
        fins.GCT = options.Gct;
        fins.SID = options.Sid;
        byteTransform.DataFormat = MapDataFormat(options.DataFormat);
        byteTransform.IsStringReverseByteWord = options.StringReverseByteWord;
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess
        ? HslReadResult.Success(result.Content)
        : HslReadResult.Failure("HSL_OMRON_FINS_READ_FAILED", FormatError(result), retryable: true);

    private string ConnectErrorCode() => _options.Transport == HslOmronFinsTransport.Tcp
        ? "HSL_OMRON_FINS_CONNECT_FAILED"
        : "HSL_OMRON_FINS_PROBE_FAILED";

    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message)
        ? $"HSL Omron FINS 操作失败，错误码 {result.ErrorCode}"
        : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";

    private static DataFormat MapDataFormat(HslDataFormat dataFormat) => dataFormat switch
    {
        HslDataFormat.ABCD => DataFormat.ABCD,
        HslDataFormat.BADC => DataFormat.BADC,
        HslDataFormat.CDAB => DataFormat.CDAB,
        HslDataFormat.DCBA => DataFormat.DCBA,
        _ => throw new ArgumentOutOfRangeException(nameof(dataFormat)),
    };

    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
