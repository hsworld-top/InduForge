using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
using Xunit;

namespace InduForge.Collector.Drivers.Keyence.Tests;

public sealed class KeyenceDriverTests
{
    [Theory]
    [InlineData("keyence.mc-3e-tcp", 6000)]
    [InlineData("keyence.mc-ascii-tcp", 6000)]
    [InlineData("keyence.kv-old-tcp", 8501)]
    [InlineData("keyence.nano-tcp", 8501)]
    [InlineData("keyence.nano-serial-over-tcp", 8501)]
    public void UsesDemoDefaultPorts(string driverId, int port)
    {
        var kind = driverId switch
        {
            "keyence.mc-3e-tcp" => KeyenceDriverKind.Mc3E,
            "keyence.mc-ascii-tcp" => KeyenceDriverKind.McAscii,
            "keyence.kv-old-tcp" => KeyenceDriverKind.KvOld,
            "keyence.nano-serial-over-tcp" => KeyenceDriverKind.NanoSerialOverTcp,
            _ => KeyenceDriverKind.Nano,
        };
        var options = KeyenceConnectionOptions.Parse(Profile(new { host = "127.0.0.1" }), kind);
        Assert.Equal(port, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Fact]
    public void NanoSerialUsesDemoDefaults()
    {
        var options = KeyenceConnectionOptions.Parse(Profile(new { portName = "COM3" }), KeyenceDriverKind.NanoSerial);

        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(8, options.DataBits);
        Assert.Equal(HslSerialParity.None, options.Parity);
        Assert.Equal(HslSerialStopBits.One, options.StopBits);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Fact]
    public void ExposesNewVariantDescriptors()
    {
        Assert.Equal("keyence.mc-ascii-tcp", new KeyenceMcAsciiTcpDriver().Descriptor.DriverId);
        Assert.Equal("keyence.nano-serial", new KeyenceNanoSerialDriver().Descriptor.DriverId);
        Assert.Equal("keyence.nano-serial-over-tcp", new KeyenceNanoSerialOverTcpDriver().Descriptor.DriverId);
    }

    [Theory]
    [InlineData("dm100", "mc", "DM100")]
    [InlineData("d100", "old", "D100")]
    [InlineData("unit=2;1000", "nano", "UNIT=2;1000")]
    public void NormalizesDocumentedAddresses(string source, string kindName, string expected)
    {
        var kind = kindName switch { "mc" => KeyenceDriverKind.Mc3E, "old" => KeyenceDriverKind.KvOld, _ => KeyenceDriverKind.Nano };
        Assert.Equal(expected, KeyenceAddress.Parse(Point(source), kind).ProtocolAddress);
    }

    [Fact]
    public async Task SessionReusesClientAndIsolatesPointFailures()
    {
        var client = new FakeClient();
        var driver = new KeyenceMc3ETcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);
        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([Point("DM100"), Point("DM101"), Point("bad address")]), CancellationToken.None);
        Assert.Equal(2, client.Requests.Count);
        Assert.True(result.Values[0].Succeeded);
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("KEYENCE_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    private static ConnectionProfile Profile(object config) => new("keyence", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string address) => new(address, JsonSerializer.SerializeToElement(new { address }), "int16", 1, JsonSerializer.SerializeToElement(new { }));
    private sealed class FakeFactory(IHslKeyenceTcpClient client) : IHslKeyenceTcpClientFactory { public IHslKeyenceTcpClient Create(HslKeyenceTcpClientOptions options) => client; }
    private sealed class FakeClient : IHslKeyenceTcpClient
    {
        public List<HslKeyenceTcpReadRequest> Requests { get; } = [];
        public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslKeyenceTcpReadRequest request, CancellationToken cancellationToken) { Requests.Add(request); return Task.FromResult(request.Address == "DM100" ? HslReadResult.Success((short)1) : HslReadResult.Failure("READ_FAILED", "失败", false)); }
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
