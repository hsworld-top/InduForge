using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.BeckhoffAds.Tests;

public sealed class BeckhoffAdsDriverTests
{
    [Fact]
    public void ExposesAdsDescriptor()
    {
        var descriptor = new BeckhoffAdsDriver().Descriptor;

        Assert.Equal("beckhoff", descriptor.ProtocolFamily);
        Assert.Equal("beckhoff.ads-tcp", descriptor.DriverId);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Fact]
    public async Task SessionReusesClientAndIsolatesPointFailures()
    {
        var client = new FakeClient
        {
            ReadHandler = request => request.Address == "s=MAIN.temperature"
                ? HslReadResult.Success(25.5f)
                : HslReadResult.Failure("HSL_BECKHOFF_ADS_READ_FAILED", "标签不存在", retryable: false),
        };
        var driver = new BeckhoffAdsDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        var result = await reader.ReadAsync(new ReadRequest([
            Point("valid", "s=MAIN.temperature"),
            Point("failed", "s=MAIN.missing"),
            new PointReadRequest("invalid", JsonSerializer.SerializeToElement(new { address = "" }), "float32", 1, JsonSerializer.SerializeToElement(new { })),
        ]), CancellationToken.None);

        Assert.Equal(2, client.ReadRequests.Count);
        Assert.Equal(25.5f, Assert.IsType<float>(result.Values[0].Value));
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("BECKHOFF_ADS_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    [Fact]
    public async Task DisposeClosesConnectedClient()
    {
        var client = new FakeClient();
        var driver = new BeckhoffAdsDriver(new FakeFactory(client));
        await using (var session = await driver.OpenSessionAsync(Profile(), CancellationToken.None))
        {
            Assert.True(session.IsConnected);
        }

        Assert.Equal(1, client.CloseCount);
    }

    private static ConnectionProfile Profile() => new(
        "beckhoff",
        JsonSerializer.SerializeToElement(new { host = "127.0.0.1" }),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest Point(string key, string address) => new(
        key,
        JsonSerializer.SerializeToElement(new { address }),
        "float32",
        1,
        JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslBeckhoffAdsClient client) : IHslBeckhoffAdsClientFactory
    {
        public IHslBeckhoffAdsClient Create(HslBeckhoffAdsClientOptions options) => client;
    }

    private sealed class FakeClient : IHslBeckhoffAdsClient
    {
        public Func<HslBeckhoffAdsReadRequest, HslReadResult> ReadHandler { get; init; } = _ => HslReadResult.Success(1);
        public List<HslBeckhoffAdsReadRequest> ReadRequests { get; } = [];
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

        public Task<HslReadResult> ReadAsync(HslBeckhoffAdsReadRequest request, CancellationToken cancellationToken)
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
