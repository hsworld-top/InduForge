using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.YASKAWA;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslYaskawaClient : IHslYaskawaClient
{
    private readonly MemobusTcpNet _device;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslYaskawaClient(HslYaskawaClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = options.Transport == HslYaskawaTransport.Udp
            ? new MemobusUdpNet(options.Host, options.Port)
            : new MemobusTcpNet(options.Host, options.Port);
        _device.CpuFrom = options.CpuFrom;
        _device.CpuTo = options.CpuTo;
        _device.ConnectTimeOut = options.ConnectTimeoutMilliseconds;
        _device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
        _device.ByteTransform.DataFormat = options.DataFormat switch
        {
            HslDataFormat.ABCD => DataFormat.ABCD,
            HslDataFormat.BADC => DataFormat.BADC,
            HslDataFormat.CDAB => DataFormat.CDAB,
            HslDataFormat.DCBA => DataFormat.DCBA,
            _ => throw new ArgumentOutOfRangeException(nameof(options)),
        };
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
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_YASKAWA_CONNECT_FAILED", exception.Message, true); }
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
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_YASKAWA_CLOSE_FAILED", exception.Message, false); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslYaskawaReadRequest request, CancellationToken cancellationToken)
    {
        if (request.ElementCount < 1) return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", false);
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "安川 Memobus 连接尚未建立", true);
            return await HslTypedDeviceReader.ReadAsync(_device, request.Address, request.DataType, request.ElementCount, "HSL_YASKAWA_READ_FAILED", "安川 Memobus", cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_YASKAWA_READ_FAILED", exception.Message, true); }
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

    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) =>
        HslOperationResult.Failure($"HSL_YASKAWA_{operation}_FAILED", string.IsNullOrWhiteSpace(result.Message) ? $"安川 Memobus 操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）", retryable);

    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
