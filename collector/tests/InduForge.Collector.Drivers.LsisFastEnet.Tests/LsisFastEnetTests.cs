using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.LsisFastEnet.Tests;
public sealed class LsisFastEnetTests
{
    [Fact] public void UsesDemoDefaults() { var value = LsisFastEnetConnectionOptions.Parse(Profile(new { host = "127.0.0.1" })); Assert.Equal(2004, value.Port); Assert.Equal(HslLsisCpuType.XGK, value.CpuType); Assert.Equal("LSIS-XGT", value.CompanyId); Assert.Equal((byte)0, value.BaseNumber); Assert.Equal((byte)3, value.SlotNumber); }
    [Theory] [InlineData("MB100")] [InlineData("MX100")] [InlineData("PX100")] [InlineData("IX0.0.0")] [InlineData("QX0.0.0")] public void AcceptsDemoAddresses(string address) => Assert.Equal(address, LsisFastEnetAddress.Parse(Point(address, "int16")).ProtocolAddress);
    [Fact] public void ExposesDescriptor() { var descriptor = new LsisFastEnetDriver().Descriptor; Assert.Equal("lsis.fast-enet", descriptor.DriverId); Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations); }
    [Theory]
    [InlineData(HslLsisSerialProtocol.CnetSerial)]
    [InlineData(HslLsisSerialProtocol.CpuSerial)]
    public void SerialVariantsUseDemoDefaults(HslLsisSerialProtocol protocol)
    {
        var options = LsisSerialVariantOptions.Parse(Profile(new { portName = "COM3" }), protocol);
        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(8, options.DataBits);
        Assert.Equal(HslSerialParity.None, options.Parity);
        Assert.Equal(HslSerialStopBits.One, options.StopBits);
        Assert.Equal((byte)1, options.Station);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }
    [Fact]
    public void CnetOverTcpUsesDemoDefaults()
    {
        var options = LsisSerialVariantOptions.Parse(Profile(new { host = "127.0.0.1" }), HslLsisSerialProtocol.CnetOverTcp);
        Assert.Equal(2000, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal((byte)1, options.Station);
    }
    [Theory]
    [InlineData("lsis.cnet")]
    [InlineData("lsis.cnet-over-tcp")]
    [InlineData("lsis.cpu-serial")]
    public void SerialVariantDriversExposeDescriptor(string driverId)
    {
        IIndustrialDriver driver = driverId switch
        {
            "lsis.cnet" => new LsisCnetDriver(),
            "lsis.cnet-over-tcp" => new LsisCnetOverTcpDriver(),
            _ => new LsisCpuSerialDriver(),
        };
        Assert.Equal(driverId, driver.Descriptor.DriverId);
        Assert.Equal("lsis", driver.Descriptor.ProtocolFamily);
        Assert.Contains(DriverOperations.PointRead, driver.Descriptor.Operations);
    }
    [Fact] public void SerialVariantsAcceptDemoAddress() => Assert.Equal("D100", LsisSerialVariantAddress.Parse(Point("D100", "int16")).Address);
    private static ConnectionProfile Profile(object config) => new("lsis", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string address, string type) => new("point", JsonSerializer.SerializeToElement(new { address }), type, 1, JsonSerializer.SerializeToElement(new { }));
}
