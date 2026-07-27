using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MitsubishiMc3ETcp;

internal sealed record MitsubishiMc3ETcpAddress(
    string HslAddress,
    HslValueType ValueType,
    int ElementCount)
{
    public static MitsubishiMc3ETcpAddress Parse(PointReadRequest point)
    {
        ArgumentNullException.ThrowIfNull(point);
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var addressValue) ||
            addressValue.ValueKind != JsonValueKind.String)
        {
            throw Invalid("MELSEC_MC_ADDRESS_INVALID", "Mitsubishi MC 地址必须包含 address 字段");
        }

        var address = addressValue.GetString()?.Trim() ?? string.Empty;
        if (address.Length is < 1 or > 128 || address.Any(char.IsWhiteSpace))
        {
            throw Invalid("MELSEC_MC_ADDRESS_INVALID", "Mitsubishi MC 设备地址无效");
        }
        if (point.ElementCount < 1)
        {
            throw Invalid("MELSEC_MC_ELEMENT_COUNT_INVALID", "Mitsubishi MC 元素数量必须大于 0");
        }

        var valueType = ParseValueType(point.DataType);
        var protocolUnits = GetWordCount(valueType, point.ElementCount);
        var maximumUnits = valueType == HslValueType.Boolean ? 7168 : 960;
        if (protocolUnits > maximumUnits)
        {
            throw Invalid("MELSEC_MC_ELEMENT_COUNT_TOO_LARGE", $"Mitsubishi MC 单变量读取长度超过协议上限 {maximumUnits}");
        }
        return new MitsubishiMc3ETcpAddress(address, valueType, point.ElementCount);
    }

    public HslMelsecMcReadRequest ToHslRequest() => new(HslAddress, ValueType, ElementCount);

    private static long GetWordCount(HslValueType valueType, int count) => valueType switch
    {
        HslValueType.Boolean => count,
        HslValueType.Signed8 or HslValueType.Unsigned8 or HslValueType.Text or HslValueType.Binary => (count + 1L) / 2,
        HslValueType.Signed16 or HslValueType.Unsigned16 => count,
        HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision => count * 2L,
        HslValueType.Signed64 or HslValueType.Unsigned64 or HslValueType.DoublePrecision => count * 4L,
        _ => count,
    };

    private static HslValueType ParseValueType(string dataType) => dataType switch
    {
        "bool" => HslValueType.Boolean,
        "int8" => HslValueType.Signed8,
        "uint8" => HslValueType.Unsigned8,
        "int16" => HslValueType.Signed16,
        "uint16" => HslValueType.Unsigned16,
        "int32" => HslValueType.Signed32,
        "uint32" => HslValueType.Unsigned32,
        "int64" => HslValueType.Signed64,
        "uint64" => HslValueType.Unsigned64,
        "float32" => HslValueType.SinglePrecision,
        "float64" => HslValueType.DoublePrecision,
        "string" => HslValueType.Text,
        "bytes" => HslValueType.Binary,
        _ => throw Invalid("MELSEC_MC_DATA_TYPE_UNSUPPORTED", "Mitsubishi MC 数据类型不受支持"),
    };

    private static MitsubishiMc3ETcpDriverException Invalid(string code, string message) => new(code, message, retryable: false);
}
