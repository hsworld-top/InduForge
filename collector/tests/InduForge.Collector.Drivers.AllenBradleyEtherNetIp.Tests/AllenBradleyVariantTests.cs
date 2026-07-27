using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.AllenBradleyEtherNetIp.Tests;

public sealed class AllenBradleyVariantTests
{
    [Theory]
    [InlineData(HslAllenBradleyProtocol.ConnectedCip)]
    [InlineData(HslAllenBradleyProtocol.MicroCip)]
    [InlineData(HslAllenBradleyProtocol.Pccc)]
    [InlineData(HslAllenBradleyProtocol.Slc)]
    public void NetworkVariantsUseDemoDefaults(HslAllenBradleyProtocol protocol)
    {
        var options = AllenBradleyVariantConnectionOptions.Parse(Profile(new { host = "192.168.1.10" }), protocol);

        Assert.Equal(44818, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal(HslDataFormat.DCBA, options.DataFormat);
        if (protocol == HslAllenBradleyProtocol.MicroCip)
        {
            Assert.Equal([0x01, 0x00], options.PortSlot);
        }
    }

    [Fact]
    public void Df1UsesDemoDefaults()
    {
        var options = AllenBradleyVariantConnectionOptions.Parse(Profile(new { portName = "COM3" }), HslAllenBradleyProtocol.Df1Serial);

        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(8, options.DataBits);
        Assert.Equal((byte)1, options.Station);
        Assert.Equal((byte)1, options.DestinationNode);
        Assert.Equal((byte)2, options.SourceNode);
        Assert.Equal(HslAllenBradleyCheckType.Crc16, options.CheckType);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Theory]
    [InlineData("A9:0", "string")]
    [InlineData("B9:0/1", "bool")]
    [InlineData("N9:0", "int16")]
    [InlineData("F9:0", "float32")]
    [InlineData("s=2;dst=1;src=2;N9:0", "int16")]
    public void AcceptsDemoLegacyAddresses(string address, string dataType)
    {
        var parsed = AllenBradleyEtherNetIpAddress.Parse(Point(address, dataType), AllenBradleyAddressKind.LegacyFile);

        Assert.Equal(address, parsed.HslAddress);
    }

    [Fact]
    public void ExposesAllVariantDescriptors()
    {
        var driverIds = new IIndustrialDriver[]
        {
            new AllenBradleyConnectedCipDriver(),
            new AllenBradleyMicroCipDriver(),
            new AllenBradleyPcccDriver(),
            new AllenBradleySlcDriver(),
            new AllenBradleyDf1SerialDriver(),
        }.Select(driver => driver.Descriptor.DriverId).ToArray();

        Assert.Equal([
            "allen-bradley.connected-cip",
            "allen-bradley.micro-cip",
            "allen-bradley.pccc",
            "allen-bradley.slc",
            "allen-bradley.df1-serial",
        ], driverIds);
    }

    private static ConnectionProfile Profile(object config) => new(
        "allen-bradley",
        JsonSerializer.SerializeToElement(config),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest Point(string address, string dataType) => new(
        "point",
        JsonSerializer.SerializeToElement(new { address }),
        dataType,
        1,
        JsonSerializer.SerializeToElement(new { }));
}
