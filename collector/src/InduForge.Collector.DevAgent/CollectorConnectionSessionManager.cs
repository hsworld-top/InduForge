using System.Collections.Concurrent;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.DevAgent;

internal sealed class CollectorConnectionSessionManager : IAsyncDisposable
{
    private static readonly TimeSpan DefaultIdleTimeout = TimeSpan.FromMinutes(30);
    private static readonly TimeSpan CleanupInterval = TimeSpan.FromMinutes(1);
    private readonly Dictionary<string, SessionEntry> _sessions = new(StringComparer.Ordinal);
    private readonly SemaphoreSlim _gate = new(1, 1);
    private readonly CancellationTokenSource _cleanupCancellation = new();
    private readonly Task _cleanupTask;
    private readonly TimeSpan _idleTimeout;
    private readonly AgentFileLogger? _logger;

    public CollectorConnectionSessionManager(AgentFileLogger? logger = null, TimeSpan? idleTimeout = null)
    {
        _logger = logger;
        _idleTimeout = idleTimeout ?? DefaultIdleTimeout;
        _cleanupTask = CleanupLoopAsync(_cleanupCancellation.Token);
    }

    public async Task<ConnectionSessionResult> OpenAsync(
        string sessionKey,
        string driverId,
        IConnectionSessionDriver driver,
        ConnectionProfile profile,
        CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_sessions.TryGetValue(sessionKey, out var existing))
            {
                if (existing.DriverId == driverId && existing.Session.IsConnected)
                {
                    existing.Touch();
                    return existing.ToResult();
                }

                _sessions.Remove(sessionKey);
                await DisposeSessionAsync(sessionKey, existing).ConfigureAwait(false);
            }

            var session = await driver.OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
            if (!session.IsConnected)
            {
                await session.DisposeAsync().ConfigureAwait(false);
                throw new CollectorTaskExecutionException(
                    "COLLECTOR_SESSION_CONNECT_FAILED",
                    "工业连接会话未进入已连接状态",
                    true);
            }

            var entry = new SessionEntry(driverId, session, DateTimeOffset.UtcNow);
            _sessions[sessionKey] = entry;
            _logger?.Info("connection.session.opened", $"sessionKey={sessionKey} driverId={driverId}");
            return entry.ToResult();
        }
        finally
        {
            _gate.Release();
        }
    }

    public async Task<IIndustrialConnectionSession> GetAsync(
        string sessionKey,
        CancellationToken cancellationToken)
    {
        var entry = await GetEntryAsync(sessionKey, cancellationToken).ConfigureAwait(false);
        return entry.Session;
    }

    public async Task<BrowseBatchResult> BrowseAsync(
        string sessionKey,
        IReadOnlyList<BrowseRequest> requests,
        bool refreshCache,
        CancellationToken cancellationToken)
    {
        var entry = await GetEntryAsync(sessionKey, cancellationToken).ConfigureAwait(false);
        return await entry.BrowseAsync(requests, refreshCache, cancellationToken).ConfigureAwait(false);
    }

    public async Task<ConnectionSessionCloseResult> CloseAsync(
        string sessionKey,
        CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (!_sessions.Remove(sessionKey, out var entry))
            {
                return new ConnectionSessionCloseResult(false);
            }

            await DisposeSessionAsync(sessionKey, entry).ConfigureAwait(false);
            return new ConnectionSessionCloseResult(true);
        }
        finally
        {
            _gate.Release();
        }
    }

    public async Task CloseAllAsync(CancellationToken cancellationToken = default)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            var sessions = _sessions.ToArray();
            _sessions.Clear();
            foreach (var pair in sessions)
            {
                await DisposeSessionAsync(pair.Key, pair.Value).ConfigureAwait(false);
            }
        }
        finally
        {
            _gate.Release();
        }
    }

    public async ValueTask DisposeAsync()
    {
        _cleanupCancellation.Cancel();
        try
        {
            await _cleanupTask.ConfigureAwait(false);
        }
        catch (OperationCanceledException)
        {
        }
        await CloseAllAsync().ConfigureAwait(false);
        _cleanupCancellation.Dispose();
        _gate.Dispose();
    }

    private async Task<SessionEntry> GetEntryAsync(string sessionKey, CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (!_sessions.TryGetValue(sessionKey, out var entry))
            {
                throw new CollectorTaskExecutionException(
                    "COLLECTOR_SESSION_REQUIRED",
                    "当前工业连接尚未建立调试长连接",
                    false);
            }
            if (!entry.Session.IsConnected)
            {
                _sessions.Remove(sessionKey);
                await DisposeSessionAsync(sessionKey, entry).ConfigureAwait(false);
                throw new CollectorTaskExecutionException(
                    "COLLECTOR_SESSION_DISCONNECTED",
                    "工业连接会话已断开，请重新连接",
                    true);
            }

            entry.Touch();
            return entry;
        }
        finally
        {
            _gate.Release();
        }
    }

    private async Task CleanupLoopAsync(CancellationToken cancellationToken)
    {
        using var timer = new PeriodicTimer(CleanupInterval);
        while (await timer.WaitForNextTickAsync(cancellationToken).ConfigureAwait(false))
        {
            await CleanupIdleSessionsAsync(cancellationToken).ConfigureAwait(false);
        }
    }

    private async Task CleanupIdleSessionsAsync(CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            var expiredAt = DateTimeOffset.UtcNow - _idleTimeout;
            var expired = _sessions
                .Where(pair => pair.Value.LastUsedAt <= expiredAt || !pair.Value.Session.IsConnected)
                .ToArray();
            foreach (var pair in expired)
            {
                _sessions.Remove(pair.Key);
                await DisposeSessionAsync(pair.Key, pair.Value).ConfigureAwait(false);
            }
        }
        finally
        {
            _gate.Release();
        }
    }

    private async Task DisposeSessionAsync(string sessionKey, SessionEntry entry)
    {
        try
        {
            await entry.Session.DisposeAsync().ConfigureAwait(false);
            _logger?.Info("connection.session.closed", $"sessionKey={sessionKey} driverId={entry.DriverId}");
        }
        catch (Exception exception)
        {
            _logger?.Warn("connection.session.close_failed", $"sessionKey={sessionKey} driverId={entry.DriverId}", exception);
        }
        finally
        {
            entry.Dispose();
        }
    }

    private sealed class SessionEntry(
        string driverId,
        IIndustrialConnectionSession session,
        DateTimeOffset connectedAt) : IDisposable
    {
        private const int BrowseConcurrency = 3;
        private readonly ConcurrentDictionary<string, Lazy<Task<BrowseResult>>> _browseCache = new(StringComparer.Ordinal);
        private readonly SemaphoreSlim _browseGate = new(BrowseConcurrency, BrowseConcurrency);

        public string DriverId { get; } = driverId;
        public IIndustrialConnectionSession Session { get; } = session;
        public DateTimeOffset ConnectedAt { get; } = connectedAt;
        public DateTimeOffset LastUsedAt { get; private set; } = connectedAt;

        public void Touch() => LastUsedAt = DateTimeOffset.UtcNow;

        public ConnectionSessionResult ToResult() => new(
            Session.IsConnected,
            Session.ServerName,
            ConnectedAt);

        public async Task<BrowseBatchResult> BrowseAsync(
            IReadOnlyList<BrowseRequest> requests,
            bool refreshCache,
            CancellationToken cancellationToken)
        {
            if (Session is not IDeviceBrowserSession browser)
            {
                throw new CollectorTaskExecutionException(
                    "COLLECTOR_SESSION_OPERATION_UNSUPPORTED",
                    "当前长连接不支持设备浏览",
                    false);
            }
            if (refreshCache)
            {
                _browseCache.Clear();
            }

            var branches = await Task.WhenAll(
                requests.Select(request => BrowseOneAsync(browser, request, cancellationToken))).ConfigureAwait(false);
            return new BrowseBatchResult(
                branches,
                branches.SelectMany(branch => branch.Diagnostics).ToArray());
        }

        public void Dispose()
        {
            _browseCache.Clear();
            _browseGate.Dispose();
        }

        private async Task<BrowseBranchResult> BrowseOneAsync(
            IDeviceBrowserSession browser,
            BrowseRequest request,
            CancellationToken cancellationToken)
        {
            var cacheKey = $"{request.MaxDepth}:{request.ParentNodeId}";
            var pending = _browseCache.GetOrAdd(
                cacheKey,
                _ => new Lazy<Task<BrowseResult>>(
                    () => ExecuteBrowseAsync(browser, request, cancellationToken),
                    LazyThreadSafetyMode.ExecutionAndPublication));
            try
            {
                var result = await pending.Value.WaitAsync(cancellationToken).ConfigureAwait(false);
                return new BrowseBranchResult(request.ParentNodeId, result.Nodes, result.Diagnostics);
            }
            catch
            {
                _browseCache.TryRemove(cacheKey, out _);
                throw;
            }
        }

        private async Task<BrowseResult> ExecuteBrowseAsync(
            IDeviceBrowserSession browser,
            BrowseRequest request,
            CancellationToken cancellationToken)
        {
            await _browseGate.WaitAsync(cancellationToken).ConfigureAwait(false);
            try
            {
                return await browser.BrowseAsync(request, cancellationToken).ConfigureAwait(false);
            }
            finally
            {
                _browseGate.Release();
            }
        }
    }
}
