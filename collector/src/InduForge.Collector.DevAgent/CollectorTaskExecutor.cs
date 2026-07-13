using System.Text.Json;
using InduForge.Collector.Contracts;
using InduForge.Collector.Drivers.OpcUa;

namespace InduForge.Collector.DevAgent;

internal sealed class CollectorTaskExecutor
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly OpcUaDriver _opcUaDriver = new();

    public async Task<CollectorTaskCompletion> ExecuteAsync(CollectorTaskEnvelope task, CancellationToken cancellationToken)
    {
        try
        {
            var request = task.Request.Deserialize<CollectorTaskRequest>(JsonOptions) ?? throw new InvalidOperationException("任务请求为空");
            var profile = CreateProfile(request.Connection);
            object result = task.Operation switch
            {
                "connection.test" => await _opcUaDriver.TestConnectionAsync(profile, cancellationToken).ConfigureAwait(false),
                "opcua.browse" => await _opcUaDriver.BrowseAsync(profile, CreateBrowseRequest(request.Input), cancellationToken).ConfigureAwait(false),
                "opcua.read" => await _opcUaDriver.ReadAsync(profile, CreateReadRequest(request.Input), cancellationToken).ConfigureAwait(false),
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
        if (!string.Equals(connection.ProtocolType, "opcua", StringComparison.OrdinalIgnoreCase))
        {
            throw new CollectorTaskExecutionException("COLLECTOR_PROTOCOL_UNSUPPORTED", "当前 Agent 不支持该协议", false);
        }
        if (!string.Equals(connection.Authentication.Type, "anonymous", StringComparison.OrdinalIgnoreCase))
        {
            throw new CollectorTaskExecutionException("COLLECTOR_AUTH_UNSUPPORTED", "当前仅支持匿名认证", false);
        }
        return new ConnectionProfile("opcua", connection.EndpointUrl, connection.SecurityMode, connection.SecurityPolicy, new ConnectionAuthentication(AuthenticationType.Anonymous), TimeSpan.FromSeconds(30));
    }

    private static BrowseRequest CreateBrowseRequest(JsonElement input) => new(
        input.TryGetProperty("parentNodeId", out var parentNodeId) ? parentNodeId.GetString() ?? "ns=0;i=85" : "ns=0;i=85",
        input.TryGetProperty("maxDepth", out var maxDepth) ? maxDepth.GetInt32() : 1);

    private static ReadRequest CreateReadRequest(JsonElement input)
    {
        if (!input.TryGetProperty("nodeIds", out var nodeIds) || nodeIds.ValueKind != JsonValueKind.Array)
        {
            throw new CollectorTaskExecutionException("OPCUA_NODE_IDS_REQUIRED", "OPC UA Read 任务缺少 nodeIds", false);
        }
        return new ReadRequest(nodeIds.EnumerateArray().Select(item => item.GetString() ?? string.Empty).Where(item => item.Length > 0).ToArray());
    }

    private static CollectorTaskCompletion Failed(string code, string message, bool retryable) => new("failed", null, new CollectorTaskFailure(code, message, retryable));
}

internal sealed record CollectorTaskRequest(CollectorTaskConnection Connection, JsonElement Input);
internal sealed record CollectorTaskConnection(string ProtocolType, string EndpointUrl, string SecurityMode, string SecurityPolicy, CollectorTaskAuthentication Authentication);
internal sealed record CollectorTaskAuthentication(string Type);
internal sealed class CollectorTaskExecutionException(string code, string message, bool retryable) : Exception(message)
{
    public string Code { get; } = code;
    public bool Retryable { get; } = retryable;
}

