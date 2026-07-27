using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.GeSrtpTcp;

internal sealed record GeSrtpTcpConnectionOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat)
{
    public static GeSrtpTcpConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "ge", StringComparison.OrdinalIgnoreCase))
            throw Invalid("GE_SRTP_PROTOCOL_MISMATCH", "连接配置不是 GE SRTP 协议");
        if (profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("GE_SRTP_CONFIG_INVALID", "GE SRTP 连接配置无效");

        var host = OptionalString(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
            throw Invalid("GE_SRTP_HOST_INVALID", "GE PLC IP / 主机名无效");
        var port = OptionalInt32(profile.Config, "port") ?? 18245;
        if (port is < 1 or > 65535)
            throw Invalid("GE_SRTP_PORT_INVALID", "GE SRTP 端口必须在 1 到 65535 之间");

        return new GeSrtpTcpConnectionOptions(
            host,
            port,
            Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000),
            ParseDataFormat(OptionalString(profile.Config, "dataFormat") ?? "DCBA"));
    }

    public HslGeSrtpClientOptions ToHslOptions() =>
        new(Host, Port, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, DataFormat);

    private static int Timeout(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 100 or > 120000)
            throw Invalid("GE_SRTP_TIMEOUT_INVALID", $"GE SRTP {name} 必须在 100 到 120000 毫秒之间");
        return value;
    }

    private static HslDataFormat ParseDataFormat(string value) => value switch
    {
        "ABCD" => HslDataFormat.ABCD,
        "BADC" => HslDataFormat.BADC,
        "CDAB" => HslDataFormat.CDAB,
        "DCBA" => HslDataFormat.DCBA,
        _ => throw Invalid("GE_SRTP_DATA_FORMAT_INVALID", "GE SRTP 数据格式无效"),
    };

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;

    private static GeSrtpTcpDriverException Invalid(string code, string message) => new(code, message, false);
}
