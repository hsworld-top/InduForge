using System.Net;
using System.Net.Http.Headers;
using System.Net.Http.Json;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace InduForge.Collector.DevAgent;

internal sealed class CenterApiClient(HttpClient httpClient)
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);

    public async Task<AgentRegistration> RegisterAsync(string centerUrl, AgentRegistrationRequest request, CancellationToken cancellationToken)
    {
        using var response = await httpClient.PostAsJsonAsync(BuildUrl(centerUrl, "/api/v1/data/collector-dev/agent/register"), request, JsonOptions, cancellationToken).ConfigureAwait(false);
        return await ReadDataAsync<AgentRegistration>(response, cancellationToken).ConfigureAwait(false);
    }

    public async Task HeartbeatAsync(AgentCredentials credentials, AgentHeartbeatRequest request, CancellationToken cancellationToken)
    {
        using var message = CreateAgentRequest(credentials, HttpMethod.Post, "/api/v1/data/collector-dev/agent/heartbeat", request);
        using var response = await httpClient.SendAsync(message, cancellationToken).ConfigureAwait(false);
        _ = await ReadDataAsync<JsonElement>(response, cancellationToken).ConfigureAwait(false);
    }

    public async Task<CollectorTaskEnvelope?> ClaimTaskAsync(AgentCredentials credentials, CancellationToken cancellationToken)
    {
        using var message = CreateAgentRequest(credentials, HttpMethod.Post, "/api/v1/data/collector-dev/agent/tasks/claim", new { });
        using var response = await httpClient.SendAsync(message, cancellationToken).ConfigureAwait(false);
        return await ReadOptionalDataAsync<CollectorTaskEnvelope>(response, cancellationToken).ConfigureAwait(false);
    }

    public async Task CompleteTaskAsync(AgentCredentials credentials, string taskId, CollectorTaskCompletion completion, CancellationToken cancellationToken)
    {
        using var message = CreateAgentRequest(credentials, HttpMethod.Post, $"/api/v1/data/collector-dev/agent/tasks/{taskId}/complete", completion);
        using var response = await httpClient.SendAsync(message, cancellationToken).ConfigureAwait(false);
        _ = await ReadDataAsync<JsonElement>(response, cancellationToken).ConfigureAwait(false);
    }

    private static HttpRequestMessage CreateAgentRequest(AgentCredentials credentials, HttpMethod method, string path, object body)
    {
        var message = new HttpRequestMessage(method, BuildUrl(credentials.CenterUrl, path));
        message.Headers.Authorization = new AuthenticationHeaderValue("Bearer", credentials.AgentToken);
        message.Content = JsonContent.Create(body, options: JsonOptions);
        return message;
    }

    private static string BuildUrl(string centerUrl, string path) => centerUrl.TrimEnd('/') + path;

    private static async Task<T> ReadDataAsync<T>(HttpResponseMessage response, CancellationToken cancellationToken)
    {
        var envelope = await response.Content.ReadFromJsonAsync<ApiEnvelope<T>>(JsonOptions, cancellationToken).ConfigureAwait(false);
        if (!response.IsSuccessStatusCode || envelope is null || envelope.Code != 0 || envelope.Data is null)
        {
            throw CreateApiException(response.StatusCode, envelope?.Msg);
        }
        return envelope.Data;
    }

    private static async Task<T?> ReadOptionalDataAsync<T>(HttpResponseMessage response, CancellationToken cancellationToken)
    {
        var envelope = await response.Content.ReadFromJsonAsync<ApiEnvelope<T>>(JsonOptions, cancellationToken).ConfigureAwait(false);
        if (!response.IsSuccessStatusCode || envelope is null || envelope.Code != 0)
        {
            throw CreateApiException(response.StatusCode, envelope?.Msg);
        }
        return envelope.Data;
    }

    private static CenterApiException CreateApiException(HttpStatusCode statusCode, string? message) =>
        new(statusCode, string.IsNullOrWhiteSpace(message) ? "中心接口请求失败" : message);
}

internal sealed class CenterApiException(HttpStatusCode statusCode, string message) : Exception(message)
{
    public HttpStatusCode StatusCode { get; } = statusCode;
    public bool IsCredentialInvalid => StatusCode is HttpStatusCode.Unauthorized or HttpStatusCode.Forbidden;
}

internal sealed record ApiEnvelope<T>(int Code, string Msg, T? Data, string ReqId);
internal sealed record AgentRegistrationRequest(string RegistrationCode, string Name, string OS, string Arch, string Version, IReadOnlyList<AgentProtocolCapability> Capabilities);
internal sealed record AgentRegistration(string AgentId, string AgentToken, string TenantId);
internal sealed record AgentProtocolCapability(string ProtocolType, string CapabilityVersion, IReadOnlyList<string> Operations);
internal sealed record AgentHeartbeatRequest(IReadOnlyList<AgentProtocolCapability> Capabilities);
internal sealed record AgentCredentials(string CenterUrl, string AgentId, string AgentToken, string TenantId, string AgentName);
internal sealed record CollectorTaskEnvelope(string TaskId, string ProjectId, string AgentId, string Operation, string Status, JsonElement Request, string DeadlineAt);
internal sealed record CollectorTaskCompletion(string Status, object? Result, CollectorTaskFailure? Error);
internal sealed record CollectorTaskFailure(string Code, string Message, bool Retryable);
