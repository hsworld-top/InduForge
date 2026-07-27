using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Xinje;

internal sealed partial record XinjeAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    public static XinjeAddress Parse(PointReadRequest point, XinjeConnectionOptions options)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String) throw Invalid("XINJE_ADDRESS_INVALID", "信捷地址必须包含 address 字段");
        var source = value.GetString()?.Trim() ?? string.Empty;
        var prefix = string.Empty;
        if (source.Contains(';'))
        {
            var parts = source.Split(';', 2);
            if (!parts[0].StartsWith("s=", StringComparison.OrdinalIgnoreCase) || !byte.TryParse(parts[0][2..], out var station)) throw Invalid("XINJE_ADDRESS_INVALID", "信捷地址站号覆盖格式无效");
            prefix = $"s={station};"; source = parts[1];
        }
        var address = source.ToUpperInvariant();
        if (address.Length < 2 || !AddressRegex().IsMatch(address)) throw Invalid("XINJE_ADDRESS_INVALID", "信捷 PLC 设备地址无效");
        if (point.ElementCount < 1) throw Invalid("XINJE_ELEMENT_COUNT_INVALID", "信捷元素数量必须大于 0");
        var type = ParseValueType(point.DataType);
        var area = AreaRegex().Match(address).Value;
        var bitAreas = options.Kind == XinjeDriverKind.InternalTcp
            ? InternalBitAreas
            : options.Series == HslXinjeSeries.XC ? XcBitAreas : XdBitAreas;
        var wordAreas = options.Kind == XinjeDriverKind.InternalTcp
            ? InternalWordAreas
            : options.Series == HslXinjeSeries.XC ? XcWordAreas : XdWordAreas;
        if (type == HslValueType.Boolean && !bitAreas.Contains(area)) throw Invalid("XINJE_BOOL_AREA_INVALID", "信捷 bool 变量地址区不支持位读取");
        if (type != HslValueType.Boolean && !wordAreas.Contains(area)) throw Invalid("XINJE_WORD_AREA_INVALID", "信捷数值变量地址区不支持字读取");
        var units = type == HslValueType.Boolean ? point.ElementCount : checked(WordWidth(type) * point.ElementCount);
        var maximum = options.Kind == XinjeDriverKind.InternalTcp ? ushort.MaxValue : type == HslValueType.Boolean ? 2000 : 125;
        if (units > maximum) throw Invalid("XINJE_ELEMENT_COUNT_TOO_LARGE", $"信捷单变量读取最多 {maximum} 个协议单位");
        return new(prefix + address, type, point.ElementCount);
    }
    public HslXinjeReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);
    private static readonly HashSet<string> XcBitAreas = ["X", "Y", "S", "M", "T", "C"];
    private static readonly HashSet<string> XcWordAreas = ["D", "F", "E", "T", "C"];
    private static readonly HashSet<string> XdBitAreas = ["X", "Y", "S", "M", "SM", "T", "C", "ET", "SEM", "HM", "HS", "HT", "HC", "HSC"];
    private static readonly HashSet<string> XdWordAreas = ["D", "ID", "QD", "SD", "TD", "CD", "ETD", "HD", "HSD", "HTD", "HCD", "HSCD", "FD", "SFD", "FS"];
    private static readonly HashSet<string> InternalBitAreas = ["M", "X", "Y", "SM", "T", "C", "HM", "HS", "HT", "HSC"];
    private static readonly HashSet<string> InternalWordAreas = ["D", "SD", "TD", "CD", "HD", "FD", "ETD", "HTD", "HCD", "HSD"];
    private static int WordWidth(HslValueType type) => type switch { HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision => 2, HslValueType.Signed64 or HslValueType.Unsigned64 or HslValueType.DoublePrecision => 4, _ => 1 };
    private static HslValueType ParseValueType(string type) => type switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw Invalid("XINJE_DATA_TYPE_UNSUPPORTED", "信捷数据类型不受支持") };
    private static XinjeDriverException Invalid(string code, string message) => new(code, message, false);
    [GeneratedRegex("^[A-Z]{1,4}[0-9]+$", RegexOptions.CultureInvariant)] private static partial Regex AddressRegex();
    [GeneratedRegex("^[A-Z]+", RegexOptions.CultureInvariant)] private static partial Regex AreaRegex();
}
