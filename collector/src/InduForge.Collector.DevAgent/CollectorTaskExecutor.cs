using System.Text.Json;
using InduForge.Collector.Contracts;
using InduForge.Collector.Drivers.OpcUa;

namespace InduForge.Collector.DevAgent;

internal sealed class CollectorTaskExecutor : IAsyncDisposable
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly DriverRegistry _registry;
    private readonly CollectorConnectionSessionManager _sessions;
    private readonly AgentFileLogger? _logger;

    public CollectorTaskExecutor()
        : this(DriverRegistry.CreateDefault())
    {
    }

    public CollectorTaskExecutor(
        DriverRegistry registry,
        AgentFileLogger? logger = null,
        CollectorConnectionSessionManager? sessions = null)
    {
        _registry = registry;
        _logger = logger;
        _sessions = sessions ?? new CollectorConnectionSessionManager(logger);
    }

    public async Task<CollectorTaskCompletion> ExecuteAsync(
        CollectorTaskEnvelope task,
        CancellationToken cancellationToken)
    {
        try
        {
            var request = task.Request.Deserialize<CollectorTaskRequest>(JsonOptions)
                ?? throw new InvalidOperationException("任务请求为空");
            var descriptor = _registry.Describe(request.DriverId);
            if (descriptor.DriverVersion != request.DriverVersion ||
                !descriptor.SchemaVersions.Contains(request.SchemaVersion))
            {
                throw new CollectorTaskExecutionException(
                    "COLLECTOR_DRIVER_VERSION_UNSUPPORTED",
                    "Agent 驱动版本或 Schema 版本不匹配",
                    false);
            }

            var driver = _registry.Create(request.DriverId);
            ConnectionProfile? profile = null;
            ConnectionProfile GetProfile() => profile ??= CreateProfile(request.Connection);
            var sessionKey = TryCreateSessionKey(request.ConnectionId, request.Input);
            object result = task.Operation switch
            {
                DriverOperations.ConnectionTest =>
                    await driver.TestConnectionAsync(GetProfile(), cancellationToken).ConfigureAwait(false),
                DriverOperations.ConnectionOpen when driver is IConnectionSessionDriver sessionDriver =>
                    await _sessions.OpenAsync(
                        RequireSessionKey(sessionKey),
                        request.DriverId,
                        sessionDriver,
                        GetProfile(),
                        cancellationToken).ConfigureAwait(false),
                DriverOperations.ConnectionClose =>
                    await _sessions.CloseAsync(RequireSessionKey(sessionKey), cancellationToken).ConfigureAwait(false),
                DriverOperations.DeviceBrowse =>
                    await BrowseAsync(driver, sessionKey, GetProfile(), request.Input, cancellationToken).ConfigureAwait(false),
                DriverOperations.PointRead =>
                    await ReadAsync(driver, sessionKey, GetProfile(), request.Input, cancellationToken).ConfigureAwait(false),
                _ => throw new CollectorTaskExecutionException(
                    "COLLECTOR_OPERATION_UNSUPPORTED",
                    "不支持的调试任务操作",
                    false),
            };
            return new CollectorTaskCompletion("succeeded", result, null);
        }
        catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
        {
            throw;
        }
        catch (OpcUaDriverException exception)
        {
            _logger?.Warn(
                "task.driver.failed",
                $"taskId={task.TaskId} operation={task.Operation} code={exception.Code}",
                exception);
            return Failed(exception.Code, exception.Message, exception.Retryable);
        }
        catch (CollectorTaskExecutionException exception)
        {
            _logger?.Warn(
                "task.validation.failed",
                $"taskId={task.TaskId} operation={task.Operation} code={exception.Code}",
                exception);
            return Failed(exception.Code, exception.Message, exception.Retryable);
        }
        catch (Exception exception)
        {
            _logger?.Error(
                "task.execution.failed",
                $"taskId={task.TaskId} operation={task.Operation}",
                exception);
            return Failed("COLLECTOR_TASK_FAILED", "调试任务执行失败", false);
        }
    }

    public Task CloseAllSessionsAsync(CancellationToken cancellationToken = default) =>
        _sessions.CloseAllAsync(cancellationToken);

    public ValueTask DisposeAsync() => _sessions.DisposeAsync();

    private async Task<object> BrowseAsync(
        IIndustrialDriver driver,
        string? sessionKey,
        ConnectionProfile profile,
        JsonElement input,
        CancellationToken cancellationToken)
    {
        var (requests, isBatch) = CreateBrowseRequests(input);
        var refreshCache = input.TryGetProperty("refreshCache", out var refreshCacheElement) &&
            refreshCacheElement.ValueKind == JsonValueKind.True;
        if (sessionKey is not null)
        {
            var batch = await _sessions.BrowseAsync(
                sessionKey,
                requests,
                refreshCache,
                cancellationToken).ConfigureAwait(false);
            if (isBatch)
            {
                return batch;
            }
            var branch = batch.Branches[0];
            return new BrowseResult(branch.Nodes, branch.Diagnostics);
        }
        if (isBatch)
        {
            throw new CollectorTaskExecutionException(
                "COLLECTOR_SESSION_REQUIRED",
                "批量设备浏览需要先建立调试长连接",
                false);
        }
        if (driver is IDeviceBrowser browser)
        {
            return await browser.BrowseAsync(profile, requests[0], cancellationToken).ConfigureAwait(false);
        }
        throw new CollectorTaskExecutionException(
            "COLLECTOR_OPERATION_UNSUPPORTED",
            "当前驱动不支持设备浏览",
            false);
    }

    private async Task<ReadResult> ReadAsync(
        IIndustrialDriver driver,
        string? sessionKey,
        ConnectionProfile profile,
        JsonElement input,
        CancellationToken cancellationToken)
    {
        var request = CreateReadRequest(input);
        if (sessionKey is not null)
        {
            var session = await _sessions.GetAsync(sessionKey, cancellationToken).ConfigureAwait(false);
            if (session is not IPointReaderSession readerSession)
            {
                throw new CollectorTaskExecutionException(
                    "COLLECTOR_SESSION_OPERATION_UNSUPPORTED",
                    "当前长连接不支持变量读取",
                    false);
            }
            return await readerSession.ReadAsync(request, cancellationToken).ConfigureAwait(false);
        }
        if (driver is IPointReader reader)
        {
            return await reader.ReadAsync(profile, request, cancellationToken).ConfigureAwait(false);
        }
        throw new CollectorTaskExecutionException(
            "COLLECTOR_OPERATION_UNSUPPORTED",
            "当前驱动不支持变量读取",
            false);
    }

    private static ConnectionProfile CreateProfile(CollectorTaskConnection connection)
    {
        if (!string.Equals(connection.ProtocolFamily, "opcua", StringComparison.OrdinalIgnoreCase))
        {
            throw new CollectorTaskExecutionException(
                "COLLECTOR_PROTOCOL_UNSUPPORTED",
                "当前 Agent 不支持该协议",
                false);
        }
        if (!string.Equals(connection.Config.AuthenticationType, "anonymous", StringComparison.OrdinalIgnoreCase))
        {
            throw new CollectorTaskExecutionException(
                "COLLECTOR_AUTH_UNSUPPORTED",
                "当前仅支持匿名认证",
                false);
        }
        return new ConnectionProfile(
            connection.ProtocolFamily,
            OpcUaEndpointBuilder.Build(
                connection.Config.Host,
                connection.Config.Port,
                connection.Config.EndpointPath),
            connection.Config.SecurityMode ?? string.Empty,
            connection.Config.SecurityPolicy ?? string.Empty,
            new ConnectionAuthentication(AuthenticationType.Anonymous),
            TimeSpan.FromMilliseconds(connection.Config.TimeoutMs ?? 30000));
    }

    private static string? TryCreateSessionKey(string connectionId, JsonElement input)
    {
        if (!input.TryGetProperty("workspaceSessionId", out var workspaceSessionIdElement))
        {
            return null;
        }
        var workspaceSessionId = workspaceSessionIdElement.GetString()?.Trim();
        if (!Guid.TryParse(workspaceSessionId, out _) || !Guid.TryParse(connectionId, out _))
        {
            throw new CollectorTaskExecutionException(
                "COLLECTOR_SESSION_ID_INVALID",
                "调试长连接租约标识无效",
                false);
        }
        return $"{workspaceSessionId}:{connectionId}";
    }

    private static string RequireSessionKey(string? sessionKey) => sessionKey
        ?? throw new CollectorTaskExecutionException(
            "COLLECTOR_SESSION_ID_REQUIRED",
            "调试长连接任务缺少 workspaceSessionId",
            false);

    private static (IReadOnlyList<BrowseRequest> Requests, bool IsBatch) CreateBrowseRequests(JsonElement input)
    {
        var maxDepth = input.TryGetProperty("maxDepth", out var maxDepthElement)
            ? maxDepthElement.GetInt32()
            : 1;
        if (input.TryGetProperty("parentNodeIds", out var parentNodeIdsElement))
        {
            if (parentNodeIdsElement.ValueKind != JsonValueKind.Array)
            {
                throw new CollectorTaskExecutionException(
                    "COLLECTOR_BROWSE_PARENT_IDS_INVALID",
                    "parentNodeIds 必须是数组",
                    false);
            }
            var parentNodeIds = parentNodeIdsElement
                .EnumerateArray()
                .Select(element => element.GetString()?.Trim() ?? string.Empty)
                .Where(value => value.Length > 0)
                .Distinct(StringComparer.Ordinal)
                .ToArray();
            if (parentNodeIds.Length is 0 or > 100)
            {
                throw new CollectorTaskExecutionException(
                    "COLLECTOR_BROWSE_PARENT_IDS_INVALID",
                    "parentNodeIds 数量必须在 1 到 100 之间",
                    false);
            }
            return (parentNodeIds.Select(nodeId => new BrowseRequest(nodeId, maxDepth)).ToArray(), true);
        }

        var parentNodeId = input.TryGetProperty("parentNodeId", out var parentNodeIdElement)
            ? parentNodeIdElement.GetString() ?? "ns=0;i=85"
            : "ns=0;i=85";
        return ([new BrowseRequest(parentNodeId, maxDepth)], false);
    }

    private static ReadRequest CreateReadRequest(JsonElement input)
    {
        if (!input.TryGetProperty("points", out var points) || points.ValueKind != JsonValueKind.Array)
        {
            throw new CollectorTaskExecutionException(
                "COLLECTOR_POINTS_REQUIRED",
                "Read 任务缺少 points",
                false);
        }
        var nodeIds = points.EnumerateArray().Select(point =>
        {
            if (!point.TryGetProperty("address", out var address))
            {
                throw new CollectorTaskExecutionException(
                    "COLLECTOR_POINT_ADDRESS_REQUIRED",
                    "采集点缺少结构化地址",
                    false);
            }
            return OpcUaAddressMapper.ParseNodeId(address);
        }).ToArray();
        return new ReadRequest(nodeIds);
    }

    private static CollectorTaskCompletion Failed(string code, string message, bool retryable) =>
        new("failed", null, new CollectorTaskFailure(code, message, retryable));
}

internal sealed record CollectorTaskRequest(
    string ConnectionId,
    string DriverId,
    string DriverVersion,
    int SchemaVersion,
    CollectorTaskConnection Connection,
    JsonElement Input);

internal sealed record CollectorTaskConnection(
    string ProtocolFamily,
    CollectorTaskConnectionConfig Config,
    JsonElement Secrets);

internal sealed record CollectorTaskConnectionConfig(
    string? Host,
    int Port,
    string? EndpointPath,
    string? SecurityMode,
    string? SecurityPolicy,
    string? AuthenticationType,
    int? TimeoutMs);

internal static class OpcUaEndpointBuilder
{
    public static string Build(string? host, int port, string? endpointPath)
    {
        var normalizedHost = host?.Trim() ?? string.Empty;
        if (normalizedHost.Length == 0 || normalizedHost.Contains("://", StringComparison.Ordinal))
        {
            throw new CollectorTaskExecutionException(
                "COLLECTOR_ENDPOINT_HOST_INVALID",
                "OPC UA 设备 IP / 主机名无效",
                false);
        }
        if (port is < 1 or > 65535)
        {
            throw new CollectorTaskExecutionException(
                "COLLECTOR_ENDPOINT_PORT_INVALID",
                "OPC UA 端口必须在 1 到 65535 之间",
                false);
        }
        if (normalizedHost.StartsWith('[') && normalizedHost.EndsWith(']'))
        {
            normalizedHost = normalizedHost[1..^1];
        }

        var normalizedPath = string.IsNullOrWhiteSpace(endpointPath) ? "/" : endpointPath.Trim();
        if (!normalizedPath.StartsWith('/')) normalizedPath = "/" + normalizedPath;
        try
        {
            return new UriBuilder("opc.tcp", normalizedHost, port, normalizedPath).Uri.AbsoluteUri;
        }
        catch (UriFormatException exception)
        {
            throw new CollectorTaskExecutionException(
                "COLLECTOR_ENDPOINT_INVALID",
                $"无法生成 OPC UA 端点地址：{exception.Message}",
                false);
        }
    }
}

internal sealed class CollectorTaskExecutionException(
    string code,
    string message,
    bool retryable) : Exception(message)
{
    public string Code { get; } = code;
    public bool Retryable { get; } = retryable;
}
