using System.Text;
using HslCommunication;
using HslCommunication.Profinet.Siemens;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslS7TcpClient : IHslS7TcpClient
{
    private readonly SiemensS7Net _client;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslS7TcpClient(HslS7TcpClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _client = new SiemensS7Net(MapPlcType(options.PlcType), options.Host)
        {
            Port = options.Port,
            Rack = options.Rack,
            Slot = options.Slot,
            ConnectTimeOut = options.TimeoutMilliseconds,
            ReceiveTimeOut = options.TimeoutMilliseconds,
        };
        if (options.LocalTsap is not null) _client.LocalTSAP = options.LocalTsap.Value;
        if (options.RemoteTsap is not null) _client.DestTSAP = options.RemoteTsap.Value;
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();

            var result = await _client.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = result.IsSuccess;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_S7_CONNECT_FAILED", FormatError(result), retryable: true);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_S7_CONNECT_FAILED", exception.Message, retryable: true);
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

            var result = await _client.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = false;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_S7_CLOSE_FAILED", FormatError(result), retryable: false);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_S7_CLOSE_FAILED", exception.Message, retryable: false);
        }
        finally
        {
            _operationGate.Release();
        }
    }

    public async Task<HslReadResult> ReadAsync(HslS7ReadRequest request, CancellationToken cancellationToken)
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
                return HslReadResult.Failure("HSL_NOT_CONNECTED", "HSL S7 通信连接尚未建立", retryable: true);
            }
            return await ReadCoreAsync(request, cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            return HslReadResult.Failure("HSL_S7_READ_FAILED", exception.Message, retryable: true);
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
        (_client as IDisposable)?.Dispose();
        _operationGate.Dispose();
    }

    private async Task<HslReadResult> ReadCoreAsync(HslS7ReadRequest request, CancellationToken cancellationToken)
    {
        var count = checked((ushort)request.ElementCount);
        return request.DataType switch
        {
            HslValueType.Boolean => request.ElementCount == 1
                ? Map(await _client.ReadBoolAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed8 => await ReadRawAsync(
                request,
                value => request.ElementCount == 1 ? unchecked((sbyte)value[0]) : value.Select(item => unchecked((sbyte)item)).ToArray(),
                cancellationToken).ConfigureAwait(false),
            HslValueType.Unsigned8 => await ReadRawAsync(
                request,
                value => request.ElementCount == 1 ? value[0] : value,
                cancellationToken).ConfigureAwait(false),
            HslValueType.Signed16 => request.ElementCount == 1
                ? Map(await _client.ReadInt16Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned16 => request.ElementCount == 1
                ? Map(await _client.ReadUInt16Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed32 => request.ElementCount == 1
                ? Map(await _client.ReadInt32Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned32 => request.ElementCount == 1
                ? Map(await _client.ReadUInt32Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed64 => request.ElementCount == 1
                ? Map(await _client.ReadInt64Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned64 => request.ElementCount == 1
                ? Map(await _client.ReadUInt64Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.SinglePrecision => request.ElementCount == 1
                ? Map(await _client.ReadFloatAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.DoublePrecision => request.ElementCount == 1
                ? Map(await _client.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Text => Map(await _client.ReadStringAsync(request.Address, count, Encoding.UTF8).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Binary => await ReadRawAsync(request, value => value, cancellationToken).ConfigureAwait(false),
            HslValueType.DateTime => Map(await _client.ReadDateTimeAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "HSL S7 读取数据类型不受支持", retryable: false),
        };
    }

    private async Task<HslReadResult> ReadRawAsync(
        HslS7ReadRequest request,
        Func<byte[], object> convert,
        CancellationToken cancellationToken)
    {
        var result = await _client.ReadAsync(request.Address, checked((ushort)request.ElementCount))
            .WaitAsync(cancellationToken)
            .ConfigureAwait(false);
        return result.IsSuccess
            ? HslReadResult.Success(convert(result.Content))
            : HslReadResult.Failure("HSL_S7_READ_FAILED", FormatError(result), retryable: true);
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess
        ? HslReadResult.Success(result.Content)
        : HslReadResult.Failure("HSL_S7_READ_FAILED", FormatError(result), retryable: true);

    private static string FormatError(OperateResult result) =>
        string.IsNullOrWhiteSpace(result.Message)
            ? $"HSL S7 操作失败，错误码 {result.ErrorCode}"
            : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";

    private static SiemensPLCS MapPlcType(HslSiemensPlc plcType) => plcType switch
    {
        HslSiemensPlc.S1200 => SiemensPLCS.S1200,
        HslSiemensPlc.S300 => SiemensPLCS.S300,
        HslSiemensPlc.S400 => SiemensPLCS.S400,
        HslSiemensPlc.S1500 => SiemensPLCS.S1500,
        HslSiemensPlc.S200Smart => SiemensPLCS.S200Smart,
        HslSiemensPlc.S200 => SiemensPLCS.S200,
        _ => throw new ArgumentOutOfRangeException(nameof(plcType)),
    };

    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
