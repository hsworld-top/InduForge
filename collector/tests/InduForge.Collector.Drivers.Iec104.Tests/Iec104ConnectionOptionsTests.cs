using System.Text.Json;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Iec104.Tests;

public sealed class Iec104ConnectionOptionsTests
{
    [Fact]
    public void UsesDemoRecommendedDefaults()
    {
        var options = Iec104ConnectionOptions.Parse(Profile(new { host = "192.168.1.10" }));

        Assert.Equal(2404, options.Port);
        Assert.Equal((ushort)1, options.CommonAddress);
        Assert.Equal((byte)20, options.InterrogationCode);
        Assert.Equal((byte)6, options.InterrogationReason);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal(3000, options.InterrogationTimeoutMilliseconds);
    }

    [Theory]
    [InlineData(1)]
    [InlineData(3)]
    [InlineData(20)]
    public void AcceptsDemoInterrogationCodes(int code)
    {
        var options = Iec104ConnectionOptions.Parse(Profile(new { host = "192.168.1.10", interrogationCode = code }));

        Assert.Equal((byte)code, options.InterrogationCode);
    }

    [Fact]
    public void RejectsInvalidCommonAddress()
    {
        var exception = Assert.Throws<Iec104DriverException>(() =>
            Iec104ConnectionOptions.Parse(Profile(new { host = "192.168.1.10", commonAddress = 0 })));

        Assert.Equal("IEC104_COMMON_ADDRESS_INVALID", exception.Code);
    }

    private static ConnectionProfile Profile(object config) => new(
        "iec",
        JsonSerializer.SerializeToElement(config),
        JsonSerializer.SerializeToElement(new { }));
}
