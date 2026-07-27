using System.IO.Ports;
using HslCommunication;
using HslCommunication.Profinet.Special;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslEcFanClient : IHslEcFanClient
{
    private readonly EcFanMachine _device;
    private readonly SemaphoreSlim _gate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslEcFanClient(HslEcFanClientOptions options)
    {
        _device = new EcFanMachine { Station = options.Station, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds };
        _device.SerialPortInni(options.PortName, options.BaudRate, options.DataBits, options.StopBits == HslSerialStopBits.Two ? StopBits.Two : StopBits.One, options.Parity switch { HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None });
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try { ThrowIfDisposed(); if (_connected) return HslOperationResult.Success(); var result = await Task.Run(_device.Open, cancellationToken).ConfigureAwait(false); _connected = result.IsSuccess; return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true); }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_EC_FAN_CONNECT_FAILED", exception.Message, true); }
        finally { _gate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try { if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); } await Task.Run(_device.Close, cancellationToken).ConfigureAwait(false); _connected = false; return HslOperationResult.Success(); }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_EC_FAN_CLOSE_FAILED", exception.Message, false); }
        finally { _gate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslEcFanMetric metric, CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "EC 风机串口尚未打开", true);
            var result = await Task.Run(() => metric switch { HslEcFanMetric.SpeedEmergency => _device.ReadSpeedEmergency(), HslEcFanMetric.SpeedMinimum => _device.ReadSpeedMin(), HslEcFanMetric.SpeedMaximum => _device.ReadSpeedMax(), _ => throw new ArgumentOutOfRangeException(nameof(metric)) }, cancellationToken).ConfigureAwait(false);
            return result.IsSuccess ? HslReadResult.Success(result.Content) : HslReadResult.Failure("HSL_EC_FAN_READ_FAILED", result.Message, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_EC_FAN_READ_FAILED", exception.Message, true); }
        finally { _gate.Release(); }
    }

    public async ValueTask DisposeAsync() { if (_disposed) return; await CloseAsync(CancellationToken.None).ConfigureAwait(false); _disposed = true; _device.Dispose(); _gate.Dispose(); }
    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure($"HSL_EC_FAN_{operation}_FAILED", result.Message, retryable);
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
