using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.AllenBradleyEtherNetIp;

internal sealed record AllenBradleyEtherNetIpAddress(
    string HslAddress,
    HslValueType ValueType,
    int ElementCount)
{
    private static readonly Regex LegacyAddressPattern = new(
        @"^(?:(?:s|dst|src)=\d+;)*(?:ST|A|B|N|F|S|C|I|O|R|T|L)\d*:\d+(?:(?:/|\.)\d+)?$",
        RegexOptions.Compiled | RegexOptions.IgnoreCase | RegexOptions.CultureInvariant);

    public static AllenBradleyEtherNetIpAddress Parse(
        PointReadRequest point,
        AllenBradleyAddressKind addressKind = AllenBradleyAddressKind.LogixTag)
    {
        ArgumentNullException.ThrowIfNull(point);
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var addressValue) ||
            addressValue.ValueKind != JsonValueKind.String)
        {
            throw Invalid("ALLEN_BRADLEY_ADDRESS_INVALID", "Allen-Bradley 标签地址必须包含 address 字段");
        }

        var address = addressValue.GetString()?.Trim() ?? string.Empty;
        if (address.Length is < 1 or > 512 || address.Any(char.IsWhiteSpace))
        {
            throw Invalid("ALLEN_BRADLEY_ADDRESS_INVALID", "Allen-Bradley 标签地址无效");
        }

        if (addressKind == AllenBradleyAddressKind.LegacyFile && !LegacyAddressPattern.IsMatch(address))
        {
            throw Invalid("ALLEN_BRADLEY_ADDRESS_INVALID", "Allen-Bradley 文件地址无效");
        }
        if (point.ElementCount is < 1 or > ushort.MaxValue)
        {
            throw Invalid("ALLEN_BRADLEY_ELEMENT_COUNT_INVALID", $"Allen-Bradley 元素数量必须在 1 到 {ushort.MaxValue} 之间");
        }

        var valueType = ParseValueType(point.DataType);
        if (addressKind == AllenBradleyAddressKind.LegacyFile && valueType is not
            (HslValueType.Boolean or HslValueType.Signed16 or HslValueType.Unsigned16 or
             HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision or
             HslValueType.Text))
        {
            throw Invalid("ALLEN_BRADLEY_DATA_TYPE_UNSUPPORTED", "当前 Allen-Bradley 文件协议不支持该数据类型");
        }
        if (valueType == HslValueType.DateTime && point.ElementCount != 1)
        {
            throw Invalid("ALLEN_BRADLEY_DATETIME_ARRAY_UNSUPPORTED", "Allen-Bradley datetime 标签只支持单元素读取");
        }
        return new AllenBradleyEtherNetIpAddress(address, valueType, point.ElementCount);
    }

    public HslAllenBradleyReadRequest ToHslRequest() => new(HslAddress, ValueType, ElementCount);

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
        "datetime" => HslValueType.DateTime,
        _ => throw Invalid("ALLEN_BRADLEY_DATA_TYPE_UNSUPPORTED", "Allen-Bradley 数据类型不受支持"),
    };

    private static AllenBradleyEtherNetIpDriverException Invalid(string code, string message) => new(code, message, retryable: false);
}

internal enum AllenBradleyAddressKind
{
    LogixTag,
    LegacyFile,
}
