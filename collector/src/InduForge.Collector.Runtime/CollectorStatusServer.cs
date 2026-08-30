using System.Net;
using System.Text;
using System.Text.Json;
using System.Security.Cryptography;

namespace InduForge.Collector.Runtime;

/// <summary>独立的只读状态端点；返回内容只包含稳定状态码，绝不转发驱动异常文本。</summary>
public sealed class CollectorStatusServer : IAsyncDisposable
{
    private readonly HttpListener _listener = new();
    private readonly CollectorLoadedConfiguration _configuration;
    private readonly DurableWal _wal;
    private readonly CollectorOrchestrator _orchestrator;
    private readonly Func<bool>? _upstreamConnected;
    private readonly DateTimeOffset _started = DateTimeOffset.UtcNow;
    private readonly CancellationTokenSource _stopping = new();
    private Task? _task;
    private string? _fatalCode;
    private string _lifecycle = "RUNNING";

    public CollectorStatusServer(string prefix, CollectorLoadedConfiguration configuration, DurableWal wal, CollectorOrchestrator orchestrator, Func<bool>? upstreamConnected = null)
    {
        if (string.IsNullOrWhiteSpace(prefix) || !Uri.TryCreate(prefix, UriKind.Absolute, out var uri) || uri.Scheme != Uri.UriSchemeHttp) throw ConfigurationError.Invalid();
        _listener.Prefixes.Add(uri.AbsoluteUri.EndsWith('/') ? uri.AbsoluteUri : uri.AbsoluteUri + "/");
        _configuration = configuration; _wal = wal; _orchestrator = orchestrator; _upstreamConnected = upstreamConnected;
    }

    public void Start()
    {
        if (_task is not null) return;
        _listener.Start(); _task = RunAsync(_stopping.Token);
    }

    public void MarkFatal(string code) => Interlocked.CompareExchange(ref _fatalCode, code, null);
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
            if (!string.Equals(context.Request.HttpMethod, "GET", StringComparison.Ordinal) || (context.Request.ContentLength64 > 0)) { context.Response.StatusCode = 405; return; }
            var path = context.Request.Url?.AbsolutePath;
            if (path is not "/health" and not "/api/v1/status") { context.Response.StatusCode = 404; return; }
            var status = CreateStatus(); var reqId = Guid.NewGuid().ToString("N");
            var envelope = new { code = 0, msg = "ok", data = status, reqId };
            var body = JsonSerializer.SerializeToUtf8Bytes(envelope);
            context.Response.StatusCode = 200; context.Response.ContentType = "application/json"; context.Response.ContentLength64 = body.Length;
            using var timeout = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken); timeout.CancelAfter(TimeSpan.FromSeconds(5));
            await context.Response.OutputStream.WriteAsync(body, timeout.Token).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { }
        finally { context.Response.Close(); }
    }

    private object CreateStatus()
    {
        var snapshot = _wal.GetSnapshot(); var now = DateTimeOffset.UtcNow; var connections = _orchestrator.Connections; var fatal = Volatile.Read(ref _fatalCode);
        var unavailable = connections.Any(item => item.State == CollectorConnectionState.Unavailable);
        var backpressured = snapshot.IsRawBackpressured || connections.Any(item => item.State == CollectorConnectionState.Backpressured);
        var health = fatal is not null || snapshot.IsCorrupted || unavailable ? "UNAVAILABLE" : backpressured || snapshot.IsDegraded ? "DEGRADED" : "HEALTHY";
        var lastError = fatal ?? connections.Select(item => item.LastErrorCode).FirstOrDefault(code => code is not null) ?? snapshot.LastError;
        var freshness = _orchestrator.LastBusinessEventAt is { } last && now - last < TimeSpan.FromMinutes(2) ? "FRESH" : _orchestrator.LastBusinessEventAt is null ? "UNKNOWN" : "STALE";
        var watermark = snapshot.IsFull ? "FULL" : snapshot.IsDegraded ? "HIGH" : "NORMAL";
        return new
        {
            schemaVersion = "runtime-health-status.v1",
            componentRole = "industrial-collector",
            siteId = "local",
            deploymentId = _configuration.Binding.DeploymentId,
            accountId = _configuration.Binding.AccountId,
            projectId = _configuration.Artifact.ProjectId,
            executionForm = OperatingSystem.IsWindows() ? "native-windows" : "native-linux",
            nodeId = NormalizeNodeId(Environment.MachineName),
            processId = Environment.ProcessId,
            lifecycleState = fatal is null ? Volatile.Read(ref _lifecycle) : "FAILED",
            healthState = health,
            version = _configuration.Artifact.CollectorVersion,
            startedAt = FormatUtc(_started),
            uptimeSeconds = Math.Max(0, (long)(now - _started).TotalSeconds),
            observedAt = FormatUtc(now),
            lastError,
            reasonCode = lastError,
            businessFreshness = new { state = freshness, lastBusinessEventAt = _orchestrator.LastBusinessEventAt is { } value ? FormatUtc(value) : null, evaluatedAt = FormatUtc(now) },
            collector = new
            {
                assignmentId = _configuration.Binding.BindingId,
                assignmentRevision = _configuration.Binding.BindingRevision,
                bindingId = _configuration.Binding.BindingId,
                bindingRevision = _configuration.Binding.BindingRevision,
                artifact = new { artifactId = _configuration.Binding.ArtifactId, artifactRevision = _configuration.Binding.ArtifactRevision, artifactDigest = _configuration.Binding.ArtifactDigest },
                ownership = new { ownerId = _configuration.Binding.OwnerId, epoch = _configuration.Binding.Epoch },
                walBacklogRecords = snapshot.BacklogRecords,
                walBytes = snapshot.BacklogBytes,
                walOldestAgeSeconds = snapshot.OldestAge is { } age ? Math.Max(0, (long)age.TotalSeconds) : 0,
                walWatermark = watermark,
                upstreamStatus = _upstreamConnected?.Invoke() == true ? (snapshot.BacklogRecords == 0 ? "CONNECTED" : "DEGRADED") : "DISCONNECTED",
            },
        };
    }
    private static string FormatUtc(DateTimeOffset value) => value.ToUniversalTime().ToString("yyyy-MM-dd'T'HH:mm:ss.FFFFFFF'Z'", System.Globalization.CultureInfo.InvariantCulture);
    private static string NormalizeNodeId(string value)
    {
        var filtered = new string(value.Where(character => char.IsAsciiLetterOrDigit(character) || character is '.' or '_' or ':' or '-').ToArray());
        if (filtered.Length is > 0 and <= 128 && char.IsAsciiLetterOrDigit(filtered[0])) return filtered;
        return "node-" + Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes(value))).ToLowerInvariant()[..16];
    }
    public async ValueTask DisposeAsync() { _stopping.Cancel(); if (_listener.IsListening) _listener.Stop(); if (_task is not null) await _task.ConfigureAwait(false); _listener.Close(); _stopping.Dispose(); }
}
