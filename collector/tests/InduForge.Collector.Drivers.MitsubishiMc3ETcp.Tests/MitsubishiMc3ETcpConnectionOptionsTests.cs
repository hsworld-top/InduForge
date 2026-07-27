using System.Text.Json;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MitsubishiMc3ETcp.Tests;

public sealed class MitsubishiMc3ETcpConnectionOptionsTests
{
    [Fact]
    public void ParseUsesConfiguredValues()
    {
        var options = MitsubishiMc3ETcpConnectionOptions.Parse(new ConnectionProfile(
            "mitsubishi",
            JsonSerializer.SerializeToElement(new
            {
                host = " 192.168.1.20 ", port = 7000, connectTimeoutMs = 8000,
                networkNumber = 1, networkStationNumber = 2, targetIoStation = 1024,
            }),
            JsonSerializer.SerializeToElement(new { })));

        Assert.Equal("192.168.1.20", options.Host);
        Assert.Equal(7000, options.Port);
        Assert.Equal(8000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(1, options.NetworkNumber);
        Assert.Equal(2, options.NetworkStationNumber);
        Assert.Equal(1024, options.TargetIoStation);
    }

    [Theory]
    [InlineData("", 6000, 0)]
    [InlineData("127.0.0.1", 0, 0)]
    [InlineData("127.0.0.1", 6000, 256)]
    public void ParseRejectsInvalidValues(string host, int port, int networkNumber)
    {
        var profile = new ConnectionProfile(
            "mitsubishi",
            JsonSerializer.SerializeToElement(new { host, port, networkNumber }),
            JsonSerializer.SerializeToElement(new { }));

        Assert.Throws<MitsubishiMc3ETcpDriverException>(() => MitsubishiMc3ETcpConnectionOptions.Parse(profile));
    }
}
