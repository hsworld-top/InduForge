using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins;

internal sealed record OmronVariantOptions(
    HslOmronVariantProtocol Protocol, string Host, int Port, string? PortName,
    int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, byte UnitNumber, HslOmronPlcType PlcType)
{
    public static OmronVariantOptions Parse(ConnectionProfile profile, HslOmronVariantProtocol protocol)
    {
        if (!string.Equals(profile.ProtocolFamily, "omron", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("OMRON_CONFIG_INVALID", "欧姆龙连接配置无效");
        var serial = protocol is HslOmronVariantProtocol.HostLink or HslOmronVariantProtocol.HostLinkCMode;
        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty;
        if (!serial && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)))
            throw Invalid("OMRON_HOST_INVALID", "欧姆龙设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? (protocol is HslOmronVariantProtocol.Cip or HslOmronVariantProtocol.ConnectedCip ? 44818 : 2000);
        if (!serial && port is < 1 or > 65535) throw Invalid("OMRON_PORT_INVALID", "欧姆龙端口无效");
        var portName = Text(profile.Config, "portName")?.Trim();
        if (serial && string.IsNullOrWhiteSpace(portName)) throw Invalid("OMRON_PORT_NAME_INVALID", "欧姆龙串口名称不能为空");
        return new(protocol, host, port, portName, Integer(profile.Config, "baudRate") ?? 9600,
            Integer(profile.Config, "dataBits") ?? 7, ParseParity(Text(profile.Config, "parity") ?? "even"),
            (Integer(profile.Config, "stopBits") ?? 1) == 2 ? HslSerialStopBits.Two : HslSerialStopBits.One,
            Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000),
            Byte(profile.Config, "unitNumber", 0), ParsePlcType(Text(profile.Config, "plcType") ?? "cscj"));
    }

    public HslOmronVariantClientOptions ToHslOptions() => new(Protocol, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits,
        ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, UnitNumber, HslDataFormat.DCBA, PlcType);
    public string ServerName => Protocol is HslOmronVariantProtocol.HostLink or HslOmronVariantProtocol.HostLinkCMode ? PortName! : $"{Host}:{Port}";
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("OMRON_TIMEOUT_INVALID", "欧姆龙超时配置无效"); return value; }
    private static byte Byte(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 0 or > 255) throw Invalid("OMRON_CONFIG_INVALID", $"欧姆龙 {name} 无效"); return checked((byte)value); }
    private static HslSerialParity ParseParity(string value) => value switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, _ => throw Invalid("OMRON_PARITY_INVALID", "欧姆龙串口校验方式无效") };
    private static HslOmronPlcType ParsePlcType(string value) => value.ToLowerInvariant() switch { "cscj" => HslOmronPlcType.CSCJ, "cv" => HslOmronPlcType.CV, _ => throw Invalid("OMRON_PLC_TYPE_INVALID", "欧姆龙 PLC 类型无效") };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;
    private static OmronFinsDriverException Invalid(string code, string message) => new(code, message, false);
}
