using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp;

internal sealed record SiemensS7TcpConnectionOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    HslSiemensPlc PlcType,
    byte Rack,
    byte Slot,
    int? LocalTsap,
    int? RemoteTsap)
{
    public static SiemensS7TcpConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "siemens", StringComparison.OrdinalIgnoreCase))
        {
            throw Invalid("S7_PROTOCOL_MISMATCH", "连接配置不是 Siemens S7 协议");
        }
        if (profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("S7_CONFIG_INVALID", "Siemens S7 TCP 连接配置无效");
        }

        var host = OptionalString(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
        {
            throw Invalid("S7_HOST_INVALID", "Siemens S7 TCP 设备 IP / 主机名无效");
        }

        var port = OptionalInt32(profile.Config, "port") ?? 102;
        if (port is < 1 or > 65535)
        {
            throw Invalid("S7_PORT_INVALID", "Siemens S7 TCP 端口必须在 1 到 65535 之间");
        }

        var timeoutMilliseconds = OptionalInt32(profile.Config, "connectTimeoutMs") ?? 5000;
        if (timeoutMilliseconds is < 100 or > 120000)
        {
            throw Invalid("S7_TIMEOUT_INVALID", "Siemens S7 TCP 连接超时必须在 100 到 120000 毫秒之间");
        }

        var plcTypeText = OptionalString(profile.Config, "plcType") ?? "S1200";
        if (!Enum.TryParse<HslSiemensPlc>(plcTypeText, ignoreCase: false, out var plcType))
        {
            throw Invalid("S7_PLC_TYPE_INVALID", "Siemens PLC 型号无效");
        }

        var rack = RequiredByte(profile.Config, "rack", 0, 7, "机架号");
        var slot = RequiredByte(profile.Config, "slot", 0, 31, "槽位号");
        var localTsap = OptionalInt32(profile.Config, "localTsap");
        var remoteTsap = OptionalInt32(profile.Config, "remoteTsap");
        ValidateTsap(localTsap, "本地 TSAP");
        ValidateTsap(remoteTsap, "远端 TSAP");

        return new SiemensS7TcpConnectionOptions(
            host,
            port,
            timeoutMilliseconds,
            plcType,
            rack,
            slot,
            localTsap,
            remoteTsap);
    }

    public HslS7TcpClientOptions ToHslOptions() => new(
        Host,
        Port,
        ConnectTimeoutMilliseconds,
        PlcType,
        Rack,
        Slot,
        LocalTsap,
        RemoteTsap);

    private static byte RequiredByte(JsonElement config, string name, int minimum, int maximum, string displayName)
    {
        var value = OptionalInt32(config, name);
        if (value is null || value < minimum || value > maximum)
        {
            throw Invalid("S7_CONFIG_INVALID", $"Siemens S7 TCP {displayName}必须在 {minimum} 到 {maximum} 之间");
        }
        return checked((byte)value.Value);
    }

    private static void ValidateTsap(int? value, string displayName)
    {
        if (value is < 0 or > 65535)
        {
            throw Invalid("S7_TSAP_INVALID", $"{displayName}必须在 0 到 65535 之间");
        }
    }

    private static string? OptionalString(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String
            ? value.GetString()
            : null;

    private static int? OptionalInt32(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result)
            ? result
            : null;

    private static SiemensS7TcpDriverException Invalid(string code, string message) =>
        new(code, message, retryable: false);
}
