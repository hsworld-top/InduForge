using System.IO.Ports;
using HslCommunication;
using HslCommunication.Core.Device;
using HslCommunication.Instrument.CJT;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslCjt188Client : IHslCjt188Client
{
    private readonly DeviceCommunication _device;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslCjt188Client(HslCjt188ClientOptions options)
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
            return HslOperationResult.Failure("HSL_CJT188_CONNECT_FAILED", exception.Message, true);
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
            return HslOperationResult.Failure("HSL_CJT188_CLOSE_FAILED", exception.Message, false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslCjt188ReadRequest request, CancellationToken cancellationToken)
    {
        if (request.ElementCount < 1) return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", false);

        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "CJT188 连接尚未建立", true);

            return await HslTypedDeviceReader.ReadAsync(
                _device,
                request.Address,
                request.DataType,
                request.ElementCount,
                "HSL_CJT188_READ_FAILED",
                "CJT188",
                cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_CJT188_READ_FAILED", exception.Message, true); }
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

    private static CJT188 CreateSerial(HslCjt188ClientOptions options)
    {
        if (string.IsNullOrWhiteSpace(options.PortName)) throw new ArgumentException("CJT188 串口名称不能为空", nameof(options));
        var device = new CJT188(options.Station)
        {
            InstrumentType = options.InstrumentType,
            EnableCodeFE = options.EnableCodeFE,
            StationMatch = options.StationMatch,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
        };
        device.SerialPortInni(options.PortName, options.BaudRate, options.DataBits, StopBitsOf(options.StopBits), ParityOf(options.Parity));
        return device;
    }

    private static CJT188OverTcp CreateTcp(HslCjt188ClientOptions options) => new(options.Station)
    {
        IpAddress = options.Host,
        Port = options.Port,
        InstrumentType = options.InstrumentType,
        EnableCodeFE = options.EnableCodeFE,
        StationMatch = options.StationMatch,
        ConnectTimeOut = options.ConnectTimeoutMilliseconds,
        ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
    };

    private static StopBits StopBitsOf(HslSerialStopBits value) => value == HslSerialStopBits.Two ? StopBits.Two : StopBits.One;
    private static Parity ParityOf(HslSerialParity value) => value switch { HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None };
    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) =>
        HslOperationResult.Failure($"HSL_CJT188_{operation}_FAILED", string.IsNullOrWhiteSpace(result.Message) ? $"CJT188 操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）", retryable);
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
