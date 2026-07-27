using System.IO.Ports;
using System.Text;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.Inovance;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslInovanceModbusVariantClient : IHslInovanceModbusVariantClient
{
    private readonly DeviceCommunication _device;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslInovanceModbusVariantClient(HslInovanceModbusVariantClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = CreateDevice(options);
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
                _ => throw new InvalidOperationException("未知的汇川通信设备类型"),
            };
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_INOVANCE_MODBUS_CONNECT_FAILED", exception.Message, true);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); }
            OperateResult result;
            if (_device is DeviceTcpNet tcp) result = await tcp.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            else if (_device is DeviceSerialPort serial) { await Task.Run(serial.Close, cancellationToken).ConfigureAwait(false); result = OperateResult.CreateSuccessResult(); }
            else throw new InvalidOperationException("未知的汇川通信设备类型");
            _connected = false;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CLOSE", result, false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_INOVANCE_MODBUS_CLOSE_FAILED", exception.Message, false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslInovanceModbusVariantReadRequest request, CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "汇川 Modbus 通信连接尚未建立", true);
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
                _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "汇川 Modbus 数据类型不受支持", false),
            };
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_INOVANCE_MODBUS_READ_FAILED", exception.Message, true); }
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

    private async Task<HslReadResult> ReadRawAsync(HslInovanceModbusVariantReadRequest request, Func<byte[], object> convert, CancellationToken cancellationToken)
    {
        var result = await _device.ReadAsync(request.Address, checked((ushort)Math.Ceiling(request.ElementCount / 2d))).WaitAsync(cancellationToken);
        return result.IsSuccess ? HslReadResult.Success(convert(result.Content.Take(request.ElementCount).ToArray())) : HslReadResult.Failure("HSL_INOVANCE_MODBUS_READ_FAILED", FormatError(result), true);
    }

    private static DeviceCommunication CreateDevice(HslInovanceModbusVariantClientOptions options)
    {
        var series = MapSeries(options.Series);
        if (options.Protocol == HslInovanceModbusVariantProtocol.Serial)
        {
            if (string.IsNullOrWhiteSpace(options.PortName)) throw new ArgumentException("汇川串口名称不能为空", nameof(options));
            var serial = new InovanceSerial(series, options.Station)
            {
                ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
                AddressStartWithZero = options.AddressStartWithZero,
                DataFormat = MapDataFormat(options.DataFormat),
                IsStringReverse = options.StringReverse,
            };
            serial.SerialPortInni(options.PortName, options.BaudRate, options.DataBits, MapStopBits(options.StopBits), MapParity(options.Parity));
            return serial;
        }
        return new InovanceSerialOverTcp(series, options.Host, options.Port, options.Station)
        {
            ConnectTimeOut = options.ConnectTimeoutMilliseconds,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
            AddressStartWithZero = options.AddressStartWithZero,
            DataFormat = MapDataFormat(options.DataFormat),
            IsStringReverse = options.StringReverse,
        };
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess ? HslReadResult.Success(result.Content) : HslReadResult.Failure("HSL_INOVANCE_MODBUS_READ_FAILED", FormatError(result), true);
    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure($"HSL_INOVANCE_MODBUS_{operation}_FAILED", FormatError(result), retryable);
    private static InovanceSeries MapSeries(HslInovanceSeries series) => series switch { HslInovanceSeries.AM => InovanceSeries.AM, HslInovanceSeries.H3U => InovanceSeries.H3U, HslInovanceSeries.H5U => InovanceSeries.H5U, HslInovanceSeries.Easy => InovanceSeries.Easy, _ => throw new ArgumentOutOfRangeException(nameof(series)) };
    private static DataFormat MapDataFormat(HslDataFormat format) => format switch { HslDataFormat.ABCD => DataFormat.ABCD, HslDataFormat.BADC => DataFormat.BADC, HslDataFormat.CDAB => DataFormat.CDAB, HslDataFormat.DCBA => DataFormat.DCBA, _ => throw new ArgumentOutOfRangeException(nameof(format)) };
    private static StopBits MapStopBits(HslSerialStopBits value) => value == HslSerialStopBits.Two ? StopBits.Two : StopBits.One;
    private static Parity MapParity(HslSerialParity value) => value switch { HslSerialParity.None => Parity.None, HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None };
    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"汇川 Modbus 操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
