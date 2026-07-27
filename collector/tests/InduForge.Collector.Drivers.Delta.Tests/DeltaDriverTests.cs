using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
using Xunit;

namespace InduForge.Collector.Drivers.Delta.Tests;

public sealed class DeltaDriverTests
{
    [Theory]
    [InlineData("tcp", 502, 1)]
    [InlineData("rtu-tcp", 502, 1)]
    [InlineData("ascii-tcp", 502, 1)]
    public void UsesDemoTcpDefaults(string kindName, int port, int station)
    {
        var options = DeltaConnectionOptions.Parse(Profile(new { host = "127.0.0.1" }), Kind(kindName));
        Assert.Equal(port, options.Port); Assert.Equal((byte)station, options.Station); Assert.Equal(HslDeltaSeries.Dvp, options.Series);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds); Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Theory]
    [InlineData("rtu")]
    [InlineData("ascii")]
    public void UsesDemoSerialDefaults(string kindName)
    {
        var options = DeltaConnectionOptions.Parse(Profile(new { portName = "COM3" }), Kind(kindName));
        Assert.Equal(9600, options.BaudRate); Assert.Equal(7, options.DataBits); Assert.Equal(HslSerialParity.Even, options.Parity);
    }

    [Theory]
    [InlineData("d100", "int16", "D100")]
    [InlineData("m100", "bool", "M100")]
    [InlineData("s=2;d100", "int16", "s=2;D100")]
    public void NormalizesDocumentedAddresses(string address, string type, string expected) => Assert.Equal(expected, DeltaAddress.Parse(Point(address, type)).ProtocolAddress);

    [Fact]
    public async Task SessionReusesClientAndIsolatesFailures()
    {
        var client = new FakeClient(); var driver = new DeltaTcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);
        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([Point("D100", "int16"), Point("D101", "int16"), Point("bad", "int16")]), CancellationToken.None);
        Assert.Equal(2, client.Requests.Count); Assert.True(result.Values[0].Succeeded); Assert.False(result.Values[1].Succeeded); Assert.Equal("DELTA_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    private static DeltaDriverKind Kind(string name) => name switch { "tcp" => DeltaDriverKind.Tcp, "rtu-tcp" => DeltaDriverKind.RtuOverTcp, "ascii-tcp" => DeltaDriverKind.AsciiOverTcp, "rtu" => DeltaDriverKind.RtuSerial, _ => DeltaDriverKind.AsciiSerial };
    private static ConnectionProfile Profile(object config) => new("delta", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string address, string type) => new(address, JsonSerializer.SerializeToElement(new { address }), type, 1, JsonSerializer.SerializeToElement(new { }));
    private sealed class FakeFactory(IHslDeltaClient client) : IHslDeltaClientFactory { public IHslDeltaClient Create(HslDeltaClientOptions options) => client; }
    private sealed class FakeClient : IHslDeltaClient
    {
        public List<HslDeltaReadRequest> Requests { get; } = []; public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslDeltaReadRequest request, CancellationToken cancellationToken) { Requests.Add(request); return Task.FromResult(request.Address == "D100" ? HslReadResult.Success((short)1) : HslReadResult.Failure("READ_FAILED", "失败", false)); }
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
