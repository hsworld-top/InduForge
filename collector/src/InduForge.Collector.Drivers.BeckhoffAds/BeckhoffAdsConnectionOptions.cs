using System.Globalization;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.BeckhoffAds;

internal sealed record BeckhoffAdsConnectionOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    bool UseAutoAmsNetId,
    int AmsPort,
    string? TargetAmsNetId,
    string? SenderAmsNetId,
    bool UseTagCache,
    HslDataFormat DataFormat)
{
    public static BeckhoffAdsConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "beckhoff", StringComparison.OrdinalIgnoreCase))
        {
            throw Invalid("BECKHOFF_ADS_PROTOCOL_MISMATCH", "连接配置不是倍福 ADS 协议");
        }
        if (profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("BECKHOFF_ADS_CONFIG_INVALID", "倍福 ADS 连接配置无效");
        }

        var host = OptionalString(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
        {
            throw Invalid("BECKHOFF_ADS_HOST_INVALID", "倍福 ADS 设备 IP / 主机名无效");
        }
        var port = OptionalInt32(profile.Config, "port") ?? 48898;
        if (port is < 1 or > 65535)
        {
            throw Invalid("BECKHOFF_ADS_PORT_INVALID", "倍福 ADS TCP 路由端口必须在 1 到 65535 之间");
        }
        var amsPort = OptionalInt32(profile.Config, "amsPort") ?? 851;
        if (amsPort is < 1 or > 65535)
        {
            throw Invalid("BECKHOFF_ADS_AMS_PORT_INVALID", "倍福 ADS AMS 端口必须在 1 到 65535 之间");
        }

        var useAutoAmsNetId = OptionalBoolean(profile.Config, "useAutoAmsNetId") ?? true;
        var targetAmsNetId = NormalizeAmsNetId(OptionalString(profile.Config, "targetAmsNetId"), "目标");
        if (!useAutoAmsNetId && targetAmsNetId is null)
        {
            throw Invalid("BECKHOFF_ADS_TARGET_AMS_NET_ID_REQUIRED", "关闭自动 AMS Net ID 后必须配置目标 AMS Net ID");
        }

        return new BeckhoffAdsConnectionOptions(
            host,
            port,
            Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 5000),
            useAutoAmsNetId,
            amsPort,
            targetAmsNetId,
            NormalizeAmsNetId(OptionalString(profile.Config, "senderAmsNetId"), "本机"),
            OptionalBoolean(profile.Config, "useTagCache") ?? true,
            ParseDataFormat(OptionalString(profile.Config, "dataFormat") ?? "DCBA"));
    }

    public HslBeckhoffAdsClientOptions ToHslOptions() => new(
        Host,
        Port,
        ConnectTimeoutMilliseconds,
        ReceiveTimeoutMilliseconds,
        UseAutoAmsNetId,
        AmsPort,
        TargetAmsNetId,
        SenderAmsNetId,
        UseTagCache,
        DataFormat);

    private static string? NormalizeAmsNetId(string? value, string fieldName)
    {
        var netId = value?.Trim();
        if (string.IsNullOrEmpty(netId)) return null;

        var parts = netId.Split('.', StringSplitOptions.None);
        if (parts.Length != 6 || parts.Any(part =>
                !byte.TryParse(part, NumberStyles.None, CultureInfo.InvariantCulture, out _)))
        {
            throw Invalid("BECKHOFF_ADS_AMS_NET_ID_INVALID", $"倍福 ADS {fieldName} AMS Net ID 必须是六段 0 到 255 数字");
        }
        return netId;
    }

    private static int Timeout(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 100 or > 120000)
        {
            throw Invalid("BECKHOFF_ADS_TIMEOUT_INVALID", $"倍福 ADS {name} 必须在 100 到 120000 毫秒之间");
        }
        return value;
    }

    private static HslDataFormat ParseDataFormat(string value) => value switch
    {
        "ABCD" => HslDataFormat.ABCD,
        "BADC" => HslDataFormat.BADC,
        "CDAB" => HslDataFormat.CDAB,
        "DCBA" => HslDataFormat.DCBA,
        _ => throw Invalid("BECKHOFF_ADS_DATA_FORMAT_INVALID", "倍福 ADS 数据格式无效"),
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

    private static BeckhoffAdsDriverException Invalid(string code, string message) => new(code, message, retryable: false);
}
