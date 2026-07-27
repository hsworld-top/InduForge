using System.IO.Ports;
using HslCommunication;
using HslCommunication.Core.Device;
using HslCommunication.Instrument.DLT;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslDltClient : IHslDltClient
{
    private readonly DeviceCommunication _device;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslDltClient(HslDltClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = CreateDevice(options);
        if (_device is DeviceSerialPort serial)
        {
            if (string.IsNullOrWhiteSpace(options.PortName)) throw new ArgumentException("DLT 串口名称不能为空", nameof(options));
            serial.SerialPortInni(options.PortName, options.BaudRate, options.DataBits, StopBitsOf(options.StopBits), ParityOf(options.Parity));
        }
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();
            var result = _device is DeviceSerialPort serial
                ? await Task.Run(serial.Open, cancellationToken).ConfigureAwait(false)
                : await ((DeviceTcpNet)_device).ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_DLT_CONNECT_FAILED", exception.Message, true); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); }
            if (_device is DeviceSerialPort serial)
            {
                await Task.Run(serial.Close, cancellationToken).ConfigureAwait(false);
                _connected = false;
                return HslOperationResult.Success();
            }
            var result = await ((DeviceTcpNet)_device).ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = false;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CLOSE", result, false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_DLT_CLOSE_FAILED", exception.Message, false); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslDltReadRequest request, CancellationToken cancellationToken)
    {
        if (request.ElementCount < 1) return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", false);
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "DLT 仪表连接尚未建立", true);
            return await HslTypedDeviceReader.ReadAsync(_device, request.Address, request.DataType, request.ElementCount, "HSL_DLT_READ_FAILED", "DLT 仪表", cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_DLT_READ_FAILED", exception.Message, true); }
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

    private static DeviceCommunication CreateDevice(HslDltClientOptions options) => options.Protocol switch
    {
        HslDltProtocol.Dlt645Serial => Configure645(new DLT645(options.Station, options.Password, options.OpCode), options),
        HslDltProtocol.Dlt645OverTcp => ConfigureTcp(Configure645(new DLT645OverTcp(options.Host, options.Port, options.Station, options.Password, options.OpCode), options), options),
        HslDltProtocol.Dlt645With1997Serial => Configure645(new DLT645With1997(options.Station), options),
        HslDltProtocol.Dlt645With1997OverTcp => ConfigureTcp(Configure645(new DLT645With1997OverTcp(options.Host, options.Port, options.Station), options), options),
        HslDltProtocol.Dlt698Serial => Configure698(new DLT698(options.Station), options),
        HslDltProtocol.Dlt698OverTcp => ConfigureTcp(Configure698(new DLT698OverTcp(options.Host, options.Port, options.Station), options), options),
        HslDltProtocol.Dlt698TcpNet => ConfigureTcp(Configure698(new DLT698TcpNet(options.Host, options.Port, options.Station), options), options),
        _ => throw new ArgumentOutOfRangeException(nameof(options)),
    };

    private static T Configure645<T>(T device, HslDltClientOptions options) where T : DeviceCommunication
    {
        dynamic dlt = device;
        dlt.EnableCodeFE = options.EnableCodeFE;
        dlt.CheckDataId = options.CheckDataId;
        device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
        return device;
    }

    private static T Configure698<T>(T device, HslDltClientOptions options) where T : DeviceCommunication
    {
        dynamic dlt = device;
        dlt.EnableCodeFE = options.EnableCodeFE;
        dlt.UseSecurityResquest = options.UseSecurityRequest;
        dlt.CA = options.ClientAddress;
        device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
        return device;
    }

    private static T ConfigureTcp<T>(T device, HslDltClientOptions options) where T : DeviceTcpNet
    {
        device.ConnectTimeOut = options.ConnectTimeoutMilliseconds;
        device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
        return device;
    }

    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure($"HSL_DLT_{operation}_FAILED", string.IsNullOrWhiteSpace(result.Message) ? $"DLT 操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）", retryable);
    private static StopBits StopBitsOf(HslSerialStopBits value) => value == HslSerialStopBits.Two ? StopBits.Two : StopBits.One;
    private static Parity ParityOf(HslSerialParity value) => value switch { HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None };
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
