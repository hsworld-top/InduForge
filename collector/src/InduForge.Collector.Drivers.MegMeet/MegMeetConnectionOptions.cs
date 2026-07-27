using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MegMeet;

internal enum MegMeetDriverKind { Tcp, RtuOverTcp, RtuSerial }

internal sealed record MegMeetConnectionOptions(
    MegMeetDriverKind Kind, HslMegMeetTransport Transport, byte Station,
    string? Host, int Port, string? PortName, int BaudRate, int DataBits,
    HslSerialParity Parity, HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds)
{
    public static MegMeetConnectionOptions Parse(ConnectionProfile profile, MegMeetDriverKind kind)
    {
        if (!string.Equals(profile.ProtocolFamily, "megmeet", StringComparison.OrdinalIgnoreCase))
            throw Invalid("MEGMEET_PROTOCOL_MISMATCH", "连接配置不是麦格米特 PLC 协议");
        if (profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("MEGMEET_CONFIG_INVALID", "麦格米特 PLC 连接配置无效");

        var transport = kind switch
        {
            MegMeetDriverKind.Tcp => HslMegMeetTransport.Tcp,
            MegMeetDriverKind.RtuOverTcp => HslMegMeetTransport.RtuOverTcp,
            MegMeetDriverKind.RtuSerial => HslMegMeetTransport.RtuSerial,
            _ => throw new ArgumentOutOfRangeException(nameof(kind)),
        };
        var isSerial = kind == MegMeetDriverKind.RtuSerial;
        var host = Text(profile.Config, "host")?.Trim();
        var portName = Text(profile.Config, "portName")?.Trim();
        if (isSerial && string.IsNullOrWhiteSpace(portName))
            throw Invalid("MEGMEET_PORT_NAME_INVALID", "麦格米特串口名称不能为空");
        if (!isSerial && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)))
            throw Invalid("MEGMEET_HOST_INVALID", "麦格米特设备 IP / 主机名无效");

        var port = Integer(profile.Config, "port") ?? 502;
        if (!isSerial && port is < 1 or > 65535)
            throw Invalid("MEGMEET_PORT_INVALID", "麦格米特端口必须在 1 到 65535 之间");

        return new(kind, transport, Byte(profile.Config, "station", 1), host, port, portName,
            Integer(profile.Config, "baudRate") ?? 9600, Integer(profile.Config, "dataBits") ?? 8,
            ParseParity(Text(profile.Config, "parity") ?? "none"), ParseStopBits(Integer(profile.Config, "stopBits") ?? 1),
            Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000));
    }

    public HslMegMeetClientOptions ToHslOptions() => new(Transport, Station, Host, Port, PortName, BaudRate, DataBits,
        Parity, StopBits, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, HslDataFormat.CDAB);

    private static int Timeout(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 100 or > 120000)
            throw Invalid("MEGMEET_TIMEOUT_INVALID", "麦格米特超时必须在 100 到 120000 毫秒之间");
        return value;
    }

    private static byte Byte(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 0 or > 255)
            throw Invalid("MEGMEET_STATION_INVALID", "麦格米特站号必须在 0 到 255 之间");
        return checked((byte)value);
    }

    private static HslSerialParity ParseParity(string value) => value.Trim().ToLowerInvariant() switch
    {
        "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even,
        _ => throw Invalid("MEGMEET_PARITY_INVALID", "麦格米特串口校验位无效"),
    };
    private static HslSerialStopBits ParseStopBits(int value) => value switch
    {
        1 => HslSerialStopBits.One, 2 => HslSerialStopBits.Two,
        _ => throw Invalid("MEGMEET_STOP_BITS_INVALID", "麦格米特串口停止位必须为 1 或 2"),
    };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static MegMeetDriverException Invalid(string code, string message) => new(code, message, false);
}
