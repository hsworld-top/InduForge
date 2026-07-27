using System.IO.Ports;
using System.Text;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.Delta;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslDeltaClient : IHslDeltaClient
{
    private readonly DeviceCommunication _device;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslDeltaClient(HslDeltaClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = CreateDevice(options);
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
            cancellationToken.ThrowIfCancellationRequested();
            var result = _device switch
            {
                DeviceTcpNet tcp => await tcp.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false),
                DeviceSerialPort serial => serial.Open(),
                _ => throw new InvalidOperationException("未知的台达通信设备类型"),
            };
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("HSL_DELTA_CONNECT_FAILED", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_DELTA_CONNECT_FAILED", exception.Message, true);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); }
            cancellationToken.ThrowIfCancellationRequested();
            OperateResult result;
            if (_device is DeviceTcpNet tcp) result = await tcp.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            else if (_device is DeviceSerialPort serial) { serial.Close(); result = OperateResult.CreateSuccessResult(); }
            else throw new InvalidOperationException("未知的台达通信设备类型");
            _connected = false;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("HSL_DELTA_CLOSE_FAILED", result, false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_DELTA_CLOSE_FAILED", exception.Message, false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslDeltaReadRequest request, CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(request);
        if (request.ElementCount < 1) return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", false);
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "台达通信连接尚未建立", true);
            return await ReadCoreAsync(request, cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_DELTA_READ_FAILED", exception.Message, true); }
        finally { _operationGate.Release(); }
    }

    public async ValueTask DisposeAsync()
    {
        if (_disposed) return;
        await CloseAsync(CancellationToken.None).ConfigureAwait(false);
        _disposed = true;
        _device.Dispose();
        _operationGate.Dispose();
    }

    private static DeviceCommunication CreateDevice(HslDeltaClientOptions options)
    {
        var series = options.Series == HslDeltaSeries.AS ? DeltaSeries.AS : DeltaSeries.Dvp;
        DeviceCommunication device = options.Transport switch
        {
            HslDeltaTransport.Tcp => new DeltaTcpNet(options.Host!, options.Port, options.Station) { Series = series, ConnectTimeOut = options.ConnectTimeoutMilliseconds, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds },
            HslDeltaTransport.RtuOverTcp => new DeltaSerialOverTcp(options.Host!, options.Port, options.Station) { Series = series, ConnectTimeOut = options.ConnectTimeoutMilliseconds, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds },
            HslDeltaTransport.AsciiOverTcp => new DeltaSerialAsciiOverTcp(options.Host!, options.Port, options.Station) { Series = series, ConnectTimeOut = options.ConnectTimeoutMilliseconds, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds },
            HslDeltaTransport.RtuSerial => CreateSerial(new DeltaSerial(options.Station) { Series = series }, options),
            HslDeltaTransport.AsciiSerial => CreateSerial(new DeltaSerialAscii(options.Station) { Series = series }, options),
            _ => throw new ArgumentOutOfRangeException(nameof(options), options.Transport, "未知的台达传输类型"),
        };
        return device;
    }

    private static DeviceSerialPort CreateSerial(DeviceSerialPort device, HslDeltaClientOptions options)
    {
        device.SerialPortInni(options.PortName!, options.BaudRate, options.DataBits, MapStopBits(options.StopBits), MapParity(options.Parity));
        device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
        return device;
    }

    private async Task<HslReadResult> ReadCoreAsync(HslDeltaReadRequest request, CancellationToken cancellationToken)
    {
        var count = checked((ushort)request.ElementCount);
        return request.DataType switch
        {
            HslValueType.Boolean => request.ElementCount == 1 ? Map(await _device.ReadBoolAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed8 => await ReadRawAsync(request, bytes => request.ElementCount == 1 ? unchecked((sbyte)bytes[0]) : bytes.Select(value => unchecked((sbyte)value)).ToArray(), cancellationToken),
            HslValueType.Unsigned8 => await ReadRawAsync(request, bytes => request.ElementCount == 1 ? bytes[0] : bytes, cancellationToken),
            HslValueType.Signed16 => request.ElementCount == 1 ? Map(await _device.ReadInt16Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned16 => request.ElementCount == 1 ? Map(await _device.ReadUInt16Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed32 => request.ElementCount == 1 ? Map(await _device.ReadInt32Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned32 => request.ElementCount == 1 ? Map(await _device.ReadUInt32Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed64 => request.ElementCount == 1 ? Map(await _device.ReadInt64Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned64 => request.ElementCount == 1 ? Map(await _device.ReadUInt64Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.SinglePrecision => request.ElementCount == 1 ? Map(await _device.ReadFloatAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.DoublePrecision => request.ElementCount == 1 ? Map(await _device.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Text => Map(await _device.ReadStringAsync(request.Address, count, Encoding.UTF8).WaitAsync(cancellationToken)),
            HslValueType.Binary => await ReadRawAsync(request, bytes => bytes, cancellationToken),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "台达读取数据类型不受支持", false),
        };
    }

    private async Task<HslReadResult> ReadRawAsync(HslDeltaReadRequest request, Func<byte[], object> convert, CancellationToken cancellationToken)
    {
        var words = checked((ushort)Math.Ceiling(request.ElementCount / 2d));
        var result = await _device.ReadAsync(request.Address, words).WaitAsync(cancellationToken);
        return result.IsSuccess ? HslReadResult.Success(convert(result.Content.Take(request.ElementCount).ToArray())) : ReadFailure(result);
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess ? HslReadResult.Success(result.Content) : ReadFailure(result);
    private static HslReadResult ReadFailure(OperateResult result) => HslReadResult.Failure("HSL_DELTA_READ_FAILED", FormatError(result), true);
    private static HslOperationResult Failure(string code, OperateResult result, bool retryable) => HslOperationResult.Failure(code, FormatError(result), retryable);
    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"台达通信操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（通信错误码 {result.ErrorCode}）";
    private static DataFormat MapDataFormat(HslDataFormat value) => value switch { HslDataFormat.ABCD => DataFormat.ABCD, HslDataFormat.BADC => DataFormat.BADC, HslDataFormat.CDAB => DataFormat.CDAB, HslDataFormat.DCBA => DataFormat.DCBA, _ => throw new ArgumentOutOfRangeException(nameof(value)) };
    private static StopBits MapStopBits(HslSerialStopBits value) => value == HslSerialStopBits.Two ? StopBits.Two : StopBits.One;
    private static Parity MapParity(HslSerialParity value) => value switch { HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None };
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
