using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusRtu;

internal sealed record ModbusRtuConnectionOptions(
    string PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int TimeoutMilliseconds,
    HslDataFormat DataFormat,
    HslModbusSerialProtocol Protocol)
{
    public static ModbusRtuConnectionOptions Parse(
        ConnectionProfile profile,
        HslModbusSerialProtocol protocol = HslModbusSerialProtocol.Rtu)
    {
        if (!string.Equals(profile.ProtocolFamily, "modbus", StringComparison.OrdinalIgnoreCase))
        {
            throw new ModbusRtuDriverException("MODBUS_PROTOCOL_MISMATCH", "连接配置不是 Modbus 协议", retryable: false);
        }
        if (profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw new ModbusRtuDriverException("MODBUS_CONFIG_INVALID", "Modbus RTU 连接配置无效", retryable: false);
        }

        var portName = OptionalString(profile.Config, "portName")?.Trim();
        if (string.IsNullOrWhiteSpace(portName))
        {
            throw new ModbusRtuDriverException("MODBUS_RTU_PORT_REQUIRED", "Modbus RTU 串口名称不能为空", retryable: false);
        }

        var baudRate = OptionalInt32(profile.Config, "baudRate") ?? 9600;
        if (baudRate <= 0)
        {
            throw new ModbusRtuDriverException("MODBUS_RTU_BAUD_RATE_INVALID", "Modbus RTU 波特率必须大于 0", retryable: false);
        }

        var dataBits = OptionalInt32(profile.Config, "dataBits") ?? 8;
        if (dataBits is not (7 or 8))
        {
            throw new ModbusRtuDriverException("MODBUS_RTU_DATA_BITS_INVALID", "Modbus RTU 数据位只支持 7 或 8", retryable: false);
        }

        var parity = ParseParity(OptionalString(profile.Config, "parity") ?? "none");
        var stopBits = ParseStopBits(OptionalString(profile.Config, "stopBits") ?? "one");
        var timeoutMilliseconds = OptionalInt32(profile.Config, "timeoutMs") ?? 10000;
        if (timeoutMilliseconds is < 100 or > 120000)
        {
            throw new ModbusRtuDriverException("MODBUS_TIMEOUT_INVALID", "Modbus RTU 通信超时必须在 100 到 120000 毫秒之间", retryable: false);
        }

        var dataFormatText = OptionalString(profile.Config, "dataFormat") ?? "CDAB";
        if (!Enum.TryParse<HslDataFormat>(dataFormatText, ignoreCase: false, out var dataFormat))
        {
            throw new ModbusRtuDriverException("MODBUS_DATA_FORMAT_INVALID", "Modbus RTU 数据格式无效", retryable: false);
        }

        return new ModbusRtuConnectionOptions(
            portName,
            baudRate,
            dataBits,
            parity,
            stopBits,
            timeoutMilliseconds,
            dataFormat,
            protocol);
    }

    public HslModbusRtuClientOptions ToHslOptions() => new(
        PortName,
        BaudRate,
        DataBits,
        Parity,
        StopBits,
        TimeoutMilliseconds,
        DataFormat,
        Protocol);

    private static HslSerialParity ParseParity(string value) => value switch
    {
        "none" => HslSerialParity.None,
        "odd" => HslSerialParity.Odd,
        "even" => HslSerialParity.Even,
        _ => throw new ModbusRtuDriverException("MODBUS_RTU_PARITY_INVALID", "Modbus RTU 校验方式无效", retryable: false),
    };

    private static HslSerialStopBits ParseStopBits(string value) => value switch
    {
        "one" => HslSerialStopBits.One,
        "two" => HslSerialStopBits.Two,
        _ => throw new ModbusRtuDriverException("MODBUS_RTU_STOP_BITS_INVALID", "Modbus RTU 停止位无效", retryable: false),
    };

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String
            ? value.GetString()
            : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result)
            ? result
            : null;
}
