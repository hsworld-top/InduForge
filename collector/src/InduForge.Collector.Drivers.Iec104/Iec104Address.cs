using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Iec104;

internal sealed record Iec104Address(
    HslIec104InformationType InformationType,
    int InformationObjectAddress)
{
    public static Iec104Address Parse(PointReadRequest point)
    {
        ArgumentNullException.ThrowIfNull(point);
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("informationType", out var typeValue) ||
            typeValue.ValueKind != JsonValueKind.String ||
            !point.Address.TryGetProperty("informationObjectAddress", out var addressValue) ||
            addressValue.ValueKind != JsonValueKind.Number ||
            !addressValue.TryGetInt32(out var informationObjectAddress))
        {
            throw Invalid("IEC104_ADDRESS_INVALID", "IEC 104 地址必须包含信息类型和信息对象地址");
        }
        if (informationObjectAddress is < 0 or > 0xFFFFFF)
        {
            throw Invalid("IEC104_INFORMATION_OBJECT_ADDRESS_INVALID", "IEC 104 信息对象地址必须在 0 到 16777215 之间");
        }
        if (point.ElementCount != 1)
        {
            throw Invalid("IEC104_ELEMENT_COUNT_UNSUPPORTED", "IEC 104 单个信息对象的元素数量必须为 1");
        }

        var informationType = ParseInformationType(typeValue.GetString());
        ValidateDataType(informationType, point.DataType);
        return new Iec104Address(informationType, informationObjectAddress);
    }

    public HslIec104ReadRequest ToHslRequest() => new(InformationType, InformationObjectAddress);

    private static HslIec104InformationType ParseInformationType(string? value) => value switch
    {
        "singlePoint" => HslIec104InformationType.SinglePoint,
        "doublePoint" => HslIec104InformationType.DoublePoint,
        "normalizedMeasured" => HslIec104InformationType.NormalizedMeasured,
        "scaledMeasured" => HslIec104InformationType.ScaledMeasured,
        "shortFloatMeasured" => HslIec104InformationType.ShortFloatMeasured,
        "bitString32" => HslIec104InformationType.BitString32,
        "integratedTotal" => HslIec104InformationType.IntegratedTotal,
        _ => throw Invalid("IEC104_INFORMATION_TYPE_INVALID", "IEC 104 信息类型不受支持"),
    };

    private static void ValidateDataType(HslIec104InformationType informationType, string dataType)
    {
        var expected = informationType switch
        {
            HslIec104InformationType.SinglePoint => "bool",
            HslIec104InformationType.DoublePoint => "uint8",
            HslIec104InformationType.NormalizedMeasured or HslIec104InformationType.ScaledMeasured => "int16",
            HslIec104InformationType.ShortFloatMeasured => "float32",
            HslIec104InformationType.BitString32 or HslIec104InformationType.IntegratedTotal => "uint32",
            _ => string.Empty,
        };
        if (!string.Equals(dataType, expected, StringComparison.OrdinalIgnoreCase))
        {
            throw Invalid("IEC104_DATA_TYPE_MISMATCH", $"IEC 104 {informationType} 信息类型必须使用 {expected} 数据类型");
        }
    }

    private static Iec104DriverException Invalid(string code, string message) => new(code, message, retryable: false);
}
