using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.AllenBradleyEtherNetIp.Tests;

public sealed class AllenBradleyEtherNetIpConnectionOptionsTests
{
    [Fact]
    public void UsesRecommendedDefaults()
    {
        var options = AllenBradleyEtherNetIpConnectionOptions.Parse(Profile(new { host = "192.168.1.10" }));

        Assert.Equal(44818, options.Port);
        Assert.Equal((byte)0, options.Slot);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.False(options.ContextCheck);
        Assert.True(options.ReadArrayUseSegment);
        Assert.Equal(HslDataFormat.DCBA, options.DataFormat);
    }

    [Fact]
    public void MapsCustomMessageRouter()
    {
        var options = AllenBradleyEtherNetIpConnectionOptions.Parse(Profile(new
        {
            host = "192.168.1.10",
            slot = 2,
            messageRouter = "1.15.2.18.1.12",
            contextCheck = true,
            readArrayUseSegment = false,
        }));

        Assert.Equal((byte)2, options.Slot);
        Assert.Equal("1.15.2.18.1.12", options.MessageRouter);
        Assert.True(options.ContextCheck);
        Assert.False(options.ReadArrayUseSegment);
    }

    [Theory]
    [InlineData("1.15.2")]
    [InlineData("1.256")]
    [InlineData("1.a")]
    public void RejectsInvalidMessageRouter(string route)
    {
        var exception = Assert.Throws<AllenBradleyEtherNetIpDriverException>(() =>
            AllenBradleyEtherNetIpConnectionOptions.Parse(Profile(new { host = "192.168.1.10", messageRouter = route })));

        Assert.Equal("ALLEN_BRADLEY_MESSAGE_ROUTER_INVALID", exception.Code);
    }

    private static ConnectionProfile Profile(object config) => new(
        "allen-bradley",
        JsonSerializer.SerializeToElement(config),
        JsonSerializer.SerializeToElement(new { }));
}
