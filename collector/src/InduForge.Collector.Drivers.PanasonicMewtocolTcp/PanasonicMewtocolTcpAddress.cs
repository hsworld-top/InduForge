using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.PanasonicMewtocolTcp;

internal sealed record PanasonicMewtocolTcpAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    public static PanasonicMewtocolTcpAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("PANASONIC_MEWTOCOL_ADDRESS_INVALID", "松下 Mewtocol 地址必须包含 address 字段");
        var address = value.GetString()?.Trim() ?? string.Empty;
        if (address.Length is < 1 or > 128 || address.Any(char.IsWhiteSpace))
            throw Invalid("PANASONIC_MEWTOCOL_ADDRESS_INVALID", "松下 Mewtocol 设备地址无效");
        if (point.ElementCount is < 1 or > ushort.MaxValue)
            throw Invalid("PANASONIC_MEWTOCOL_ELEMENT_COUNT_INVALID", "松下 Mewtocol 元素数量必须在 1 到 65535 之间");
        return new PanasonicMewtocolTcpAddress(address, ParseValueType(point.DataType), point.ElementCount);
    }

    public HslPanasonicMewtocolReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);

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
        _ => throw Invalid("PANASONIC_MEWTOCOL_DATA_TYPE_UNSUPPORTED", "松下 Mewtocol 数据类型不受支持"),
    };

    private static PanasonicMewtocolTcpDriverException Invalid(string code, string message) => new(code, message, false);
}
