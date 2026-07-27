using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Fuji;

internal enum FujiDriverKind { CommandSettingTypeTcp, SphTcp, SpbOverTcp, SpbSerial }

internal sealed record FujiConnectionOptions(
    FujiDriverKind Kind, HslFujiProtocol Protocol, string? Host, int Port, string? PortName,
    int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits,
    byte Station, byte ConnectionId, bool DataSwap, int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds, HslDataFormat DataFormat)
{
    public static FujiConnectionOptions Parse(ConnectionProfile profile, FujiDriverKind kind)
    {
        if (!string.Equals(profile.ProtocolFamily, "fuji", StringComparison.OrdinalIgnoreCase))
            throw Invalid("FUJI_PROTOCOL_MISMATCH", "连接配置不是富士 PLC 协议");
        if (profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("FUJI_CONFIG_INVALID", "富士 PLC 连接配置无效");

        var protocol = kind switch
        {
            FujiDriverKind.CommandSettingTypeTcp => HslFujiProtocol.CommandSettingTypeTcp,
            FujiDriverKind.SphTcp => HslFujiProtocol.SphTcp,
            FujiDriverKind.SpbOverTcp => HslFujiProtocol.SpbOverTcp,
            FujiDriverKind.SpbSerial => HslFujiProtocol.SpbSerial,
            _ => throw new ArgumentOutOfRangeException(nameof(kind)),
        };
        var isSerial = kind == FujiDriverKind.SpbSerial;
        var host = Text(profile.Config, "host")?.Trim();
        var portName = Text(profile.Config, "portName")?.Trim();
        if (isSerial && string.IsNullOrWhiteSpace(portName)) throw Invalid("FUJI_PORT_NAME_INVALID", "富士串口名称不能为空");
        if (!isSerial && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))) throw Invalid("FUJI_HOST_INVALID", "富士设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? DefaultPort(kind);
        if (!isSerial && port is < 1 or > 65535) throw Invalid("FUJI_PORT_INVALID", "富士端口必须在 1 到 65535 之间");

        var dataFormat = kind == FujiDriverKind.SphTcp
            ? ParseDataFormat(Text(profile.Config, "dataFormat") ?? "DCBA")
            : kind == FujiDriverKind.CommandSettingTypeTcp ? HslDataFormat.ABCD : HslDataFormat.DCBA;
        return new(kind, protocol, host, port, portName, Integer(profile.Config, "baudRate") ?? 9600,
            Integer(profile.Config, "dataBits") ?? 8, ParseParity(Text(profile.Config, "parity") ?? "none"),
            ParseStopBits(Integer(profile.Config, "stopBits") ?? 1), Byte(profile.Config, "station", 0, "FUJI_STATION_INVALID", "富士站号"),
            Byte(profile.Config, "connectionId", 254, "FUJI_CONNECTION_ID_INVALID", "富士连接 ID"), Boolean(profile.Config, "dataSwap") ?? false,
            Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000), dataFormat);
    }

    public HslFujiClientOptions ToHslOptions() => new(Protocol, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits,
        Station, ConnectionId, DataSwap, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, DataFormat);

    private static int DefaultPort(FujiDriverKind kind) => kind switch
    {
        FujiDriverKind.CommandSettingTypeTcp => 7000,
        FujiDriverKind.SphTcp => 507,
        FujiDriverKind.SpbOverTcp => 9600,
        _ => 0,
    };
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("FUJI_TIMEOUT_INVALID", "富士超时必须在 100 到 120000 毫秒之间"); return value; }
    private static byte Byte(JsonElement config, string name, int fallback, string code, string label) { var value = Integer(config, name) ?? fallback; if (value is < 0 or > 255) throw Invalid(code, $"{label}必须在 0 到 255 之间"); return checked((byte)value); }
    private static HslDataFormat ParseDataFormat(string value) => value.Trim().ToUpperInvariant() switch { "ABCD" => HslDataFormat.ABCD, "BADC" => HslDataFormat.BADC, "CDAB" => HslDataFormat.CDAB, "DCBA" => HslDataFormat.DCBA, _ => throw Invalid("FUJI_DATA_FORMAT_INVALID", "富士数据格式无效") };
    private static HslSerialParity ParseParity(string value) => value.Trim().ToLowerInvariant() switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, _ => throw Invalid("FUJI_PARITY_INVALID", "富士串口校验位无效") };
    private static HslSerialStopBits ParseStopBits(int value) => value switch { 1 => HslSerialStopBits.One, 2 => HslSerialStopBits.Two, _ => throw Invalid("FUJI_STOP_BITS_INVALID", "富士串口停止位必须为 1 或 2") };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static bool? Boolean(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : null;
    private static FujiDriverException Invalid(string code, string message) => new(code, message, false);
}
