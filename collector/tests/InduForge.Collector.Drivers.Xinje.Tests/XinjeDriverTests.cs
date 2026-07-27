using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
using Xunit;
namespace InduForge.Collector.Drivers.Xinje.Tests;
public sealed class XinjeDriverTests
{
    [Theory]
    [InlineData("tcp", 1, "ABCD")]
    [InlineData("rtu-tcp", 1, "ABCD")]
    [InlineData("internal", 0, "CDAB")]
    public void UsesDemoTcpDefaults(string kindName, int station, string format)
    {
        var options = XinjeConnectionOptions.Parse(Profile(new { host = "127.0.0.1" }), Kind(kindName));
        Assert.Equal(502, options.Port); Assert.Equal((byte)station, options.Station); Assert.Equal(format, options.DataFormat.ToString());
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds); Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }
    [Fact]
    public void UsesDemoSerialDefaults()
    {
        var options = XinjeConnectionOptions.Parse(Profile(new { portName = "COM3" }), XinjeDriverKind.RtuSerial);
        Assert.Equal(9600, options.BaudRate); Assert.Equal(8, options.DataBits); Assert.Equal(HslSerialParity.None, options.Parity);
    }
    [Theory]
    [InlineData("d100", "int16", "XC", "D100")]
    [InlineData("m100", "bool", "XC", "M100")]
    [InlineData("hsc100", "bool", "XD", "HSC100")]
    [InlineData("s=2;hd100", "int16", "XL", "s=2;HD100")]
    public void NormalizesSeriesAddresses(string address, string type, string series, string expected)
    {
        var options = XinjeConnectionOptions.Parse(Profile(new { host = "127.0.0.1", series }), XinjeDriverKind.Tcp);
        Assert.Equal(expected, XinjeAddress.Parse(Point(address, type), options).ProtocolAddress);
    }
    [Fact]
    public async Task SessionReusesClientAndIsolatesFailures()
    {
        var client = new FakeClient(); var driver = new XinjeTcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);
        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([Point("D100", "int16"), Point("D101", "int16"), Point("bad", "int16")]), CancellationToken.None);
        Assert.Equal(2, client.Requests.Count); Assert.True(result.Values[0].Succeeded); Assert.False(result.Values[1].Succeeded); Assert.Equal("XINJE_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }
    private static XinjeDriverKind Kind(string value) => value switch { "tcp" => XinjeDriverKind.Tcp, "rtu-tcp" => XinjeDriverKind.RtuOverTcp, _ => XinjeDriverKind.InternalTcp };
    private static ConnectionProfile Profile(object config) => new("xinje", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string address, string type) => new(address, JsonSerializer.SerializeToElement(new { address }), type, 1, JsonSerializer.SerializeToElement(new { }));
    private sealed class FakeFactory(IHslXinjeClient client) : IHslXinjeClientFactory { public IHslXinjeClient Create(HslXinjeClientOptions options) => client; }
    private sealed class FakeClient : IHslXinjeClient
    {
        public List<HslXinjeReadRequest> Requests { get; } = []; public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslXinjeReadRequest request, CancellationToken cancellationToken) { Requests.Add(request); return Task.FromResult(request.Address == "D100" ? HslReadResult.Success((short)1) : HslReadResult.Failure("READ_FAILED", "失败", false)); }
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
