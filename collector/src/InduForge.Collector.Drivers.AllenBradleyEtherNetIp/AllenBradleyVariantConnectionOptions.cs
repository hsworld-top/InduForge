using System.Globalization;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.AllenBradleyEtherNetIp;

internal sealed record AllenBradleyVariantConnectionOptions(
    HslAllenBradleyProtocol Protocol,
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte Slot,
    IReadOnlyList<byte>? PortSlot,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    byte Station,
    byte DestinationNode,
    byte SourceNode,
    HslAllenBradleyCheckType CheckType,
    HslDataFormat DataFormat)
{
    public static AllenBradleyVariantConnectionOptions Parse(ConnectionProfile profile, HslAllenBradleyProtocol protocol)
    {
        if (!string.Equals(profile.ProtocolFamily, "allen-bradley", StringComparison.OrdinalIgnoreCase))
        {
            throw Invalid("ALLEN_BRADLEY_PROTOCOL_MISMATCH", "连接配置不是 Allen-Bradley 协议");
        }
        if (profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("ALLEN_BRADLEY_CONFIG_INVALID", "Allen-Bradley 连接配置无效");
        }

        var serial = protocol == HslAllenBradleyProtocol.Df1Serial;
        var host = OptionalString(profile.Config, "host")?.Trim() ?? string.Empty;
        if (!serial && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)))
        {
            throw Invalid("ALLEN_BRADLEY_HOST_INVALID", "Allen-Bradley 设备 IP / 主机名无效");
        }

        var port = OptionalInt32(profile.Config, "port") ?? 44818;
        if (!serial && port is < 1 or > 65535)
        {
            throw Invalid("ALLEN_BRADLEY_PORT_INVALID", "Allen-Bradley 端口必须在 1 到 65535 之间");
        }

        var slot = Byte(profile.Config, "slot", 0, "槽位号");
        var portName = OptionalString(profile.Config, "portName")?.Trim();
        if (serial && string.IsNullOrWhiteSpace(portName))
        {
            throw Invalid("ALLEN_BRADLEY_PORT_NAME_INVALID", "Allen-Bradley DF1 串口名称不能为空");
        }

        return new AllenBradleyVariantConnectionOptions(
            protocol,
            host,
            port,
            Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000),
            slot,
            protocol == HslAllenBradleyProtocol.MicroCip ? ParsePortSlot(OptionalString(profile.Config, "portSlot") ?? "01 00") : null,
            portName,
            SerialInt(profile.Config, "baudRate", 9600, 1200, 115200),
            SerialInt(profile.Config, "dataBits", 8, 7, 8),
            ParseParity(OptionalString(profile.Config, "parity") ?? "none"),
            ParseStopBits(OptionalInt32(profile.Config, "stopBits") ?? 1),
            Byte(profile.Config, "station", 1, "站号"),
            Byte(profile.Config, "destinationNode", 1, "目标节点"),
            Byte(profile.Config, "sourceNode", 2, "来源节点"),
            ParseCheckType(OptionalString(profile.Config, "checkType") ?? "crc16"),
            ParseDataFormat(OptionalString(profile.Config, "dataFormat") ?? "DCBA"));
    }

    public HslAllenBradleyClientOptions ToHslOptions() => new(
        Host,
        Port,
        ConnectTimeoutMilliseconds,
        ReceiveTimeoutMilliseconds,
        Slot,
        null,
        false,
        true,
        DataFormat,
        Protocol,
        PortName,
        BaudRate,
        DataBits,
        Parity,
        StopBits,
        Station,
        DestinationNode,
        SourceNode,
        CheckType,
        PortSlot);

    public string ServerName => Protocol == HslAllenBradleyProtocol.Df1Serial ? PortName! : $"{Host}:{Port}";

    private static byte[] ParsePortSlot(string value)
    {
        var parts = value.Split([' ', '-', ':'], StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);
        if (parts.Length is < 2 or > 32)
        {
            throw Invalid("ALLEN_BRADLEY_PORT_SLOT_INVALID", "MicroCIP PortSlot 必须包含 2 到 32 个十六进制字节");
        }
        try
        {
            return parts.Select(part => byte.Parse(part, NumberStyles.AllowHexSpecifier, CultureInfo.InvariantCulture)).ToArray();
        }
        catch (FormatException)
        {
            throw Invalid("ALLEN_BRADLEY_PORT_SLOT_INVALID", "MicroCIP PortSlot 必须使用十六进制字节，例如 01 00");
        }
    }

    private static int Timeout(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 100 or > 120000)
        {
            throw Invalid("ALLEN_BRADLEY_TIMEOUT_INVALID", $"Allen-Bradley {name} 必须在 100 到 120000 毫秒之间");
        }
        return value;
    }

    private static byte Byte(JsonElement config, string name, byte defaultValue, string displayName)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < byte.MinValue or > byte.MaxValue)
        {
            throw Invalid("ALLEN_BRADLEY_BYTE_OPTION_INVALID", $"Allen-Bradley {displayName}必须在 0 到 255 之间");
        }
        return checked((byte)value);
    }

    private static int SerialInt(JsonElement config, string name, int defaultValue, int minimum, int maximum)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value < minimum || value > maximum)
        {
            throw Invalid("ALLEN_BRADLEY_SERIAL_OPTION_INVALID", $"Allen-Bradley {name} 配置无效");
        }
        return value;
    }

    private static HslSerialParity ParseParity(string value) => value switch
    {
        "none" => HslSerialParity.None,
        "odd" => HslSerialParity.Odd,
        "even" => HslSerialParity.Even,
        _ => throw Invalid("ALLEN_BRADLEY_PARITY_INVALID", "Allen-Bradley 串口校验方式无效"),
    };

    private static HslSerialStopBits ParseStopBits(int value) => value switch
    {
        1 => HslSerialStopBits.One,
        2 => HslSerialStopBits.Two,
        _ => throw Invalid("ALLEN_BRADLEY_STOP_BITS_INVALID", "Allen-Bradley 串口停止位无效"),
    };

    private static HslAllenBradleyCheckType ParseCheckType(string value) => value.ToLowerInvariant() switch
    {
        "bcc" => HslAllenBradleyCheckType.Bcc,
        "crc16" => HslAllenBradleyCheckType.Crc16,
        _ => throw Invalid("ALLEN_BRADLEY_CHECK_TYPE_INVALID", "Allen-Bradley DF1 校验类型无效"),
    };

    private static HslDataFormat ParseDataFormat(string value) => value switch
    {
        "ABCD" => HslDataFormat.ABCD,
        "BADC" => HslDataFormat.BADC,
        "CDAB" => HslDataFormat.CDAB,
        "DCBA" => HslDataFormat.DCBA,
        _ => throw Invalid("ALLEN_BRADLEY_DATA_FORMAT_INVALID", "Allen-Bradley 数据格式无效"),
    };

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;

    private static AllenBradleyEtherNetIpDriverException Invalid(string code, string message) => new(code, message, false);
}
