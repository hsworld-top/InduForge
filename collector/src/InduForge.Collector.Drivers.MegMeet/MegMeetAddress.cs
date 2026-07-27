using System.Text.Json;
using System.Text.RegularExpressions;
using System.Globalization;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MegMeet;

internal sealed partial record MegMeetAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    public static MegMeetAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("MEGMEET_ADDRESS_INVALID", "麦格米特地址必须包含 address 字段");

        var source = value.GetString()?.Trim() ?? string.Empty;
        var stationPrefix = string.Empty;
        if (source.Contains(';'))
        {
            var parts = source.Split(';', 2);
            if (!parts[0].StartsWith("s=", StringComparison.OrdinalIgnoreCase) || !byte.TryParse(parts[0][2..], out var station))
                throw Invalid("MEGMEET_ADDRESS_INVALID", "麦格米特地址站号覆盖格式无效");
            stationPrefix = $"s={station};";
            source = parts[1];
        }

        var address = source.ToUpperInvariant();
        var match = AddressRegex().Match(address);
        if (!match.Success)
            throw Invalid("MEGMEET_ADDRESS_INVALID", "麦格米特 PLC 设备地址无效");
        if (point.ElementCount < 1)
            throw Invalid("MEGMEET_ELEMENT_COUNT_INVALID", "麦格米特元素数量必须大于 0");

        var valueType = ParseValueType(point.DataType);
        var area = match.Groups[1].Value;
        var index = int.Parse(match.Groups[2].Value, CultureInfo.InvariantCulture);
        var bitArea = area is "X" or "Y" or "M" or "SM" or "S" or "T" or "C";
        var wordArea = area is "D" or "SD" or "Z" or "R" or "T" or "C";
        if (valueType == HslValueType.Boolean && !bitArea)
            throw Invalid("MEGMEET_BOOL_AREA_INVALID", "麦格米特 bool 变量地址区不支持位读取");
        if (valueType != HslValueType.Boolean && !wordArea)
            throw Invalid("MEGMEET_WORD_AREA_INVALID", "麦格米特非 bool 变量地址区不支持字读取");
        if (area == "C" && index >= 200 && valueType is not (HslValueType.Signed32 or HslValueType.Unsigned32))
            throw Invalid("MEGMEET_COUNTER_TYPE_INVALID", "麦格米特 C200 及以上计数器必须使用 int32 或 uint32");

        var units = valueType == HslValueType.Boolean ? point.ElementCount : checked(WordWidth(valueType) * point.ElementCount);
        var maximum = valueType == HslValueType.Boolean ? 2000 : 125;
        if (units > maximum)
            throw Invalid("MEGMEET_ELEMENT_COUNT_TOO_LARGE", $"麦格米特单变量读取最多 {maximum} 个协议单位");
        return new(stationPrefix + address, valueType, point.ElementCount);
    }

    public HslMegMeetReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);

    private static int WordWidth(HslValueType type) => type switch
    {
        HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision => 2,
        HslValueType.Signed64 or HslValueType.Unsigned64 or HslValueType.DoublePrecision => 4,
        _ => 1,
    };

    private static HslValueType ParseValueType(string type) => type switch
    {
        "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8,
        "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32,
        "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64,
        "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision,
        "string" => HslValueType.Text, "bytes" => HslValueType.Binary,
        _ => throw Invalid("MEGMEET_DATA_TYPE_UNSUPPORTED", "麦格米特数据类型不受支持"),
    };

    private static MegMeetDriverException Invalid(string code, string message) => new(code, message, false);

    [GeneratedRegex("^(X|Y|M|SM|S|D|SD|Z|R|T|C)([0-9]+)(?:\\.[0-9]+)?$", RegexOptions.CultureInvariant)]
    private static partial Regex AddressRegex();
}
