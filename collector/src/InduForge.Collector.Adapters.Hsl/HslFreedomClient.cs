using System.IO.Ports;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.Freedom;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslFreedomClient : IHslFreedomClient
{
    private readonly DeviceCommunication _device;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslFreedomClient(HslFreedomClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = CreateDevice(options);
        _device.ByteTransform.DataFormat = MapDataFormat(options.DataFormat);
        _device.ByteTransform.IsStringReverseByteWord = options.StringReverse;
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
                _ => throw new InvalidOperationException("未知的自由协议通信设备类型"),
            };
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_FREEDOM_CONNECT_FAILED", exception.Message, true); }
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
            else throw new InvalidOperationException("未知的自由协议通信设备类型");
            _connected = false;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CLOSE", result, false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_FREEDOM_CLOSE_FAILED", exception.Message, false); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslFreedomReadRequest request, CancellationToken cancellationToken)
    {
        if (request.ElementCount < 1) return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", false);
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "自由协议连接尚未建立", true);
            return await HslTypedDeviceReader.ReadAsync(_device, request.Address, request.DataType, request.ElementCount, "HSL_FREEDOM_READ_FAILED", "自由协议", cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_FREEDOM_READ_FAILED", exception.Message, true); }
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

    private static DeviceCommunication CreateDevice(HslFreedomClientOptions options) => options.Transport switch
    {
        HslFreedomTransport.Tcp => new FreedomTcpNet(options.Host, options.Port) { ConnectTimeOut = options.ConnectTimeoutMilliseconds, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds },
        HslFreedomTransport.Udp => new FreedomUdpNet(options.Host, options.Port) { ConnectTimeOut = options.ConnectTimeoutMilliseconds, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds },
        HslFreedomTransport.Serial => ConfigureSerial(new FreedomSerial { ReceiveTimeOut = options.ReceiveTimeoutMilliseconds, RtsEnable = options.RtsEnable }, options),
        _ => throw new ArgumentOutOfRangeException(nameof(options)),
    };

    private static FreedomSerial ConfigureSerial(FreedomSerial device, HslFreedomClientOptions options)
    {
        if (string.IsNullOrWhiteSpace(options.PortName)) throw new ArgumentException("自由协议串口名称不能为空", nameof(options));
        device.SerialPortInni(options.PortName, options.BaudRate, options.DataBits, options.StopBits == HslSerialStopBits.Two ? StopBits.Two : StopBits.One, options.Parity switch { HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None });
        return device;
    }

    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure($"HSL_FREEDOM_{operation}_FAILED", FormatError(result), retryable);
    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"自由协议操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    private static DataFormat MapDataFormat(HslDataFormat value) => value switch { HslDataFormat.ABCD => DataFormat.ABCD, HslDataFormat.BADC => DataFormat.BADC, HslDataFormat.CDAB => DataFormat.CDAB, HslDataFormat.DCBA => DataFormat.DCBA, _ => throw new ArgumentOutOfRangeException(nameof(value)) };
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
