using System.Globalization;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.FatekProgramTcp;

internal sealed record FatekProgramAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    private static readonly string[] BitAreas = ["M", "X", "Y", "S", "T", "C"];
    private static readonly string[] WordAreas = ["RT", "RC", "D", "R"];

    public static FatekProgramAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("FATEK_PROGRAM_ADDRESS_INVALID", "永宏编程口地址必须包含 address 字段");
        var address = Normalize(value.GetString(), point.DataType == "bool");
        if (point.ElementCount is < 1 or > ushort.MaxValue)
            throw Invalid("FATEK_PROGRAM_ELEMENT_COUNT_INVALID", "永宏编程口元素数量必须在 1 到 65535 之间");
        return new(address, ParseValueType(point.DataType), point.ElementCount);
    }

    public HslFatekProgramReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);

    private static string Normalize(string? value, bool boolean)
    {
        var address = value?.Trim() ?? string.Empty;
        if (address.Length is < 2 or > 128 || address.Any(char.IsWhiteSpace)) throw Invalid("FATEK_PROGRAM_ADDRESS_INVALID", "永宏 PLC 设备地址无效");
        var body = address;
        string? stationPrefix = null;
        var separator = address.IndexOf(';');
        if (separator >= 0)
        {
            var stationText = address[..separator];
            if (!stationText.StartsWith("s=", StringComparison.OrdinalIgnoreCase) || !byte.TryParse(stationText[2..], NumberStyles.None, CultureInfo.InvariantCulture, out var station))
                throw Invalid("FATEK_PROGRAM_ADDRESS_INVALID", "永宏 PLC 设备地址无效");
            stationPrefix = $"s={station};";
            body = address[(separator + 1)..];
        }
        body = body.ToUpperInvariant();
        var areas = boolean ? BitAreas : WordAreas;
        var area = areas.FirstOrDefault(candidate => body.StartsWith(candidate, StringComparison.Ordinal));
        if (area is null || !int.TryParse(body[area.Length..], NumberStyles.None, CultureInfo.InvariantCulture, out var offset) || offset < 0)
            throw Invalid(boolean ? "FATEK_PROGRAM_BOOL_AREA_INVALID" : "FATEK_PROGRAM_WORD_AREA_INVALID",
                boolean ? "永宏 bool 变量只支持 M、X、Y、S、T、C 地址" : "永宏数值变量只支持 RT、RC、D、R 地址");
        return $"{stationPrefix}{area}{offset.ToString(CultureInfo.InvariantCulture)}";
    }

    private static HslValueType ParseValueType(string dataType) => dataType switch
    {
        "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8,
        "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32,
        "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64,
        "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision,
        "string" => HslValueType.Text, "bytes" => HslValueType.Binary,
        _ => throw Invalid("FATEK_PROGRAM_DATA_TYPE_UNSUPPORTED", "永宏编程口数据类型不受支持"),
    };
    private static FatekProgramDriverException Invalid(string code, string message) => new(code, message, false);
}
