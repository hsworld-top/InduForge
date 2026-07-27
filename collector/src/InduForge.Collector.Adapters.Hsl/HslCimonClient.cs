using HslCommunication;
using HslCommunication.Profinet.Cimon;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslCimonClient : IHslCimonClient
{
    private readonly CimonHmiProtocol _client;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslCimonClient(HslCimonClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _client = new CimonHmiProtocol(options.Host, options.Port) { FrameNo = options.FrameNumber, ConnectTimeOut = options.ConnectTimeoutMilliseconds, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds };
    }

    public bool IsConnected => _connected && !_disposed;
    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { await _operationGate.WaitAsync(cancellationToken); try { ThrowIfDisposed(); if (_connected) return HslOperationResult.Success(); var result = await _client.ConnectServerAsync().WaitAsync(cancellationToken); _connected = result.IsSuccess; return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true); } catch (OperationCanceledException) { throw; } catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_CIMON_CONNECT_FAILED", exception.Message, true); } finally { _operationGate.Release(); } }
    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { await _operationGate.WaitAsync(cancellationToken); try { if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); } var result = await _client.ConnectCloseAsync().WaitAsync(cancellationToken); _connected = false; return result.IsSuccess ? HslOperationResult.Success() : Failure("CLOSE", result, false); } catch (OperationCanceledException) { throw; } catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_CIMON_CLOSE_FAILED", exception.Message, false); } finally { _operationGate.Release(); } }
    public async Task<HslReadResult> ReadAsync(HslCimonReadRequest request, CancellationToken cancellationToken) { if (request.ElementCount < 1) return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", false); await _operationGate.WaitAsync(cancellationToken); try { ThrowIfDisposed(); if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "Cimon HMI 连接尚未建立", true); return await HslTypedDeviceReader.ReadAsync(_client, request.Address, request.DataType, request.ElementCount, "HSL_CIMON_READ_FAILED", "Cimon HMI", cancellationToken); } catch (OperationCanceledException) { throw; } catch (Exception exception) { return HslReadResult.Failure("HSL_CIMON_READ_FAILED", exception.Message, true); } finally { _operationGate.Release(); } }
    public async ValueTask DisposeAsync() { if (_disposed) return; await CloseAsync(CancellationToken.None); _disposed = true; _client.Dispose(); _operationGate.Dispose(); }
    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure($"HSL_CIMON_{operation}_FAILED", string.IsNullOrWhiteSpace(result.Message) ? $"Cimon HMI 操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）", retryable);
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
