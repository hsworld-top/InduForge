using System.Text;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Profinet.Yokogawa;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslYokogawaClient : IHslYokogawaClient
{
    private readonly YokogawaLinkTcp _device;
    private readonly SemaphoreSlim _gate = new(1, 1);
    private bool _connected;
    private bool _disposed;
    public HslYokogawaClient(HslYokogawaClientOptions options)
    {
        _device = new YokogawaLinkTcp(options.Host, options.Port) { CpuNumber = options.CpuNumber, ConnectTimeOut = options.ConnectTimeoutMilliseconds, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds };
        _device.ByteTransform.DataFormat = MapFormat(options.DataFormat);
    }
    public bool IsConnected => _connected && !_disposed;
    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false); try { ThrowIfDisposed(); if (_connected) return HslOperationResult.Success(); var r = await _device.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false); _connected = r.IsSuccess; return r.IsSuccess ? HslOperationResult.Success() : Fail("HSL_YOKOGAWA_CONNECT_FAILED", r, true); } catch (OperationCanceledException) { throw; } catch (Exception e) { _connected = false; return HslOperationResult.Failure("HSL_YOKOGAWA_CONNECT_FAILED", e.Message, true); } finally { _gate.Release(); }
    }
    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false); try { if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); } var r = await _device.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false); _connected = false; return r.IsSuccess ? HslOperationResult.Success() : Fail("HSL_YOKOGAWA_CLOSE_FAILED", r, false); } catch (OperationCanceledException) { throw; } catch (Exception e) { _connected = false; return HslOperationResult.Failure("HSL_YOKOGAWA_CLOSE_FAILED", e.Message, false); } finally { _gate.Release(); }
    }
    public async Task<HslReadResult> ReadAsync(HslYokogawaReadRequest request, CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false); try { ThrowIfDisposed(); if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "横河通信连接尚未建立", true); return await ReadCoreAsync(request, cancellationToken).ConfigureAwait(false); } catch (OperationCanceledException) { throw; } catch (Exception e) { return HslReadResult.Failure("HSL_YOKOGAWA_READ_FAILED", e.Message, true); } finally { _gate.Release(); }
    }
    public async ValueTask DisposeAsync() { if (_disposed) return; if (_connected) await CloseAsync(CancellationToken.None).ConfigureAwait(false); _disposed = true; _device.Dispose(); _gate.Dispose(); }
    private async Task<HslReadResult> ReadCoreAsync(HslYokogawaReadRequest request, CancellationToken cancellationToken)
    {
        var count = checked((ushort)request.ElementCount); return request.DataType switch
        {
            HslValueType.Boolean => request.ElementCount == 1 ? Map(await _device.ReadBoolAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed8 => await Raw(request, b => request.ElementCount == 1 ? unchecked((sbyte)b[0]) : b.Select(x => unchecked((sbyte)x)).ToArray(), cancellationToken),
            HslValueType.Unsigned8 => await Raw(request, b => request.ElementCount == 1 ? b[0] : b, cancellationToken),
            HslValueType.Signed16 => request.ElementCount == 1 ? Map(await _device.ReadInt16Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned16 => request.ElementCount == 1 ? Map(await _device.ReadUInt16Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed32 => request.ElementCount == 1 ? Map(await _device.ReadInt32Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned32 => request.ElementCount == 1 ? Map(await _device.ReadUInt32Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed64 => request.ElementCount == 1 ? Map(await _device.ReadInt64Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned64 => request.ElementCount == 1 ? Map(await _device.ReadUInt64Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.SinglePrecision => request.ElementCount == 1 ? Map(await _device.ReadFloatAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.DoublePrecision => request.ElementCount == 1 ? Map(await _device.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Text => Map(await _device.ReadStringAsync(request.Address, count, Encoding.UTF8).WaitAsync(cancellationToken)),
            HslValueType.Binary => await Raw(request, b => b, cancellationToken),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "横河读取数据类型不受支持", false),
        };
    }
    private async Task<HslReadResult> Raw(HslYokogawaReadRequest request, Func<byte[], object> convert, CancellationToken token) { var words = checked((ushort)Math.Ceiling(request.ElementCount / 2d)); var r = await _device.ReadAsync(request.Address, words).WaitAsync(token); return r.IsSuccess ? HslReadResult.Success(convert(r.Content.Take(request.ElementCount).ToArray())) : ReadFail(r); }
    private static HslReadResult Map<T>(OperateResult<T> r) => r.IsSuccess ? HslReadResult.Success(r.Content) : ReadFail(r);
    private static HslReadResult ReadFail(OperateResult r) => HslReadResult.Failure("HSL_YOKOGAWA_READ_FAILED", Message(r), true);
    private static HslOperationResult Fail(string code, OperateResult r, bool retryable) => HslOperationResult.Failure(code, Message(r), retryable);
    private static string Message(OperateResult r) => string.IsNullOrWhiteSpace(r.Message) ? $"横河通信操作失败，错误码 {r.ErrorCode}" : $"{r.Message}（通信错误码 {r.ErrorCode}）";
    private static DataFormat MapFormat(HslDataFormat v) => v switch { HslDataFormat.ABCD => DataFormat.ABCD, HslDataFormat.BADC => DataFormat.BADC, HslDataFormat.CDAB => DataFormat.CDAB, HslDataFormat.DCBA => DataFormat.DCBA, _ => throw new ArgumentOutOfRangeException(nameof(v)) };
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
