using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MitsubishiMc3ETcp;

internal sealed record MitsubishiMc3ETcpConnectionOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    byte NetworkNumber,
    byte NetworkStationNumber,
    ushort TargetIoStation)
{
    public static MitsubishiMc3ETcpConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "mitsubishi", StringComparison.OrdinalIgnoreCase))
        {
            throw Invalid("MELSEC_MC_PROTOCOL_MISMATCH", "连接配置不是 Mitsubishi MC 协议");
        }
        if (profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("MELSEC_MC_CONFIG_INVALID", "Mitsubishi MC 3E TCP 连接配置无效");
        }

        var host = OptionalString(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
        {
            throw Invalid("MELSEC_MC_HOST_INVALID", "Mitsubishi MC 设备 IP / 主机名无效");
        }
        var port = OptionalInt32(profile.Config, "port") ?? 6000;
        if (port is < 1 or > 65535)
        {
            throw Invalid("MELSEC_MC_PORT_INVALID", "Mitsubishi MC 端口必须在 1 到 65535 之间");
        }
        var timeout = OptionalInt32(profile.Config, "connectTimeoutMs") ?? 5000;
        if (timeout is < 100 or > 120000)
        {
            throw Invalid("MELSEC_MC_TIMEOUT_INVALID", "Mitsubishi MC 连接超时必须在 100 到 120000 毫秒之间");
        }

        return new MitsubishiMc3ETcpConnectionOptions(
            host,
            port,
            timeout,
            RequiredByte(profile.Config, "networkNumber", 0),
            RequiredByte(profile.Config, "networkStationNumber", 0),
            RequiredUInt16(profile.Config, "targetIoStation", 1023));
    }

    public HslMelsecMcClientOptions ToHslOptions() => new(
        Host,
        Port,
        ConnectTimeoutMilliseconds,
        NetworkNumber,
        NetworkStationNumber,
        TargetIoStation);

    private static byte RequiredByte(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 0 or > 255)
        {
            throw Invalid("MELSEC_MC_CONFIG_INVALID", $"Mitsubishi MC {name} 必须在 0 到 255 之间");
        }
        return checked((byte)value);
    }

    private static ushort RequiredUInt16(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 0 or > 65535)
        {
            throw Invalid("MELSEC_MC_CONFIG_INVALID", $"Mitsubishi MC {name} 必须在 0 到 65535 之间");
        }
        return checked((ushort)value);
    }

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result)
            ? result
            : null;

    private static MitsubishiMc3ETcpDriverException Invalid(string code, string message) => new(code, message, retryable: false);
}
