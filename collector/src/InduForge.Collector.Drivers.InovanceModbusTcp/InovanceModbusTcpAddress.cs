using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InovanceModbusTcp;

internal sealed record InovanceModbusTcpAddress(string ProtocolAddress, HslValueType ValueType, int ElementCount)
{
    public static InovanceModbusTcpAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("INOVANCE_MODBUS_TCP_ADDRESS_INVALID", "汇川 Modbus TCP 地址必须包含 address 字段");
        var address = value.GetString()?.Trim().ToUpperInvariant() ?? string.Empty;
        if (address.Length is < 1 or > 128 || address.Any(char.IsWhiteSpace))
            throw Invalid("INOVANCE_MODBUS_TCP_ADDRESS_INVALID", "汇川 PLC 设备地址无效");
        ValidateElementCount(point.DataType, point.ElementCount);
        return new InovanceModbusTcpAddress(address, ParseValueType(point.DataType), point.ElementCount);
    }

    public HslInovanceModbusTcpReadRequest ToHslRequest() => new(ProtocolAddress, ValueType, ElementCount);

    private static void ValidateElementCount(string dataType, int elementCount)
    {
        if (elementCount < 1)
            throw Invalid("INOVANCE_MODBUS_TCP_ELEMENT_COUNT_INVALID", "汇川 Modbus TCP 元素数量必须大于 0");
        var registerCount = dataType switch
        {
            "bool" => elementCount,
            "int8" or "uint8" or "string" or "bytes" => (elementCount + 1L) / 2L,
            "int16" or "uint16" => elementCount,
            "int32" or "uint32" or "float32" => elementCount * 2L,
            "int64" or "uint64" or "float64" => elementCount * 4L,
            _ => throw Invalid("INOVANCE_MODBUS_TCP_DATA_TYPE_UNSUPPORTED", "汇川 Modbus TCP 数据类型不受支持"),
        };
        var maximum = dataType == "bool" ? 2000 : 125;
        if (registerCount > maximum)
            throw Invalid("INOVANCE_MODBUS_TCP_READ_LENGTH_INVALID", "汇川 Modbus TCP 单变量读取长度超过协议上限");
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
        _ => throw Invalid("INOVANCE_MODBUS_TCP_DATA_TYPE_UNSUPPORTED", "汇川 Modbus TCP 数据类型不受支持"),
    };

    private static InovanceModbusTcpDriverException Invalid(string code, string message) => new(code, message, false);
}
