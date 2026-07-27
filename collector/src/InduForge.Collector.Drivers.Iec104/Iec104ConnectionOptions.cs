using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Iec104;

internal sealed record Iec104ConnectionOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    ushort CommonAddress,
    byte InterrogationCode,
    byte InterrogationReason,
    int InterrogationTimeoutMilliseconds)
{
    public static Iec104ConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "iec", StringComparison.OrdinalIgnoreCase))
        {
            throw Invalid("IEC104_PROTOCOL_MISMATCH", "连接配置不是 IEC 60870-5-104 协议");
        }
        if (profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("IEC104_CONFIG_INVALID", "IEC 104 连接配置无效");
        }

        var host = OptionalString(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
        {
            throw Invalid("IEC104_HOST_INVALID", "IEC 104 设备 IP / 主机名无效");
        }
        var port = OptionalInt32(profile.Config, "port") ?? 2404;
        if (port is < 1 or > 65535)
        {
            throw Invalid("IEC104_PORT_INVALID", "IEC 104 端口必须在 1 到 65535 之间");
        }
        var commonAddress = OptionalInt32(profile.Config, "commonAddress") ?? 1;
        if (commonAddress is < 1 or > ushort.MaxValue)
        {
            throw Invalid("IEC104_COMMON_ADDRESS_INVALID", "IEC 104 公共地址必须在 1 到 65535 之间");
        }
        var interrogationCode = OptionalInt32(profile.Config, "interrogationCode") ?? 20;
        if (interrogationCode is not (1 or 3 or 20))
        {
            throw Invalid("IEC104_INTERROGATION_CODE_INVALID", "IEC 104 召唤限定词只支持 1、3 或 20");
        }
        var interrogationReason = OptionalInt32(profile.Config, "interrogationReason") ?? 6;
        if (interrogationReason is < 1 or > 10)
        {
            throw Invalid("IEC104_INTERROGATION_REASON_INVALID", "IEC 104 传送原因必须在 1 到 10 之间");
        }

        return new Iec104ConnectionOptions(
            host,
            port,
            Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000),
            checked((ushort)commonAddress),
            checked((byte)interrogationCode),
            checked((byte)interrogationReason),
            Timeout(profile.Config, "interrogationTimeoutMs", 3000));
    }

    public HslIec104ClientOptions ToHslOptions() => new(
        Host,
        Port,
        ConnectTimeoutMilliseconds,
        ReceiveTimeoutMilliseconds,
        CommonAddress,
        InterrogationCode,
        InterrogationReason,
        InterrogationTimeoutMilliseconds);

    private static int Timeout(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 100 or > 120000)
        {
            throw Invalid("IEC104_TIMEOUT_INVALID", $"IEC 104 {name} 必须在 100 到 120000 毫秒之间");
        }
        return value;
    }

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result)
            ? result
            : null;

    private static Iec104DriverException Invalid(string code, string message) => new(code, message, retryable: false);
}
