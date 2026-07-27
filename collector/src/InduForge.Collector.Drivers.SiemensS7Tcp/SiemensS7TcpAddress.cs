using System.Globalization;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp;

internal enum SiemensS7Area
{
    Input,
    Output,
    Marker,
    DataBlock,
    Timer,
    Counter,
}

internal sealed record SiemensS7TcpAddress(
    SiemensS7Area Area,
    int? DataBlockNumber,
    int ByteOffset,
    byte? BitOffset,
    string HslAddress,
    HslValueType ValueType,
    int ElementCount)
{
    public static SiemensS7TcpAddress Parse(PointReadRequest point)
    {
        ArgumentNullException.ThrowIfNull(point);
        if (point.Address.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("S7_ADDRESS_INVALID", "Siemens S7 地址必须是结构化对象");
        }

        var areaText = RequiredString(point.Address, "area", "S7_AREA_REQUIRED", "Siemens S7 地址缺少区域");
        var area = areaText switch
        {
            "input" => SiemensS7Area.Input,
            "output" => SiemensS7Area.Output,
            "marker" => SiemensS7Area.Marker,
            "dataBlock" => SiemensS7Area.DataBlock,
            "timer" => SiemensS7Area.Timer,
            "counter" => SiemensS7Area.Counter,
            _ => throw Invalid("S7_AREA_INVALID", "Siemens S7 地址区域无效"),
        };

        var byteOffset = RequiredInt32(point.Address, "byteOffset", "S7_OFFSET_REQUIRED", "Siemens S7 地址缺少字节偏移");
        if (byteOffset < 0)
        {
            throw Invalid("S7_OFFSET_INVALID", "Siemens S7 字节偏移不能小于 0");
        }

        var dataBlockNumber = OptionalInt32(point.Address, "dbNumber");
        if (area == SiemensS7Area.DataBlock && dataBlockNumber is not (>= 1 and <= 65535))
        {
            throw Invalid("S7_DB_NUMBER_REQUIRED", "数据块地址必须配置 1 到 65535 的 DB 编号");
        }
        if (area != SiemensS7Area.DataBlock && dataBlockNumber is not null)
        {
            throw Invalid("S7_DB_NUMBER_NOT_ALLOWED", "只有数据块地址可以配置 DB 编号");
        }

        var bitOffset = OptionalInt32(point.Address, "bitOffset");
        if (bitOffset is < 0 or > 7)
        {
            throw Invalid("S7_BIT_OFFSET_INVALID", "Siemens S7 位偏移必须在 0 到 7 之间");
        }

        var valueType = ParseValueType(point.DataType);
        var isBitArea = area is SiemensS7Area.Input or SiemensS7Area.Output or SiemensS7Area.Marker or SiemensS7Area.DataBlock;
        if (valueType == HslValueType.Boolean && (!isBitArea || bitOffset is null))
        {
            throw Invalid("S7_BIT_OFFSET_REQUIRED", "Siemens S7 bool 变量必须配置可位寻址区域和位偏移");
        }
        if (valueType != HslValueType.Boolean && bitOffset is not null)
        {
            throw Invalid("S7_BIT_OFFSET_NOT_ALLOWED", "只有 Siemens S7 bool 变量可以配置位偏移");
        }
        if (area is SiemensS7Area.Timer or SiemensS7Area.Counter &&
            valueType is not (HslValueType.Signed16 or HslValueType.Unsigned16))
        {
            throw Invalid("S7_TIMER_COUNTER_TYPE_INVALID", "定时器和计数器只支持 int16 或 uint16 数据类型");
        }
        if (point.ElementCount < 1)
        {
            throw Invalid("S7_ELEMENT_COUNT_INVALID", "Siemens S7 元素数量必须大于 0");
        }
        if (valueType == HslValueType.DateTime && point.ElementCount != 1)
        {
            throw Invalid("S7_DATETIME_COUNT_INVALID", "Siemens S7 datetime 变量只支持单元素读取");
        }

        var prefix = area switch
        {
            SiemensS7Area.Input => "I",
            SiemensS7Area.Output => "Q",
            SiemensS7Area.Marker => "M",
            SiemensS7Area.DataBlock => $"DB{dataBlockNumber}.",
            SiemensS7Area.Timer => "T",
            SiemensS7Area.Counter => "C",
            _ => throw new InvalidOperationException("Siemens S7 地址区域未映射"),
        };
        var address = prefix + byteOffset.ToString(CultureInfo.InvariantCulture);
        if (bitOffset is not null)
        {
            address += "." + bitOffset.Value.ToString(CultureInfo.InvariantCulture);
        }

        return new SiemensS7TcpAddress(
            area,
            dataBlockNumber,
            byteOffset,
            bitOffset is null ? null : checked((byte)bitOffset.Value),
            address,
            valueType,
            point.ElementCount);
    }

    public HslS7ReadRequest ToHslRequest() => new(HslAddress, ValueType, ElementCount);

    internal static HslValueType ParseValueType(string dataType) => dataType switch
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
        _ => throw Invalid("S7_DATA_TYPE_UNSUPPORTED", "Siemens S7 数据类型不受支持"),
    };

    private static string RequiredString(JsonElement address, string name, string code, string message) =>
        address.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String && !string.IsNullOrWhiteSpace(value.GetString())
            ? value.GetString()!
            : throw Invalid(code, message);

    private static int RequiredInt32(JsonElement address, string name, string code, string message) =>
        OptionalInt32(address, name) ?? throw Invalid(code, message);

    private static int? OptionalInt32(JsonElement address, string name) =>
        address.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result)
            ? result
            : null;

    private static SiemensS7TcpDriverException Invalid(string code, string message) =>
        new(code, message, retryable: false);
}
