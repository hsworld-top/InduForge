using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins.Tests;

public sealed class OmronVariantTests
{
    [Theory]
    [InlineData(HslOmronVariantProtocol.Cip, 44818)]
    [InlineData(HslOmronVariantProtocol.ConnectedCip, 44818)]
    [InlineData(HslOmronVariantProtocol.HostLinkOverTcp, 2000)]
    [InlineData(HslOmronVariantProtocol.HostLinkCModeOverTcp, 2000)]
    public void NetworkProtocolsUseDemoDefaults(HslOmronVariantProtocol protocol, int expectedPort)
    {
        var options = OmronVariantOptions.Parse(Profile(new { host = "192.168.1.10" }), protocol);

        Assert.Equal(expectedPort, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal((byte)0, options.UnitNumber);
        Assert.Equal(HslDataFormat.DCBA, options.ToHslOptions().DataFormat);
    }

    [Theory]
    [InlineData(HslOmronVariantProtocol.HostLink)]
    [InlineData(HslOmronVariantProtocol.HostLinkCMode)]
    public void HostLinkSerialUsesDemoDefaults(HslOmronVariantProtocol protocol)
    {
        var options = OmronVariantOptions.Parse(Profile(new { portName = "COM3" }), protocol);

        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(7, options.DataBits);
        Assert.Equal(HslSerialParity.Even, options.Parity);
        Assert.Equal(HslSerialStopBits.One, options.StopBits);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal((byte)0, options.UnitNumber);
        Assert.Equal(HslOmronPlcType.CSCJ, options.PlcType);
    }

    [Theory]
    [InlineData("D0")]
    [InlineData("DM0")]
    [InlineData("CIO0")]
    [InlineData("H5.2")]
    [InlineData("E0.100.2")]
    [InlineData("TIM0")]
    [InlineData("CNT0")]
    public void HostLinkCModeAcceptsDemoAddresses(string address)
    {
        var result = OmronVariantAddress.Parse(Point(address, "int16", 1), true);

        Assert.Equal(address, result.Address);
    }

    [Theory]
    [InlineData("A1")]
    [InlineData("type=0xDA;A2")]
    [InlineData("Program:MainProgram.A1")]
    [InlineData("slot=2;A1")]
    [InlineData("C[0,1]")]
    public void CipAcceptsDemoTagAddresses(string address)
    {
        var result = OmronVariantAddress.Parse(Point(address, "int16", 1), false);

        Assert.Equal(address, result.Address);
    }

    [Fact]
    public void CipRejectsDateTimeArray()
    {
        var exception = Assert.Throws<OmronFinsDriverException>(() =>
            OmronVariantAddress.Parse(Point("Timestamp", "datetime", 2), false));

        Assert.Equal("OMRON_ELEMENT_COUNT_INVALID", exception.Code);
    }

    [Theory]
    [InlineData("omron.cip")]
    [InlineData("omron.connected-cip")]
    [InlineData("omron.hostlink")]
    [InlineData("omron.hostlink-over-tcp")]
    [InlineData("omron.hostlink-cmode")]
    [InlineData("omron.hostlink-cmode-over-tcp")]
    public void VariantDriversExposeExpectedDescriptor(string driverId)
    {
        IIndustrialDriver driver = driverId switch
        {
            "omron.cip" => new OmronCipDriver(),
            "omron.connected-cip" => new OmronConnectedCipDriver(),
            "omron.hostlink" => new OmronHostLinkDriver(),
            "omron.hostlink-over-tcp" => new OmronHostLinkOverTcpDriver(),
            "omron.hostlink-cmode" => new OmronHostLinkCModeDriver(),
            _ => new OmronHostLinkCModeOverTcpDriver(),
        };

        Assert.Equal(driverId, driver.Descriptor.DriverId);
        Assert.Equal("omron", driver.Descriptor.ProtocolFamily);
        Assert.Equal([1], driver.Descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, driver.Descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, driver.Descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, driver.Descriptor.Operations);
    }

    private static ConnectionProfile Profile(object config) => new(
        "omron",
        JsonSerializer.SerializeToElement(config),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest Point(string address, string dataType, int elementCount) => new(
        "point",
        JsonSerializer.SerializeToElement(new { address }),
        dataType,
        elementCount,
        JsonSerializer.SerializeToElement(new { }));
}
