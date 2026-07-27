using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Cimon;
internal sealed record CimonOptions(string Host, int Port, int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, byte FrameNumber)
{
    public static CimonOptions Parse(ConnectionProfile profile) { if (!string.Equals(profile.ProtocolFamily, "cimon", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("CIMON_CONFIG_INVALID", "Cimon HMI 连接配置无效"); var host = Text(profile.Config, "host")?.Trim(); if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)) throw Invalid("CIMON_HOST_INVALID", "Cimon HMI 设备 IP / 主机名无效"); var port = Integer(profile.Config, "port") ?? 10260; if (port is < 1 or > 65535) throw Invalid("CIMON_PORT_INVALID", "Cimon HMI 端口无效"); var frame = Integer(profile.Config, "frameNumber") ?? 1; if (frame is < 0 or > 255) throw Invalid("CIMON_FRAME_INVALID", "Cimon HMI FrameNo 必须在 0 到 255 之间"); return new(host, port, Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000), checked((byte)frame)); }
    public HslCimonClientOptions ToHslOptions() => new(Host, Port, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, FrameNumber);
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("CIMON_TIMEOUT_INVALID", "Cimon HMI 超时配置无效"); return value; }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;
    private static CimonDriverException Invalid(string code, string message) => new(code, message, false);
}
