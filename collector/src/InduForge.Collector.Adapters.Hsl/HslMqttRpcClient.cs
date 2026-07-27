using HslCommunication;
using HslCommunication.MQTT;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslMqttRpcClient : IHslMqttRpcClient
{
    private readonly MqttRpcDevice _device;
    private readonly SemaphoreSlim _gate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslMqttRpcClient(HslMqttRpcClientOptions options)
    {
        var connection = new MqttConnectionOptions { IpAddress = options.Host, Port = options.Port, ClientId = options.ClientId, UseRSAProvider = options.UseRsa };
        if (!string.IsNullOrEmpty(options.UserName) || !string.IsNullOrEmpty(options.Password)) connection.Credentials = new MqttCredential(options.UserName ?? string.Empty, options.Password ?? string.Empty);
        _device = new MqttRpcDevice(connection, options.DeviceTopic);
    }

    public bool IsConnected => _connected && !_disposed;
    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try { ThrowIfDisposed(); if (_connected) return HslOperationResult.Success(); var result = await _device.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false); _connected = result.IsSuccess; return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true); }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_MQTT_RPC_CONNECT_FAILED", exception.Message, true); }
        finally { _gate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try { if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); } var result = _device.ConnectClose(); _connected = false; return result.IsSuccess ? HslOperationResult.Success() : Failure("CLOSE", result, false); }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_MQTT_RPC_CLOSE_FAILED", exception.Message, false); }
        finally { _gate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslMqttRpcReadRequest request, CancellationToken cancellationToken)
    {
        if (request.ElementCount is < 1 or > ushort.MaxValue) return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须在 1 到 65535 之间", false);
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "MQTT RPC 设备尚未连接", true);
            var count = checked((ushort)request.ElementCount);
            return request.DataType switch
            {
                HslValueType.Boolean => Result(await _device.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
                HslValueType.Signed16 => Result(await _device.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
                HslValueType.Unsigned16 => Result(await _device.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
                HslValueType.Signed32 => Result(await _device.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
                HslValueType.Unsigned32 => Result(await _device.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
                HslValueType.Signed64 => Result(await _device.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
                HslValueType.Unsigned64 => Result(await _device.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
                HslValueType.SinglePrecision => Result(await _device.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
                HslValueType.DoublePrecision => Result(await _device.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false), request.ElementCount),
                HslValueType.Text => Result(await _device.ReadStringAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
                HslValueType.Binary => Result(await _device.ReadAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
                _ => HslReadResult.Failure("HSL_MQTT_RPC_DATA_TYPE_UNSUPPORTED", "MQTT RPC 数据类型不受支持", false),
            };
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_MQTT_RPC_READ_FAILED", exception.Message, true); }
        finally { _gate.Release(); }
    }

    public async ValueTask DisposeAsync() { if (_disposed) return; await CloseAsync(CancellationToken.None).ConfigureAwait(false); _disposed = true; _gate.Dispose(); }
    private static HslReadResult Result<T>(OperateResult<T[]> result, int count) => result.IsSuccess ? HslReadResult.Success(count == 1 ? result.Content[0] : result.Content) : HslReadResult.Failure("HSL_MQTT_RPC_READ_FAILED", result.Message, true);
    private static HslReadResult Result<T>(OperateResult<T> result) => result.IsSuccess ? HslReadResult.Success(result.Content) : HslReadResult.Failure("HSL_MQTT_RPC_READ_FAILED", result.Message, true);
    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure($"HSL_MQTT_RPC_{operation}_FAILED", result.Message, retryable);
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
