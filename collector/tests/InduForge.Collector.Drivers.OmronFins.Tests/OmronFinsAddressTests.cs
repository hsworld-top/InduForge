using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins.Tests;

public sealed class OmronFinsAddressTests
{
    [Theory]
    [InlineData("D100", "int16", HslValueType.Signed16)]
    [InlineData("D100.0", "bool", HslValueType.Boolean)]
    [InlineData("E0.100", "float32", HslValueType.SinglePrecision)]
    public void ParsesNativeAddress(string address, string dataType, HslValueType expectedType)
    {
        var result = OmronFinsAddress.Parse(Point(address, dataType, 1));

        Assert.Equal(address, result.HslAddress);
        Assert.Equal(expectedType, result.ValueType);
    }

    [Theory]
    [InlineData("")]
    [InlineData("D 100")]
    public void RejectsInvalidAddress(string address)
    {
        var exception = Assert.Throws<OmronFinsDriverException>(() => OmronFinsAddress.Parse(Point(address, "int16", 1)));

        Assert.Equal("OMRON_FINS_ADDRESS_INVALID", exception.Code);
    }

    [Fact]
    public void RejectsReadLargerThanPlatformLimit()
    {
        var exception = Assert.Throws<OmronFinsDriverException>(() => OmronFinsAddress.Parse(Point("D100", "float64", 16384)));

        Assert.Equal("OMRON_FINS_ELEMENT_COUNT_TOO_LARGE", exception.Code);
    }

    private static PointReadRequest Point(string address, string dataType, int count) => new(
        "point",
        JsonSerializer.SerializeToElement(new { address }),
        dataType,
        count,
        JsonSerializer.SerializeToElement(new { }));
}
