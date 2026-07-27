using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.AllenBradleyEtherNetIp.Tests;

public sealed class AllenBradleyEtherNetIpDriverTests
{
    [Fact]
    public void ExposesEtherNetIpDescriptor()
    {
        var descriptor = new AllenBradleyEtherNetIpDriver().Descriptor;

        Assert.Equal("allen-bradley", descriptor.ProtocolFamily);
        Assert.Equal("allen-bradley.ethernet-ip", descriptor.DriverId);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
    }

    [Fact]
    public async Task SessionReusesClientAndIsolatesPointFailures()
    {
        var client = new FakeClient
        {
            ReadHandler = request => request.Address == "Temperature"
                ? HslReadResult.Success(25.5f)
                : HslReadResult.Failure("HSL_ALLEN_BRADLEY_READ_FAILED", "标签不存在", retryable: false),
        };
        var driver = new AllenBradleyEtherNetIpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        var result = await reader.ReadAsync(new ReadRequest([
            Point("valid", "Temperature"),
            Point("failed", "MissingTag"),
            new PointReadRequest("invalid", JsonSerializer.SerializeToElement(new { address = "" }), "float32", 1, JsonSerializer.SerializeToElement(new { })),
        ]), CancellationToken.None);

        Assert.Equal(2, client.ReadRequests.Count);
        Assert.Equal(25.5f, Assert.IsType<float>(result.Values[0].Value));
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("ALLEN_BRADLEY_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    [Fact]
    public async Task DisposeClosesConnectedClient()
    {
        var client = new FakeClient();
        var driver = new AllenBradleyEtherNetIpDriver(new FakeFactory(client));
        await using (var session = await driver.OpenSessionAsync(Profile(), CancellationToken.None))
        {
            Assert.True(session.IsConnected);
        }

        Assert.Equal(1, client.CloseCount);
    }

    private static ConnectionProfile Profile() => new(
        "allen-bradley",
        JsonSerializer.SerializeToElement(new { host = "127.0.0.1", port = 44818 }),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest Point(string key, string address) => new(
        key,
        JsonSerializer.SerializeToElement(new { address }),
        "float32",
        1,
        JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslAllenBradleyClient client) : IHslAllenBradleyClientFactory
    {
        public IHslAllenBradleyClient Create(HslAllenBradleyClientOptions options) => client;
    }

    private sealed class FakeClient : IHslAllenBradleyClient
    {
        public Func<HslAllenBradleyReadRequest, HslReadResult> ReadHandler { get; init; } = _ => HslReadResult.Success(1);
        public List<HslAllenBradleyReadRequest> ReadRequests { get; } = [];
        public int CloseCount { get; private set; }
        public bool IsConnected { get; private set; }

        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
        {
            IsConnected = true;
            return Task.FromResult(HslOperationResult.Success());
        }

        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
        {
            CloseCount++;
            IsConnected = false;
            return Task.FromResult(HslOperationResult.Success());
        }

        public Task<HslReadResult> ReadAsync(HslAllenBradleyReadRequest request, CancellationToken cancellationToken)
        {
            ReadRequests.Add(request);
            return Task.FromResult(ReadHandler(request));
        }

        public async ValueTask DisposeAsync()
        {
            if (IsConnected) await CloseAsync(CancellationToken.None);
        }
    }
}
