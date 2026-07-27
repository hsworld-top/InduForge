using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Keyence;

internal enum KeyenceDriverKind { Mc3E, McAscii, KvOld, Nano, NanoSerial, NanoSerialOverTcp }

internal sealed record KeyenceConnectionOptions(
    KeyenceDriverKind Kind, string Host, int Port, int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds,
    HslKeyenceProtocol Protocol, byte NetworkNumber, byte NetworkStationNumber, ushort TargetIoStation,
    bool EnableWriteBitToWordRegister, bool StringReverse, byte Station, bool UseStation,
    string? PortName, int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits)
{
    public static KeyenceConnectionOptions Parse(ConnectionProfile profile, KeyenceDriverKind kind)
    {
        if (!string.Equals(profile.ProtocolFamily, "keyence", StringComparison.OrdinalIgnoreCase))
            throw Invalid("KEYENCE_PROTOCOL_MISMATCH", "连接配置不是基恩士协议");
        if (profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("KEYENCE_CONFIG_INVALID", "基恩士连接配置无效");
        var serial = kind == KeyenceDriverKind.NanoSerial;
        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty;
        if (!serial && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)))
            throw Invalid("KEYENCE_HOST_INVALID", "基恩士设备 IP / 主机名无效");
        var defaultPort = kind is KeyenceDriverKind.Mc3E or KeyenceDriverKind.McAscii ? 6000 : 8501;
        var port = Integer(profile.Config, "port") ?? defaultPort;
        if (!serial && port is < 1 or > 65535) throw Invalid("KEYENCE_PORT_INVALID", "基恩士端口必须在 1 到 65535 之间");
        var connectTimeout = Timeout(profile.Config, "connectTimeoutMs", 5000);
        var receiveTimeout = Timeout(profile.Config, "receiveTimeoutMs", 10000);
        var frameFormat = Text(profile.Config, "frameFormat")?.Trim().ToLowerInvariant() ?? "binary";
        var protocol = kind switch
        {
            KeyenceDriverKind.Mc3E when frameFormat == "binary" => HslKeyenceProtocol.McBinary,
            KeyenceDriverKind.Mc3E when frameFormat == "ascii" => HslKeyenceProtocol.McAscii,
            KeyenceDriverKind.McAscii => HslKeyenceProtocol.McAscii,
            KeyenceDriverKind.KvOld => HslKeyenceProtocol.KvOld,
            KeyenceDriverKind.Nano => HslKeyenceProtocol.Nano,
            KeyenceDriverKind.NanoSerial => HslKeyenceProtocol.NanoSerial,
            KeyenceDriverKind.NanoSerialOverTcp => HslKeyenceProtocol.NanoSerialOverTcp,
            _ => throw Invalid("KEYENCE_FRAME_FORMAT_INVALID", "基恩士 MC 报文格式必须为 binary 或 ascii"),
        };
        var portName = Text(profile.Config, "portName")?.Trim();
        if (serial && string.IsNullOrWhiteSpace(portName)) throw Invalid("KEYENCE_PORT_NAME_INVALID", "基恩士 Nano 串口名称不能为空");
        var baudRate = Integer(profile.Config, "baudRate") ?? 9600;
        var dataBits = Integer(profile.Config, "dataBits") ?? 8;
        var stopBits = Integer(profile.Config, "stopBits") ?? 1;
        if (serial && (baudRate is < 1200 or > 115200 || dataBits is < 7 or > 8 || stopBits is < 1 or > 2))
            throw Invalid("KEYENCE_SERIAL_CONFIG_INVALID", "基恩士 Nano 串口参数无效");
        return new(kind, host, port, connectTimeout, receiveTimeout, protocol,
            Byte(profile.Config, "networkNumber", 0), Byte(profile.Config, "networkStationNumber", 0),
            UInt16(profile.Config, "targetIoStation", 1023), Boolean(profile.Config, "enableWriteBitToWordRegister"),
            Boolean(profile.Config, "stringReverse"), Byte(profile.Config, "station", 0), Boolean(profile.Config, "useStation"),
            portName, baudRate, dataBits, ParseParity(Text(profile.Config, "parity") ?? "none"),
            stopBits == 2 ? HslSerialStopBits.Two : HslSerialStopBits.One);
    }

    public HslKeyenceTcpClientOptions ToHslOptions() => new(Protocol, Host, Port, ConnectTimeoutMilliseconds,
        ReceiveTimeoutMilliseconds, HslDataFormat.DCBA, StringReverse, NetworkNumber, NetworkStationNumber,
        TargetIoStation, EnableWriteBitToWordRegister, Station, UseStation,
        PortName, BaudRate, DataBits, Parity, StopBits);

    private static HslSerialParity ParseParity(string value) => value switch
    {
        "none" => HslSerialParity.None,
        "odd" => HslSerialParity.Odd,
        "even" => HslSerialParity.Even,
        _ => throw Invalid("KEYENCE_PARITY_INVALID", "基恩士 Nano 串口校验方式无效"),
    };

    private static int Timeout(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 100 or > 120000) throw Invalid("KEYENCE_TIMEOUT_INVALID", "基恩士超时必须在 100 到 120000 毫秒之间");
        return value;
    }
    private static byte Byte(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 0 or > 255) throw Invalid("KEYENCE_CONFIG_INVALID", $"基恩士 {name} 必须在 0 到 255 之间");
        return checked((byte)value);
    }
    private static ushort UInt16(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 0 or > 65535) throw Invalid("KEYENCE_CONFIG_INVALID", $"基恩士 {name} 必须在 0 到 65535 之间");
        return checked((ushort)value);
    }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static bool Boolean(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False && value.GetBoolean();
    private static KeyenceDriverException Invalid(string code, string message) => new(code, message, false);
}
