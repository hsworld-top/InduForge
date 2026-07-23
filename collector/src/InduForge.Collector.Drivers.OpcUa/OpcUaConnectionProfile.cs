using System.Text.Json;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OpcUa;

internal sealed record OpcUaConnectionProfile(string EndpointUrl, TimeSpan Timeout)
{
    public static OpcUaConnectionProfile Parse(ConnectionProfile profile, int defaultTimeoutMilliseconds)
    {
        if (!string.Equals(profile.ProtocolFamily, "opcua", StringComparison.OrdinalIgnoreCase))
        {
            throw new OpcUaDriverException("OPCUA_PROTOCOL_MISMATCH", "连接配置不是 OPC UA 协议", retryable: false);
        }
        if (profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw new OpcUaDriverException("OPCUA_CONFIG_INVALID", "OPC UA 连接配置无效", retryable: false);
        }

        var host = RequiredString(profile.Config, "host", "OPCUA_HOST_REQUIRED", "OPC UA 设备 IP / 主机名不能为空");
        if (host.Contains("://", StringComparison.Ordinal))
        {
            throw new OpcUaDriverException("OPCUA_HOST_INVALID", "OPC UA 设备 IP / 主机名不能包含协议前缀", retryable: false);
        }

        var port = RequiredInt32(profile.Config, "port", "OPCUA_PORT_REQUIRED", "OPC UA 端口不能为空");
        if (port is < 1 or > 65535)
        {
            throw new OpcUaDriverException("OPCUA_PORT_INVALID", "OPC UA 端口必须在 1 到 65535 之间", retryable: false);
        }

        var securityMode = OptionalString(profile.Config, "securityMode") ?? "None";
        var securityPolicy = OptionalString(profile.Config, "securityPolicy") ?? "None";
        if (!string.Equals(securityMode, "None", StringComparison.OrdinalIgnoreCase) ||
            !string.Equals(securityPolicy, "None", StringComparison.OrdinalIgnoreCase))
        {
            throw new OpcUaDriverException(
                "OPCUA_SECURITY_UNSUPPORTED",
                "当前版本只支持 SecurityMode=None 和 SecurityPolicy=None",
                retryable: false);
        }

        var authenticationType = OptionalString(profile.Config, "authenticationType") ?? "anonymous";
        if (!string.Equals(authenticationType, "anonymous", StringComparison.OrdinalIgnoreCase))
        {
            throw new OpcUaDriverException("OPCUA_AUTH_UNSUPPORTED", "当前版本只支持 OPC UA 匿名认证", retryable: false);
        }

        var timeoutMilliseconds = OptionalInt32(profile.Config, "timeoutMs") ?? defaultTimeoutMilliseconds;
        if (timeoutMilliseconds is < 1000 or > 120000)
        {
            throw new OpcUaDriverException("OPCUA_TIMEOUT_INVALID", "OPC UA 通信超时必须在 1000 到 120000 毫秒之间", retryable: false);
        }

        var endpointPath = OptionalString(profile.Config, "endpointPath") ?? "/";
        if (!endpointPath.StartsWith('/')) endpointPath = "/" + endpointPath;
        if (host.StartsWith('[') && host.EndsWith(']')) host = host[1..^1];

        try
        {
            var endpointUrl = new UriBuilder("opc.tcp", host, port, endpointPath).Uri.AbsoluteUri;
            return new OpcUaConnectionProfile(endpointUrl, TimeSpan.FromMilliseconds(timeoutMilliseconds));
        }
        catch (UriFormatException exception)
        {
            throw new OpcUaDriverException("OPCUA_ENDPOINT_INVALID", "OPC UA Endpoint 地址无效", retryable: false, exception);
        }
    }

    private static string RequiredString(JsonElement config, string name, string code, string message)
    {
        var value = OptionalString(config, name);
        return string.IsNullOrWhiteSpace(value)
            ? throw new OpcUaDriverException(code, message, retryable: false)
            : value.Trim();
    }

    private static int RequiredInt32(JsonElement config, string name, string code, string message) =>
        OptionalInt32(config, name) ?? throw new OpcUaDriverException(code, message, retryable: false);

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String
            ? value.GetString()
            : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result)
            ? result
            : null;
}
