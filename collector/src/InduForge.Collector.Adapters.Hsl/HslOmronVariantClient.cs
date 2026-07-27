using System.IO.Ports;
using System.Text;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.AllenBradley;
using HslCommunication.Profinet.Omron;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslOmronVariantClient : IHslOmronVariantClient
{
    private readonly DeviceCommunication _device;
    private readonly IReadWriteCip? _cipDevice;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslOmronVariantClient(HslOmronVariantClientOptions options)
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
                _ => throw new InvalidOperationException("未知的欧姆龙通信设备类型"),
            };
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("HSL_OMRON_CONNECT_FAILED", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_OMRON_CONNECT_FAILED", exception.Message, true); }
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
            else throw new InvalidOperationException("未知的欧姆龙通信设备类型");
            _connected = false;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("HSL_OMRON_CLOSE_FAILED", result, false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_OMRON_CLOSE_FAILED", exception.Message, false); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslOmronVariantReadRequest request, CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(request);
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "欧姆龙通信连接尚未建立", true);
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
                HslValueType.DateTime when _cipDevice is not null => Map(await _cipDevice.ReadDateAsync(request.Address).WaitAsync(cancellationToken)),
                _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "当前欧姆龙协议不支持该数据类型", false),
            };
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_OMRON_READ_FAILED", exception.Message, true); }
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

    private static DeviceCommunication CreateDevice(HslOmronVariantClientOptions options)
    {
        DeviceCommunication device = options.Protocol switch
        {
            HslOmronVariantProtocol.Cip => new OmronCipNet(options.Host, options.Port),
            HslOmronVariantProtocol.ConnectedCip => new OmronConnectedCipNet(options.Host, options.Port),
            HslOmronVariantProtocol.HostLinkOverTcp => new OmronHostLinkOverTcp(options.Host, options.Port)
            {
                UnitNumber = options.UnitNumber,
                PlcType = MapPlcType(options.PlcType),
            },
            HslOmronVariantProtocol.HostLink => CreateHostLinkSerial(options),
            HslOmronVariantProtocol.HostLinkCModeOverTcp => new OmronHostLinkCModeOverTcp(options.Host, options.Port) { UnitNumber = options.UnitNumber },
            HslOmronVariantProtocol.HostLinkCMode => CreateCModeSerial(options),
            _ => throw new ArgumentOutOfRangeException(nameof(options), options.Protocol, "不支持的欧姆龙协议"),
        };
        device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
        if (device is DeviceTcpNet tcp) tcp.ConnectTimeOut = options.ConnectTimeoutMilliseconds;
        return device;
    }

    private static OmronHostLink CreateHostLinkSerial(HslOmronVariantClientOptions options)
    {
        if (string.IsNullOrWhiteSpace(options.PortName)) throw new ArgumentException("欧姆龙 HostLink 串口名称不能为空", nameof(options));
        var device = new OmronHostLink { UnitNumber = options.UnitNumber, PlcType = MapPlcType(options.PlcType) };
        InitializeSerialPort(device, options);
        return device;
    }

    private static OmronHostLinkCMode CreateCModeSerial(HslOmronVariantClientOptions options)
    {
        if (string.IsNullOrWhiteSpace(options.PortName)) throw new ArgumentException("欧姆龙 C-Mode 串口名称不能为空", nameof(options));
        var device = new OmronHostLinkCMode { UnitNumber = options.UnitNumber };
        InitializeSerialPort(device, options);
        return device;
    }

    private static void InitializeSerialPort(DeviceSerialPort device, HslOmronVariantClientOptions options)
    {
        device.SerialPortInni(options.PortName, options.BaudRate, options.DataBits,
            options.StopBits == HslSerialStopBits.Two ? StopBits.Two : StopBits.One,
            options.Parity switch { HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None });
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess ? HslReadResult.Success(result.Content) : HslReadResult.Failure("HSL_OMRON_READ_FAILED", FormatError(result), true);
    private static HslOperationResult Failure(string code, OperateResult result, bool retryable) => HslOperationResult.Failure(code, FormatError(result), retryable);
    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"欧姆龙通信失败，错误码 {result.ErrorCode}" : $"{result.Message}（通信错误码 {result.ErrorCode}）";
    private static DataFormat MapDataFormat(HslDataFormat value) => value switch { HslDataFormat.ABCD => DataFormat.ABCD, HslDataFormat.BADC => DataFormat.BADC, HslDataFormat.CDAB => DataFormat.CDAB, HslDataFormat.DCBA => DataFormat.DCBA, _ => throw new ArgumentOutOfRangeException(nameof(value)) };
    private static OmronPlcType MapPlcType(HslOmronPlcType value) => value == HslOmronPlcType.CV ? OmronPlcType.CV : OmronPlcType.CSCJ;
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
