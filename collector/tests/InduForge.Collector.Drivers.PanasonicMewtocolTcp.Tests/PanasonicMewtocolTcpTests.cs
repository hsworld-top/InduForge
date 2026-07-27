using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.PanasonicMewtocolTcp.Tests;

public sealed class PanasonicMewtocolTcpTests
{
    [Fact]
    public void UsesDemoDefaults()
    {
        var options = PanasonicMewtocolTcpConnectionOptions.Parse(Profile(new { host = "192.168.1.10" }));
        Assert.Equal(2000, options.Port);
        Assert.Equal((byte)0xEE, options.Station);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal(HslDataFormat.DCBA, options.DataFormat);
    }

    [Theory]
    [InlineData("D0")]
    [InlineData("F0")]
    [InlineData("K0")]
    [InlineData("T0")]
    [InlineData("C0")]
    [InlineData("s=2;D100")]
    public void AcceptsDemoAddressForms(string address)
    {
        var parsed = PanasonicMewtocolTcpAddress.Parse(Point("point", address, "int16"));
        Assert.Equal(address, parsed.ProtocolAddress);
    }

    [Fact]
    public async Task SessionReusesClientAndIsolatesFailures()
    {
        var client = new FakeClient();
        var driver = new PanasonicMewtocolTcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);
        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([
            Point("valid", "D0", "int16"),
            Point("failed", "D1", "int16"),
            Point("invalid", "D 2", "int16"),
        ]), CancellationToken.None);

        Assert.Equal(2, client.Requests.Count);
        Assert.Equal((short)123, Assert.IsType<short>(result.Values[0].Value));
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("PANASONIC_MEWTOCOL_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    [Fact]
    public void ExposesDescriptor()
    {
        var descriptor = new PanasonicMewtocolTcpDriver().Descriptor;
        Assert.Equal("panasonic.mewtocol-tcp", descriptor.DriverId);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Fact]
    public void SerialUsesDemoDefaults()
    {
        var options = PanasonicMewtocolSerialConnectionOptions.Parse(Profile(new { portName = "COM9" }));
        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(8, options.DataBits);
        Assert.Equal((byte)238, options.Station);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal("panasonic.mewtocol-serial", new PanasonicMewtocolSerialDriver().Descriptor.DriverId);
    }

    private static ConnectionProfile Profile(object config) => new("panasonic", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string key, string address, string dataType) => new(key, JsonSerializer.SerializeToElement(new { address }), dataType, 1, JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslPanasonicMewtocolClient client) : IHslPanasonicMewtocolClientFactory
    {
        public IHslPanasonicMewtocolClient Create(HslPanasonicMewtocolClientOptions options) => client;
    }

    private sealed class FakeClient : IHslPanasonicMewtocolClient
    {
        public List<HslPanasonicMewtocolReadRequest> Requests { get; } = [];
        public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslPanasonicMewtocolReadRequest request, CancellationToken cancellationToken)
        {
            Requests.Add(request);
            return Task.FromResult(request.Address == "D0" ? HslReadResult.Success((short)123) : HslReadResult.Failure("HSL_READ_FAILED", "地址不存在", false));
        }
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
