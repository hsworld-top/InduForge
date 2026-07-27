using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Iec104.Tests;

public sealed class Iec104AddressTests
{
    [Theory]
    [InlineData("singlePoint", "bool", HslIec104InformationType.SinglePoint)]
    [InlineData("doublePoint", "uint8", HslIec104InformationType.DoublePoint)]
    [InlineData("normalizedMeasured", "int16", HslIec104InformationType.NormalizedMeasured)]
    [InlineData("scaledMeasured", "int16", HslIec104InformationType.ScaledMeasured)]
    [InlineData("shortFloatMeasured", "float32", HslIec104InformationType.ShortFloatMeasured)]
    [InlineData("bitString32", "uint32", HslIec104InformationType.BitString32)]
    [InlineData("integratedTotal", "uint32", HslIec104InformationType.IntegratedTotal)]
    public void MapsInformationTypes(string informationType, string dataType, HslIec104InformationType expected)
    {
        var address = Iec104Address.Parse(Point(informationType, 100, dataType, 1));

        Assert.Equal(expected, address.InformationType);
        Assert.Equal(100, address.InformationObjectAddress);
    }

    [Fact]
    public void RejectsMismatchedDataType()
    {
        var exception = Assert.Throws<Iec104DriverException>(() =>
            Iec104Address.Parse(Point("singlePoint", 1, "uint8", 1)));

        Assert.Equal("IEC104_DATA_TYPE_MISMATCH", exception.Code);
    }

    [Fact]
    public void RejectsArrayElementCount()
    {
        var exception = Assert.Throws<Iec104DriverException>(() =>
            Iec104Address.Parse(Point("shortFloatMeasured", 1, "float32", 2)));

        Assert.Equal("IEC104_ELEMENT_COUNT_UNSUPPORTED", exception.Code);
    }

    private static PointReadRequest Point(string informationType, int address, string dataType, int elementCount) => new(
        "point",
        JsonSerializer.SerializeToElement(new { informationType, informationObjectAddress = address }),
        dataType,
        elementCount,
        JsonSerializer.SerializeToElement(new { }));
}
