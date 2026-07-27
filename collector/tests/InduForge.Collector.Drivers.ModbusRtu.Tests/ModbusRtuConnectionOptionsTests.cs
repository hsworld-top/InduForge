using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusRtu.Tests;

public sealed class ModbusRtuConnectionOptionsTests
{
    [Fact]
    public void ParseUsesConfiguredConnectionValues()
    {
        var profile = new ConnectionProfile(
            "modbus",
            JsonSerializer.SerializeToElement(new
            {
                portName = " COM12 ",
                baudRate = 19200,
                dataBits = 7,
                parity = "even",
                stopBits = "two",
                timeoutMs = 8000,
                dataFormat = "CDAB",
            }),
            JsonSerializer.SerializeToElement(new { }));

        var result = ModbusRtuConnectionOptions.Parse(profile);

        Assert.Equal("COM12", result.PortName);
        Assert.Equal(19200, result.BaudRate);
        Assert.Equal(7, result.DataBits);
        Assert.Equal(HslSerialParity.Even, result.Parity);
        Assert.Equal(HslSerialStopBits.Two, result.StopBits);
        Assert.Equal(8000, result.TimeoutMilliseconds);
        Assert.Equal(HslDataFormat.CDAB, result.DataFormat);
    }

    [Theory]
    [InlineData("", 9600, 8, "none", "one", "ABCD")]
    [InlineData("COM1", 0, 8, "none", "one", "ABCD")]
    [InlineData("COM1", 9600, 6, "none", "one", "ABCD")]
    [InlineData("COM1", 9600, 8, "mark", "one", "ABCD")]
    [InlineData("COM1", 9600, 8, "none", "onePointFive", "ABCD")]
    [InlineData("COM1", 9600, 8, "none", "one", "INVALID")]
    public void ParseRejectsInvalidConnectionValues(
        string portName,
        int baudRate,
        int dataBits,
        string parity,
        string stopBits,
        string dataFormat)
    {
        var profile = new ConnectionProfile(
            "modbus",
            JsonSerializer.SerializeToElement(new
            {
                portName,
                baudRate,
                dataBits,
                parity,
                stopBits,
                dataFormat,
            }),
            JsonSerializer.SerializeToElement(new { }));

        Assert.Throws<ModbusRtuDriverException>(() => ModbusRtuConnectionOptions.Parse(profile));
    }
}
