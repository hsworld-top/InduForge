using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusTcp;

internal sealed record ModbusTcpConnectionOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat,
    HslModbusNetworkProtocol Protocol)
{
    public static ModbusTcpConnectionOptions Parse(
        ConnectionProfile profile,
        HslModbusNetworkProtocol protocol = HslModbusNetworkProtocol.Tcp)
    {
        if (!string.Equals(profile.ProtocolFamily, "modbus", StringComparison.OrdinalIgnoreCase))
        {
            throw new ModbusTcpDriverException("MODBUS_PROTOCOL_MISMATCH", "连接配置不是 Modbus 协议", retryable: false);
        }
        if (profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw new ModbusTcpDriverException("MODBUS_CONFIG_INVALID", "Modbus TCP 连接配置无效", retryable: false);
        }

        var host = OptionalString(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
        {
            throw new ModbusTcpDriverException("MODBUS_HOST_INVALID", "Modbus TCP 设备 IP / 主机名无效", retryable: false);
        }

        var port = OptionalInt32(profile.Config, "port") ?? 502;
        if (port is < 1 or > 65535)
        {
            throw new ModbusTcpDriverException("MODBUS_PORT_INVALID", "Modbus TCP 端口必须在 1 到 65535 之间", retryable: false);
        }

        var timeoutMilliseconds = OptionalInt32(profile.Config, "connectTimeoutMs") ?? 5000;
        if (timeoutMilliseconds is < 100 or > 120000)
        {
            throw new ModbusTcpDriverException("MODBUS_TIMEOUT_INVALID", "Modbus TCP 连接超时必须在 100 到 120000 毫秒之间", retryable: false);
        }

        var receiveTimeoutMilliseconds = OptionalInt32(profile.Config, "receiveTimeoutMs") ?? 10000;
        if (receiveTimeoutMilliseconds is < 100 or > 120000)
        {
            throw new ModbusTcpDriverException("MODBUS_RECEIVE_TIMEOUT_INVALID", "Modbus 网络接收超时必须在 100 到 120000 毫秒之间", retryable: false);
        }

        var dataFormatText = OptionalString(profile.Config, "dataFormat") ?? "CDAB";
        if (!Enum.TryParse<HslDataFormat>(dataFormatText, ignoreCase: false, out var dataFormat))
        {
            throw new ModbusTcpDriverException("MODBUS_DATA_FORMAT_INVALID", "Modbus TCP 数据格式无效", retryable: false);
        }

        return new ModbusTcpConnectionOptions(host, port, timeoutMilliseconds, receiveTimeoutMilliseconds, dataFormat, protocol);
    }

    public HslModbusTcpClientOptions ToHslOptions() => new(
        Host,
        Port,
        ConnectTimeoutMilliseconds,
        DataFormat,
        Protocol,
        ReceiveTimeoutMilliseconds);

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String
            ? value.GetString()
            : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result)
            ? result
            : null;
}
