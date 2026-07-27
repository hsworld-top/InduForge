using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.AllenBradleyEtherNetIp.Tests;

public sealed class AllenBradleyEtherNetIpAddressTests
{
    [Theory]
    [InlineData("Temperature", "float32", HslValueType.SinglePrecision)]
    [InlineData("ArrayTag[10]", "int32", HslValueType.Signed32)]
    [InlineData("Program:MainProgram.LocalTag", "bool", HslValueType.Boolean)]
    [InlineData("slot=2;TagName", "uint16", HslValueType.Unsigned16)]
    public void ParsesLogixTag(string address, string dataType, HslValueType expectedType)
    {
        var result = AllenBradleyEtherNetIpAddress.Parse(Point(address, dataType, 1));

        Assert.Equal(address, result.HslAddress);
        Assert.Equal(expectedType, result.ValueType);
    }

    [Fact]
    public void RejectsWhitespaceInTag()
    {
        var exception = Assert.Throws<AllenBradleyEtherNetIpDriverException>(() =>
            AllenBradleyEtherNetIpAddress.Parse(Point("Bad Tag", "int16", 1)));

        Assert.Equal("ALLEN_BRADLEY_ADDRESS_INVALID", exception.Code);
    }

    [Fact]
    public void RejectsDateTimeArray()
    {
        var exception = Assert.Throws<AllenBradleyEtherNetIpDriverException>(() =>
            AllenBradleyEtherNetIpAddress.Parse(Point("Timestamp", "datetime", 2)));

        Assert.Equal("ALLEN_BRADLEY_DATETIME_ARRAY_UNSUPPORTED", exception.Code);
    }

    private static PointReadRequest Point(string address, string dataType, int count) => new(
        "point",
        JsonSerializer.SerializeToElement(new { address }),
        dataType,
        count,
        JsonSerializer.SerializeToElement(new { }));
}
