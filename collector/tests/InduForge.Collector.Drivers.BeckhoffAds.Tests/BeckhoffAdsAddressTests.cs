using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.BeckhoffAds.Tests;

public sealed class BeckhoffAdsAddressTests
{
    [Theory]
    [InlineData("M100")]
    [InlineData("M100.0")]
    [InlineData("s=MAIN.a")]
    [InlineData("i=1235467;")]
    [InlineData("IG=0xF020;0")]
    public void AcceptsAdsAddressForms(string address)
    {
        var parsed = BeckhoffAdsAddress.Parse(Point(address, "int16", 1));

        Assert.Equal(address, parsed.ProtocolAddress);
        Assert.Equal(HslValueType.Signed16, parsed.ValueType);
    }

    [Fact]
    public void RejectsWhitespaceInsideAddress()
    {
        var exception = Assert.Throws<BeckhoffAdsDriverException>(() =>
            BeckhoffAdsAddress.Parse(Point("s=MAIN. a", "int16", 1)));

        Assert.Equal("BECKHOFF_ADS_ADDRESS_INVALID", exception.Code);
    }

    [Fact]
    public void RejectsElementCountAboveProtocolLimit()
    {
        var exception = Assert.Throws<BeckhoffAdsDriverException>(() =>
            BeckhoffAdsAddress.Parse(Point("M100", "int16", ushort.MaxValue + 1)));

        Assert.Equal("BECKHOFF_ADS_ELEMENT_COUNT_INVALID", exception.Code);
    }

    [Fact]
    public void RejectsDateTimeUntilProtocolMappingExists()
    {
        var exception = Assert.Throws<BeckhoffAdsDriverException>(() =>
            BeckhoffAdsAddress.Parse(Point("s=MAIN.timestamp", "datetime", 1)));

        Assert.Equal("BECKHOFF_ADS_DATA_TYPE_UNSUPPORTED", exception.Code);
    }

    private static PointReadRequest Point(string address, string dataType, int count) => new(
        "point",
        JsonSerializer.SerializeToElement(new { address }),
        dataType,
        count,
        JsonSerializer.SerializeToElement(new { }));
}
