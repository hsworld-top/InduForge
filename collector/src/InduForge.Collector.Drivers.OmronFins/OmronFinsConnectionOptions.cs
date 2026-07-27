using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins;

internal sealed record OmronFinsConnectionOptions(
    HslOmronFinsTransport Transport,
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslOmronPlcType PlcType,
    int ReadSplits,
    byte Gct,
    byte Sid,
    HslDataFormat DataFormat,
    bool StringReverseByteWord,
    bool ReceiveUntilEmpty)
{
    public static OmronFinsConnectionOptions Parse(ConnectionProfile profile, HslOmronFinsTransport transport)
    {
        if (!string.Equals(profile.ProtocolFamily, "omron", StringComparison.OrdinalIgnoreCase))
        {
            throw Invalid("OMRON_FINS_PROTOCOL_MISMATCH", "连接配置不是 Omron FINS 协议");
        }
        if (profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("OMRON_FINS_CONFIG_INVALID", "Omron FINS 连接配置无效");
        }

        var host = OptionalString(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
        {
            throw Invalid("OMRON_FINS_HOST_INVALID", "Omron FINS 设备 IP / 主机名无效");
        }
        var port = OptionalInt32(profile.Config, "port") ?? 9600;
        if (port is < 1 or > 65535)
        {
            throw Invalid("OMRON_FINS_PORT_INVALID", "Omron FINS 端口必须在 1 到 65535 之间");
        }
        var connectTimeout = Timeout(profile.Config, "connectTimeoutMs", 5000);
        var receiveTimeout = Timeout(profile.Config, "receiveTimeoutMs", 5000);
        var readSplits = OptionalInt32(profile.Config, "readSplits") ?? 500;
        if (readSplits is < 1 or > 999)
        {
            throw Invalid("OMRON_FINS_READ_SPLITS_INVALID", "Omron FINS 分段读取长度必须在 1 到 999 之间");
        }

        return new OmronFinsConnectionOptions(
            transport,
            host,
            port,
            connectTimeout,
            receiveTimeout,
            ParsePlcType(OptionalString(profile.Config, "plcType") ?? "CSCJ"),
            readSplits,
            ByteValue(profile.Config, "gct", 2),
            ByteValue(profile.Config, "sid", 0),
            ParseDataFormat(OptionalString(profile.Config, "dataFormat") ?? "CDAB"),
            OptionalBoolean(profile.Config, "stringReverseByteWord") ?? true,
            OptionalBoolean(profile.Config, "receiveUntilEmpty") ?? false);
    }

    public HslOmronFinsClientOptions ToHslOptions() => new(
        Transport,
        Host,
        Port,
        ConnectTimeoutMilliseconds,
        ReceiveTimeoutMilliseconds,
        PlcType,
        ReadSplits,
        Gct,
        Sid,
        DataFormat,
        StringReverseByteWord,
        ReceiveUntilEmpty);

    private static int Timeout(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 100 or > 120000)
        {
            throw Invalid("OMRON_FINS_TIMEOUT_INVALID", $"Omron FINS {name} 必须在 100 到 120000 毫秒之间");
        }
        return value;
    }

    private static byte ByteValue(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 0 or > 255)
        {
            throw Invalid("OMRON_FINS_CONFIG_INVALID", $"Omron FINS {name} 必须在 0 到 255 之间");
        }
        return checked((byte)value);
    }

    private static HslOmronPlcType ParsePlcType(string value) => value switch
    {
        "CSCJ" => HslOmronPlcType.CSCJ,
        "CV" => HslOmronPlcType.CV,
        _ => throw Invalid("OMRON_FINS_PLC_TYPE_INVALID", "Omron FINS PLC 类型只支持 CSCJ 或 CV"),
    };

    private static HslDataFormat ParseDataFormat(string value) => value switch
    {
        "ABCD" => HslDataFormat.ABCD,
        "BADC" => HslDataFormat.BADC,
        "CDAB" => HslDataFormat.CDAB,
        "DCBA" => HslDataFormat.DCBA,
        _ => throw Invalid("OMRON_FINS_DATA_FORMAT_INVALID", "Omron FINS 数据格式无效"),
    };

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result)
            ? result
            : null;

    private static bool? OptionalBoolean(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False
            ? value.GetBoolean()
            : null;

    private static OmronFinsDriverException Invalid(string code, string message) => new(code, message, retryable: false);
}
