using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins.Tests;

public sealed class OmronFinsConnectionOptionsTests
{
    [Fact]
    public void TcpUsesRecommendedDefaults()
    {
        var options = OmronFinsConnectionOptions.Parse(Profile(new { host = "192.168.1.10" }), HslOmronFinsTransport.Tcp);

        Assert.Equal(9600, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(5000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal(HslOmronPlcType.CSCJ, options.PlcType);
        Assert.Equal(500, options.ReadSplits);
        Assert.Equal(HslDataFormat.CDAB, options.DataFormat);
        Assert.True(options.StringReverseByteWord);
    }

    [Fact]
    public void UdpMapsAdvancedSettings()
    {
        var options = OmronFinsConnectionOptions.Parse(Profile(new
        {
            host = "192.168.1.10",
            port = 9601,
            plcType = "CV",
            readSplits = 999,
            gct = 3,
            sid = 9,
            dataFormat = "ABCD",
            stringReverseByteWord = false,
        }), HslOmronFinsTransport.Udp);

        Assert.Equal(HslOmronFinsTransport.Udp, options.Transport);
        Assert.Equal(HslOmronPlcType.CV, options.PlcType);
        Assert.Equal(999, options.ReadSplits);
        Assert.Equal((byte)3, options.Gct);
        Assert.Equal((byte)9, options.Sid);
        Assert.False(options.StringReverseByteWord);
    }

    [Theory]
    [InlineData("invalid", "OMRON_FINS_PLC_TYPE_INVALID")]
    [InlineData("CSCJ", null)]
    public void ValidatesPlcType(string plcType, string? errorCode)
    {
        var exception = Record.Exception(() => OmronFinsConnectionOptions.Parse(
            Profile(new { host = "192.168.1.10", plcType }),
            HslOmronFinsTransport.Tcp));

        if (errorCode is null)
        {
            Assert.Null(exception);
            return;
        }
        Assert.Equal(errorCode, Assert.IsType<OmronFinsDriverException>(exception).Code);
    }

    [Fact]
    public void RejectsOversizedReadSplits()
    {
        var exception = Assert.Throws<OmronFinsDriverException>(() => OmronFinsConnectionOptions.Parse(
            Profile(new { host = "192.168.1.10", readSplits = 1000 }),
            HslOmronFinsTransport.Tcp));

        Assert.Equal("OMRON_FINS_READ_SPLITS_INVALID", exception.Code);
    }

    private static ConnectionProfile Profile(object config) => new(
        "omron",
        JsonSerializer.SerializeToElement(config),
        JsonSerializer.SerializeToElement(new { }));
}
