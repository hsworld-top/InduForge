using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
using Xunit;

namespace InduForge.Collector.Drivers.Fuji.Tests;

public sealed class FujiDriverTests
{
    [Theory]
    [InlineData("command", 7000)]
    [InlineData("sph", 507)]
    [InlineData("spb-tcp", 9600)]
    public void UsesDemoTcpDefaults(string kindName, int port)
    {
        var options = FujiConnectionOptions.Parse(Profile(new { host = "127.0.0.1" }), Kind(kindName));
        Assert.Equal(port, options.Port); Assert.Equal(5000, options.ConnectTimeoutMilliseconds); Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Fact]
    public void UsesDemoProtocolSpecificDefaults()
    {
        var command = FujiConnectionOptions.Parse(Profile(new { host = "127.0.0.1" }), FujiDriverKind.CommandSettingTypeTcp);
        var sph = FujiConnectionOptions.Parse(Profile(new { host = "127.0.0.1" }), FujiDriverKind.SphTcp);
        var spb = FujiConnectionOptions.Parse(Profile(new { host = "127.0.0.1" }), FujiDriverKind.SpbOverTcp);
        Assert.False(command.DataSwap); Assert.Equal(HslDataFormat.ABCD, command.DataFormat);
        Assert.Equal((byte)254, sph.ConnectionId); Assert.Equal(HslDataFormat.DCBA, sph.DataFormat);
        Assert.Equal((byte)0, spb.Station); Assert.Equal(HslDataFormat.DCBA, spb.DataFormat);
    }

    [Fact]
    public void UsesDemoSerialDefaults()
    {
        var options = FujiConnectionOptions.Parse(Profile(new { portName = "COM3" }), FujiDriverKind.SpbSerial);
        Assert.Equal(9600, options.BaudRate); Assert.Equal(8, options.DataBits); Assert.Equal(HslSerialParity.None, options.Parity); Assert.Equal(HslSerialStopBits.One, options.StopBits);
    }

    [Theory]
    [InlineData("command", "w9.0", "W9.0")]
    [InlineData("command", "ts0", "TS0")]
    [InlineData("sph", "m10.100.5", "M10.100.5")]
    [InlineData("sph", "i0", "I0")]
    [InlineData("spb-tcp", "s=02;d10.12", "s=2;D10.12")]
    [InlineData("spb", "tn0", "TN0")]
    public void NormalizesDemoAddresses(string kindName, string address, string expected) => Assert.Equal(expected, FujiAddress.Parse(Point(address, "int16"), Kind(kindName)).ProtocolAddress);

    [Fact]
    public async Task SessionReusesClientAndIsolatesFailures()
    {
        var client = new FakeClient(); var driver = new FujiSpbOverTcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);
        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([Point("D100", "int16"), Point("D101", "int16"), Point("bad", "int16")]), CancellationToken.None);
        Assert.Equal(2, client.Requests.Count); Assert.True(result.Values[0].Succeeded); Assert.False(result.Values[1].Succeeded); Assert.Equal("FUJI_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    private static FujiDriverKind Kind(string name) => name switch { "command" => FujiDriverKind.CommandSettingTypeTcp, "sph" => FujiDriverKind.SphTcp, "spb-tcp" => FujiDriverKind.SpbOverTcp, _ => FujiDriverKind.SpbSerial };
    private static ConnectionProfile Profile(object config) => new("fuji", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string address, string type) => new(address, JsonSerializer.SerializeToElement(new { address }), type, 1, JsonSerializer.SerializeToElement(new { }));
    private sealed class FakeFactory(IHslFujiClient client) : IHslFujiClientFactory { public IHslFujiClient Create(HslFujiClientOptions options) => client; }
    private sealed class FakeClient : IHslFujiClient
    {
        public List<HslFujiReadRequest> Requests { get; } = []; public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslFujiReadRequest request, CancellationToken cancellationToken) { Requests.Add(request); return Task.FromResult(request.Address == "D100" ? HslReadResult.Success((short)1) : HslReadResult.Failure("READ_FAILED", "失败", false)); }
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
