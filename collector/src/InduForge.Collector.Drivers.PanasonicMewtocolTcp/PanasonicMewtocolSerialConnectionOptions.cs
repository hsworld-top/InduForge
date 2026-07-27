using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.PanasonicMewtocolTcp;

internal sealed record PanasonicMewtocolSerialConnectionOptions(
    string PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int ReceiveTimeoutMilliseconds,
    byte Station)
{
    public static PanasonicMewtocolSerialConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "panasonic", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("PANASONIC_MEWTOCOL_CONFIG_INVALID", "松下 Mewtocol 串口连接配置无效");
        var portName = Text(profile.Config, "portName")?.Trim();
        if (string.IsNullOrWhiteSpace(portName)) throw Invalid("PANASONIC_MEWTOCOL_PORT_REQUIRED", "松下 Mewtocol 串口名称不能为空");
        var baudRate = Integer(profile.Config, "baudRate") ?? 9600;
        if (baudRate <= 0) throw Invalid("PANASONIC_MEWTOCOL_BAUD_RATE_INVALID", "松下 Mewtocol 波特率必须大于 0");
        var dataBits = Integer(profile.Config, "dataBits") ?? 8;
        if (dataBits is not (7 or 8)) throw Invalid("PANASONIC_MEWTOCOL_DATA_BITS_INVALID", "松下 Mewtocol 数据位只支持 7 或 8");
        var station = Integer(profile.Config, "station") ?? 238;
        if (station is < 0 or > 255) throw Invalid("PANASONIC_MEWTOCOL_STATION_INVALID", "松下 Mewtocol 站号必须在 0 到 255 之间");
        var timeout = Integer(profile.Config, "receiveTimeoutMs") ?? 10000;
        if (timeout is < 100 or > 120000) throw Invalid("PANASONIC_MEWTOCOL_TIMEOUT_INVALID", "松下 Mewtocol 接收超时必须在 100 到 120000 毫秒之间");
        return new(portName, baudRate, dataBits, ParseParity(Text(profile.Config, "parity") ?? "none"), ParseStopBits(Integer(profile.Config, "stopBits") ?? 1), timeout, checked((byte)station));
    }

    public HslPanasonicMewtocolClientOptions ToHslOptions() => new(
        "", 0, 5000, ReceiveTimeoutMilliseconds, Station, HslDataFormat.DCBA,
        HslPanasonicMewtocolTransport.Serial, PortName, BaudRate, DataBits, Parity, StopBits);

    private static HslSerialParity ParseParity(string value) => value switch
    {
        "none" => HslSerialParity.None,
        "odd" => HslSerialParity.Odd,
        "even" => HslSerialParity.Even,
        _ => throw Invalid("PANASONIC_MEWTOCOL_PARITY_INVALID", "松下 Mewtocol 校验方式无效"),
    };
    private static HslSerialStopBits ParseStopBits(int value) => value switch
    {
        1 => HslSerialStopBits.One,
        2 => HslSerialStopBits.Two,
        _ => throw Invalid("PANASONIC_MEWTOCOL_STOP_BITS_INVALID", "松下 Mewtocol 停止位无效"),
    };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static PanasonicMewtocolTcpDriverException Invalid(string code, string message) => new(code, message, false);
}
