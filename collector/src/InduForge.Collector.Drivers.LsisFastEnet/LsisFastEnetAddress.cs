using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.LsisFastEnet;
internal sealed record LsisFastEnetAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    public static LsisFastEnetAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var raw) || raw.ValueKind != JsonValueKind.String) throw Invalid("LSIS_FAST_ENET_ADDRESS_INVALID", "LSIS Fast Enet 地址必须包含 address 字段");
        var address = raw.GetString()?.Trim() ?? "";
        if (address.Length is < 1 or > 128 || address.Any(char.IsWhiteSpace)) throw Invalid("LSIS_FAST_ENET_ADDRESS_INVALID", "LSIS Fast Enet 设备地址无效");
        if (point.ElementCount is < 1 or > ushort.MaxValue) throw Invalid("LSIS_FAST_ENET_ELEMENT_COUNT_INVALID", "LSIS Fast Enet 元素数量必须在 1 到 65535 之间");
        return new(address, Type(point.DataType), point.ElementCount);
    }
    public HslLsisFastEnetReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);
    private static HslValueType Type(string value) => value switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw Invalid("LSIS_FAST_ENET_DATA_TYPE_UNSUPPORTED", "LSIS Fast Enet 数据类型不受支持") };
    private static LsisFastEnetDriverException Invalid(string code, string message) => new(code, message, false);
}
