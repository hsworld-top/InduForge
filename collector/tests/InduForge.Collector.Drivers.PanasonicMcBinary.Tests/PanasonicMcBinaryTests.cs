using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
using Xunit;

namespace InduForge.Collector.Drivers.PanasonicMcBinary.Tests;

public sealed class PanasonicMcBinaryTests
{
    [Fact]
    public void UsesDemoDefaults()
    {
        var options = PanasonicMcOptions.Parse(Profile(new { host = "127.0.0.1" }));
        Assert.Equal(5000, options.Port);
        Assert.Equal((byte)0, options.NetworkStationNumber);
        Assert.Equal((ushort)1023, options.TargetIoStation);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal(HslDataFormat.DCBA, options.DataFormat);
    }

    [Theory]
    [InlineData("r2.1", "bool", "R2.1")]
    [InlineData("tn0", "int16", "TN0")]
    [InlineData("dt100", "float32", "DT100")]
    public void NormalizesDemoAddresses(string address, string dataType, string expected)
    {
        Assert.Equal(expected, PanasonicMcAddress.Parse(Point(address, dataType)).Address);
    }

    [Fact]
    public async Task ReusesConnectionAndIsolatesPointFailures()
    {
        var client = new FakeClient();
        var driver = new PanasonicMcBinaryDriver(new Factory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);
        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([Point("D0", "int16"), Point("X0", "int16"), Point("D1", "int16")]), CancellationToken.None);
        Assert.Equal(2, client.Requests.Count);
        Assert.True(result.Values[0].Succeeded);
        Assert.Equal("PANASONIC_MC_WORD_AREA_INVALID", result.Values[1].ErrorCode);
        Assert.False(result.Values[2].Succeeded);
    }

    private static ConnectionProfile Profile(object config) => new("panasonic", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string address, string dataType) => new(address, JsonSerializer.SerializeToElement(new { address }), dataType, 1, JsonSerializer.SerializeToElement(new { }));
    private sealed class Factory(IHslPanasonicMcClient client) : IHslPanasonicMcClientFactory { public IHslPanasonicMcClient Create(HslPanasonicMcClientOptions options) => client; }
    private sealed class FakeClient : IHslPanasonicMcClient
    {
        public List<HslPanasonicMcReadRequest> Requests { get; } = [];
        public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslPanasonicMcReadRequest request, CancellationToken cancellationToken) { Requests.Add(request); return Task.FromResult(request.Address == "D0" ? HslReadResult.Success((short)1) : HslReadResult.Failure("READ_FAILED", "失败", false)); }
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
