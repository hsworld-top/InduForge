using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusRtu.Tests;

public sealed class ModbusRtuDriverTests
{
    [Fact]
    public async Task OpenSessionConnectsAndDisposeClosesClient()
    {
        var client = new FakeHslModbusRtuClient();
        var driver = new ModbusRtuDriver(new FakeFactory(client));

        await using (var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None))
        {
            Assert.True(session.IsConnected);
            Assert.Equal(1, client.ConnectCount);
        }

        Assert.Equal(1, client.CloseCount);
    }

    [Fact]
    public async Task ReadMergesContiguousPointsWithSameType()
    {
        var client = new FakeHslModbusRtuClient
        {
            ReadHandler = request => HslReadResult.Success(new short[] { 10, 20 }),
        };
        var driver = new ModbusRtuDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        var result = await reader.ReadAsync(
            new ReadRequest([
                CreatePoint("point-1", 100, "int16"),
                CreatePoint("point-2", 101, "int16"),
            ]),
            CancellationToken.None);

        var request = Assert.Single(client.ReadRequests);
        Assert.Equal("s=1;100", request.Address);
        Assert.Equal(2, request.ElementCount);
        Assert.Equal(10, Assert.IsType<short>(result.Values[0].Value));
        Assert.Equal(20, Assert.IsType<short>(result.Values[1].Value));
        Assert.All(result.Values, value => Assert.True(value.Succeeded));
    }

    [Fact]
    public async Task ReadFallsBackToIndividualPointsWhenMergedReadFails()
    {
        var client = new FakeHslModbusRtuClient
        {
            ReadHandler = request => request.ElementCount switch
            {
                2 => HslReadResult.Failure("HSL_READ_FAILED", "批量失败", retryable: true),
                _ when request.Address.EndsWith("100", StringComparison.Ordinal) => HslReadResult.Success((short)10),
                _ => HslReadResult.Failure("HSL_READ_FAILED", "单点失败", retryable: true),
            },
        };
        var driver = new ModbusRtuDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        var result = await reader.ReadAsync(
            new ReadRequest([
                CreatePoint("point-1", 100, "int16"),
                CreatePoint("point-2", 101, "int16"),
            ]),
            CancellationToken.None);

        Assert.Equal(3, client.ReadRequests.Count);
        Assert.True(result.Values[0].Succeeded);
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("HSL_READ_FAILED", result.Values[1].ErrorCode);
    }

    [Fact]
    public async Task InvalidAddressDoesNotPreventValidPointRead()
    {
        var client = new FakeHslModbusRtuClient
        {
            ReadHandler = _ => HslReadResult.Success((short)12),
        };
        var driver = new ModbusRtuDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        var result = await reader.ReadAsync(
            new ReadRequest([
                CreatePoint("invalid", 100, "bool"),
                CreatePoint("valid", 101, "int16"),
            ]),
            CancellationToken.None);

        Assert.Single(client.ReadRequests);
        Assert.False(result.Values[0].Succeeded);
        Assert.Equal("MODBUS_BIT_INDEX_REQUIRED", result.Values[0].ErrorCode);
        Assert.True(result.Values[1].Succeeded);
    }

    [Fact]
    public async Task ReadSplitsContiguousRegistersAtProtocolLimit()
    {
        var client = new FakeHslModbusRtuClient
        {
            ReadHandler = request => HslReadResult.Success(
                Enumerable.Range(0, request.ElementCount).Select(value => (short)value).ToArray()),
        };
        var driver = new ModbusRtuDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);
        var points = Enumerable.Range(0, 126)
            .Select(index => CreatePoint($"point-{index}", 100 + index, "int16"))
            .ToArray();

        var result = await reader.ReadAsync(new ReadRequest(points), CancellationToken.None);

        Assert.Equal(2, client.ReadRequests.Count);
        Assert.Equal(125, client.ReadRequests[0].ElementCount);
        Assert.Equal(1, client.ReadRequests[1].ElementCount);
        Assert.All(result.Values, value => Assert.True(value.Succeeded));
    }

    [Fact]
    public async Task ReadSeparatesDifferentStations()
    {
        var client = new FakeHslModbusRtuClient
        {
            ReadHandler = _ => HslReadResult.Success((short)1),
        };
        var driver = new ModbusRtuDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        await reader.ReadAsync(
            new ReadRequest([
                CreatePoint("station-1", 100, "int16", station: 1),
                CreatePoint("station-2", 101, "int16", station: 2),
            ]),
            CancellationToken.None);

        Assert.Equal(2, client.ReadRequests.Count);
        Assert.Contains(client.ReadRequests, request => request.Address.StartsWith("s=1;", StringComparison.Ordinal));
        Assert.Contains(client.ReadRequests, request => request.Address.StartsWith("s=2;", StringComparison.Ordinal));
    }

    private static ConnectionProfile CreateProfile() => new(
        "modbus",
        JsonSerializer.SerializeToElement(new
        {
            portName = "COM9",
            baudRate = 9600,
            dataBits = 8,
            parity = "none",
            stopBits = "one",
            timeoutMs = 3000,
            dataFormat = "ABCD",
        }),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest CreatePoint(string key, int address, string dataType, int station = 1) => new(
        key,
        JsonSerializer.SerializeToElement(new { station, area = "holdingRegister", address }),
        dataType,
        1,
        JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslModbusRtuClient client) : IHslModbusRtuClientFactory
    {
        public IHslModbusRtuClient Create(HslModbusRtuClientOptions options) => client;
    }

    private sealed class FakeHslModbusRtuClient : IHslModbusRtuClient
    {
        public Func<HslReadRequest, HslReadResult> ReadHandler { get; init; } = _ => HslReadResult.Success((short)1);
        public List<HslReadRequest> ReadRequests { get; } = [];
        public int ConnectCount { get; private set; }
        public int CloseCount { get; private set; }
        public bool IsConnected { get; private set; }

        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
        {
            ConnectCount++;
            IsConnected = true;
            return Task.FromResult(HslOperationResult.Success());
        }

        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
        {
            CloseCount++;
            IsConnected = false;
            return Task.FromResult(HslOperationResult.Success());
        }

        public Task<HslReadResult> ReadAsync(HslReadRequest request, CancellationToken cancellationToken)
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
