using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.BeckhoffAds;

internal sealed record BeckhoffAdsAddress(
    string ProtocolAddress,
    HslValueType ValueType,
    int ElementCount)
{
    public static BeckhoffAdsAddress Parse(PointReadRequest point)
    {
        ArgumentNullException.ThrowIfNull(point);
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var addressValue) ||
            addressValue.ValueKind != JsonValueKind.String)
        {
            throw Invalid("BECKHOFF_ADS_ADDRESS_INVALID", "倍福 ADS 地址必须包含 address 字段");
        }

        var address = addressValue.GetString()?.Trim() ?? string.Empty;
        if (address.Length is < 1 or > 512 || address.Any(char.IsWhiteSpace))
        {
            throw Invalid("BECKHOFF_ADS_ADDRESS_INVALID", "倍福 ADS 地址无效");
        }
        if (point.ElementCount is < 1 or > ushort.MaxValue)
        {
            throw Invalid("BECKHOFF_ADS_ELEMENT_COUNT_INVALID", $"倍福 ADS 元素数量必须在 1 到 {ushort.MaxValue} 之间");
        }

        return new BeckhoffAdsAddress(address, ParseValueType(point.DataType), point.ElementCount);
    }

    public HslBeckhoffAdsReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);

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
        _ => throw Invalid("BECKHOFF_ADS_DATA_TYPE_UNSUPPORTED", "倍福 ADS 数据类型不受支持"),
    };

    private static BeckhoffAdsDriverException Invalid(string code, string message) => new(code, message, retryable: false);
}
