using System.Globalization;
using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Vigor;
internal sealed partial record VigorAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    public static VigorAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String) throw Invalid("VIGOR_ADDRESS_INVALID", "丰炜地址必须包含 address 字段");
        var source = value.GetString()?.Trim() ?? string.Empty; var prefix = string.Empty;
        if (source.Contains(';')) { var parts = source.Split(';', 2); if (!parts[0].StartsWith("s=", StringComparison.OrdinalIgnoreCase) || !byte.TryParse(parts[0][2..], out var station)) throw Invalid("VIGOR_ADDRESS_INVALID", "丰炜地址站号覆盖格式无效"); prefix = $"s={station};"; source = parts[1]; }
        var address = source.ToUpperInvariant(); var match = AddressRegex().Match(address); if (!match.Success) throw Invalid("VIGOR_ADDRESS_INVALID", "丰炜 PLC 设备地址无效");
        if (point.ElementCount < 1) throw Invalid("VIGOR_ELEMENT_COUNT_INVALID", "丰炜元素数量必须大于 0");
        var type = ParseType(point.DataType); var area = match.Groups[1].Value; var bit = area is "X" or "Y" or "M" or "SM" or "S" or "TS" or "TC" or "CS" or "CC"; var word = area is "D" or "SD" or "R" or "T" or "C";
        if (type == HslValueType.Boolean && !bit) throw Invalid("VIGOR_BOOL_AREA_INVALID", "丰炜 bool 变量地址区不支持位读取"); if (type != HslValueType.Boolean && !word) throw Invalid("VIGOR_WORD_AREA_INVALID", "丰炜非 bool 变量地址区不支持字读取");
        if (area == "C") { var index = int.Parse(match.Groups[2].Value, CultureInfo.InvariantCulture); if (index is >= 200 and <= 255 && type is not (HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision)) throw Invalid("VIGOR_COUNTER_TYPE_INVALID", "丰炜 C200 至 C255 必须使用 32 位数据类型"); }
        var units = type == HslValueType.Boolean ? point.ElementCount : checked(Width(type) * point.ElementCount); var maximum = type == HslValueType.Boolean ? 1024 : 32; if (units > maximum) throw Invalid("VIGOR_ELEMENT_COUNT_TOO_LARGE", $"丰炜单变量读取最多 {maximum} 个协议单位");
        return new(prefix + address, type, point.ElementCount);
    }
    public HslVigorReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);
    private static int Width(HslValueType t) => t switch { HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision => 2, HslValueType.Signed64 or HslValueType.Unsigned64 or HslValueType.DoublePrecision => 4, _ => 1 };
    private static HslValueType ParseType(string t) => t switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw Invalid("VIGOR_DATA_TYPE_UNSUPPORTED", "丰炜数据类型不受支持") };
    private static VigorDriverException Invalid(string code, string message) => new(code, message, false);
    [GeneratedRegex("^(X|Y|M|SM|S|TS|TC|CS|CC|D|SD|R|T|C)([0-9]+)$", RegexOptions.CultureInvariant)] private static partial Regex AddressRegex();
}
