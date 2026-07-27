using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Freedom;

internal sealed record FreedomOptions(HslFreedomTransport Transport, string Host, int Port, string? PortName, int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits, bool RtsEnable, int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, HslDataFormat DataFormat, bool StringReverse)
{
    public static FreedomOptions Parse(ConnectionProfile profile, HslFreedomTransport transport)
    {
        if (!string.Equals(profile.ProtocolFamily, "freedom", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("FREEDOM_CONFIG_INVALID", "自由协议连接配置无效");
        var serial = transport == HslFreedomTransport.Serial;
        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty;
        if (!serial && (host.Length == 0 || host.Contains("://", StringComparison.Ordinal))) throw Invalid("FREEDOM_HOST_INVALID", "自由协议设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? 6000;
        if (!serial && port is < 1 or > 65535) throw Invalid("FREEDOM_PORT_INVALID", "自由协议端口无效");
        var portName = Text(profile.Config, "portName")?.Trim();
        if (serial && string.IsNullOrWhiteSpace(portName)) throw Invalid("FREEDOM_PORT_NAME_INVALID", "自由协议串口名称不能为空");
        var baudRate = Integer(profile.Config, "baudRate") ?? 9600; var dataBits = Integer(profile.Config, "dataBits") ?? 8; var stopBits = Integer(profile.Config, "stopBits") ?? 1;
        if (serial && (baudRate <= 0 || dataBits is < 5 or > 8 || stopBits is < 1 or > 2)) throw Invalid("FREEDOM_SERIAL_CONFIG_INVALID", "自由协议串口参数无效");
        return new(transport, host, port, portName, baudRate, dataBits, ParseParity(Text(profile.Config, "parity") ?? "none"), stopBits == 2 ? HslSerialStopBits.Two : HslSerialStopBits.One, Boolean(profile.Config, "rtsEnable") ?? false, Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000), Format(Text(profile.Config, "dataFormat") ?? "DCBA"), Boolean(profile.Config, "stringReverse") ?? false);
    }
    public HslFreedomClientOptions ToHslOptions() => new(Transport, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits, RtsEnable, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, DataFormat, StringReverse);
    public string ServerName => Transport == HslFreedomTransport.Serial ? PortName! : $"{Host}:{Port}";
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("FREEDOM_TIMEOUT_INVALID", "自由协议超时配置无效"); return value; }
    private static HslSerialParity ParseParity(string value) => value switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, _ => throw Invalid("FREEDOM_PARITY_INVALID", "自由协议校验方式无效") };
    private static HslDataFormat Format(string value) => value.ToUpperInvariant() switch { "ABCD" => HslDataFormat.ABCD, "BADC" => HslDataFormat.BADC, "CDAB" => HslDataFormat.CDAB, "DCBA" => HslDataFormat.DCBA, _ => throw Invalid("FREEDOM_DATA_FORMAT_INVALID", "自由协议数据格式无效") };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;
    private static bool? Boolean(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : null;
    private static FreedomDriverException Invalid(string code, string message) => new(code, message, false);
}
