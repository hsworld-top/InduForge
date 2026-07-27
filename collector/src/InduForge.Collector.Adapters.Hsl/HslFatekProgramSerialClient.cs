using System.IO.Ports;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Profinet.FATEK;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslFatekProgramSerialClient : IHslFatekProgramClient
{
    private readonly FatekProgram _client;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslFatekProgramSerialClient(HslFatekProgramSerialClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        if (string.IsNullOrWhiteSpace(options.PortName))
        {
            throw new ArgumentException("永宏编程口串口名称不能为空", nameof(options));
        }

        _client = new FatekProgram
        {
            Station = options.Station,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
        };
        _client.SerialPortInni(
            options.PortName,
            options.BaudRate,
            options.DataBits,
            options.StopBits == HslSerialStopBits.Two ? StopBits.Two : StopBits.One,
            options.Parity switch
            {
                HslSerialParity.Odd => Parity.Odd,
                HslSerialParity.Even => Parity.Even,
                _ => Parity.None,
            });
        _client.ByteTransform.DataFormat = DataFormat.DCBA;
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();
            var result = await Task.Run(_client.Open, cancellationToken).ConfigureAwait(false);
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_FATEK_PROGRAM_SERIAL_CONNECT_FAILED", exception.Message, true); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); }
            await Task.Run(_client.Close, cancellationToken).ConfigureAwait(false);
            _connected = false;
            return HslOperationResult.Success();
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_FATEK_PROGRAM_SERIAL_CLOSE_FAILED", exception.Message, false); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslFatekProgramReadRequest request, CancellationToken cancellationToken)
    {
        if (request.ElementCount < 1) return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", false);
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "永宏编程口串口通信连接尚未建立", true);
            return await HslFatekProgramReader.ReadAsync(
                _client,
                request,
                "HSL_FATEK_PROGRAM_SERIAL_READ_FAILED",
                "永宏编程口串口",
                cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_FATEK_PROGRAM_SERIAL_READ_FAILED", exception.Message, true); }
        finally { _operationGate.Release(); }
    }

    public async ValueTask DisposeAsync()
    {
        if (_disposed) return;
        await CloseAsync(CancellationToken.None).ConfigureAwait(false);
        _disposed = true;
        _client.Dispose();
        _operationGate.Dispose();
    }

    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) =>
        HslOperationResult.Failure($"HSL_FATEK_PROGRAM_SERIAL_{operation}_FAILED", FormatError(result), retryable);

    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message)
        ? $"永宏编程口串口操作失败，错误码 {result.ErrorCode}"
        : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";

    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
