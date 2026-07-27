using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins;

internal sealed record OmronFinsAddress(
    string HslAddress,
    HslValueType ValueType,
    int ElementCount)
{
    public static OmronFinsAddress Parse(PointReadRequest point)
    {
        ArgumentNullException.ThrowIfNull(point);
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var addressValue) ||
            addressValue.ValueKind != JsonValueKind.String)
        {
            throw Invalid("OMRON_FINS_ADDRESS_INVALID", "Omron FINS 地址必须包含 address 字段");
        }

        var address = addressValue.GetString()?.Trim() ?? string.Empty;
        if (address.Length is < 1 or > 128 || address.Any(char.IsWhiteSpace))
        {
            throw Invalid("OMRON_FINS_ADDRESS_INVALID", "Omron FINS 设备地址无效");
        }
        if (point.ElementCount < 1)
        {
            throw Invalid("OMRON_FINS_ELEMENT_COUNT_INVALID", "Omron FINS 元素数量必须大于 0");
        }

        var valueType = ParseValueType(point.DataType);
        var protocolUnits = GetProtocolUnits(valueType, point.ElementCount);
        if (protocolUnits > ushort.MaxValue)
        {
            throw Invalid("OMRON_FINS_ELEMENT_COUNT_TOO_LARGE", $"Omron FINS 单变量读取长度超过平台上限 {ushort.MaxValue}");
        }
        return new OmronFinsAddress(address, valueType, point.ElementCount);
    }

    public HslOmronFinsReadRequest ToHslRequest() => new(HslAddress, ValueType, ElementCount);

    private static long GetProtocolUnits(HslValueType valueType, int count) => valueType switch
    {
        HslValueType.Boolean => count,
        HslValueType.Signed8 or HslValueType.Unsigned8 or HslValueType.Text or HslValueType.Binary => (count + 1L) / 2,
        HslValueType.Signed16 or HslValueType.Unsigned16 => count,
        HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision => count * 2L,
        HslValueType.Signed64 or HslValueType.Unsigned64 or HslValueType.DoublePrecision => count * 4L,
        _ => count,
    };

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
        _ => throw Invalid("OMRON_FINS_DATA_TYPE_UNSUPPORTED", "Omron FINS 数据类型不受支持"),
    };

    private static OmronFinsDriverException Invalid(string code, string message) => new(code, message, retryable: false);
}
