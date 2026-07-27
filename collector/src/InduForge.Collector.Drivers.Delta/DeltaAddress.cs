using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Delta;

internal sealed partial record DeltaAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    public static DeltaAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("DELTA_ADDRESS_INVALID", "台达地址必须包含 address 字段");
        var source = value.GetString()?.Trim() ?? string.Empty;
        var stationPrefix = string.Empty;
        if (source.Contains(';'))
        {
            var parts = source.Split(';', 2);
            if (!parts[0].StartsWith("s=", StringComparison.OrdinalIgnoreCase) || !byte.TryParse(parts[0][2..], out var station))
                throw Invalid("DELTA_ADDRESS_INVALID", "台达地址站号覆盖格式无效");
            stationPrefix = $"s={station};";
            source = parts[1];
        }
        var address = source.ToUpperInvariant();
        if (address.Length < 2 || !AddressRegex().IsMatch(address)) throw Invalid("DELTA_ADDRESS_INVALID", "台达 PLC 设备地址无效");
        if (point.ElementCount < 1) throw Invalid("DELTA_ELEMENT_COUNT_INVALID", "台达元素数量必须大于 0");
        var valueType = ParseValueType(point.DataType);
        var area = AreaRegex().Match(address).Value;
        var bitArea = area is "X" or "Y" or "M" or "SM" or "S" or "T" or "C" or "HC";
        if (valueType == HslValueType.Boolean && !bitArea) throw Invalid("DELTA_BOOL_AREA_INVALID", "台达 bool 变量地址区不支持位读取");
        var units = valueType == HslValueType.Boolean ? point.ElementCount : checked(WordWidth(valueType) * point.ElementCount);
        var maximum = valueType == HslValueType.Boolean ? 2000 : 125;
        if (units > maximum) throw Invalid("DELTA_ELEMENT_COUNT_TOO_LARGE", $"台达单变量读取最多 {maximum} 个协议单位");
        return new(stationPrefix + address, valueType, point.ElementCount);
    }

    public HslDeltaReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);
    private static int WordWidth(HslValueType type) => type switch { HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision => 2, HslValueType.Signed64 or HslValueType.Unsigned64 or HslValueType.DoublePrecision => 4, _ => 1 };
    private static HslValueType ParseValueType(string type) => type switch
    {
        "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8,
        "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32,
        "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64,
        "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary,
        _ => throw Invalid("DELTA_DATA_TYPE_UNSUPPORTED", "台达数据类型不受支持"),
    };
    private static DeltaDriverException Invalid(string code, string message) => new(code, message, false);
    [GeneratedRegex("^(?:X|Y|M|SM|S|SR|D|T|C|HC|E)[0-9]+(?:\\.[0-9]+)?$", RegexOptions.CultureInvariant)] private static partial Regex AddressRegex();
    [GeneratedRegex("^[A-Z]+", RegexOptions.CultureInvariant)] private static partial Regex AreaRegex();
}
