using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
using InduForge.Collector.Drivers.SiemensS7Tcp;

namespace InduForge.Collector.Drivers.SiemensS7Tcp.Tests;

public sealed class SiemensS7TcpConnectionOptionsTests
{
    [Fact]
    public void ParsesConnectionOptions()
    {
        var options = SiemensS7TcpConnectionOptions.Parse(CreateProfile(new
        {
            host = "192.168.1.10",
            port = 1102,
            plcType = "S1500",
            rack = 0,
            slot = 1,
            localTsap = 0x100,
            remoteTsap = 0x102,
            connectTimeoutMs = 8000,
        }));

        Assert.Equal("192.168.1.10", options.Host);
        Assert.Equal(1102, options.Port);
        Assert.Equal(HslSiemensPlc.S1500, options.PlcType);
        Assert.Equal((byte)0, options.Rack);
        Assert.Equal((byte)1, options.Slot);
        Assert.Equal(0x100, options.LocalTsap);
        Assert.Equal(0x102, options.RemoteTsap);
        Assert.Equal(8000, options.ConnectTimeoutMilliseconds);
    }

    [Theory]
    [InlineData("S7_PLC_TYPE_INVALID", "invalid", 0, 1)]
    [InlineData("S7_CONFIG_INVALID", "S1200", 8, 1)]
    [InlineData("S7_CONFIG_INVALID", "S1200", 0, 32)]
    public void RejectsInvalidConnectionOptions(string code, string plcType, int rack, int slot)
    {
        var exception = Assert.Throws<SiemensS7TcpDriverException>(() => SiemensS7TcpConnectionOptions.Parse(CreateProfile(new
        {
            host = "127.0.0.1",
            plcType,
            rack,
            slot,
        })));

        Assert.Equal(code, exception.Code);
    }

    private static ConnectionProfile CreateProfile(object config) => new(
        "siemens",
        JsonSerializer.SerializeToElement(config),
        JsonSerializer.SerializeToElement(new { }));
}
