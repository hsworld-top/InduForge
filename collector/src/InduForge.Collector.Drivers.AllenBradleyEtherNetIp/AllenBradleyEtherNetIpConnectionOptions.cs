using System.Globalization;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.AllenBradleyEtherNetIp;

internal sealed record AllenBradleyEtherNetIpConnectionOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte Slot,
    string? MessageRouter,
    bool ContextCheck,
    bool ReadArrayUseSegment,
    HslDataFormat DataFormat)
{
    public static AllenBradleyEtherNetIpConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "allen-bradley", StringComparison.OrdinalIgnoreCase))
        {
            throw Invalid("ALLEN_BRADLEY_PROTOCOL_MISMATCH", "连接配置不是 Allen-Bradley EtherNet/IP 协议");
        }
        if (profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("ALLEN_BRADLEY_CONFIG_INVALID", "Allen-Bradley EtherNet/IP 连接配置无效");
        }

        var host = OptionalString(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
        {
            throw Invalid("ALLEN_BRADLEY_HOST_INVALID", "Allen-Bradley 设备 IP / 主机名无效");
        }
        var port = OptionalInt32(profile.Config, "port") ?? 44818;
        if (port is < 1 or > 65535)
        {
            throw Invalid("ALLEN_BRADLEY_PORT_INVALID", "Allen-Bradley 端口必须在 1 到 65535 之间");
        }
        var slot = OptionalInt32(profile.Config, "slot") ?? 0;
        if (slot is < 0 or > 255)
        {
            throw Invalid("ALLEN_BRADLEY_SLOT_INVALID", "Allen-Bradley 槽位号必须在 0 到 255 之间");
        }

        return new AllenBradleyEtherNetIpConnectionOptions(
            host,
            port,
            Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000),
            checked((byte)slot),
            NormalizeMessageRouter(OptionalString(profile.Config, "messageRouter")),
            OptionalBoolean(profile.Config, "contextCheck") ?? false,
            OptionalBoolean(profile.Config, "readArrayUseSegment") ?? true,
            ParseDataFormat(OptionalString(profile.Config, "dataFormat") ?? "DCBA"));
    }

    public HslAllenBradleyClientOptions ToHslOptions() => new(
        Host,
        Port,
        ConnectTimeoutMilliseconds,
        ReceiveTimeoutMilliseconds,
        Slot,
        MessageRouter,
        ContextCheck,
        ReadArrayUseSegment,
        DataFormat);

    private static string? NormalizeMessageRouter(string? value)
    {
        var route = value?.Trim();
        if (string.IsNullOrEmpty(route)) return null;

        var parts = route.Split('.', StringSplitOptions.None);
        if (parts.Length < 2 || parts.Length % 2 != 0 || parts.Any(part =>
                !byte.TryParse(part, NumberStyles.None, CultureInfo.InvariantCulture, out _)))
        {
            throw Invalid("ALLEN_BRADLEY_MESSAGE_ROUTER_INVALID", "Allen-Bradley 消息路由必须是偶数个 0 到 255 数字段，例如 1.15.2.18.1.12");
        }
        return route;
    }

    private static int Timeout(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 100 or > 120000)
        {
            throw Invalid("ALLEN_BRADLEY_TIMEOUT_INVALID", $"Allen-Bradley {name} 必须在 100 到 120000 毫秒之间");
        }
        return value;
    }

    private static HslDataFormat ParseDataFormat(string value) => value switch
    {
        "ABCD" => HslDataFormat.ABCD,
        "BADC" => HslDataFormat.BADC,
        "CDAB" => HslDataFormat.CDAB,
        "DCBA" => HslDataFormat.DCBA,
        _ => throw Invalid("ALLEN_BRADLEY_DATA_FORMAT_INVALID", "Allen-Bradley 数据格式无效"),
    };

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result)
            ? result
            : null;

    private static bool? OptionalBoolean(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False
            ? value.GetBoolean()
            : null;

    private static AllenBradleyEtherNetIpDriverException Invalid(string code, string message) => new(code, message, retryable: false);
}
