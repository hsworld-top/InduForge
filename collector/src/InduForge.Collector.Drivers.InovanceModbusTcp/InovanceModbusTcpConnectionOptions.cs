using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InovanceModbusTcp;

internal sealed record InovanceModbusTcpConnectionOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte Station,
    HslInovanceSeries Series,
    bool AddressStartWithZero,
    HslDataFormat DataFormat,
    bool StringReverse)
{
    public static InovanceModbusTcpConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "inovance", StringComparison.OrdinalIgnoreCase))
            throw Invalid("INOVANCE_MODBUS_TCP_PROTOCOL_MISMATCH", "连接配置不是汇川 Modbus TCP 协议");
        if (profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("INOVANCE_MODBUS_TCP_CONFIG_INVALID", "汇川 Modbus TCP 连接配置无效");

        var host = OptionalString(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
            throw Invalid("INOVANCE_MODBUS_TCP_HOST_INVALID", "汇川 PLC IP / 主机名无效");
        var port = OptionalInt32(profile.Config, "port") ?? 502;
        if (port is < 1 or > 65535)
            throw Invalid("INOVANCE_MODBUS_TCP_PORT_INVALID", "汇川 Modbus TCP 端口必须在 1 到 65535 之间");
        var station = OptionalInt32(profile.Config, "station") ?? 1;
        if (station is < 0 or > 255)
            throw Invalid("INOVANCE_MODBUS_TCP_STATION_INVALID", "汇川 PLC 站号必须在 0 到 255 之间");

        return new InovanceModbusTcpConnectionOptions(
            host,
            port,
            Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000),
            checked((byte)station),
            ParseSeries(OptionalString(profile.Config, "series") ?? "AM"),
            OptionalBoolean(profile.Config, "addressStartWithZero") ?? true,
            ParseDataFormat(OptionalString(profile.Config, "dataFormat") ?? "CDAB"),
            OptionalBoolean(profile.Config, "stringReverse") ?? false);
    }

    public HslInovanceModbusTcpClientOptions ToHslOptions() => new(
        Host, Port, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, Station, Series,
        AddressStartWithZero, DataFormat, StringReverse);

    private static int Timeout(JsonElement config, string name, int defaultValue)
    {
        var value = OptionalInt32(config, name) ?? defaultValue;
        if (value is < 100 or > 120000)
            throw Invalid("INOVANCE_MODBUS_TCP_TIMEOUT_INVALID", $"汇川 Modbus TCP {name} 必须在 100 到 120000 毫秒之间");
        return value;
    }

    private static HslInovanceSeries ParseSeries(string value) => value switch
    {
        "AM" => HslInovanceSeries.AM,
        "H3U" => HslInovanceSeries.H3U,
        "H5U" => HslInovanceSeries.H5U,
        "Easy" => HslInovanceSeries.Easy,
        _ => throw Invalid("INOVANCE_MODBUS_TCP_SERIES_INVALID", "汇川 PLC 系列无效"),
    };

    private static HslDataFormat ParseDataFormat(string value) => value switch
    {
        "ABCD" => HslDataFormat.ABCD,
        "BADC" => HslDataFormat.BADC,
        "CDAB" => HslDataFormat.CDAB,
        "DCBA" => HslDataFormat.DCBA,
        _ => throw Invalid("INOVANCE_MODBUS_TCP_DATA_FORMAT_INVALID", "汇川 Modbus TCP 数据格式无效"),
    };

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;

    private static bool? OptionalBoolean(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : null;

    private static InovanceModbusTcpDriverException Invalid(string code, string message) => new(code, message, false);
}
