using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.BeckhoffAds.Tests;

public sealed class BeckhoffAdsConnectionOptionsTests
{
    [Fact]
    public void UsesRecommendedDefaults()
    {
        var options = BeckhoffAdsConnectionOptions.Parse(Profile(new { host = "192.168.1.10" }));

        Assert.Equal(48898, options.Port);
        Assert.True(options.UseAutoAmsNetId);
        Assert.Equal(851, options.AmsPort);
        Assert.Null(options.TargetAmsNetId);
        Assert.True(options.UseTagCache);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(5000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal(HslDataFormat.DCBA, options.DataFormat);
    }

    [Fact]
    public void MapsManualAmsNetIds()
    {
        var options = BeckhoffAdsConnectionOptions.Parse(Profile(new
        {
            host = "192.168.1.10",
            useAutoAmsNetId = false,
            amsPort = 852,
            targetAmsNetId = "192.168.1.10.1.1",
            senderAmsNetId = "192.168.1.20.1.1",
            useTagCache = false,
        }));

        Assert.False(options.UseAutoAmsNetId);
        Assert.Equal(852, options.AmsPort);
        Assert.Equal("192.168.1.10.1.1", options.TargetAmsNetId);
        Assert.Equal("192.168.1.20.1.1", options.SenderAmsNetId);
        Assert.False(options.UseTagCache);
    }

    [Fact]
    public void ManualModeRequiresTargetAmsNetId()
    {
        var exception = Assert.Throws<BeckhoffAdsDriverException>(() =>
            BeckhoffAdsConnectionOptions.Parse(Profile(new { host = "192.168.1.10", useAutoAmsNetId = false })));

        Assert.Equal("BECKHOFF_ADS_TARGET_AMS_NET_ID_REQUIRED", exception.Code);
    }

    [Theory]
    [InlineData("192.168.1.10.1")]
    [InlineData("192.168.1.10.1.256")]
    [InlineData("192.168.1.10.1.a")]
    [InlineData("192.168.1.10.1.1:851")]
    public void RejectsInvalidAmsNetId(string netId)
    {
        var exception = Assert.Throws<BeckhoffAdsDriverException>(() =>
            BeckhoffAdsConnectionOptions.Parse(Profile(new { host = "192.168.1.10", targetAmsNetId = netId })));

        Assert.Equal("BECKHOFF_ADS_AMS_NET_ID_INVALID", exception.Code);
    }

    private static ConnectionProfile Profile(object config) => new(
        "beckhoff",
        JsonSerializer.SerializeToElement(config),
        JsonSerializer.SerializeToElement(new { }));
}
