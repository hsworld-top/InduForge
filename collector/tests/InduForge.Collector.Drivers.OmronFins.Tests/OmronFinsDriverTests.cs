using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins.Tests;

public sealed class OmronFinsDriverTests
{
    [Fact]
    public void DriversExposeIndependentIds()
    {
        Assert.Equal("omron.fins-tcp", new OmronFinsTcpDriver().Descriptor.DriverId);
        Assert.Equal("omron.fins-udp", new OmronFinsUdpDriver().Descriptor.DriverId);
    }

    [Theory]
    [InlineData(true, HslOmronFinsTransport.Tcp)]
    [InlineData(false, HslOmronFinsTransport.Udp)]
    public async Task DriverPassesExpectedTransport(bool tcp, HslOmronFinsTransport expected)
    {
        var client = new FakeClient();
        var factory = new FakeFactory(client);
        var driver = tcp ? (OmronFinsDriverBase)new OmronFinsTcpDriver(factory) : new OmronFinsUdpDriver(factory);

        await using var session = await driver.OpenSessionAsync(Profile(), CancellationToken.None);

        Assert.Equal(expected, factory.Options!.Transport);
        Assert.True(session.IsConnected);
    }

    [Fact]
    public async Task SessionReusesClientAndIsolatesPointFailures()
    {
        var client = new FakeClient
        {
            ReadHandler = request => request.Address == "D100"
                ? HslReadResult.Success((short)25)
                : HslReadResult.Failure("HSL_OMRON_FINS_READ_FAILED", "地址不存在", retryable: false),
        };
        var driver = new OmronFinsTcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        var result = await reader.ReadAsync(new ReadRequest([
            Point("valid", "D100"),
            Point("failed", "D101"),
            new PointReadRequest("invalid", JsonSerializer.SerializeToElement(new { address = "" }), "int16", 1, JsonSerializer.SerializeToElement(new { })),
        ]), CancellationToken.None);

        Assert.Equal(2, client.ReadRequests.Count);
        Assert.Equal((short)25, Assert.IsType<short>(result.Values[0].Value));
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("OMRON_FINS_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    [Fact]
    public async Task DisposeClosesOpenedSession()
    {
        var client = new FakeClient();
        var driver = new OmronFinsUdpDriver(new FakeFactory(client));
        await using (var session = await driver.OpenSessionAsync(Profile(), CancellationToken.None))
        {
            Assert.True(session.IsConnected);
        }

        Assert.Equal(1, client.CloseCount);
    }

    private static ConnectionProfile Profile() => new(
        "omron",
        JsonSerializer.SerializeToElement(new { host = "127.0.0.1", port = 9600 }),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest Point(string key, string address) => new(
        key,
        JsonSerializer.SerializeToElement(new { address }),
        "int16",
        1,
        JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslOmronFinsClient client) : IHslOmronFinsClientFactory
    {
        public HslOmronFinsClientOptions? Options { get; private set; }

        public IHslOmronFinsClient Create(HslOmronFinsClientOptions options)
        {
            Options = options;
            return client;
        }
    }

    private sealed class FakeClient : IHslOmronFinsClient
    {
        public Func<HslOmronFinsReadRequest, HslReadResult> ReadHandler { get; init; } = _ => HslReadResult.Success((short)1);
        public List<HslOmronFinsReadRequest> ReadRequests { get; } = [];
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

        public Task<HslReadResult> ReadAsync(HslOmronFinsReadRequest request, CancellationToken cancellationToken)
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
