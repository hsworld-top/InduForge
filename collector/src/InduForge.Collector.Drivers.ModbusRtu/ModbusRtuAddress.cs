using System.Globalization;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusRtu;

internal enum ModbusRtuArea
{
    Coil,
    DiscreteInput,
    InputRegister,
    HoldingRegister,
}

internal sealed record ModbusRtuAddress(
    byte Station,
    ModbusRtuArea Area,
    ushort StartAddress,
    byte? BitIndex,
    string HslAddress,
    HslModbusReadArea ReadArea,
    HslValueType ValueType,
    int ElementCount)
{
    public static ModbusRtuAddress Parse(PointReadRequest point)
    {
        ArgumentNullException.ThrowIfNull(point);
        if (point.Address.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("MODBUS_ADDRESS_INVALID", "Modbus 地址必须是结构化对象");
        }

        var station = RequiredInt32(point.Address, "station", "MODBUS_STATION_REQUIRED", "Modbus 地址缺少站号");
        if (station is < 1 or > 247)
        {
            throw Invalid("MODBUS_STATION_INVALID", "Modbus RTU 站号必须在 1 到 247 之间");
        }

        var address = RequiredInt32(point.Address, "address", "MODBUS_ADDRESS_REQUIRED", "Modbus 地址缺少寄存器或线圈地址");
        if (address is < 0 or > 65535)
        {
            throw Invalid("MODBUS_ADDRESS_OUT_OF_RANGE", "Modbus 地址必须在 0 到 65535 之间");
        }

        var areaText = RequiredString(point.Address, "area", "MODBUS_AREA_REQUIRED", "Modbus 地址缺少区域");
        var area = areaText switch
        {
            "coil" => ModbusRtuArea.Coil,
            "discreteInput" => ModbusRtuArea.DiscreteInput,
            "inputRegister" => ModbusRtuArea.InputRegister,
            "holdingRegister" => ModbusRtuArea.HoldingRegister,
            _ => throw Invalid("MODBUS_AREA_INVALID", "Modbus 地址区域无效"),
        };

        var bitIndex = OptionalInt32(point.Address, "bitIndex");
        if (bitIndex is < 0 or > 15)
        {
            throw Invalid("MODBUS_BIT_INDEX_INVALID", "Modbus 寄存器位索引必须在 0 到 15 之间");
        }
        if (point.ElementCount < 1)
        {
            throw Invalid("MODBUS_ELEMENT_COUNT_INVALID", "Modbus 变量元素数量必须大于 0");
        }

        var valueType = ParseValueType(point.DataType);
        var registerArea = area is ModbusRtuArea.InputRegister or ModbusRtuArea.HoldingRegister;
        if (!registerArea && valueType != HslValueType.Boolean)
        {
            throw Invalid("MODBUS_AREA_DATA_TYPE_INVALID", "线圈和离散输入只支持 bool 数据类型");
        }
        if (!registerArea && bitIndex is not null)
        {
            throw Invalid("MODBUS_BIT_INDEX_UNSUPPORTED", "线圈和离散输入不能配置寄存器位索引");
        }
        if (registerArea && valueType == HslValueType.Boolean && bitIndex is null)
        {
            throw Invalid("MODBUS_BIT_INDEX_REQUIRED", "寄存器 bool 变量必须配置位索引");
        }
        if (registerArea && valueType != HslValueType.Boolean && bitIndex is not null)
        {
            throw Invalid("MODBUS_BIT_INDEX_DATA_TYPE_INVALID", "只有寄存器 bool 变量可以配置位索引");
        }

        // RTU 与 TCP 使用相同功能码上限，保存和调试阶段都拒绝单点超限读取。
        var protocolUnits = GetProtocolReadUnits(valueType, point.ElementCount, bitIndex);
        var maximumUnits = registerArea ? 125 : 2000;
        if (protocolUnits > maximumUnits)
        {
            throw Invalid("MODBUS_ELEMENT_COUNT_TOO_LARGE", $"Modbus 单变量读取长度超过协议上限 {maximumUnits}");
        }

        var prefix = area == ModbusRtuArea.InputRegister
            ? $"s={station};x=4;"
            : $"s={station};";
        var addressText = address.ToString(CultureInfo.InvariantCulture);
        var suffix = bitIndex is null ? addressText : $"{addressText}.{bitIndex}";
        var readArea = area switch
        {
            ModbusRtuArea.Coil => HslModbusReadArea.Coil,
            ModbusRtuArea.DiscreteInput => HslModbusReadArea.DiscreteInput,
            _ => HslModbusReadArea.Register,
        };

        return new ModbusRtuAddress(
            checked((byte)station),
            area,
            checked((ushort)address),
            bitIndex is null ? null : checked((byte)bitIndex.Value),
            prefix + suffix,
            readArea,
            valueType,
            point.ElementCount);
    }

    public HslReadRequest ToHslRequest() => new(HslAddress, ReadArea, ValueType, ElementCount);

    private static long GetProtocolReadUnits(HslValueType valueType, int elementCount, int? bitIndex) => valueType switch
    {
        HslValueType.Boolean when bitIndex is not null => (bitIndex.Value + (long)elementCount + 15) / 16,
        HslValueType.Boolean => elementCount,
        HslValueType.Signed8 or HslValueType.Unsigned8 or HslValueType.Text or HslValueType.Binary =>
            ((long)elementCount + 1) / 2,
        HslValueType.Signed16 or HslValueType.Unsigned16 => elementCount,
        HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision => (long)elementCount * 2,
        HslValueType.Signed64 or HslValueType.Unsigned64 or HslValueType.DoublePrecision => (long)elementCount * 4,
        _ => elementCount,
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
        _ => throw Invalid("MODBUS_DATA_TYPE_UNSUPPORTED", "Modbus RTU 不支持该数据类型"),
    };

    private static string RequiredString(JsonElement address, string name, string code, string message)
    {
        if (!address.TryGetProperty(name, out var value) || value.ValueKind != JsonValueKind.String || string.IsNullOrWhiteSpace(value.GetString()))
        {
            throw Invalid(code, message);
        }
        return value.GetString()!;
    }

    private static int RequiredInt32(JsonElement address, string name, string code, string message) =>
        OptionalInt32(address, name) ?? throw Invalid(code, message);

    private static int? OptionalInt32(JsonElement address, string name) =>
        address.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result)
            ? result
            : null;

    private static ModbusRtuDriverException Invalid(string code, string message) => new(code, message, retryable: false);
}
