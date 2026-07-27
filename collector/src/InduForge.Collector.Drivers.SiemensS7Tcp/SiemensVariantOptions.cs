using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp;

internal sealed record SiemensVariantOptions(
    HslSiemensVariantProtocol Protocol,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    byte Station,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    bool HandshakeCheck,
    string? UserName,
    string? Password,
    bool UseHttps)
{
    public static SiemensVariantOptions Parse(ConnectionProfile profile, HslSiemensVariantProtocol protocol)
    {
        if (!string.Equals(profile.ProtocolFamily, "siemens", StringComparison.OrdinalIgnoreCase) ||
            profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("SIEMENS_CONFIG_INVALID", "Siemens 连接配置无效");
        }

        var serial = protocol is HslSiemensVariantProtocol.PpiSerial or HslSiemensVariantProtocol.MpiSerial;
        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty;
        if (!serial && (host.Length == 0 || host.Contains("://", StringComparison.Ordinal)))
        {
            throw Invalid("SIEMENS_HOST_INVALID", "Siemens 设备 IP / 主机名无效");
        }

        var port = Integer(profile.Config, "port") ?? DefaultPort(protocol);
        if (!serial && port is < 1 or > 65535)
        {
            throw Invalid("SIEMENS_PORT_INVALID", "Siemens 端口必须在 1 到 65535 之间");
        }

        var portName = Text(profile.Config, "portName")?.Trim();
        if (serial && string.IsNullOrWhiteSpace(portName))
        {
            throw Invalid("SIEMENS_PORT_NAME_INVALID", "Siemens 串口名称不能为空");
        }

        var baudRate = Integer(profile.Config, "baudRate") ?? 9600;
        var dataBits = Integer(profile.Config, "dataBits") ?? 8;
        var stopBits = Integer(profile.Config, "stopBits") ?? 1;
        if (serial && (baudRate <= 0 || dataBits is < 5 or > 8 || stopBits is < 1 or > 2))
        {
            throw Invalid("SIEMENS_SERIAL_CONFIG_INVALID", "Siemens 串口参数无效");
        }

        return new(
            protocol,
            host,
            port,
            portName,
            baudRate,
            dataBits,
            ParseParity(Text(profile.Config, "parity") ?? "even"),
            stopBits == 2 ? HslSerialStopBits.Two : HslSerialStopBits.One,
            Byte(profile.Config, "station", 2),
            Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000),
            Boolean(profile.Config, "handshakeCheck") ?? true,
            Text(profile.Config, "userName")?.Trim() ?? (protocol == HslSiemensVariantProtocol.WebApi ? "admin" : null),
            Text(profile.Config, "password"),
            Boolean(profile.Config, "useHttps") ?? true);
    }

    public HslSiemensVariantClientOptions ToHslOptions() => new(
        Protocol,
        Host,
        Port,
        PortName,
        BaudRate,
        DataBits,
        Parity,
        StopBits,
        Station,
        ConnectTimeoutMilliseconds,
        ReceiveTimeoutMilliseconds,
        HandshakeCheck,
        UserName,
        Password,
        UseHttps);

    public string ServerName => Protocol switch
    {
        HslSiemensVariantProtocol.PpiSerial or HslSiemensVariantProtocol.MpiSerial => $"{PortName} / Station={Station}",
        HslSiemensVariantProtocol.PpiOverTcp => $"{Host}:{Port} / Station={Station}",
        _ => $"{Host}:{Port}",
    };

    private static int DefaultPort(HslSiemensVariantProtocol protocol) => protocol switch
    {
        HslSiemensVariantProtocol.PpiOverTcp => 2000,
        HslSiemensVariantProtocol.FetchWrite => 102,
        HslSiemensVariantProtocol.WebApi => 443,
        HslSiemensVariantProtocol.S7Plus => 102,
        _ => 0,
    };

    private static int Timeout(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 100 or > 120000) throw Invalid("SIEMENS_TIMEOUT_INVALID", "Siemens 超时配置必须在 100 到 120000 毫秒之间");
        return value;
    }

    private static byte Byte(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 0 or > 255) throw Invalid("SIEMENS_STATION_INVALID", "Siemens 站号必须在 0 到 255 之间");
        return checked((byte)value);
    }

    private static HslSerialParity ParseParity(string value) => value switch
    {
        "none" => HslSerialParity.None,
        "odd" => HslSerialParity.Odd,
        "even" => HslSerialParity.Even,
        _ => throw Invalid("SIEMENS_PARITY_INVALID", "Siemens 串口校验方式无效"),
    };

    private static string? Text(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;

    private static int? Integer(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;

    private static bool? Boolean(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : null;

    private static SiemensVariantDriverException Invalid(string code, string message) => new(code, message, false);
}
