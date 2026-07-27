using System.IO.Ports;
using System.Text;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.AllenBradley;
using HslCommunication.Serial;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslAllenBradleyClient : IHslAllenBradleyClient
{
    private readonly DeviceCommunication _device;
    private readonly IReadWriteCip? _cipDevice;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslAllenBradleyClient(HslAllenBradleyClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = CreateDevice(options);
        _cipDevice = _device as IReadWriteCip;
        _device.ByteTransform.DataFormat = MapDataFormat(options.DataFormat);
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();
            var result = _device switch
            {
                DeviceTcpNet tcp => await tcp.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false),
                DeviceSerialPort serial => await Task.Run(serial.Open, cancellationToken).ConfigureAwait(false),
                _ => throw new InvalidOperationException("未知的 Allen-Bradley 通信设备类型"),
            };
            _connected = result.IsSuccess;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_ALLEN_BRADLEY_CONNECT_FAILED", FormatError(result), retryable: true);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_ALLEN_BRADLEY_CONNECT_FAILED", exception.Message, retryable: true);
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
            OperateResult result;
            if (_device is DeviceTcpNet tcp)
            {
                result = await tcp.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            }
            else if (_device is DeviceSerialPort serial)
            {
                await Task.Run(serial.Close, cancellationToken).ConfigureAwait(false);
                result = OperateResult.CreateSuccessResult();
            }
            else
            {
                throw new InvalidOperationException("未知的 Allen-Bradley 通信设备类型");
            }
            _connected = false;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_ALLEN_BRADLEY_CLOSE_FAILED", FormatError(result), retryable: false);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_ALLEN_BRADLEY_CLOSE_FAILED", exception.Message, retryable: false);
        }
        finally
        {
            _operationGate.Release();
        }
    }

    public async Task<HslReadResult> ReadAsync(HslAllenBradleyReadRequest request, CancellationToken cancellationToken)
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
                return HslReadResult.Failure("HSL_NOT_CONNECTED", "HSL Allen-Bradley 通信连接尚未建立", retryable: true);
            }
            return await ReadCoreAsync(request, cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            return HslReadResult.Failure("HSL_ALLEN_BRADLEY_READ_FAILED", exception.Message, retryable: true);
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
        _device.Dispose();
        _operationGate.Dispose();
    }

    private async Task<HslReadResult> ReadCoreAsync(HslAllenBradleyReadRequest request, CancellationToken cancellationToken)
    {
        var count = checked((ushort)request.ElementCount);
        return request.DataType switch
        {
            HslValueType.Boolean => request.ElementCount == 1
                ? Map(await _device.ReadBoolAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed8 => await ReadRawAsync(request, value => request.ElementCount == 1
                ? unchecked((sbyte)value[0])
                : value.Select(item => unchecked((sbyte)item)).ToArray(), cancellationToken).ConfigureAwait(false),
            HslValueType.Unsigned8 => await ReadRawAsync(
                request,
                value => request.ElementCount == 1 ? value[0] : value,
                cancellationToken).ConfigureAwait(false),
            HslValueType.Signed16 => Map(await _device.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
            HslValueType.Unsigned16 => Map(await _device.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
            HslValueType.Signed32 => Map(await _device.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
            HslValueType.Unsigned32 => Map(await _device.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
            HslValueType.Signed64 => Map(await _device.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
            HslValueType.Unsigned64 => Map(await _device.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
            HslValueType.SinglePrecision => Map(await _device.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
            HslValueType.DoublePrecision => Map(await _device.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
            HslValueType.Text => await ReadTextAsync(request, count, cancellationToken).ConfigureAwait(false),
            HslValueType.Binary => await ReadRawAsync(request, value => value, cancellationToken).ConfigureAwait(false),
            HslValueType.DateTime => _cipDevice is null
                ? HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "当前 Allen-Bradley 协议不支持 datetime 数据类型", retryable: false)
                : Map(await _cipDevice.ReadDateAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "HSL Allen-Bradley 读取数据类型不受支持", retryable: false),
        };
    }

    private async Task<HslReadResult> ReadRawAsync(
        HslAllenBradleyReadRequest request,
        Func<byte[], object> convert,
        CancellationToken cancellationToken)
    {
        var result = await _device.ReadAsync(request.Address, checked((ushort)request.ElementCount)).WaitAsync(cancellationToken).ConfigureAwait(false);
        return result.IsSuccess
            ? HslReadResult.Success(convert(result.Content.Take(request.ElementCount).ToArray()))
            : HslReadResult.Failure("HSL_ALLEN_BRADLEY_READ_FAILED", FormatError(result), retryable: true);
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess
        ? HslReadResult.Success(result.Content)
        : HslReadResult.Failure("HSL_ALLEN_BRADLEY_READ_FAILED", FormatError(result), retryable: true);

    private static HslReadResult Map<T>(OperateResult<T[]> result, int elementCount) => result.IsSuccess
        ? HslReadResult.Success(elementCount == 1 ? result.Content[0] : result.Content)
        : HslReadResult.Failure("HSL_ALLEN_BRADLEY_READ_FAILED", FormatError(result), retryable: true);

    private async Task<HslReadResult> ReadTextAsync(
        HslAllenBradleyReadRequest request,
        ushort count,
        CancellationToken cancellationToken)
    {
        if (_device is AllenBradleyNet tagDevice)
        {
            return Map(await tagDevice.ReadStringAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false));
        }
        if (_device is AllenBradleyPcccNet pcccDevice)
        {
            return Map(await pcccDevice.ReadStringAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false));
        }
        return Map(await _device.ReadStringAsync(request.Address, count, Encoding.ASCII).WaitAsync(cancellationToken).ConfigureAwait(false));
    }

    private static DeviceCommunication CreateDevice(HslAllenBradleyClientOptions options)
    {
        return options.Protocol switch
        {
            HslAllenBradleyProtocol.EtherNetIp => CreateEtherNetIp(options),
            HslAllenBradleyProtocol.ConnectedCip => ConfigureNetwork(new AllenBradleyConnectedCipNet(options.Host, options.Port), options),
            HslAllenBradleyProtocol.MicroCip => CreateMicroCip(options),
            HslAllenBradleyProtocol.Pccc => ConfigureNetwork(new AllenBradleyPcccNet(options.Host, options.Port), options),
            HslAllenBradleyProtocol.Slc => ConfigureNetwork(new AllenBradleySLCNet(options.Host, options.Port), options),
            HslAllenBradleyProtocol.Df1Serial => CreateDf1Serial(options),
            _ => throw new ArgumentOutOfRangeException(nameof(options), "不支持的 Allen-Bradley 协议"),
        };
    }

    private static AllenBradleyNet CreateEtherNetIp(HslAllenBradleyClientOptions options)
    {
        var device = ConfigureNetwork(new AllenBradleyNet(options.Host, options.Port), options);
        device.Slot = options.Slot;
        device.ContextCheck = options.ContextCheck;
        device.ReadArrayUseSegment = options.ReadArrayUseSegment;
        if (!string.IsNullOrWhiteSpace(options.MessageRouter))
        {
            device.MessageRouter = new MessageRouter(options.MessageRouter);
        }
        return device;
    }

    private static AllenBradleyMicroCip CreateMicroCip(HslAllenBradleyClientOptions options)
    {
        var device = ConfigureNetwork(new AllenBradleyMicroCip(options.Host, options.Port), options);
        device.Slot = options.Slot;
        device.PortSlot = options.PortSlot?.ToArray() ?? [0x01, 0x00];
        return device;
    }

    private static AllenBradleyDF1Serial CreateDf1Serial(HslAllenBradleyClientOptions options)
    {
        if (string.IsNullOrWhiteSpace(options.PortName))
        {
            throw new ArgumentException("Allen-Bradley DF1 串口名称不能为空", nameof(options));
        }
        var device = new AllenBradleyDF1Serial
        {
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
            Station = options.Station,
            DstNode = options.DestinationNode,
            SrcNode = options.SourceNode,
            CheckType = options.CheckType == HslAllenBradleyCheckType.Bcc ? CheckType.BCC : CheckType.CRC16,
        };
        device.SerialPortInni(
            options.PortName,
            options.BaudRate,
            options.DataBits,
            options.StopBits == HslSerialStopBits.Two ? StopBits.Two : StopBits.One,
            options.Parity switch
            {
                HslSerialParity.Odd => Parity.Odd,
                HslSerialParity.Even => Parity.Even,
                _ => Parity.None,
            });
        return device;
    }

    private static T ConfigureNetwork<T>(T device, HslAllenBradleyClientOptions options) where T : DeviceTcpNet
    {
        device.ConnectTimeOut = options.ConnectTimeoutMilliseconds;
        device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
        return device;
    }

    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message)
        ? $"HSL Allen-Bradley 操作失败，错误码 {result.ErrorCode}"
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
