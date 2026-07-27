using System.Globalization;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Freedom;

internal sealed record FreedomAddress(string ProtocolAddress, HslValueType DataType, int ElementCount)
{
    public static FreedomAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("requestHex", out var value) || value.ValueKind != JsonValueKind.String) throw Invalid("FREEDOM_REQUEST_HEX_REQUIRED", "自由协议地址必须包含请求报文十六进制");
        var compact = new string((value.GetString() ?? string.Empty).Where(character => !char.IsWhiteSpace(character) && character != '-').ToArray());
        if (compact.Length < 2 || compact.Length % 2 != 0 || !compact.All(Uri.IsHexDigit)) throw Invalid("FREEDOM_REQUEST_HEX_INVALID", "自由协议请求报文必须是偶数长度的十六进制字节");
        var offset = point.Address.TryGetProperty("responseOffset", out var offsetValue) && offsetValue.TryGetInt32(out var parsedOffset) ? parsedOffset : 0;
        if (offset is < 0 or > ushort.MaxValue) throw Invalid("FREEDOM_RESPONSE_OFFSET_INVALID", "自由协议响应偏移必须在 0 到 65535 之间");
        if (point.ElementCount is < 1 or > ushort.MaxValue) throw Invalid("FREEDOM_ELEMENT_COUNT_INVALID", "自由协议元素数量必须在 1 到 65535 之间");
        var bytes = Enumerable.Range(0, compact.Length / 2).Select(index => compact.Substring(index * 2, 2).ToUpperInvariant());
        var address = string.Join(' ', bytes);
        if (offset > 0) address = $"stx={offset.ToString(CultureInfo.InvariantCulture)};{address}";
        return new(address, ParseType(point.DataType), point.ElementCount);
    }
    public HslFreedomReadRequest ToHslRequest() => new(ProtocolAddress, DataType, ElementCount);
    private static HslValueType ParseType(string value) => value switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw Invalid("FREEDOM_DATA_TYPE_UNSUPPORTED", "自由协议数据类型不受支持") };
    private static FreedomDriverException Invalid(string code, string message) => new(code, message, false);
}
