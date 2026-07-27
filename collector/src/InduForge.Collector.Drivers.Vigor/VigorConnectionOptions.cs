using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Vigor;
internal enum VigorDriverKind { SerialOverTcp, Serial }
internal sealed record VigorConnectionOptions(VigorDriverKind Kind, HslVigorTransport Transport, byte Station, string? Host, int Port, string? PortName, int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits, int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds)
{
    public static VigorConnectionOptions Parse(ConnectionProfile profile, VigorDriverKind kind)
    {
        if (!string.Equals(profile.ProtocolFamily, "vigor", StringComparison.OrdinalIgnoreCase)) throw Invalid("VIGOR_PROTOCOL_MISMATCH", "连接配置不是丰炜 PLC 协议");
        if (profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("VIGOR_CONFIG_INVALID", "丰炜 PLC 连接配置无效");
        var isSerial = kind == VigorDriverKind.Serial; var host = Text(profile.Config, "host")?.Trim(); var portName = Text(profile.Config, "portName")?.Trim();
        if (isSerial && string.IsNullOrWhiteSpace(portName)) throw Invalid("VIGOR_PORT_NAME_INVALID", "丰炜串口名称不能为空");
        if (!isSerial && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))) throw Invalid("VIGOR_HOST_INVALID", "丰炜设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? 5000; if (!isSerial && port is < 1 or > 65535) throw Invalid("VIGOR_PORT_INVALID", "丰炜端口必须在 1 到 65535 之间");
        return new(kind, isSerial ? HslVigorTransport.Serial : HslVigorTransport.SerialOverTcp, Byte(profile.Config, "station", 0), host, port, portName,
            Integer(profile.Config, "baudRate") ?? 9600, Integer(profile.Config, "dataBits") ?? 8, ParseParity(Text(profile.Config, "parity") ?? "none"), ParseStopBits(Integer(profile.Config, "stopBits") ?? 1),
            Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000));
    }
    public HslVigorClientOptions ToHslOptions() => new(Transport, Station, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, HslDataFormat.DCBA);
    private static int Timeout(JsonElement c, string n, int f) { var v = Integer(c, n) ?? f; if (v is < 100 or > 120000) throw Invalid("VIGOR_TIMEOUT_INVALID", "丰炜超时必须在 100 到 120000 毫秒之间"); return v; }
    private static byte Byte(JsonElement c, string n, int f) { var v = Integer(c, n) ?? f; if (v is < 0 or > 255) throw Invalid("VIGOR_STATION_INVALID", "丰炜站号必须在 0 到 255 之间"); return checked((byte)v); }
    private static HslSerialParity ParseParity(string v) => v.Trim().ToLowerInvariant() switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, _ => throw Invalid("VIGOR_PARITY_INVALID", "丰炜串口校验位无效") };
    private static HslSerialStopBits ParseStopBits(int v) => v switch { 1 => HslSerialStopBits.One, 2 => HslSerialStopBits.Two, _ => throw Invalid("VIGOR_STOP_BITS_INVALID", "丰炜串口停止位必须为 1 或 2") };
    private static string? Text(JsonElement c, string n) => c.TryGetProperty(n, out var v) && v.ValueKind == JsonValueKind.String ? v.GetString() : null;
    private static int? Integer(JsonElement c, string n) => c.TryGetProperty(n, out var v) && v.TryGetInt32(out var r) ? r : null;
    private static VigorDriverException Invalid(string code, string message) => new(code, message, false);
}
