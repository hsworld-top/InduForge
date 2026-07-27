using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Cimon;
internal sealed record CimonAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static CimonAddress Parse(PointReadRequest point) { if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String) throw Invalid("CIMON_ADDRESS_REQUIRED", "Cimon HMI 地址必须包含 address 字段"); var address = value.GetString()?.Trim() ?? string.Empty; if (address.Length is < 2 or > 128 || address.Any(char.IsWhiteSpace)) throw Invalid("CIMON_ADDRESS_INVALID", "Cimon HMI 设备地址无效"); if (point.ElementCount is < 1 or > ushort.MaxValue) throw Invalid("CIMON_ELEMENT_COUNT_INVALID", "Cimon HMI 元素数量必须在 1 到 65535 之间"); return new(address.ToUpperInvariant(), Type(point.DataType), point.ElementCount); }
    public HslCimonReadRequest ToHslRequest() => new(Address, DataType, ElementCount);
    private static HslValueType Type(string value) => value switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw Invalid("CIMON_DATA_TYPE_UNSUPPORTED", "Cimon HMI 数据类型不受支持") };
    private static CimonDriverException Invalid(string code, string message) => new(code, message, false);
}
