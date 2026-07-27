using System.Net;
using System.Net.Http.Headers;
using System.Net.Http.Json;
using System.Diagnostics;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace InduForge.Collector.DevAgent;

internal sealed class CenterApiClient(HttpClient httpClient, AgentFileLogger? logger = null)
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);

    public async Task<AgentRegistration> RegisterAsync(string centerUrl, AgentRegistrationRequest request, CancellationToken cancellationToken)
    {
        using var message = new HttpRequestMessage(HttpMethod.Post, BuildUrl(centerUrl, "/api/v1/data/collector-dev/agent/register"))
        {
            Content = JsonContent.Create(request, options: JsonOptions),
        };
        using var response = await SendAsync(message, cancellationToken).ConfigureAwait(false);
        return await ReadDataAsync<AgentRegistration>(response, cancellationToken).ConfigureAwait(false);
    }

    public async Task DisconnectAsync(AgentCredentials credentials, CancellationToken cancellationToken)
    {
        await SendAgentCommandAsync(credentials, "/api/v1/data/collector-dev/agent/disconnect", cancellationToken).ConfigureAwait(false);
    }

    public async Task RevokeAsync(AgentCredentials credentials, CancellationToken cancellationToken)
    {
        await SendAgentCommandAsync(credentials, "/api/v1/data/collector-dev/agent/revoke", cancellationToken).ConfigureAwait(false);
    }

    public async Task HeartbeatAsync(AgentCredentials credentials, AgentHeartbeatRequest request, CancellationToken cancellationToken)
    {
        using var message = CreateAgentRequest(credentials, HttpMethod.Post, "/api/v1/data/collector-dev/agent/heartbeat", request);
        using var response = await SendAsync(message, cancellationToken).ConfigureAwait(false);
        _ = await ReadDataAsync<JsonElement>(response, cancellationToken).ConfigureAwait(false);
    }

    public async Task<CollectorTaskEnvelope?> ClaimTaskAsync(AgentCredentials credentials, CancellationToken cancellationToken)
    {
        using var message = CreateAgentRequest(credentials, HttpMethod.Post, "/api/v1/data/collector-dev/agent/tasks/claim?waitSeconds=10", new { });
        using var response = await SendAsync(message, cancellationToken, false).ConfigureAwait(false);
        return await ReadOptionalDataAsync<CollectorTaskEnvelope>(response, cancellationToken).ConfigureAwait(false);
    }

    public async Task CompleteTaskAsync(AgentCredentials credentials, string taskId, CollectorTaskCompletion completion, CancellationToken cancellationToken)
    {
        using var message = CreateAgentRequest(credentials, HttpMethod.Post, $"/api/v1/data/collector-dev/agent/tasks/{taskId}/complete", completion);
        using var response = await SendAsync(message, cancellationToken).ConfigureAwait(false);
        _ = await ReadDataAsync<JsonElement>(response, cancellationToken).ConfigureAwait(false);
    }

    private async Task SendAgentCommandAsync(AgentCredentials credentials, string path, CancellationToken cancellationToken)
    {
        using var message = CreateAgentRequest(credentials, HttpMethod.Post, path, new { });
        using var response = await SendAsync(message, cancellationToken).ConfigureAwait(false);
        _ = await ReadDataAsync<JsonElement>(response, cancellationToken).ConfigureAwait(false);
    }

    private async Task<HttpResponseMessage> SendAsync(HttpRequestMessage message, CancellationToken cancellationToken, bool logFailure = true)
    {
        var startedAt = Stopwatch.GetTimestamp();
        var requestPath = message.RequestUri?.AbsolutePath ?? "unknown";
        try
        {
            var response = await httpClient.SendAsync(message, cancellationToken).ConfigureAwait(false);
            if (logFailure && !response.IsSuccessStatusCode)
            {
                logger?.Warn("center.http.failed", $"method={message.Method} path={requestPath} status={(int)response.StatusCode} elapsedMs={Stopwatch.GetElapsedTime(startedAt).TotalMilliseconds:F0}");
            }
            return response;
        }
        catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
        {
            throw;
        }
        catch (Exception exception)
        {
            if (logFailure)
            {
                logger?.Error("center.http.exception", $"method={message.Method} path={requestPath} elapsedMs={Stopwatch.GetElapsedTime(startedAt).TotalMilliseconds:F0}", exception);
            }
            throw;
        }
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
internal sealed record AgentRegistrationRequest(string RegistrationCode, string MachineId, string Name, string OS, string Arch, string Version, IReadOnlyList<AgentProtocolCapability> Capabilities);
internal sealed record AgentRegistration(string AgentId, string AgentToken, string TenantId);
internal sealed record AgentProtocolCapability(
    string DriverId,
    string DriverVersion,
    IReadOnlyList<int> SchemaVersions,
    IReadOnlyList<string> Operations,
    AgentProtocolResources? Resources = null);
internal sealed record AgentProtocolResources(IReadOnlyList<string> SerialPorts);
internal sealed record AgentHeartbeatRequest(IReadOnlyList<AgentProtocolCapability> Capabilities);
internal sealed record AgentCredentials(string CenterUrl, string AgentId, string AgentToken, string TenantId, string AgentName);
internal sealed record CollectorTaskEnvelope(string TaskId, string ProjectId, string AgentId, string Operation, string Status, JsonElement Request, string DeadlineAt);
internal sealed record CollectorTaskCompletion(string Status, object? Result, CollectorTaskFailure? Error);
internal sealed record CollectorTaskFailure(string Code, string Message, bool Retryable);
