using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.GeSrtpTcp.Tests;

public sealed class GeSrtpTcpTests
{
    [Fact]
    public void UsesDemoDefaults()
    {
        var options = GeSrtpTcpConnectionOptions.Parse(Profile(new { host = "192.168.1.10" }));
        Assert.Equal(18245, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal(HslDataFormat.DCBA, options.DataFormat);
    }

    [Theory]
    [InlineData("R1")]
    [InlineData("I1")]
    [InlineData("Q1")]
    [InlineData("M1")]
    [InlineData("T1")]
    [InlineData("SA1")]
    [InlineData("SB1")]
    [InlineData("SC1")]
    [InlineData("S1")]
    [InlineData("G1")]
    [InlineData("AI1")]
    [InlineData("AQ1")]
    public void AcceptsDemoAddressAreas(string address)
    {
        Assert.Equal(address, GeSrtpTcpAddress.Parse(Point("point", address, "int16")).ProtocolAddress);
    }

    [Fact]
    public void NormalizesAddressAndRejectsBoolWordArea()
    {
        Assert.Equal("M1", GeSrtpTcpAddress.Parse(Point("point", " m01 ", "bool")).ProtocolAddress);
        var exception = Assert.Throws<GeSrtpTcpDriverException>(() => GeSrtpTcpAddress.Parse(Point("point", "R1", "bool")));
        Assert.Equal("GE_SRTP_BOOL_AREA_INVALID", exception.Code);
    }

    [Fact]
    public async Task SessionReusesClientAndIsolatesFailures()
    {
        var client = new FakeClient();
        var driver = new GeSrtpTcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);
        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([
            Point("valid", "R1", "int16"),
            Point("failed", "R2", "int16"),
            Point("invalid", "D100", "int16"),
        ]), CancellationToken.None);

        Assert.Equal(2, client.Requests.Count);
        Assert.Equal((short)123, Assert.IsType<short>(result.Values[0].Value));
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("GE_SRTP_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    [Fact]
    public void ExposesDescriptor()
    {
        var descriptor = new GeSrtpTcpDriver().Descriptor;
        Assert.Equal("ge.srtp-tcp", descriptor.DriverId);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    private static ConnectionProfile Profile(object config) =>
        new("ge", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest Point(string key, string address, string dataType) =>
        new(key, JsonSerializer.SerializeToElement(new { address }), dataType, 1, JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslGeSrtpClient client) : IHslGeSrtpClientFactory
    {
        public IHslGeSrtpClient Create(HslGeSrtpClientOptions options) => client;
    }

    private sealed class FakeClient : IHslGeSrtpClient
    {
        public List<HslGeSrtpReadRequest> Requests { get; } = [];
        public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
        {
            IsConnected = true;
            return Task.FromResult(HslOperationResult.Success());
        }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
        {
            IsConnected = false;
            return Task.FromResult(HslOperationResult.Success());
        }
        public Task<HslReadResult> ReadAsync(HslGeSrtpReadRequest request, CancellationToken cancellationToken)
        {
            Requests.Add(request);
            return Task.FromResult(request.Address == "R1"
                ? HslReadResult.Success((short)123)
                : HslReadResult.Failure("HSL_READ_FAILED", "地址不存在", false));
        }
        public async ValueTask DisposeAsync()
        {
            if (IsConnected) await CloseAsync(CancellationToken.None);
        }
    }
}
