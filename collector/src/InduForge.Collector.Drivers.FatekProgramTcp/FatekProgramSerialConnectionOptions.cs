using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.FatekProgramTcp;

internal sealed record FatekProgramSerialConnectionOptions(
    string PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int ReceiveTimeoutMilliseconds,
    byte Station)
{
    public static FatekProgramSerialConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "fatek", StringComparison.OrdinalIgnoreCase) ||
            profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("FATEK_PROGRAM_SERIAL_CONFIG_INVALID", "永宏编程口串口连接配置无效");
        }

        var portName = Text(profile.Config, "portName")?.Trim();
        if (string.IsNullOrWhiteSpace(portName))
        {
            throw Invalid("FATEK_PROGRAM_SERIAL_PORT_NAME_INVALID", "永宏编程口串口名称不能为空");
        }

        var baudRate = Integer(profile.Config, "baudRate") ?? 9600;
        var dataBits = Integer(profile.Config, "dataBits") ?? 7;
        var stopBits = Integer(profile.Config, "stopBits") ?? 2;
        if (baudRate <= 0 || dataBits is < 5 or > 8 || stopBits is < 1 or > 2)
        {
            throw Invalid("FATEK_PROGRAM_SERIAL_CONFIG_INVALID", "永宏编程口串口参数无效");
        }

        var station = Integer(profile.Config, "station") ?? 1;
        if (station is < 0 or > 255)
        {
            throw Invalid("FATEK_PROGRAM_SERIAL_STATION_INVALID", "永宏 PLC 站号必须在 0 到 255 之间");
        }

        return new(
            portName,
            baudRate,
            dataBits,
            ParseParity(Text(profile.Config, "parity") ?? "even"),
            stopBits == 2 ? HslSerialStopBits.Two : HslSerialStopBits.One,
            Timeout(profile.Config, "receiveTimeoutMs", 10000),
            checked((byte)station));
    }

    public HslFatekProgramSerialClientOptions ToHslOptions() =>
        new(PortName, BaudRate, DataBits, Parity, StopBits, ReceiveTimeoutMilliseconds, Station);

    private static int Timeout(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 100 or > 120000)
        {
            throw Invalid("FATEK_PROGRAM_SERIAL_TIMEOUT_INVALID", "永宏编程口串口接收超时必须在 100 到 120000 毫秒之间");
        }
        return value;
    }

    private static HslSerialParity ParseParity(string value) => value switch
    {
        "none" => HslSerialParity.None,
        "odd" => HslSerialParity.Odd,
        "even" => HslSerialParity.Even,
        _ => throw Invalid("FATEK_PROGRAM_SERIAL_PARITY_INVALID", "永宏编程口串口校验方式无效"),
    };

    private static string? Text(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;

    private static int? Integer(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;

    private static FatekProgramDriverException Invalid(string code, string message) => new(code, message, false);
}
