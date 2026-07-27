using System.Text;
using HslCommunication;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.Melsec;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslMelsecNetworkClient : IHslMelsecNetworkClient
{
    private readonly DeviceTcpNet _device;
    private readonly MelsecCipNet? _cipDevice;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslMelsecNetworkClient(HslMelsecNetworkClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = CreateDevice(options);
        _cipDevice = _device as MelsecCipNet;
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();
            var result = await _device.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_MELSEC_NETWORK_CONNECT_FAILED", exception.Message, true);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); }
            var result = await _device.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = false;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CLOSE", result, false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_MELSEC_NETWORK_CLOSE_FAILED", exception.Message, false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslMelsecNetworkReadRequest request, CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "三菱网络通信连接尚未建立", true);
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
                HslValueType.Text => Map(await _device.ReadStringAsync(request.Address, count, Encoding.ASCII).WaitAsync(cancellationToken)),
                HslValueType.Binary => await ReadRawAsync(request, bytes => bytes, cancellationToken),
                HslValueType.DateTime when _cipDevice is not null => Map(await _cipDevice.ReadDateAsync(request.Address).WaitAsync(cancellationToken)),
                _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "当前三菱网络协议不支持该数据类型", false),
            };
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_MELSEC_NETWORK_READ_FAILED", exception.Message, true); }
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

    private async Task<HslReadResult> ReadRawAsync(HslMelsecNetworkReadRequest request, Func<byte[], object> convert, CancellationToken cancellationToken)
    {
        var wordCount = checked((ushort)Math.Ceiling(request.ElementCount / 2d));
        var result = await _device.ReadAsync(request.Address, wordCount).WaitAsync(cancellationToken);
        return result.IsSuccess ? HslReadResult.Success(convert(result.Content.Take(request.ElementCount).ToArray())) : HslReadResult.Failure("HSL_MELSEC_NETWORK_READ_FAILED", FormatError(result), true);
    }

    private static DeviceTcpNet CreateDevice(HslMelsecNetworkClientOptions options)
    {
        DeviceTcpNet device = options.Protocol switch
        {
            HslMelsecNetworkProtocol.A1EAsciiTcp => new MelsecA1EAsciiNet(options.Host, options.Port),
            HslMelsecNetworkProtocol.A1EBinaryTcp => new MelsecA1ENet(options.Host, options.Port),
            HslMelsecNetworkProtocol.McAsciiTcp => ConfigureMc(new MelsecMcAsciiNet(options.Host, options.Port), options),
            HslMelsecNetworkProtocol.McAsciiUdp => ConfigureMc(new MelsecMcAsciiUdp(options.Host, options.Port), options),
            HslMelsecNetworkProtocol.McBinaryUdp => ConfigureMc(new MelsecMcUdp(options.Host, options.Port), options),
            HslMelsecNetworkProtocol.McRBinaryTcp => ConfigureMcR(new MelsecMcRNet(options.Host, options.Port), options),
            HslMelsecNetworkProtocol.CipTcp => new MelsecCipNet(options.Host, options.Port),
            _ => throw new ArgumentOutOfRangeException(nameof(options), "不支持的三菱网络协议"),
        };
        device.ConnectTimeOut = options.ConnectTimeoutMilliseconds;
        device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
        device.ByteTransform.IsStringReverseByteWord = options.StringReverse;
        return device;
    }

    private static T ConfigureMc<T>(T device, HslMelsecNetworkClientOptions options) where T : MelsecMcNet
    {
        device.NetworkNumber = options.NetworkNumber;
        device.NetworkStationNumber = options.NetworkStationNumber;
        device.TargetIOStation = options.TargetIoStation;
        device.EnableWriteBitToWordRegister = options.EnableWriteBitToWordRegister;
        return device;
    }

    private static MelsecMcRNet ConfigureMcR(MelsecMcRNet device, HslMelsecNetworkClientOptions options)
    {
        device.NetworkNumber = options.NetworkNumber;
        device.NetworkStationNumber = options.NetworkStationNumber;
        device.TargetIOStation = options.TargetIoStation;
        device.EnableWriteBitToWordRegister = options.EnableWriteBitToWordRegister;
        return device;
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess ? HslReadResult.Success(result.Content) : HslReadResult.Failure("HSL_MELSEC_NETWORK_READ_FAILED", FormatError(result), true);
    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure($"HSL_MELSEC_NETWORK_{operation}_FAILED", FormatError(result), retryable);
    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"三菱网络操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
