using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Xinje;

internal enum XinjeDriverKind { Tcp, RtuOverTcp, RtuSerial, InternalTcp }

internal sealed record XinjeConnectionOptions(
    XinjeDriverKind Kind, HslXinjeTransport Transport, HslXinjeSeries Series, byte Station, string? Host, int Port,
    string? PortName, int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, HslDataFormat DataFormat)
{
    public static XinjeConnectionOptions Parse(ConnectionProfile profile, XinjeDriverKind kind)
    {
        if (!string.Equals(profile.ProtocolFamily, "xinje", StringComparison.OrdinalIgnoreCase)) throw Invalid("XINJE_PROTOCOL_MISMATCH", "连接配置不是信捷 PLC 协议");
        if (profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("XINJE_CONFIG_INVALID", "信捷 PLC 连接配置无效");
        var transport = kind switch { XinjeDriverKind.Tcp => HslXinjeTransport.Tcp, XinjeDriverKind.RtuOverTcp => HslXinjeTransport.RtuOverTcp, XinjeDriverKind.RtuSerial => HslXinjeTransport.RtuSerial, XinjeDriverKind.InternalTcp => HslXinjeTransport.InternalTcp, _ => throw new ArgumentOutOfRangeException(nameof(kind)) };
        var isSerial = kind == XinjeDriverKind.RtuSerial;
        var host = Text(profile.Config, "host")?.Trim();
        var portName = Text(profile.Config, "portName")?.Trim();
        if (isSerial && string.IsNullOrWhiteSpace(portName)) throw Invalid("XINJE_PORT_NAME_INVALID", "信捷串口名称不能为空");
        if (!isSerial && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))) throw Invalid("XINJE_HOST_INVALID", "信捷设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? 502;
        if (!isSerial && port is < 1 or > 65535) throw Invalid("XINJE_PORT_INVALID", "信捷端口必须在 1 到 65535 之间");
        var series = ParseSeries(Text(profile.Config, "series") ?? "XC");
        var station = Byte(profile.Config, "station", kind == XinjeDriverKind.InternalTcp ? 0 : 1);
        var format = kind == XinjeDriverKind.InternalTcp ? HslDataFormat.CDAB : ParseDataFormat(Text(profile.Config, "dataFormat") ?? "ABCD");
        return new(kind, transport, series, station, host, port, portName, Integer(profile.Config, "baudRate") ?? 9600,
            Integer(profile.Config, "dataBits") ?? 8, ParseParity(Text(profile.Config, "parity") ?? "none"),
            ParseStopBits(Integer(profile.Config, "stopBits") ?? 1), Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000), format);
    }
    public HslXinjeClientOptions ToHslOptions() => new(Transport, Series, Station, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, DataFormat);
    private static HslXinjeSeries ParseSeries(string value) => value.Trim().ToUpperInvariant() switch { "XC" => HslXinjeSeries.XC, "XD" => HslXinjeSeries.XD, "XL" => HslXinjeSeries.XL, _ => throw Invalid("XINJE_SERIES_INVALID", "信捷系列必须为 XC、XD 或 XL") };
    private static HslDataFormat ParseDataFormat(string value) => value.Trim().ToUpperInvariant() switch { "ABCD" => HslDataFormat.ABCD, "BADC" => HslDataFormat.BADC, "CDAB" => HslDataFormat.CDAB, "DCBA" => HslDataFormat.DCBA, _ => throw Invalid("XINJE_DATA_FORMAT_INVALID", "信捷数据格式无效") };
    private static HslSerialParity ParseParity(string value) => value.Trim().ToLowerInvariant() switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, _ => throw Invalid("XINJE_PARITY_INVALID", "信捷串口校验位无效") };
    private static HslSerialStopBits ParseStopBits(int value) => value switch { 1 => HslSerialStopBits.One, 2 => HslSerialStopBits.Two, _ => throw Invalid("XINJE_STOP_BITS_INVALID", "信捷串口停止位必须为 1 或 2") };
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("XINJE_TIMEOUT_INVALID", "信捷超时必须在 100 到 120000 毫秒之间"); return value; }
    private static byte Byte(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 0 or > 255) throw Invalid("XINJE_STATION_INVALID", "信捷站号必须在 0 到 255 之间"); return checked((byte)value); }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static XinjeDriverException Invalid(string code, string message) => new(code, message, false);
}
