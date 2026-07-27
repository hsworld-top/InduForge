using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.PanasonicMewtocolTcp;

internal sealed record PanasonicMewtocolTcpConnectionOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte Station,
    HslDataFormat DataFormat)
{
    public static PanasonicMewtocolTcpConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "panasonic", StringComparison.OrdinalIgnoreCase))
            throw Invalid("PANASONIC_MEWTOCOL_PROTOCOL_MISMATCH", "连接配置不是松下 Mewtocol 协议");
        if (profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("PANASONIC_MEWTOCOL_CONFIG_INVALID", "松下 Mewtocol 连接配置无效");

        var host = OptionalString(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
            throw Invalid("PANASONIC_MEWTOCOL_HOST_INVALID", "松下 PLC IP / 主机名无效");
        var port = OptionalInt32(profile.Config, "port") ?? 2000;
        if (port is < 1 or > 65535)
            throw Invalid("PANASONIC_MEWTOCOL_PORT_INVALID", "松下 Mewtocol 端口必须在 1 到 65535 之间");
        var station = OptionalInt32(profile.Config, "station") ?? 0xEE;
        if (station is < 0 or > 255)
            throw Invalid("PANASONIC_MEWTOCOL_STATION_INVALID", "松下 Mewtocol 站号必须在 0 到 255 之间");

        return new PanasonicMewtocolTcpConnectionOptions(
            host,
            port,
            Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000),
            checked((byte)station),
            ParseDataFormat(OptionalString(profile.Config, "dataFormat") ?? "DCBA"));
    }

    public HslPanasonicMewtocolClientOptions ToHslOptions() => new(
        Host, Port, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, Station, DataFormat);

    private static int Timeout(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 100 or > 120000)
            throw Invalid("PANASONIC_MEWTOCOL_TIMEOUT_INVALID", $"松下 Mewtocol {name} 必须在 100 到 120000 毫秒之间");
        return value;
    }

    private static HslDataFormat ParseDataFormat(string value) => value switch
    {
        "ABCD" => HslDataFormat.ABCD,
        "BADC" => HslDataFormat.BADC,
        "CDAB" => HslDataFormat.CDAB,
        "DCBA" => HslDataFormat.DCBA,
        _ => throw Invalid("PANASONIC_MEWTOCOL_DATA_FORMAT_INVALID", "松下 Mewtocol 数据格式无效"),
    };

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;
    private static PanasonicMewtocolTcpDriverException Invalid(string code, string message) => new(code, message, false);
}
