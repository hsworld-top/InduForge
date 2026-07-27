using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MitsubishiMc3ETcp.Tests;

public sealed class MitsubishiMc3ETcpAddressTests
{
    [Theory]
    [InlineData(" D100 ", "int16", "D100", HslValueType.Signed16)]
    [InlineData("M0", "bool", "M0", HslValueType.Boolean)]
    [InlineData("D100.3", "bool", "D100.3", HslValueType.Boolean)]
    public void ParseKeepsNativeDeviceAddress(string address, string dataType, string expected, HslValueType expectedType)
    {
        var parsed = MitsubishiMc3ETcpAddress.Parse(CreatePoint(address, dataType));

        Assert.Equal(expected, parsed.HslAddress);
        Assert.Equal(expectedType, parsed.ValueType);
    }

    [Theory]
    [InlineData("")]
    [InlineData("D 100")]
    public void ParseRejectsInvalidAddress(string address) =>
        Assert.Throws<MitsubishiMc3ETcpDriverException>(() => MitsubishiMc3ETcpAddress.Parse(CreatePoint(address, "int16")));

    [Fact]
    public void ParseRejectsReadAboveWordLimit()
    {
        var point = CreatePoint("D100", "float64", 241);
        var exception = Assert.Throws<MitsubishiMc3ETcpDriverException>(() => MitsubishiMc3ETcpAddress.Parse(point));
        Assert.Equal("MELSEC_MC_ELEMENT_COUNT_TOO_LARGE", exception.Code);
    }

    private static PointReadRequest CreatePoint(string address, string dataType, int count = 1) => new(
        "point-1",
        JsonSerializer.SerializeToElement(new { address }),
        dataType,
        count,
        JsonSerializer.SerializeToElement(new { }));
}
