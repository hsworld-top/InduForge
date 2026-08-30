using System.Net;
using System.Text;
using System.Text.Json;
using System.Security.Cryptography;

namespace InduForge.Collector.Runtime;

/// <summary>独立的只读状态端点；返回内容只包含稳定状态码，绝不转发驱动异常文本。</summary>
public sealed class CollectorStatusServer : IAsyncDisposable
{
    private readonly HttpListener _listener = new();
    private readonly string _siteId;
    private readonly CollectorLoadedConfiguration _configuration;
    private readonly DurableWal _wal;
    private readonly CollectorOrchestrator _orchestrator;
    private readonly Func<bool>? _upstreamConnected;
    private readonly DateTimeOffset _started = DateTimeOffset.UtcNow;
    private readonly CancellationTokenSource _stopping = new();
    private Task? _task;
    private string? _fatalCode;
    private string _lifecycle = "STARTING";

    public CollectorStatusServer(string prefix, string siteId, CollectorLoadedConfiguration configuration, DurableWal wal, CollectorOrchestrator orchestrator, Func<bool>? upstreamConnected = null)
    {
        if (string.IsNullOrWhiteSpace(prefix) || !Uri.TryCreate(prefix, UriKind.Absolute, out var uri) || uri.Scheme != Uri.UriSchemeHttp) throw ConfigurationError.Invalid();
        _listener.Prefixes.Add(uri.AbsoluteUri.EndsWith('/') ? uri.AbsoluteUri : uri.AbsoluteUri + "/");
        _siteId = StrictJson.RequireStableId(siteId);
        _configuration = configuration; _wal = wal; _orchestrator = orchestrator; _upstreamConnected = upstreamConnected;
    }

    public void Start()
    {
        if (_task is not null) return;
        _listener.Start(); _task = RunAsync(_stopping.Token);
    }

    public void MarkFatal(string code) => Interlocked.CompareExchange(ref _fatalCode, code, null);
    public void MarkRunning() => Volatile.Write(ref _lifecycle, "RUNNING");
    public void MarkStopping() => Volatile.Write(ref _lifecycle, "STOPPING");
    public void MarkStopped() => Volatile.Write(ref _lifecycle, "STOPPED");

    private async Task RunAsync(CancellationToken cancellationToken)
    {
        while (!cancellationToken.IsCancellationRequested)
        {
            HttpListenerContext context;
            try { context = await _listener.GetContextAsync().WaitAsync(cancellationToken).ConfigureAwait(false); }
            catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested) { break; }
            catch (HttpListenerException) when (cancellationToken.IsCancellationRequested) { break; }
            _ = RespondAsync(context, cancellationToken);
        }
    }

    private async Task RespondAsync(HttpListenerContext context, CancellationToken cancellationToken)
    {
        try
        {
            var path = context.Request.Url?.AbsolutePath;
            if (path is not "/health" and not "/api/v1/status")
            {
                await WriteEnvelopeAsync(context, 404, 404, "NOT_FOUND", null, cancellationToken).ConfigureAwait(false);
                return;
            }
            if (!string.Equals(context.Request.HttpMethod, "GET", StringComparison.Ordinal) || context.Request.HasEntityBody)
            {
                context.Response.AddHeader("Allow", "GET");
                await WriteEnvelopeAsync(context, 405, 405, "METHOD_NOT_ALLOWED", null, cancellationToken).ConfigureAwait(false);
                return;
            }
            if (path == "/health")
            {
                var health = CreateHealth();
                await WriteEnvelopeAsync(context, health.Up ? 200 : 503, health.Up ? 0 : 503, health.Up ? "ok" : "SERVICE_UNAVAILABLE", new { status = health.Status, observedAt = FormatUtc(health.ObservedAt) }, cancellationToken).ConfigureAwait(false);
                return;
            }
            await WriteEnvelopeAsync(context, 200, 0, "ok", CreateStatus(), cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { }
        finally { context.Response.Close(); }
    }

    private static async Task WriteEnvelopeAsync(HttpListenerContext context, int httpStatus, int code, string msg, object? data, CancellationToken cancellationToken)
    {
        var body = JsonSerializer.SerializeToUtf8Bytes(new { code, msg, data, reqId = Guid.NewGuid().ToString("N") });
        context.Response.StatusCode = httpStatus;
        context.Response.ContentType = "application/json";
        context.Response.ContentLength64 = body.Length;
        using var timeout = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken); timeout.CancelAfter(TimeSpan.FromSeconds(5));
        await context.Response.OutputStream.WriteAsync(body, timeout.Token).ConfigureAwait(false);
    }

    private (bool Up, string Status, DateTimeOffset ObservedAt) CreateHealth()
    {
        var fatal = Volatile.Read(ref _fatalCode);
        var unavailable = _orchestrator.Connections.Any(item => item.State == CollectorConnectionState.Unavailable);
        // 探活不读取 backlog 或 payload 汇总，避免高频 /health 触发 WAL 的 O(n) 遍历。
        var up = fatal is null && !unavailable && !_wal.IsUnavailableForHealth() && Volatile.Read(ref _lifecycle) == "RUNNING";
        return (up, up ? "UP" : "DOWN", DateTimeOffset.UtcNow);
    }

    private object CreateStatus()
    {
        var status = CreateStatusSnapshot();
        return new
        {
            schemaVersion = "runtime-health-status.v1",
            componentRole = "industrial-collector",
            siteId = _siteId,
            deploymentId = _configuration.Binding.DeploymentId,
            accountId = _configuration.Binding.AccountId,
            projectId = _configuration.Artifact.ProjectId,
            executionForm = OperatingSystem.IsWindows() ? "native-windows" : "native-linux",
            nodeId = NormalizeNodeId(Environment.MachineName),
            processId = Environment.ProcessId,
            lifecycleState = status.Lifecycle,
            healthState = status.Health,
            version = _configuration.Artifact.CollectorVersion,
            startedAt = FormatUtc(_started),
            uptimeSeconds = Math.Max(0, (long)(status.ObservedAt - _started).TotalSeconds),
            observedAt = FormatUtc(status.ObservedAt),
            lastError = status.LastError,
            reasonCode = status.LastError,
            businessFreshness = new { state = status.Freshness, lastBusinessEventAt = _orchestrator.LastBusinessEventAt is { } value ? FormatUtc(value) : null, evaluatedAt = FormatUtc(status.ObservedAt) },
            collector = new
            {
                assignmentId = _configuration.Binding.BindingId,
                assignmentRevision = _configuration.Binding.BindingRevision,
                bindingId = _configuration.Binding.BindingId,
                bindingRevision = _configuration.Binding.BindingRevision,
                artifact = new { artifactId = _configuration.Binding.ArtifactId, artifactRevision = _configuration.Binding.ArtifactRevision, artifactDigest = _configuration.Binding.ArtifactDigest },
                ownership = new { ownerId = _configuration.Binding.OwnerId, epoch = _configuration.Binding.Epoch },
                walBacklogRecords = status.Snapshot.BacklogRecords,
                walBytes = status.Snapshot.BacklogBytes,
                walOldestAgeSeconds = status.Snapshot.OldestAge is { } age ? Math.Max(0, (long)age.TotalSeconds) : 0,
                walWatermark = status.Watermark,
                upstreamStatus = _upstreamConnected?.Invoke() == true ? (status.Snapshot.BacklogRecords == 0 ? "CONNECTED" : "DEGRADED") : "DISCONNECTED",
            },
        };
    }

    private StatusSnapshot CreateStatusSnapshot()
    {
        var snapshot = _wal.GetSnapshot(); var now = DateTimeOffset.UtcNow; var connections = _orchestrator.Connections; var fatal = Volatile.Read(ref _fatalCode);
        var unavailable = connections.Any(item => item.State == CollectorConnectionState.Unavailable);
        var backpressured = snapshot.IsRawBackpressured || connections.Any(item => item.State == CollectorConnectionState.Backpressured);
        var health = fatal is not null || snapshot.IsCorrupted || unavailable ? "UNAVAILABLE" : backpressured || snapshot.IsDegraded ? "DEGRADED" : "HEALTHY";
        var lastError = fatal ?? connections.Select(item => item.LastErrorCode).FirstOrDefault(code => code is not null) ?? snapshot.LastError;
        var freshness = _orchestrator.LastBusinessEventAt is { } last && now - last < TimeSpan.FromMinutes(2) ? "FRESH" : _orchestrator.LastBusinessEventAt is null ? "UNKNOWN" : "STALE";
        var watermark = snapshot.IsFull ? "FULL" : snapshot.IsDegraded ? "HIGH" : "NORMAL";
        return new StatusSnapshot(snapshot, now, fatal is null ? Volatile.Read(ref _lifecycle) : "FAILED", health, lastError, freshness, watermark);
    }
    private sealed record StatusSnapshot(WalSnapshot Snapshot, DateTimeOffset ObservedAt, string Lifecycle, string Health, string? LastError, string Freshness, string Watermark);
    private static string FormatUtc(DateTimeOffset value) => value.ToUniversalTime().ToString("yyyy-MM-dd'T'HH:mm:ss.FFFFFFF'Z'", System.Globalization.CultureInfo.InvariantCulture);
    private static string NormalizeNodeId(string value)
    {
        var filtered = new string(value.Where(character => char.IsAsciiLetterOrDigit(character) || character is '.' or '_' or ':' or '-').ToArray());
        if (filtered.Length is > 0 and <= 128 && char.IsAsciiLetterOrDigit(filtered[0])) return filtered;
        return "node-" + Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes(value))).ToLowerInvariant()[..16];
    }
    public async ValueTask DisposeAsync() { _stopping.Cancel(); if (_listener.IsListening) _listener.Stop(); if (_task is not null) await _task.ConfigureAwait(false); _listener.Close(); _stopping.Dispose(); }
}
