using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Cimon.Tests;
public sealed class CimonTests
{
    [Fact] public void UsesDemoDefaults() { var options = CimonOptions.Parse(Profile(new { host = "127.0.0.1" })); Assert.Equal(10260, options.Port); Assert.Equal((byte)1, options.FrameNumber); Assert.Equal(5000, options.ConnectTimeoutMilliseconds); Assert.Equal(10000, options.ReceiveTimeoutMilliseconds); }
    [Fact] public void AcceptsDemoAddress() => Assert.Equal("D100", CimonAddress.Parse(Point("p", "D100", "int16")).Address);
    [Fact] public async Task ReusesConnection() { var client = new FakeClient(); var driver = new CimonDriver(new Factory(client)); await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None); var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([Point("p", "D100", "int16")]), CancellationToken.None); Assert.Single(client.Requests); Assert.True(result.Values[0].Succeeded); }
    [Fact] public void ExposesDescriptor() => Assert.Equal("cimon.hmi-protocol", new CimonDriver().Descriptor.DriverId);
    private static ConnectionProfile Profile(object config) => new("cimon", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { })); private static PointReadRequest Point(string key, string address, string type) => new(key, JsonSerializer.SerializeToElement(new { address }), type, 1, JsonSerializer.SerializeToElement(new { }));
    private sealed class Factory(IHslCimonClient client) : IHslCimonClientFactory { public IHslCimonClient Create(HslCimonClientOptions options) => client; }
    private sealed class FakeClient : IHslCimonClient { public List<HslCimonReadRequest> Requests { get; } = []; public bool IsConnected { get; private set; } public Task<HslOperationResult> ConnectAsync(CancellationToken token) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); } public Task<HslOperationResult> CloseAsync(CancellationToken token) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); } public Task<HslReadResult> ReadAsync(HslCimonReadRequest request, CancellationToken token) { Requests.Add(request); return Task.FromResult(HslReadResult.Success((short)1)); } public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); } }
}
