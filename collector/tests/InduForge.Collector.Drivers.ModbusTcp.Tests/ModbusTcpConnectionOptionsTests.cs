using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusTcp.Tests;

public sealed class ModbusTcpConnectionOptionsTests
{
    [Fact]
    public void ParseUsesConfiguredConnectionValues()
    {
        var profile = new ConnectionProfile(
            "modbus",
            JsonSerializer.SerializeToElement(new
            {
                host = "192.168.1.20",
                port = 1502,
                connectTimeoutMs = 8000,
                dataFormat = "CDAB",
            }),
            JsonSerializer.SerializeToElement(new { }));

        var result = ModbusTcpConnectionOptions.Parse(profile);

        Assert.Equal("192.168.1.20", result.Host);
        Assert.Equal(1502, result.Port);
        Assert.Equal(8000, result.ConnectTimeoutMilliseconds);
        Assert.Equal(HslDataFormat.CDAB, result.DataFormat);
    }

    [Theory]
    [InlineData("", 502, "ABCD")]
    [InlineData("127.0.0.1", 0, "ABCD")]
    [InlineData("127.0.0.1", 502, "INVALID")]
    public void ParseRejectsInvalidConnectionValues(string host, int port, string dataFormat)
    {
        var profile = new ConnectionProfile(
            "modbus",
            JsonSerializer.SerializeToElement(new { host, port, dataFormat }),
            JsonSerializer.SerializeToElement(new { }));

        Assert.Throws<ModbusTcpDriverException>(() => ModbusTcpConnectionOptions.Parse(profile));
    }
}
