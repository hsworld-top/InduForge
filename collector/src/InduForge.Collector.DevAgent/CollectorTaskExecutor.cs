using System.Text.Json;
using InduForge.Collector.Contracts;
using InduForge.Collector.Drivers.OpcUa;

namespace InduForge.Collector.DevAgent;

internal sealed class CollectorTaskExecutor
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly DriverRegistry _registry;

    public CollectorTaskExecutor()
        : this(DriverRegistry.CreateDefault())
    {
    }

    public CollectorTaskExecutor(DriverRegistry registry)
    {
        _registry = registry;
    }

    public async Task<CollectorTaskCompletion> ExecuteAsync(CollectorTaskEnvelope task, CancellationToken cancellationToken)
    {
        try
        {
            var request = task.Request.Deserialize<CollectorTaskRequest>(JsonOptions) ?? throw new InvalidOperationException("任务请求为空");
            var descriptor = _registry.Describe(request.DriverId);
            if (descriptor.DriverVersion != request.DriverVersion || !descriptor.SchemaVersions.Contains(request.SchemaVersion))
            {
                throw new CollectorTaskExecutionException("COLLECTOR_DRIVER_VERSION_UNSUPPORTED", "Agent 驱动版本或 Schema 版本不匹配", false);
            }
            var driver = _registry.Create(request.DriverId);
            var profile = CreateProfile(request.Connection);
            object result = task.Operation switch
            {
                DriverOperations.ConnectionTest => await driver.TestConnectionAsync(profile, cancellationToken).ConfigureAwait(false),
                DriverOperations.DeviceBrowse when driver is IDeviceBrowser browser => await browser.BrowseAsync(profile, CreateBrowseRequest(request.Input), cancellationToken).ConfigureAwait(false),
                DriverOperations.PointRead when driver is IPointReader reader => await reader.ReadAsync(profile, CreateReadRequest(request.Input), cancellationToken).ConfigureAwait(false),
                _ => throw new CollectorTaskExecutionException("COLLECTOR_OPERATION_UNSUPPORTED", "不支持的调试任务操作", false),
            };
            return new CollectorTaskCompletion("succeeded", result, null);
        }
        catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
        {
            throw;
        }
        catch (OpcUaDriverException exception)
        {
            return Failed(exception.Code, exception.Message, exception.Retryable);
        }
        catch (CollectorTaskExecutionException exception)
        {
            return Failed(exception.Code, exception.Message, exception.Retryable);
        }
        catch (Exception)
        {
            return Failed("COLLECTOR_TASK_FAILED", "调试任务执行失败", false);
        }
    }

    private static ConnectionProfile CreateProfile(CollectorTaskConnection connection)
    {
        if (!string.Equals(connection.ProtocolFamily, "opcua", StringComparison.OrdinalIgnoreCase))
        {
            throw new CollectorTaskExecutionException("COLLECTOR_PROTOCOL_UNSUPPORTED", "当前 Agent 不支持该协议", false);
        }
        if (!string.Equals(connection.Config.AuthenticationType, "anonymous", StringComparison.OrdinalIgnoreCase))
        {
            throw new CollectorTaskExecutionException("COLLECTOR_AUTH_UNSUPPORTED", "当前仅支持匿名认证", false);
        }
        return new ConnectionProfile(
            connection.ProtocolFamily,
            connection.Config.EndpointUrl,
            connection.Config.SecurityMode,
            connection.Config.SecurityPolicy,
            new ConnectionAuthentication(AuthenticationType.Anonymous),
            TimeSpan.FromMilliseconds(connection.Config.TimeoutMs ?? 30000));
    }

    private static BrowseRequest CreateBrowseRequest(JsonElement input) => new(
        input.TryGetProperty("parentNodeId", out var parentNodeId) ? parentNodeId.GetString() ?? "ns=0;i=85" : "ns=0;i=85",
        input.TryGetProperty("maxDepth", out var maxDepth) ? maxDepth.GetInt32() : 1);

    private static ReadRequest CreateReadRequest(JsonElement input)
    {
        if (!input.TryGetProperty("points", out var points) || points.ValueKind != JsonValueKind.Array)
        {
            throw new CollectorTaskExecutionException("COLLECTOR_POINTS_REQUIRED", "Read 任务缺少 points", false);
        }
        var nodeIds = points.EnumerateArray().Select(point =>
        {
            if (!point.TryGetProperty("address", out var address))
            {
                throw new CollectorTaskExecutionException("COLLECTOR_POINT_ADDRESS_REQUIRED", "采集点缺少结构化地址", false);
            }
            return OpcUaAddressMapper.ParseNodeId(address);
        }).ToArray();
        return new ReadRequest(nodeIds);
    }

    private static CollectorTaskCompletion Failed(string code, string message, bool retryable) => new("failed", null, new CollectorTaskFailure(code, message, retryable));
}

internal sealed record CollectorTaskRequest(string DriverId, string DriverVersion, int SchemaVersion, CollectorTaskConnection Connection, JsonElement Input);
internal sealed record CollectorTaskConnection(string ProtocolFamily, CollectorTaskConnectionConfig Config, JsonElement Secrets);
internal sealed record CollectorTaskConnectionConfig(string EndpointUrl, string SecurityMode, string SecurityPolicy, string AuthenticationType, int? TimeoutMs);
internal sealed class CollectorTaskExecutionException(string code, string message, bool retryable) : Exception(message)
{
    public string Code { get; } = code;
    public bool Retryable { get; } = retryable;
}

