using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Fuji;

internal sealed partial record FujiAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    public static FujiAddress Parse(PointReadRequest point, FujiDriverKind kind)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("FUJI_ADDRESS_INVALID", "富士地址必须包含 address 字段");
        var source = value.GetString()?.Trim() ?? string.Empty;
        var stationPrefix = string.Empty;
        if (source.Contains(';'))
        {
            if (kind is not (FujiDriverKind.SpbOverTcp or FujiDriverKind.SpbSerial)) throw Invalid("FUJI_ADDRESS_INVALID", "当前富士协议不支持地址内覆盖站号");
            var parts = source.Split(';', 2);
            if (!parts[0].StartsWith("s=", StringComparison.OrdinalIgnoreCase) || !byte.TryParse(parts[0][2..], out var station)) throw Invalid("FUJI_ADDRESS_INVALID", "富士地址站号覆盖格式无效");
            stationPrefix = $"s={station};"; source = parts[1];
        }
        var address = source.ToUpperInvariant();
        if (!Matches(kind, address)) throw Invalid("FUJI_ADDRESS_INVALID", "富士 PLC 设备地址无效");
        if (point.ElementCount < 1) throw Invalid("FUJI_ELEMENT_COUNT_INVALID", "富士元素数量必须大于 0");
        var valueType = ParseValueType(point.DataType);
        var units = valueType == HslValueType.Boolean ? point.ElementCount : checked(WordWidth(valueType) * point.ElementCount);
        var maximum = kind is FujiDriverKind.SpbOverTcp or FujiDriverKind.SpbSerial
            ? valueType == HslValueType.Boolean ? 1680 : 105
            : valueType == HslValueType.Boolean ? 2000 : 125;
        if (units > maximum) throw Invalid("FUJI_ELEMENT_COUNT_TOO_LARGE", $"富士单变量读取最多 {maximum} 个协议单位");
        return new(stationPrefix + address, valueType, point.ElementCount);
    }

    public HslFujiReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);
    private static bool Matches(FujiDriverKind kind, string address) => kind switch
    {
        FujiDriverKind.CommandSettingTypeTcp => CommandAddressRegex().IsMatch(address),
        FujiDriverKind.SphTcp => SphAddressRegex().IsMatch(address),
        FujiDriverKind.SpbOverTcp or FujiDriverKind.SpbSerial => SpbAddressRegex().IsMatch(address),
        _ => false,
    };
    private static int WordWidth(HslValueType type) => type switch { HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision => 2, HslValueType.Signed64 or HslValueType.Unsigned64 or HslValueType.DoublePrecision => 4, _ => 1 };
    private static HslValueType ParseValueType(string type) => type switch
    {
        "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8,
        "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32,
        "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64,
        "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary,
        _ => throw Invalid("FUJI_DATA_TYPE_UNSUPPORTED", "富士数据类型不受支持"),
    };
    private static FujiDriverException Invalid(string code, string message) => new(code, message, false);

    [GeneratedRegex("^(?:B|M|K|F|A|D|S|W|TS|TR|CS|CR|BD|WL)[0-9]+(?:\\.[0-9]+)?$", RegexOptions.CultureInvariant)] private static partial Regex CommandAddressRegex();
    [GeneratedRegex("^(?:M(?:1|3|10)\\.[0-9]+(?:\\.[0-9]+)?|[IQ][0-9]+(?:\\.[0-9]+)?)$", RegexOptions.CultureInvariant)] private static partial Regex SphAddressRegex();
    [GeneratedRegex("^(?:X|Y|L|M|D|TN|CN|TC|CC|R|W)[0-9]+(?:\\.[0-9]+)?$", RegexOptions.CultureInvariant)] private static partial Regex SpbAddressRegex();
}
