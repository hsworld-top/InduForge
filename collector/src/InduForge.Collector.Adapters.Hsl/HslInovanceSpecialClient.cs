using System.IO.Ports;
using System.Text;
using HslCommunication;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.Inovance;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslInovanceSpecialClient : IHslInovanceSpecialClient
{
    private readonly DeviceCommunication _device;
    private readonly InovanceConnectedCipNet? _cipDevice;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslInovanceSpecialClient(HslInovanceSpecialClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = CreateDevice(options);
        _cipDevice = _device as InovanceConnectedCipNet;
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
            return HslOperationResult.Failure("HSL_INOVANCE_CONNECT_FAILED", exception.Message, true);
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
            return HslOperationResult.Failure("HSL_INOVANCE_CLOSE_FAILED", exception.Message, false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslInovanceSpecialReadRequest request, CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "汇川通信连接尚未建立", true);
            var count = checked((ushort)request.ElementCount);
            return request.DataType switch
            {
                HslValueType.Boolean => request.ElementCount == 1 ? Map(await _device.ReadBoolAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Signed16 => request.ElementCount == 1 ? Map(await _device.ReadInt16Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Unsigned16 => request.ElementCount == 1 ? Map(await _device.ReadUInt16Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Signed32 => request.ElementCount == 1 ? Map(await _device.ReadInt32Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Unsigned32 => request.ElementCount == 1 ? Map(await _device.ReadUInt32Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Signed64 => request.ElementCount == 1 ? Map(await _device.ReadInt64Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Unsigned64 => request.ElementCount == 1 ? Map(await _device.ReadUInt64Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.SinglePrecision => request.ElementCount == 1 ? Map(await _device.ReadFloatAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.DoublePrecision => request.ElementCount == 1 ? Map(await _device.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Text => Map(await _device.ReadStringAsync(request.Address, count, Encoding.ASCII).WaitAsync(cancellationToken)),
                HslValueType.Binary => await ReadRawAsync(request, cancellationToken),
                HslValueType.DateTime when _cipDevice is not null => Map(await _cipDevice.ReadDateAsync(request.Address).WaitAsync(cancellationToken)),
                _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "当前汇川协议不支持该数据类型", false),
            };
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_INOVANCE_READ_FAILED", exception.Message, true); }
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

    private async Task<HslReadResult> ReadRawAsync(HslInovanceSpecialReadRequest request, CancellationToken cancellationToken)
    {
        var result = await _device.ReadAsync(request.Address, checked((ushort)request.ElementCount)).WaitAsync(cancellationToken);
        return result.IsSuccess ? HslReadResult.Success(result.Content) : HslReadResult.Failure("HSL_INOVANCE_READ_FAILED", FormatError(result), true);
    }

    private static DeviceCommunication CreateDevice(HslInovanceSpecialClientOptions options)
    {
        return options.Protocol switch
        {
            HslInovanceSpecialProtocol.ConnectedCip => new InovanceConnectedCipNet(options.Host, options.Port)
            {
                ConnectTimeOut = options.ConnectTimeoutMilliseconds,
                ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
                TOConnectionId = options.ToConnectionId,
                OTConnectionId = options.OtConnectionId,
            },
            HslInovanceSpecialProtocol.EasyNet => new InovanceEasyNet(options.Host, options.Port)
            {
                ConnectTimeOut = options.ConnectTimeoutMilliseconds,
                ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
            },
            HslInovanceSpecialProtocol.ComputerLink => CreateComputerLink(options),
            _ => throw new ArgumentOutOfRangeException(nameof(options), "不支持的汇川专用协议"),
        };
    }

    private static InovanceComputerLink CreateComputerLink(HslInovanceSpecialClientOptions options)
    {
        if (string.IsNullOrWhiteSpace(options.PortName)) throw new ArgumentException("汇川 ComputerLink 串口名称不能为空", nameof(options));
        var device = new InovanceComputerLink
        {
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
            Station = options.Station,
            WaittingTime = options.WaitingTime,
            SumCheck = options.SumCheck,
            Format = options.Format,
        };
        device.SerialPortInni(options.PortName, options.BaudRate, options.DataBits, MapStopBits(options.StopBits), MapParity(options.Parity));
        return device;
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess ? HslReadResult.Success(result.Content) : HslReadResult.Failure("HSL_INOVANCE_READ_FAILED", FormatError(result), true);
    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure($"HSL_INOVANCE_{operation}_FAILED", FormatError(result), retryable);
    private static StopBits MapStopBits(HslSerialStopBits value) => value == HslSerialStopBits.Two ? StopBits.Two : StopBits.One;
    private static Parity MapParity(HslSerialParity value) => value switch { HslSerialParity.None => Parity.None, HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None };
    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"汇川操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
