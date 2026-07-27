using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.LsisFastEnet;

internal sealed record LsisFastEnetConnectionOptions(string Host, int Port, int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, HslLsisCpuType CpuType, string CompanyId, byte BaseNumber, byte SlotNumber, HslDataFormat DataFormat)
{
    public static LsisFastEnetConnectionOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "lsis", StringComparison.OrdinalIgnoreCase)) throw Invalid("LSIS_FAST_ENET_PROTOCOL_MISMATCH", "连接配置不是 LSIS Fast Enet 协议");
        if (profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("LSIS_FAST_ENET_CONFIG_INVALID", "LSIS Fast Enet 连接配置无效");
        var host = Text(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)) throw Invalid("LSIS_FAST_ENET_HOST_INVALID", "LSIS PLC IP / 主机名无效");
        var port = Number(profile.Config, "port") ?? 2004;
        if (port is < 1 or > 65535) throw Invalid("LSIS_FAST_ENET_PORT_INVALID", "LSIS Fast Enet 端口必须在 1 到 65535 之间");
        var baseNumber = ByteValue(profile.Config, "baseNumber", 0);
        var slotNumber = ByteValue(profile.Config, "slotNumber", 3);
        return new(host, port, Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 5000), ParseCpuType(Text(profile.Config, "cpuType") ?? "XGK"), ParseCompanyId(Text(profile.Config, "companyId") ?? "LSIS-XGT"), baseNumber, slotNumber, ParseDataFormat(Text(profile.Config, "dataFormat") ?? "DCBA"));
    }
    public HslLsisFastEnetClientOptions ToHslOptions() => new(Host, Port, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, CpuType, CompanyId, BaseNumber, SlotNumber, DataFormat);
    private static HslLsisCpuType ParseCpuType(string value) => value switch { "XGK" => HslLsisCpuType.XGK, "XGI" => HslLsisCpuType.XGI, "XGR" => HslLsisCpuType.XGR, "XGB_MK" => HslLsisCpuType.XgbMk, "XGB_IEC" => HslLsisCpuType.XgbIec, "XGB" => HslLsisCpuType.XGB, _ => throw Invalid("LSIS_FAST_ENET_CPU_TYPE_INVALID", "LSIS CPU 型号无效") };
    private static string ParseCompanyId(string value) => value is "LSIS-XGT" or "LGIS-GLOGA" or "MASTER-K" ? value : throw Invalid("LSIS_FAST_ENET_COMPANY_ID_INVALID", "LSIS 公司标识无效");
    private static HslDataFormat ParseDataFormat(string value) => value switch { "ABCD" => HslDataFormat.ABCD, "BADC" => HslDataFormat.BADC, "CDAB" => HslDataFormat.CDAB, "DCBA" => HslDataFormat.DCBA, _ => throw Invalid("LSIS_FAST_ENET_DATA_FORMAT_INVALID", "LSIS 数据格式无效") };
    private static byte ByteValue(JsonElement config, string name, int fallback) { var value = Number(config, name) ?? fallback; if (value is < 0 or > 255) throw Invalid("LSIS_FAST_ENET_CONFIG_INVALID", $"LSIS {name} 必须在 0 到 255 之间"); return checked((byte)value); }
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Number(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("LSIS_FAST_ENET_TIMEOUT_INVALID", $"LSIS {name} 必须在 100 到 120000 毫秒之间"); return value; }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Number(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static LsisFastEnetDriverException Invalid(string code, string message) => new(code, message, false);
}
