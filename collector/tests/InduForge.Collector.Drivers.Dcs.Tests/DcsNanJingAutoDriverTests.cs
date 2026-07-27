using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Dcs.Tests;

public sealed class DcsNanJingAutoDriverTests
{
    [Fact]
    public void UsesDemoDefaults()
    {
        var options = DcsNanJingAutoOptions.Parse(Profile(new { host = "127.0.0.1" }));

        Assert.Equal(502, options.Port);
        Assert.Equal(1, options.Station);
        Assert.True(options.AddressStartWithZero);
        Assert.Equal(HslDataFormat.ABCD, options.DataFormat);
    }

    [Theory]
    [InlineData("100", "100")]
    [InlineData("s=2;x=3;100", "s=2;x=3;100")]
    public void KeepsSupportedDeviceAddress(string value, string expected)
    {
        Assert.Equal(expected, DcsNanJingAutoAddress.Parse(Point(value)).Address);
    }

    [Fact]
    public async Task IsolatesInvalidAddress()
    {
        var client = new FakeClient();
        var driver = new DcsNanJingAutoDriver(new Factory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);

        var result = await ((IPointReaderSession)session).ReadAsync(
            new ReadRequest([Point("100"), Point("BAD ADDRESS")]),
            CancellationToken.None);

        Assert.Single(client.Requests);
        Assert.Equal("DCS_NANJING_AUTO_ADDRESS_INVALID", result.Values[1].ErrorCode);
    }

    private static ConnectionProfile Profile(object config) => new("dcs", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string address) => new(address, JsonSerializer.SerializeToElement(new { address }), "int16", 1, JsonSerializer.SerializeToElement(new { }));

    private sealed class Factory(IHslSpecializedNetworkClient client) : IHslSpecializedNetworkClientFactory
    {
        public IHslSpecializedNetworkClient Create(HslSpecializedNetworkClientOptions options) => client;
    }

    private sealed class FakeClient : IHslSpecializedNetworkClient
    {
        public List<HslSpecializedNetworkReadRequest> Requests { get; } = [];
        public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslSpecializedNetworkReadRequest request, CancellationToken cancellationToken) { Requests.Add(request); return Task.FromResult(HslReadResult.Success((short)1)); }
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
