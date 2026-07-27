using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Delta;

internal enum DeltaDriverKind { Tcp, RtuOverTcp, AsciiOverTcp, RtuSerial, AsciiSerial }

internal sealed record DeltaConnectionOptions(
    DeltaDriverKind Kind, HslDeltaTransport Transport, HslDeltaSeries Series, byte Station,
    string? Host, int Port, string? PortName, int BaudRate, int DataBits, HslSerialParity Parity,
    HslSerialStopBits StopBits, int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds)
{
    public static DeltaConnectionOptions Parse(ConnectionProfile profile, DeltaDriverKind kind)
    {
        if (!string.Equals(profile.ProtocolFamily, "delta", StringComparison.OrdinalIgnoreCase))
            throw Invalid("DELTA_PROTOCOL_MISMATCH", "连接配置不是台达 PLC 协议");
        if (profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("DELTA_CONFIG_INVALID", "台达 PLC 连接配置无效");
        var transport = kind switch
        {
            DeltaDriverKind.Tcp => HslDeltaTransport.Tcp,
            DeltaDriverKind.RtuOverTcp => HslDeltaTransport.RtuOverTcp,
            DeltaDriverKind.AsciiOverTcp => HslDeltaTransport.AsciiOverTcp,
            DeltaDriverKind.RtuSerial => HslDeltaTransport.RtuSerial,
            DeltaDriverKind.AsciiSerial => HslDeltaTransport.AsciiSerial,
            _ => throw new ArgumentOutOfRangeException(nameof(kind)),
        };
        var isSerial = kind is DeltaDriverKind.RtuSerial or DeltaDriverKind.AsciiSerial;
        var host = Text(profile.Config, "host")?.Trim();
        var portName = Text(profile.Config, "portName")?.Trim();
        if (isSerial && string.IsNullOrWhiteSpace(portName)) throw Invalid("DELTA_PORT_NAME_INVALID", "台达串口名称不能为空");
        if (!isSerial && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))) throw Invalid("DELTA_HOST_INVALID", "台达设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? 502;
        if (!isSerial && port is < 1 or > 65535) throw Invalid("DELTA_PORT_INVALID", "台达端口必须在 1 到 65535 之间");
        var seriesText = Text(profile.Config, "series")?.Trim().ToUpperInvariant() ?? "DVP";
        var series = seriesText switch { "DVP" => HslDeltaSeries.Dvp, "AS" => HslDeltaSeries.AS, _ => throw Invalid("DELTA_SERIES_INVALID", "台达系列必须为 DVP 或 AS") };
        return new(kind, transport, series, Byte(profile.Config, "station", 1), host, port, portName,
            Integer(profile.Config, "baudRate") ?? 9600, Integer(profile.Config, "dataBits") ?? 7,
            ParseParity(Text(profile.Config, "parity") ?? "even"), ParseStopBits(Integer(profile.Config, "stopBits") ?? 1),
            Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000));
    }

    public HslDeltaClientOptions ToHslOptions() => new(Transport, Series, Station, Host, Port, PortName, BaudRate, DataBits,
        Parity, StopBits, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, HslDataFormat.CDAB);
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("DELTA_TIMEOUT_INVALID", "台达超时必须在 100 到 120000 毫秒之间"); return value; }
    private static byte Byte(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 0 or > 255) throw Invalid("DELTA_STATION_INVALID", "台达站号必须在 0 到 255 之间"); return checked((byte)value); }
    private static HslSerialParity ParseParity(string value) => value.Trim().ToLowerInvariant() switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, _ => throw Invalid("DELTA_PARITY_INVALID", "台达串口校验位无效") };
    private static HslSerialStopBits ParseStopBits(int value) => value switch { 1 => HslSerialStopBits.One, 2 => HslSerialStopBits.Two, _ => throw Invalid("DELTA_STOP_BITS_INVALID", "台达串口停止位必须为 1 或 2") };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static DeltaDriverException Invalid(string code, string message) => new(code, message, false);
}
