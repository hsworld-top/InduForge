using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
using InduForge.Collector.Drivers.SiemensS7Tcp;

namespace InduForge.Collector.Drivers.SiemensS7Tcp.Tests;

public sealed class SiemensS7TcpDriverTests
{
    [Fact]
    public async Task OpensAndClosesLongLivedSession()
    {
        var client = new FakeHslS7TcpClient();
        var driver = new SiemensS7TcpDriver(new FakeFactory(client));

        await using (var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None))
        {
            Assert.True(session.IsConnected);
            Assert.Equal("S1200 127.0.0.1:102", session.ServerName);
        }

        Assert.Equal(1, client.ConnectCount);
        Assert.Equal(1, client.CloseCount);
    }

    [Fact]
    public async Task ReadsPointsInOriginalOrderOnOneConnection()
    {
        var client = new FakeHslS7TcpClient
        {
            ReadHandler = request => request.Address switch
            {
                "M0.0" => HslReadResult.Success(true),
                "DB1.2" => HslReadResult.Success((short)25),
                _ => HslReadResult.Failure("HSL_S7_READ_FAILED", "地址不存在", retryable: false),
            },
        };
        var driver = new SiemensS7TcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        var result = await reader.ReadAsync(
            new ReadRequest([
                CreatePoint("enabled", new { area = "marker", byteOffset = 0, bitOffset = 0 }, "bool"),
                CreatePoint("temperature", new { area = "dataBlock", dbNumber = 1, byteOffset = 2 }, "int16"),
            ]),
            CancellationToken.None);

        Assert.Equal(["M0.0", "DB1.2"], client.ReadRequests.Select(item => item.Address));
        Assert.True(Assert.IsType<bool>(result.Values[0].Value));
        Assert.Equal((short)25, Assert.IsType<short>(result.Values[1].Value));
        Assert.All(result.Values, value => Assert.Equal("Good", value.Quality));
    }

    [Fact]
    public async Task InvalidPointDoesNotPreventOtherPointsFromReading()
    {
        var client = new FakeHslS7TcpClient
        {
            ReadHandler = _ => HslReadResult.Success((short)25),
        };
        var driver = new SiemensS7TcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        var result = await reader.ReadAsync(
            new ReadRequest([
                CreatePoint("invalid", new { area = "marker", byteOffset = 0 }, "bool"),
                CreatePoint("valid", new { area = "dataBlock", dbNumber = 1, byteOffset = 2 }, "int16"),
            ]),
            CancellationToken.None);

        Assert.Single(client.ReadRequests);
        Assert.False(result.Values[0].Succeeded);
        Assert.Equal("S7_BIT_OFFSET_REQUIRED", result.Values[0].ErrorCode);
        Assert.Equal("Bad", result.Values[0].Quality);
        Assert.True(result.Values[1].Succeeded);
    }

    [Fact]
    public async Task ConnectionFailureKeepsAdapterError()
    {
        var client = new FakeHslS7TcpClient
        {
            ConnectResult = HslOperationResult.Failure("HSL_S7_CONNECT_FAILED", "连接失败", retryable: true),
        };
        var driver = new SiemensS7TcpDriver(new FakeFactory(client));

        var exception = await Assert.ThrowsAsync<SiemensS7TcpDriverException>(() =>
            driver.OpenSessionAsync(CreateProfile(), CancellationToken.None));

        Assert.Equal("HSL_S7_CONNECT_FAILED", exception.Code);
        Assert.True(exception.Retryable);
    }

    private static ConnectionProfile CreateProfile() => new(
        "siemens",
        JsonSerializer.SerializeToElement(new
        {
            host = "127.0.0.1",
            port = 102,
            plcType = "S1200",
            rack = 0,
            slot = 1,
            connectTimeoutMs = 5000,
        }),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest CreatePoint(string key, object address, string dataType) => new(
        key,
        JsonSerializer.SerializeToElement(address),
        dataType,
        1,
        JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslS7TcpClient client) : IHslS7TcpClientFactory
    {
        public IHslS7TcpClient Create(HslS7TcpClientOptions options) => client;
    }

    private sealed class FakeHslS7TcpClient : IHslS7TcpClient
    {
        public HslOperationResult ConnectResult { get; init; } = HslOperationResult.Success();
        public Func<HslS7ReadRequest, HslReadResult> ReadHandler { get; init; } = _ => HslReadResult.Success((short)1);
        public List<HslS7ReadRequest> ReadRequests { get; } = [];
        public int ConnectCount { get; private set; }
        public int CloseCount { get; private set; }
        public bool IsConnected { get; private set; }

        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
        {
            ConnectCount++;
            IsConnected = ConnectResult.Succeeded;
            return Task.FromResult(ConnectResult);
        }

        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
        {
            CloseCount++;
            IsConnected = false;
            return Task.FromResult(HslOperationResult.Success());
        }

        public Task<HslReadResult> ReadAsync(HslS7ReadRequest request, CancellationToken cancellationToken)
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
