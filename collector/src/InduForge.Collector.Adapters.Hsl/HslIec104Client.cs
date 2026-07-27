using System.Collections.Concurrent;
using HslCommunication;
using HslCommunication.Instrument.IEC;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslIec104Client : IHslIec104Client
{
    private readonly IEC104 _client;
    private readonly HslIec104ClientOptions _options;
    private readonly ConcurrentDictionary<CacheKey, Snapshot> _snapshots = new();
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private readonly object _interrogationSync = new();
    private TaskCompletionSource<bool>? _interrogationCompletion;
    private bool _connected;
    private bool _disposed;
    private string? _lastMessageError;

    public HslIec104Client(HslIec104ClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _options = options;
        _client = new IEC104(options.Host, options.Port)
        {
            ConnectTimeOut = options.ConnectTimeoutMilliseconds,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
            Station = options.CommonAddress,
        };
        _client.OnIEC104MessageReceived += OnMessageReceived;
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
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_IEC104_CONNECT_FAILED", FormatError(result), retryable: true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_IEC104_CONNECT_FAILED", exception.Message, retryable: true);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected)
            {
                _connected = false;
                return HslOperationResult.Success();
            }

            var result = await _client.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = false;
            CompleteInterrogation();
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_IEC104_CLOSE_FAILED", FormatError(result), retryable: false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            CompleteInterrogation();
            return HslOperationResult.Failure("HSL_IEC104_CLOSE_FAILED", exception.Message, retryable: false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<IReadOnlyList<HslIec104PointReadResult>> ReadAsync(
        IReadOnlyList<HslIec104ReadRequest> requests,
        CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(requests);
        if (requests.Count == 0) return [];

        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected)
            {
                return FailureForAll(requests, "HSL_NOT_CONNECTED", "IEC 104 通信连接尚未建立", retryable: true);
            }

            ClearStationSnapshots(_options.CommonAddress);
            _lastMessageError = null;
            var completion = new TaskCompletionSource<bool>(TaskCreationOptions.RunContinuationsAsynchronously);
            lock (_interrogationSync) _interrogationCompletion = completion;

            var sendResult = _client.TotalSubscriptions(
                _options.CommonAddress,
                _options.InterrogationCode,
                _options.InterrogationReason);
            if (!sendResult.IsSuccess)
            {
                ClearInterrogation(completion);
                return FailureForAll(requests, "HSL_IEC104_INTERROGATION_FAILED", FormatError(sendResult), retryable: true);
            }

            var timedOut = false;
            try
            {
                await completion.Task
                    .WaitAsync(TimeSpan.FromMilliseconds(_options.InterrogationTimeoutMilliseconds), cancellationToken)
                    .ConfigureAwait(false);
            }
            catch (TimeoutException)
            {
                timedOut = true;
            }
            finally
            {
                ClearInterrogation(completion);
            }

            return requests.Select(request => MapSnapshot(request, timedOut)).ToArray();
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            ClearInterrogation();
            return FailureForAll(requests, "HSL_IEC104_READ_FAILED", exception.Message, retryable: true);
        }
        finally { _operationGate.Release(); }
    }

    public async ValueTask DisposeAsync()
    {
        if (_disposed) return;
        await CloseAsync(CancellationToken.None).ConfigureAwait(false);
        _disposed = true;
        _client.OnIEC104MessageReceived -= OnMessageReceived;
        _client.Dispose();
        _operationGate.Dispose();
    }

    internal static HslIec104InformationType? ResolveInformationType(byte typeId) => typeId switch
    {
        IEC104.TypeID.M_SP_NA_1 or IEC104.TypeID.M_SP_TB_1 => HslIec104InformationType.SinglePoint,
        IEC104.TypeID.M_DP_NA_1 or IEC104.TypeID.M_DP_TB_1 => HslIec104InformationType.DoublePoint,
        IEC104.TypeID.M_ME_NA_1 or IEC104.TypeID.M_ME_ND_1 or IEC104.TypeID.M_ME_TD_1 => HslIec104InformationType.NormalizedMeasured,
        IEC104.TypeID.M_ME_NB_1 or IEC104.TypeID.M_ME_TE_1 => HslIec104InformationType.ScaledMeasured,
        IEC104.TypeID.M_ME_NC_1 or IEC104.TypeID.M_ME_TF_1 => HslIec104InformationType.ShortFloatMeasured,
        IEC104.TypeID.M_BO_NA_1 or IEC104.TypeID.M_BO_TB_1 => HslIec104InformationType.BitString32,
        IEC104.TypeID.M_IT_NA_1 or IEC104.TypeID.M_IT_TB_1 => HslIec104InformationType.IntegratedTotal,
        _ => null,
    };

    private void OnMessageReceived(object? sender, IEC104MessageEventArgs args)
    {
        try
        {
            if (args.TypeID == IEC104.TypeID.C_IC_NA_1)
            {
                // 与 HSL Demo 一致：激活确认后清空旧快照，激活终止表示本轮总召唤结束。
                if (args.TransmissionReason == 7) ClearStationSnapshots(args.StationAddress);
                if (args.TransmissionReason == 10) CompleteInterrogation();
                return;
            }

            foreach (var snapshot in ParseSnapshots(args))
            {
                _snapshots[new CacheKey(args.StationAddress, snapshot.InformationType, snapshot.InformationObjectAddress)] = snapshot;
            }
        }
        catch (Exception exception)
        {
            // HSL 在接收线程触发事件，解析异常必须被隔离，避免终止整个代理进程。
            _lastMessageError = exception.Message;
        }
    }

    private static IEnumerable<Snapshot> ParseSnapshots(IEC104MessageEventArgs args)
    {
        var informationType = ResolveInformationType(args.TypeID);
        if (informationType is null) return [];

        return informationType.Value switch
        {
            HslIec104InformationType.SinglePoint =>
                IecValueObject<byte>.ParseYaoXinValue(args).Select(value => Snapshot.From(value, informationType.Value, value.Value != 0)),
            HslIec104InformationType.DoublePoint =>
                IecValueObject<byte>.ParseYaoXinValue(args).Select(value => Snapshot.From(value, informationType.Value, value.Value)),
            HslIec104InformationType.NormalizedMeasured or HslIec104InformationType.ScaledMeasured =>
                IecValueObject<short>.ParseInt16Value(args).Select(value => Snapshot.From(value, informationType.Value, value.Value)),
            HslIec104InformationType.ShortFloatMeasured =>
                IecValueObject<float>.ParseFloatValue(args).Select(value => Snapshot.From(value, informationType.Value, value.Value)),
            HslIec104InformationType.BitString32 or HslIec104InformationType.IntegratedTotal =>
                IecValueObject<uint>.ParseUInt32Value(args).Select(value => Snapshot.From(value, informationType.Value, value.Value)),
            _ => [],
        };
    }

    private HslIec104PointReadResult MapSnapshot(HslIec104ReadRequest request, bool timedOut)
    {
        var key = new CacheKey(_options.CommonAddress, request.InformationType, request.InformationObjectAddress);
        if (_snapshots.TryGetValue(key, out var snapshot))
        {
            return HslIec104PointReadResult.Success(request, snapshot.Value, snapshot.Quality, snapshot.SourceTimestamp);
        }

        var message = _lastMessageError is null
            ? timedOut
                ? "IEC 104 总召唤超时，未收到该信息对象"
                : "IEC 104 总召唤已结束，但未返回该信息对象"
            : $"IEC 104 报文解析失败：{_lastMessageError}";
        return HslIec104PointReadResult.Failure(
            request,
            timedOut ? "HSL_IEC104_INTERROGATION_TIMEOUT" : "HSL_IEC104_POINT_NOT_RECEIVED",
            message,
            retryable: true);
    }

    private void ClearStationSnapshots(int stationAddress)
    {
        foreach (var key in _snapshots.Keys.Where(key => key.StationAddress == stationAddress))
        {
            _snapshots.TryRemove(key, out _);
        }
    }

    private void CompleteInterrogation()
    {
        lock (_interrogationSync) _interrogationCompletion?.TrySetResult(true);
    }

    private void ClearInterrogation(TaskCompletionSource<bool>? expected = null)
    {
        lock (_interrogationSync)
        {
            if (expected is null || ReferenceEquals(_interrogationCompletion, expected)) _interrogationCompletion = null;
        }
    }

    private static HslIec104PointReadResult[] FailureForAll(
        IReadOnlyList<HslIec104ReadRequest> requests,
        string errorCode,
        string errorMessage,
        bool retryable) =>
        requests.Select(request => HslIec104PointReadResult.Failure(request, errorCode, errorMessage, retryable)).ToArray();

    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);

    private static string FormatError(OperateResult result) =>
        string.IsNullOrWhiteSpace(result.Message) ? $"HSL 错误码 {result.ErrorCode}" : result.Message;

    private sealed record CacheKey(int StationAddress, HslIec104InformationType InformationType, int InformationObjectAddress);

    private sealed record Snapshot(
        HslIec104InformationType InformationType,
        int InformationObjectAddress,
        object Value,
        byte Quality,
        DateTimeOffset? SourceTimestamp)
    {
        public static Snapshot From<T>(IecValueObject<T> source, HslIec104InformationType informationType, object value) => new(
            informationType,
            source.Address,
            value,
            source.Quality,
            source.Time == default ? null : new DateTimeOffset(source.Time));
    }
}
