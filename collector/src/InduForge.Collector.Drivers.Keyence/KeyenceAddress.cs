using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Keyence;

internal sealed partial record KeyenceAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    public static KeyenceAddress Parse(PointReadRequest point, KeyenceDriverKind kind)
    {
        ArgumentNullException.ThrowIfNull(point);
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("KEYENCE_ADDRESS_INVALID", "基恩士地址必须包含 address 字段");
        var address = value.GetString()?.Trim().ToUpperInvariant() ?? string.Empty;
        if (address.Length is < 2 or > 128 || address.Any(char.IsWhiteSpace) || !Pattern(kind).IsMatch(address))
            throw Invalid("KEYENCE_ADDRESS_INVALID", "基恩士设备地址格式无效");
        if (point.ElementCount is < 1 or > ushort.MaxValue)
            throw Invalid("KEYENCE_ELEMENT_COUNT_INVALID", "基恩士元素数量必须在 1 到 65535 之间");
        var valueType = ParseValueType(point.DataType);
        if (kind == KeyenceDriverKind.Mc3E)
        {
            var protocolUnits = valueType == HslValueType.Boolean ? point.ElementCount : checked(GetWordWidth(valueType) * point.ElementCount);
            var maximumUnits = valueType == HslValueType.Boolean ? 7168 : 960;
            if (protocolUnits > maximumUnits)
                throw Invalid("KEYENCE_ELEMENT_COUNT_TOO_LARGE", $"基恩士 MC 单次读取最多 {maximumUnits} 个协议单位");
        }
        return new(address, valueType, point.ElementCount);
    }

    public HslKeyenceTcpReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);

    private static Regex Pattern(KeyenceDriverKind kind) => kind switch
    {
        KeyenceDriverKind.Mc3E => McAddressRegex(),
        KeyenceDriverKind.KvOld => KvOldAddressRegex(),
        KeyenceDriverKind.Nano => NanoAddressRegex(),
        _ => throw new ArgumentOutOfRangeException(nameof(kind)),
    };
    private static int GetWordWidth(HslValueType type) => type switch
    {
        HslValueType.Signed8 or HslValueType.Unsigned8 or HslValueType.Signed16 or HslValueType.Unsigned16 => 1,
        HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision => 2,
        HslValueType.Signed64 or HslValueType.Unsigned64 or HslValueType.DoublePrecision => 4,
        HslValueType.Text or HslValueType.Binary => 1,
        _ => 1,
    };
    private static HslValueType ParseValueType(string type) => type switch
    {
        "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8,
        "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32,
        "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64,
        "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision,
        "string" => HslValueType.Text, "bytes" => HslValueType.Binary,
        _ => throw Invalid("KEYENCE_DATA_TYPE_UNSUPPORTED", "基恩士数据类型不受支持"),
    };
    private static KeyenceDriverException Invalid(string code, string message) => new(code, message, false);

    [GeneratedRegex("^(?:R|MR|LR|CR|CM|DM|EM|FM|ZF|W|TN|TS|CN|CS)[0-9A-F]+$", RegexOptions.CultureInvariant)]
    private static partial Regex McAddressRegex();
    [GeneratedRegex("^[A-Z]{1,4}[0-9]+(?:\\.[0-9]+)?$", RegexOptions.CultureInvariant)]
    private static partial Regex KvOldAddressRegex();
    [GeneratedRegex("^(?:(?:R|B|MR|LR|CR|VB|DM|EM|FM|ZF|W|TM|Z|AT|CM|VM|T|C|TC|CC|TS|CS)[0-9]+|UNIT=[0-9]+;[0-9]+)$", RegexOptions.CultureInvariant)]
    private static partial Regex NanoAddressRegex();
}
