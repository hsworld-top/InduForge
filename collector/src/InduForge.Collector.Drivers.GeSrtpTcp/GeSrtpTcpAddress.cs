using System.Globalization;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.GeSrtpTcp;

internal sealed record GeSrtpTcpAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    private static readonly string[] AddressAreas = ["AI", "AQ", "SA", "SB", "SC", "I", "Q", "M", "T", "S", "G", "R"];

    public static GeSrtpTcpAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("GE_SRTP_ADDRESS_INVALID", "GE SRTP 地址必须包含 address 字段");

        var address = Normalize(value.GetString());
        var area = AddressAreas.First(candidate => address.StartsWith(candidate, StringComparison.Ordinal));
        if (point.DataType == "bool" && area is "AI" or "AQ" or "R")
            throw Invalid("GE_SRTP_BOOL_AREA_INVALID", "GE SRTP AI、AQ、R 字寄存器不支持 bool 数据类型");
        if (point.ElementCount is < 1 or > ushort.MaxValue)
            throw Invalid("GE_SRTP_ELEMENT_COUNT_INVALID", "GE SRTP 元素数量必须在 1 到 65535 之间");
        return new GeSrtpTcpAddress(address, ParseValueType(point.DataType), point.ElementCount);
    }

    public HslGeSrtpReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);

    private static string Normalize(string? value)
    {
        var address = value?.Trim().ToUpperInvariant() ?? string.Empty;
        if (address.Length is < 2 or > 128 || address.Any(char.IsWhiteSpace))
            throw Invalid("GE_SRTP_ADDRESS_INVALID", "GE SRTP 设备地址无效");
        var area = AddressAreas.FirstOrDefault(candidate => address.StartsWith(candidate, StringComparison.Ordinal));
        if (area is null || !int.TryParse(address[area.Length..], NumberStyles.None, CultureInfo.InvariantCulture, out var offset) || offset < 1)
            throw Invalid("GE_SRTP_ADDRESS_INVALID", "GE SRTP 设备地址无效");
        return $"{area}{offset.ToString(CultureInfo.InvariantCulture)}";
    }

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
        _ => throw Invalid("GE_SRTP_DATA_TYPE_UNSUPPORTED", "GE SRTP 数据类型不受支持"),
    };

    private static GeSrtpTcpDriverException Invalid(string code, string message) => new(code, message, false);
}
