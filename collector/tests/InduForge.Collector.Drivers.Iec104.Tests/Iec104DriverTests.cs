using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Iec104.Tests;

public sealed class Iec104DriverTests
{
    [Fact]
    public void ExposesIec104Descriptor()
    {
        var descriptor = new Iec104Driver().Descriptor;

        Assert.Equal("iec", descriptor.ProtocolFamily);
        Assert.Equal("iec.60870-5-104", descriptor.DriverId);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Fact]
    public async Task ReadsValidPointsInOneInterrogationAndIsolatesInvalidPoint()
    {
        var client = new FakeClient();
        var driver = new Iec104Driver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        var result = await reader.ReadAsync(new ReadRequest([
            Point("single", "singlePoint", 1, "bool"),
            Point("float", "shortFloatMeasured", 2, "float32"),
            Point("invalid", "singlePoint", 3, "uint8"),
        ]), CancellationToken.None);

        Assert.Equal(1, client.ReadCount);
        Assert.Equal(2, client.LastRequests.Count);
        Assert.True(Assert.IsType<bool>(result.Values[0].Value));
        Assert.Equal("Good", result.Values[0].Quality);
        Assert.Equal(12.5f, Assert.IsType<float>(result.Values[1].Value));
        Assert.Equal("Bad", result.Values[1].Quality);
        Assert.Equal("IEC104_DATA_TYPE_MISMATCH", result.Values[2].ErrorCode);
    }

    [Fact]
    public async Task DisposeClosesConnectedClient()
    {
        var client = new FakeClient();
        var driver = new Iec104Driver(new FakeFactory(client));
        await using (var session = await driver.OpenSessionAsync(Profile(), CancellationToken.None))
        {
            Assert.True(session.IsConnected);
        }

        Assert.Equal(1, client.CloseCount);
    }

    private static ConnectionProfile Profile() => new(
        "iec",
        JsonSerializer.SerializeToElement(new { host = "127.0.0.1" }),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest Point(string key, string informationType, int address, string dataType) => new(
        key,
        JsonSerializer.SerializeToElement(new { informationType, informationObjectAddress = address }),
        dataType,
        1,
        JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslIec104Client client) : IHslIec104ClientFactory
    {
        public IHslIec104Client Create(HslIec104ClientOptions options) => client;
    }

    private sealed class FakeClient : IHslIec104Client
    {
        public int ReadCount { get; private set; }
        public int CloseCount { get; private set; }
        public bool IsConnected { get; private set; }
        public IReadOnlyList<HslIec104ReadRequest> LastRequests { get; private set; } = [];

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

        public Task<IReadOnlyList<HslIec104PointReadResult>> ReadAsync(
            IReadOnlyList<HslIec104ReadRequest> requests,
            CancellationToken cancellationToken)
        {
            ReadCount++;
            LastRequests = requests;
            return Task.FromResult<IReadOnlyList<HslIec104PointReadResult>>([
                HslIec104PointReadResult.Success(requests[0], true, 0, null),
                HslIec104PointReadResult.Success(requests[1], 12.5f, 0x80, DateTimeOffset.UtcNow),
            ]);
        }

        public async ValueTask DisposeAsync()
        {
            if (IsConnected) await CloseAsync(CancellationToken.None);
        }
    }
}
