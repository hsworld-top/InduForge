using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.FatekProgramTcp;

internal sealed record FatekProgramTcpConnectionOptions(
    string Host, int Port, int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, byte Station)
{
    public static FatekProgramTcpConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "fatek", StringComparison.OrdinalIgnoreCase))
            throw Invalid("FATEK_PROGRAM_PROTOCOL_MISMATCH", "连接配置不是永宏编程口协议");
        if (profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("FATEK_PROGRAM_TCP_CONFIG_INVALID", "永宏编程口 TCP 连接配置无效");
        var host = Text(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
            throw Invalid("FATEK_PROGRAM_TCP_HOST_INVALID", "永宏 PLC IP / 主机名无效");
        var port = Number(profile.Config, "port") ?? 2000;
        if (port is < 1 or > 65535)
            throw Invalid("FATEK_PROGRAM_TCP_PORT_INVALID", "永宏编程口 TCP 端口必须在 1 到 65535 之间");
        var station = Number(profile.Config, "station") ?? 1;
        if (station is < 0 or > 255)
            throw Invalid("FATEK_PROGRAM_TCP_STATION_INVALID", "永宏 PLC 站号必须在 0 到 255 之间");
        return new(host, port, Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000), checked((byte)station));
    }

    public HslFatekProgramTcpClientOptions ToHslOptions() => new(Host, Port, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, Station);
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Number(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("FATEK_PROGRAM_TCP_TIMEOUT_INVALID", $"永宏编程口 TCP {name} 必须在 100 到 120000 毫秒之间"); return value; }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Number(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static FatekProgramDriverException Invalid(string code, string message) => new(code, message, false);
}
