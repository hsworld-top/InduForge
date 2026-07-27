using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins;

internal sealed partial record OmronVariantAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static OmronVariantAddress Parse(PointReadRequest point, bool cMode)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("OMRON_ADDRESS_INVALID", "欧姆龙地址必须包含 address 字段");
        var address = value.GetString()?.Trim() ?? string.Empty;
        if (address.Length is < 1 or > 512 || address.Any(char.IsWhiteSpace) || (cMode && !CModePattern().IsMatch(address)))
            throw Invalid("OMRON_ADDRESS_INVALID", "欧姆龙设备地址无效");
        if (point.ElementCount is < 1 or > ushort.MaxValue) throw Invalid("OMRON_ELEMENT_COUNT_INVALID", "欧姆龙元素数量无效");
        if (point.DataType == "datetime" && point.ElementCount != 1) throw Invalid("OMRON_ELEMENT_COUNT_INVALID", "欧姆龙 datetime 标签只支持单元素读取");
        var dataType = point.DataType switch
        {
            "bool" => HslValueType.Boolean, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16,
            "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64,
            "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision,
            "string" => HslValueType.Text, "datetime" when !cMode => HslValueType.DateTime,
            _ => throw Invalid("OMRON_DATA_TYPE_UNSUPPORTED", "当前欧姆龙协议不支持该数据类型"),
        };
        return new(address, dataType, point.ElementCount);
    }
    public HslOmronVariantReadRequest ToHslRequest() => new(Address, DataType, ElementCount);
    [GeneratedRegex(@"^(?:D|DM|C|CIO|LR|H|HR|A|AR|TIM|CNT)\d+(?:\.\d+)?$|^(?:E|EM)[0-9A-Fa-f]+\.\d+(?:\.\d+)?$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)]
    private static partial Regex CModePattern();
    private static OmronFinsDriverException Invalid(string code, string message) => new(code, message, false);
}
