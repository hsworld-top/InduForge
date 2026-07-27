using System.IO.Ports;
using HslCommunication;
using HslCommunication.Core.Device;
using HslCommunication.Instrument.RKC;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslRkcClient : IHslRkcClient
{
    private readonly DeviceCommunication _device;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslRkcClient(HslRkcClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = options.Transport == HslInstrumentTransport.Serial ? CreateSerial(options) : CreateTcp(options);
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
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_RKC_CONNECT_FAILED", exception.Message, true);
        }
        finally { _operationGate.Release(); }
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
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_RKC_CLOSE_FAILED", exception.Message, false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslRkcReadRequest request, CancellationToken cancellationToken)
    {
        if (request.DataType != HslValueType.DoublePrecision || request.ElementCount != 1)
        {
            return HslReadResult.Failure("HSL_RKC_READ_SHAPE_INVALID", "RKC 温控器仅支持读取单个 float64 数值", false);
        }

        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "RKC 温控器连接尚未建立", true);

            var result = await _device.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false);
            return result.IsSuccess
                ? HslReadResult.Success(result.Content)
                : HslReadResult.Failure("HSL_RKC_READ_FAILED", Format(result), true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_RKC_READ_FAILED", exception.Message, true); }
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

    private static TemperatureController CreateSerial(HslRkcClientOptions options)
    {
        if (string.IsNullOrWhiteSpace(options.PortName)) throw new ArgumentException("RKC 温控器串口名称不能为空", nameof(options));
        var device = new TemperatureController { Station = options.Station, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds };
        device.SerialPortInni(options.PortName, options.BaudRate, options.DataBits, StopBitsOf(options.StopBits), ParityOf(options.Parity));
        return device;
    }

    private static TemperatureControllerOverTcp CreateTcp(HslRkcClientOptions options) => new(options.Host, options.Port)
    {
        Station = options.Station,
        ConnectTimeOut = options.ConnectTimeoutMilliseconds,
        ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
    };

    private static StopBits StopBitsOf(HslSerialStopBits value) => value == HslSerialStopBits.Two ? StopBits.Two : StopBits.One;
    private static Parity ParityOf(HslSerialParity value) => value switch { HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None };
    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) =>
        HslOperationResult.Failure($"HSL_RKC_{operation}_FAILED", string.IsNullOrWhiteSpace(result.Message) ? $"RKC 温控器操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）", retryable);
    private static string Format(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"RKC 温控器读取失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
