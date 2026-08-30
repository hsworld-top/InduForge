using System.Diagnostics;
using System.Globalization;
using System.Text.Json;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Runtime;

public enum CollectorConnectionState { Starting, Connected, Unavailable, Backpressured, Stopped }
public sealed record CollectorConnectionStatus(string ConnectionId, CollectorConnectionState State, string? LastErrorCode);

/// <summary>
/// 以单调时钟编排已启用映射。每个连接只有一个持久 session 和一个串行读循环，避免同一设备的并发轮询。
/// </summary>
public sealed class CollectorOrchestrator : IAsyncDisposable
{
    private readonly CollectorLoadedConfiguration _configuration;
    private readonly CollectorRuntimeHost _host;
    private readonly DurableWal _wal;
    private readonly ICollectorResourceResolver _resources;
    private readonly ICollectorSecretResolver _secrets;
    private readonly Dictionary<string, ConnectionWorker> _workers = new(StringComparer.Ordinal);
    private readonly CancellationTokenSource _stopping = new();
    private Task? _completion;

    public CollectorOrchestrator(CollectorLoadedConfiguration configuration, CollectorRuntimeHost host, DurableWal wal, ICollectorResourceResolver resources, ICollectorSecretResolver secrets)
    {
        _configuration = configuration ?? throw new ArgumentNullException(nameof(configuration));
        _host = host ?? throw new ArgumentNullException(nameof(host)); _wal = wal ?? throw new ArgumentNullException(nameof(wal));
        _resources = resources ?? throw new ArgumentNullException(nameof(resources)); _secrets = secrets ?? throw new ArgumentNullException(nameof(secrets));
        var usedResources = new HashSet<string>(StringComparer.Ordinal);
        foreach (var connection in _configuration.Binding.Connections.Values)
            if (!usedResources.Add(connection.ResourceRef)) throw ConfigurationError.Invalid();
        if (!usedResources.Add(_configuration.Binding.NatsResourceRef)) throw ConfigurationError.Invalid();
    }

    public IReadOnlyList<CollectorConnectionStatus> Connections => _workers.Values.Select(worker => worker.Status).ToArray();
    public DateTimeOffset? LastBusinessEventAt { get; private set; }
    public Task Completion => _completion ?? Task.CompletedTask;

    public Task StartAsync(CancellationToken cancellationToken = default)
    {
        if (_completion is not null) return Task.CompletedTask;
        foreach (var connection in _configuration.Artifact.Connections.Where(item => item.Enabled))
        {
            var binding = _configuration.Binding.Connections[connection.ConnectionId];
            var driver = _host.CreateSessionDriver(new RuntimeDriverConfiguration(connection.DriverId, connection.ProtocolFamily, connection.DriverVersion, connection.SchemaVersion, connection.RequiredOperations));
            var points = _configuration.Artifact.Points.Where(point => point.Enabled && point.ConnectionId == connection.ConnectionId).ToArray();
            _workers.Add(connection.ConnectionId, new ConnectionWorker(this, connection, binding, points, driver));
        }
        _completion = Task.WhenAll(_workers.Values.Select(worker => worker.RunAsync(_stopping.Token, cancellationToken)));
        return Task.CompletedTask;
    }

    public async Task StopAsync(CancellationToken cancellationToken = default)
    {
        _stopping.Cancel();
        if (_completion is not null)
        {
            try { await _completion.WaitAsync(cancellationToken).ConfigureAwait(false); }
            catch (OperationCanceledException) when (_stopping.IsCancellationRequested) { }
        }
        foreach (var worker in _workers.Values) await worker.DisposeAsync().ConfigureAwait(false);
    }

    internal async ValueTask<ConnectionProfile> ResolveProfileAsync(CollectorConnection connection, BindingConnection binding, CancellationToken cancellationToken)
    {
        var resource = await _resources.ResolveResourceAsync(binding.ResourceRef, cancellationToken).ConfigureAwait(false);
        ValidateConnectionResource(connection, resource.Value);
        var secretProperties = new Dictionary<string, JsonElement>(StringComparer.Ordinal);
        foreach (var reference in binding.SecretRefs)
        {
            using var secret = await _secrets.ResolveSecretAsync(reference, cancellationToken).ConfigureAwait(false);
            if (secret.Value.ValueKind != JsonValueKind.Object) throw ConfigurationError.Invalid();
            foreach (var property in secret.Value.EnumerateObject())
            {
                if (!secretProperties.TryAdd(property.Name, property.Value.Clone())) throw ConfigurationError.Invalid();
            }
        }
        using var secretDocument = JsonDocument.Parse(JsonSerializer.Serialize(secretProperties));
        return new ConnectionProfile(connection.ProtocolFamily, resource.Value, secretDocument.RootElement.Clone());
    }

    private static void ValidateConnectionResource(CollectorConnection connection, JsonElement resource)
    {
        StrictJson.RequireObject(resource, "connection resource");
        if (connection.DriverId == "modbus.tcp")
        {
            StrictJson.RequireOnly(resource, "host", "port", "connectTimeoutMs", "receiveTimeoutMs", "dataFormat");
            var host = StrictJson.RequireNonEmptyString(resource, "host"); if (host.Contains("://", StringComparison.Ordinal)) throw ConfigurationError.Invalid();
            _ = StrictJson.RequireInt32(resource, "port", 1, 65535);
            if (resource.TryGetProperty("connectTimeoutMs", out var connect)) _ = StrictJson.RequireInt32Value(connect, 100, 120000);
            if (resource.TryGetProperty("receiveTimeoutMs", out var receive)) _ = StrictJson.RequireInt32Value(receive, 100, 120000);
            if (resource.TryGetProperty("dataFormat", out var format) && (format.ValueKind != JsonValueKind.String || !new[] { "ABCD", "BADC", "CDAB", "DCBA" }.Contains(format.GetString(), StringComparer.Ordinal))) throw ConfigurationError.Invalid();
            return;
        }
        if (connection.DriverId == "opcua.standard")
        {
            StrictJson.RequireOnly(resource, "host", "port", "endpointPath", "securityMode", "securityPolicy", "authenticationType", "timeoutMs");
            var host = StrictJson.RequireNonEmptyString(resource, "host"); if (host.Contains("://", StringComparison.Ordinal)) throw ConfigurationError.Invalid();
            _ = StrictJson.RequireInt32(resource, "port", 1, 65535); _ = StrictJson.RequireNonEmptyString(resource, "endpointPath");
            StrictJson.RequireString(resource, "securityMode", "None"); StrictJson.RequireString(resource, "securityPolicy", "None"); StrictJson.RequireString(resource, "authenticationType", "anonymous");
            if (resource.TryGetProperty("timeoutMs", out var timeout)) _ = StrictJson.RequireInt32Value(timeout, 1000, 120000);
            return;
        }
        throw ConfigurationError.Invalid();
    }

    internal async Task AppendAsync(CollectorPoint point, PendingPointSample sample, WalRawBatchReservation? reservation, CancellationToken cancellationToken)
    {
        var source = NormalizeUtc(sample.SourceTimestamp);
        var server = NormalizeUtc(sample.ServerTimestamp);
        var received = NormalizeUtc(sample.ReceivedAt);
        var binding = _configuration.Binding;
        Func<long, WalAppendRequest> factory = sequence =>
        {
            var eventId = CollectorV1EventValidator.ComputeRawEventId(binding.DeploymentId, point.DatapointId, binding.OwnerId, binding.Epoch, sequence);
            byte[] Build(JsonElement value, string quality) => JsonSerializer.SerializeToUtf8Bytes(new
            {
                schemaVersion = "data.raw.v1",
                subject = "data.raw." + point.DatapointId,
                eventId,
                deploymentId = binding.DeploymentId,
                accountId = binding.AccountId,
                pointId = point.DatapointId,
                ownerId = binding.OwnerId,
                epoch = binding.Epoch,
                sequence,
                value,
                quality,
                sourceTimestamp = source,
                serverTimestamp = server,
                receivedAt = received,
                source = new { collectorId = binding.CollectorId, connectionId = point.ConnectionId, variableId = point.VariableId },
            });
            var payload = Build(sample.Normalized, sample.Quality);
            // 动态值在读前无法完全估算；不截断，保留同一采样时间作为转换失败事实。
            if (payload.Length > DurableWal.MaximumRawEventPayloadBytes) payload = Build(JsonSerializer.SerializeToElement<object?>(null), "bad");
            if (payload.Length > DurableWal.MaximumRawEventPayloadBytes) throw new WalUnavailableException("raw fallback 超出冻结 wire 上限");
            return new WalAppendRequest("data.raw." + point.DatapointId, eventId, payload, binding.OwnerId, binding.Epoch);
        };
        var appended = reservation is null
            ? await _wal.AppendDataAsync(factory, cancellationToken).ConfigureAwait(false)
            : await _wal.AppendReservedDataAsync(reservation, factory, cancellationToken).ConfigureAwait(false);
        if (!appended.Accepted) throw new WalRawBackpressureException();
        LastBusinessEventAt = DateTimeOffset.UtcNow;
    }

    internal static string NormalizeQuality(string? quality, bool succeeded) => !succeeded ? "bad" : quality?.ToLowerInvariant() switch { "good" => "good", "bad" => "bad", _ => "unknown" };
    private static string NormalizeUtc(DateTimeOffset value) => value.ToUniversalTime().ToString("yyyy-MM-dd'T'HH:mm:ss.FFFFFFF'Z'", CultureInfo.InvariantCulture);

    public async ValueTask DisposeAsync() { await StopAsync().ConfigureAwait(false); _stopping.Dispose(); }

    private sealed class ConnectionWorker : IAsyncDisposable
    {
        private readonly CollectorOrchestrator _owner; private readonly CollectorConnection _connection; private readonly BindingConnection _binding; private readonly CollectorPoint[] _points; private readonly IConnectionSessionDriver _driver;
        private readonly Dictionary<string, PointState> _states; private IIndustrialConnectionSession? _session; private CollectorConnectionStatus _status; private readonly Queue<PendingPointSample> _pending = new();
        internal ConnectionWorker(CollectorOrchestrator owner, CollectorConnection connection, BindingConnection binding, CollectorPoint[] points, IConnectionSessionDriver driver)
        { _owner = owner; _connection = connection; _binding = binding; _points = points; _driver = driver; _states = points.ToDictionary(point => point.DatapointId, _ => new PointState(), StringComparer.Ordinal); _status = new(connection.ConnectionId, CollectorConnectionState.Starting, null); }
        internal CollectorConnectionStatus Status => Volatile.Read(ref _status);
        private void SetStatus(CollectorConnectionState state, string? code) => Volatile.Write(ref _status, new CollectorConnectionStatus(_connection.ConnectionId, state, code));

        internal async Task RunAsync(CancellationToken stopping, CancellationToken startup)
        {
            var started = Stopwatch.GetTimestamp(); var retry = TimeSpan.FromMilliseconds(250);
            while (!stopping.IsCancellationRequested)
            {
                try
                {
                    if (_session is null || !_session.IsConnected)
                    {
                        await DisposeSessionAsync().ConfigureAwait(false);
                        var profile = await _owner.ResolveProfileAsync(_connection, _binding, stopping).ConfigureAwait(false);
                        _session = await _driver.OpenSessionAsync(profile, stopping).ConfigureAwait(false);
                        if (_session is not IPointReaderSession) throw new CollectorRuntimeConfigurationException("COLLECTOR_DRIVER_READ_UNAVAILABLE");
                        SetStatus(CollectorConnectionState.Connected, null); retry = TimeSpan.FromMilliseconds(250);
                    }
                    if (_pending.Count > 0)
                    {
                        throw new WalUnavailableException("发现无 WAL reservation 的已观测采样");
                    }
                    if (_owner._wal.GetSnapshot().IsRawBackpressured || _owner._wal.GetSnapshot().IsDegraded)
                    {
                        SetStatus(CollectorConnectionState.Backpressured, "WAL_BACKPRESSURE");
                        await Task.Delay(TimeSpan.FromMilliseconds(100), stopping).ConfigureAwait(false); continue;
                    }
                    if (_points.Length == 0)
                    {
                        // 启用连接可以暂时没有启用点位；维持 session 供状态和后续部署观察，但不进入空批重连循环。
                        await Task.Delay(TimeSpan.FromSeconds(1), stopping).ConfigureAwait(false);
                        continue;
                    }
                    var elapsed = Stopwatch.GetElapsedTime(started);
                    var due = _points.Where(point => elapsed >= _states[point.DatapointId].NextDue).ToArray();
                    if (due.Length == 0)
                    {
                        var next = _points.Min(point => _states[point.DatapointId].NextDue);
                        await Task.Delay(next > elapsed ? next - elapsed : TimeSpan.FromMilliseconds(1), stopping).ConfigureAwait(false); continue;
                    }
                    using var reservation = await _owner._wal.ReserveRawBatchAsync(due.Length, stopping).ConfigureAwait(false);
                    if (reservation is null)
                    {
                        SetStatus(CollectorConnectionState.Backpressured, "WAL_BACKPRESSURE");
                        await Task.Delay(TimeSpan.FromMilliseconds(100), stopping).ConfigureAwait(false);
                        continue;
                    }
                    var chunk = due.Take(reservation.Count).ToArray();
                    foreach (var point in chunk)
                    {
                        var state = _states[point.DatapointId]; var interval = TimeSpan.FromMilliseconds(point.Acquisition.IntervalMs);
                        do { state.NextDue += interval; } while (state.NextDue <= elapsed);
                    }
                    var request = new ReadRequest(chunk.Select(point => new PointReadRequest(point.DatapointId, point.Address, point.DataType, point.ElementCount, point.ReadOptions)).ToArray());
                    var result = await ((IPointReaderSession)_session).ReadAsync(request, stopping).ConfigureAwait(false);
                    var resultByKey = result.Values.GroupBy(value => value.Key, StringComparer.Ordinal).ToDictionary(group => group.Key, group => group.Last(), StringComparer.Ordinal);
                    foreach (var point in chunk)
                    {
                        var value = resultByKey.TryGetValue(point.DatapointId, out var resultValue) ? resultValue : new PointReadValue(point.DatapointId, false, null, null, "unknown", null, null, "READ_MISSING", null);
                        HandlePoint(point, value);
                    }
                    await AppendPendingAsync(reservation, stopping).ConfigureAwait(false);
                }
                catch (WalRawBackpressureException) { SetStatus(CollectorConnectionState.Backpressured, "WAL_BACKPRESSURE"); }
                catch (OperationCanceledException) when (stopping.IsCancellationRequested) { break; }
                catch (Exception exception)
                {
                    if (exception is CollectorRuntimeConfigurationException or WalUnavailableException or WalCorruptionException)
                        throw;
                    var retryable = exception is IndustrialDriverException { Retryable: true } || exception is IOException || exception is TimeoutException;
                    SetStatus(CollectorConnectionState.Unavailable, ErrorCode(exception));
                    await DisposeSessionAsync().ConfigureAwait(false);
                    if (!retryable) { await Task.Delay(TimeSpan.FromSeconds(30), stopping).ConfigureAwait(false); retry = TimeSpan.FromMilliseconds(250); }
                    else { await Task.Delay(Jitter(retry), stopping).ConfigureAwait(false); retry = TimeSpan.FromMilliseconds(Math.Min(retry.TotalMilliseconds * 2, 30_000)); }
                }
            }
            SetStatus(CollectorConnectionState.Stopped, null);
        }

        private void HandlePoint(CollectorPoint point, PointReadValue value)
        {
            var quality = NormalizeQuality(value.Quality, value.Succeeded);
            JsonElement normalized;
            try { normalized = CollectorValueNormalizer.Normalize(value.Value, point.DataType, point.ElementCount); }
            catch (Exception) { normalized = JsonSerializer.SerializeToElement<object?>(null); quality = "bad"; }
            var state = _states[point.DatapointId];
            if (!ShouldPublish(state, point.Acquisition, normalized, quality)) return;
            var received = DateTimeOffset.UtcNow;
            _pending.Enqueue(new PendingPointSample(point, value, normalized.Clone(), quality, value.SourceTimestamp ?? received, value.ServerTimestamp ?? received, received));
        }
        private async Task AppendPendingAsync(WalRawBatchReservation? reservation, CancellationToken cancellationToken)
        {
            while (_pending.Count > 0)
            {
                var pending = _pending.Peek();
                try
                {
                    await _owner.AppendAsync(pending.Point, pending, reservation, cancellationToken).ConfigureAwait(false);
                    var state = _states[pending.Point.DatapointId]; state.Value = pending.Normalized.Clone(); state.Quality = pending.Quality; state.HasValue = true; _pending.Dequeue();
                    SetStatus(CollectorConnectionState.Connected, null);
                }
                catch (WalRawBackpressureException)
                {
                    if (reservation is not null) throw new WalUnavailableException("已授予 WAL reservation 后发生 backpressure");
                    SetStatus(CollectorConnectionState.Backpressured, "WAL_BACKPRESSURE");
                    await Task.Delay(TimeSpan.FromMilliseconds(100), cancellationToken).ConfigureAwait(false);
                    return;
                }
            }
        }
        private static bool ShouldPublish(PointState state, Acquisition acquisition, JsonElement current, string quality)
        {
            if (!state.HasValue || !acquisition.ChangeOnly || !string.Equals(state.Quality, quality, StringComparison.Ordinal)) return true;
            if (quality != "good") return false;
            if (TryDecimal(current, out var now) && TryDecimal(state.Value, out var before)) return decimal.Abs(now - before) > acquisition.Deadband;
            return !JsonElement.DeepEquals(current, state.Value);
        }
        private static bool TryDecimal(JsonElement element, out decimal value)
        {
            value = default;
            return element.ValueKind == JsonValueKind.Number && element.TryGetDecimal(out value);
        }
        private static TimeSpan Jitter(TimeSpan baseDelay) => TimeSpan.FromMilliseconds(baseDelay.TotalMilliseconds * (0.8 + Random.Shared.NextDouble() * 0.4));
        private static string ErrorCode(Exception exception) => exception is IndustrialDriverException driver ? driver.Code : exception is CollectorRuntimeConfigurationException ? "COLLECTOR_CONFIGURATION_INVALID" : exception is WalUnavailableException ? "WAL_UNAVAILABLE" : "CONNECTION_FAILURE";
        private async ValueTask DisposeSessionAsync() { if (_session is not null) { await _session.DisposeAsync().ConfigureAwait(false); _session = null; } }
        public async ValueTask DisposeAsync() => await DisposeSessionAsync().ConfigureAwait(false);
        private sealed class PointState { internal TimeSpan NextDue; internal bool HasValue; internal JsonElement Value; internal string? Quality; }
    }
}

internal sealed record PendingPointSample(CollectorPoint Point, PointReadValue Value, JsonElement Normalized, string Quality, DateTimeOffset SourceTimestamp, DateTimeOffset ServerTimestamp, DateTimeOffset ReceivedAt);

public sealed class WalRawBackpressureException : Exception { internal WalRawBackpressureException() : base("WAL_BACKPRESSURE") { } }

internal static class CollectorValueNormalizer
{
    internal static JsonElement Normalize(object? value, string type, int elementCount = 1)
    {
        ArgumentOutOfRangeException.ThrowIfLessThan(elementCount, 1);
        if (elementCount > 1)
        {
            if (value is string or byte[] || value is not System.Collections.IEnumerable values) throw new InvalidCastException();
            var items = values.Cast<object?>().ToArray();
            if (items.Length != elementCount) throw new InvalidCastException();
            return JsonSerializer.SerializeToElement(items.Select(item => NormalizeScalar(item, type)).ToArray());
        }
        return JsonSerializer.SerializeToElement(NormalizeScalar(value, type));
    }

    private static object? NormalizeScalar(object? value, string type)
    {
        object? normalized = type switch
        {
            "bool" => value is bool b ? b : throw new InvalidCastException(),
            "int8" => Signed(value, sbyte.MinValue, sbyte.MaxValue),
            "uint8" => Unsigned(value, byte.MaxValue),
            "int16" => Signed(value, short.MinValue, short.MaxValue),
            "uint16" => Unsigned(value, ushort.MaxValue),
            "int32" => Signed(value, int.MinValue, int.MaxValue),
            "uint32" => Unsigned(value, uint.MaxValue),
            "int64" => Signed(value, long.MinValue, long.MaxValue),
            "uint64" => Unsigned(value, ulong.MaxValue),
            "float32" => value is float single ? Finite(single) : throw new InvalidCastException(),
            "float64" => value is double number ? Finite(number) : throw new InvalidCastException(),
            "decimal" => DecimalValue(value),
            "string" => value as string ?? throw new InvalidCastException(),
            "bytes" => Convert.ToBase64String(value as byte[] ?? throw new InvalidCastException()),
            "datetime" => value switch { DateTimeOffset dto => dto.ToUniversalTime().ToString("O", CultureInfo.InvariantCulture), DateTime { Kind: DateTimeKind.Utc } dt => new DateTimeOffset(dt).ToString("O", CultureInfo.InvariantCulture), DateTime { Kind: DateTimeKind.Local } dt => new DateTimeOffset(dt).ToUniversalTime().ToString("O", CultureInfo.InvariantCulture), _ => throw new InvalidCastException() },
            _ => throw new InvalidCastException(),
        };
        return normalized;
    }
    private static float Finite(float value) => float.IsFinite(value) ? value : throw new OverflowException();
    private static double Finite(double value) => double.IsFinite(value) ? value : throw new OverflowException();
    private static long Signed(object? value, long minimum, long maximum)
    {
        var number = value switch { sbyte x => x, short x => x, int x => x, long x => x, byte x => x, ushort x => x, uint x => (long)x, _ => throw new InvalidCastException() };
        return number >= minimum && number <= maximum ? number : throw new OverflowException();
    }
    private static ulong Unsigned(object? value, ulong maximum)
    {
        var number = value switch { byte x => (ulong)x, ushort x => x, uint x => x, ulong x => x, sbyte x when x >= 0 => (ulong)x, short x when x >= 0 => (ulong)x, int x when x >= 0 => (ulong)x, long x when x >= 0 => (ulong)x, _ => throw new InvalidCastException() };
        return number <= maximum ? number : throw new OverflowException();
    }
    private static decimal DecimalValue(object? value) => value switch
    {
        decimal number => number,
        sbyte number => number,
        byte number => number,
        short number => number,
        ushort number => number,
        int number => number,
        uint number => number,
        long number => number,
        ulong number => number,
        _ => throw new InvalidCastException(),
    };
}
