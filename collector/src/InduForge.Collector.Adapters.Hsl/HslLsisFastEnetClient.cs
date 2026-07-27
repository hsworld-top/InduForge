using System.Text;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Profinet.LSIS;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslLsisFastEnetClient : IHslLsisFastEnetClient
{
    private readonly LSFastEnet _client;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslLsisFastEnetClient(HslLsisFastEnetClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _client = new LSFastEnet(options.Host, options.Port)
        {
            ConnectTimeOut = options.ConnectTimeoutMilliseconds,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
            SetCpuType = MapCpuType(options.CpuType),
            CompanyID = options.CompanyId,
            BaseNo = options.BaseNumber,
            SlotNo = options.SlotNumber,
        };
        _client.ByteTransform.DataFormat = MapDataFormat(options.DataFormat);
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();
            var result = await _client.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_LSIS_FAST_ENET_CONNECT_FAILED", exception.Message, true); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); }
            var result = await _client.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = false;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CLOSE", result, false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure("HSL_LSIS_FAST_ENET_CLOSE_FAILED", exception.Message, false); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslLsisFastEnetReadRequest request, CancellationToken cancellationToken)
    {
        if (request.ElementCount < 1) return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", false);
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "LSIS Fast Enet 通信连接尚未建立", true);
            return await ReadCoreAsync(request, cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_LSIS_FAST_ENET_READ_FAILED", exception.Message, true); }
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

    private async Task<HslReadResult> ReadCoreAsync(HslLsisFastEnetReadRequest request, CancellationToken cancellationToken)
    {
        var count = checked((ushort)request.ElementCount);
        return request.DataType switch
        {
            HslValueType.Boolean => request.ElementCount == 1 ? Map(await _client.ReadBoolAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _client.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed8 => await ReadRawAsync(request, bytes => request.ElementCount == 1 ? unchecked((sbyte)bytes[0]) : bytes.Select(value => unchecked((sbyte)value)).ToArray(), cancellationToken),
            HslValueType.Unsigned8 => request.ElementCount == 1 ? Map(await _client.ReadByteAsync(request.Address).WaitAsync(cancellationToken)) : await ReadRawAsync(request, bytes => bytes, cancellationToken),
            HslValueType.Signed16 => request.ElementCount == 1 ? Map(await _client.ReadInt16Async(request.Address).WaitAsync(cancellationToken)) : Map(await _client.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned16 => request.ElementCount == 1 ? Map(await _client.ReadUInt16Async(request.Address).WaitAsync(cancellationToken)) : Map(await _client.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed32 => request.ElementCount == 1 ? Map(await _client.ReadInt32Async(request.Address).WaitAsync(cancellationToken)) : Map(await _client.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned32 => request.ElementCount == 1 ? Map(await _client.ReadUInt32Async(request.Address).WaitAsync(cancellationToken)) : Map(await _client.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed64 => request.ElementCount == 1 ? Map(await _client.ReadInt64Async(request.Address).WaitAsync(cancellationToken)) : Map(await _client.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned64 => request.ElementCount == 1 ? Map(await _client.ReadUInt64Async(request.Address).WaitAsync(cancellationToken)) : Map(await _client.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.SinglePrecision => request.ElementCount == 1 ? Map(await _client.ReadFloatAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _client.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.DoublePrecision => request.ElementCount == 1 ? Map(await _client.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _client.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Text => Map(await _client.ReadStringAsync(request.Address, count, Encoding.UTF8).WaitAsync(cancellationToken)),
            HslValueType.Binary => await ReadRawAsync(request, bytes => bytes, cancellationToken),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "LSIS Fast Enet 数据类型不受支持", false),
        };
    }

    private async Task<HslReadResult> ReadRawAsync(HslLsisFastEnetReadRequest request, Func<byte[], object> convert, CancellationToken cancellationToken)
    {
        var result = await _client.ReadAsync(request.Address, checked((ushort)request.ElementCount)).WaitAsync(cancellationToken);
        return result.IsSuccess ? HslReadResult.Success(convert(result.Content.Take(request.ElementCount).ToArray())) : HslReadResult.Failure("HSL_LSIS_FAST_ENET_READ_FAILED", FormatError(result), true);
    }

    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) =>
        HslOperationResult.Failure($"HSL_LSIS_FAST_ENET_{operation}_FAILED", FormatError(result), retryable);
    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess ? HslReadResult.Success(result.Content) : HslReadResult.Failure("HSL_LSIS_FAST_ENET_READ_FAILED", FormatError(result), true);
    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"LSIS Fast Enet 操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    private static DataFormat MapDataFormat(HslDataFormat format) => format switch { HslDataFormat.ABCD => DataFormat.ABCD, HslDataFormat.BADC => DataFormat.BADC, HslDataFormat.CDAB => DataFormat.CDAB, HslDataFormat.DCBA => DataFormat.DCBA, _ => throw new ArgumentOutOfRangeException(nameof(format)) };
    private static string MapCpuType(HslLsisCpuType cpuType) => cpuType switch
    {
        HslLsisCpuType.XGK => "XGK",
        HslLsisCpuType.XGI => "XGI",
        HslLsisCpuType.XGR => "XGR",
        HslLsisCpuType.XgbMk => "XGB_MK",
        HslLsisCpuType.XgbIec => "XGB_IEC",
        HslLsisCpuType.XGB => "XGB",
        _ => throw new ArgumentOutOfRangeException(nameof(cpuType)),
    };
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
