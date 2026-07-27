using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp.Tests;

public sealed class SiemensVariantDriverTests
{
    [Fact]
    public void UsesPpiSerialDemoDefaults()
    {
        var options = SiemensVariantOptions.Parse(Profile(new { portName = "COM3" }), HslSiemensVariantProtocol.PpiSerial);

        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(8, options.DataBits);
        Assert.Equal(HslSerialParity.Even, options.Parity);
        Assert.Equal(HslSerialStopBits.One, options.StopBits);
        Assert.Equal((byte)2, options.Station);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Fact]
    public void UsesPpiOverTcpDemoDefaults()
    {
        var options = SiemensVariantOptions.Parse(Profile(new { host = "127.0.0.1" }), HslSiemensVariantProtocol.PpiOverTcp);

        Assert.Equal(2000, options.Port);
        Assert.Equal((byte)2, options.Station);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Fact]
    public void UsesMpiAndWebApiDemoDefaults()
    {
        var mpi = SiemensVariantOptions.Parse(Profile(new { portName = "COM3" }), HslSiemensVariantProtocol.MpiSerial);
        var webApi = SiemensVariantOptions.Parse(Profile(new { host = "127.0.0.1" }), HslSiemensVariantProtocol.WebApi);

        Assert.True(mpi.HandshakeCheck);
        Assert.Equal((byte)2, mpi.Station);
        Assert.Equal(443, webApi.Port);
        Assert.Equal("admin", webApi.UserName);
        Assert.True(webApi.UseHttps);
    }

    [Fact]
    public void UsesFetchWriteDemoDefaults()
    {
        var options = SiemensVariantOptions.Parse(Profile(new { host = "127.0.0.1" }), HslSiemensVariantProtocol.FetchWrite);

        Assert.Equal(102, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Fact]
    public void UsesS7PlusDemoDefaults()
    {
        var options = SiemensVariantOptions.Parse(Profile(new { host = "127.0.0.1" }), HslSiemensVariantProtocol.S7Plus);
        Assert.Equal(102, options.Port);
    }

    [Fact]
    public async Task PpiSessionReusesConnectionAndIsolatesFailures()
    {
        var client = new FakeClient();
        var driver = new SiemensPpiDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { portName = "COM3" }), CancellationToken.None);

        var result = await ((IPointReaderSession)session).ReadAsync(
            new ReadRequest([
                Point("ok", new { area = "marker", byteOffset = 0, bitOffset = 0 }, "bool"),
                Point("failed", new { area = "dataBlock", dbNumber = 1, byteOffset = 2 }, "int16"),
                Point("invalid", new { area = "marker", byteOffset = 1 }, "bool"),
            ]),
            CancellationToken.None);

        Assert.Equal(2, client.Requests.Count);
        Assert.True(Assert.IsType<bool>(result.Values[0].Value));
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("S7_BIT_OFFSET_REQUIRED", result.Values[2].ErrorCode);
    }

    [Fact]
    public void FetchWriteRejectsSingleBitAddress()
    {
        var exception = Assert.Throws<SiemensVariantDriverException>(() => SiemensVariantAddress.Parse(
            Point("p", new { area = "marker", byteOffset = 0, bitOffset = 0 }, "bool"),
            HslSiemensVariantProtocol.FetchWrite));

        Assert.Equal("SIEMENS_FETCH_WRITE_BOOL_UNSUPPORTED", exception.Code);
    }

    [Fact]
    public void WebApiUsesTagAddressAndAllowsDateTime()
    {
        var address = SiemensVariantAddress.Parse(
            Point("p", new { tag = "DataBlock_1.Temperature" }, "datetime"),
            HslSiemensVariantProtocol.WebApi);

        Assert.Equal("DataBlock_1.Temperature", address.Address);
        Assert.Equal(HslValueType.DateTime, address.DataType);
    }

    [Fact]
    public void ExposesAllDescriptors()
    {
        Assert.Equal("siemens.ppi", new SiemensPpiDriver().Descriptor.DriverId);
        Assert.Equal("siemens.ppi-over-tcp", new SiemensPpiOverTcpDriver().Descriptor.DriverId);
        Assert.Equal("siemens.mpi", new SiemensMpiDriver().Descriptor.DriverId);
        Assert.Equal("siemens.fetch-write", new SiemensFetchWriteDriver().Descriptor.DriverId);
        Assert.Equal("siemens.web-api", new SiemensWebApiDriver().Descriptor.DriverId);
        Assert.Equal("siemens.s7-plus", new SiemensS7PlusDriver().Descriptor.DriverId);
    }

    private static ConnectionProfile Profile(object config) => new(
        "siemens",
        JsonSerializer.SerializeToElement(config),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest Point(string key, object address, string dataType) => new(
        key,
        JsonSerializer.SerializeToElement(address),
        dataType,
        1,
        JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslSiemensVariantClient client) : IHslSiemensVariantClientFactory
    {
        public IHslSiemensVariantClient Create(HslSiemensVariantClientOptions options) => client;
    }

    private sealed class FakeClient : IHslSiemensVariantClient
    {
        public List<HslS7ReadRequest> Requests { get; } = [];
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

        public Task<HslReadResult> ReadAsync(HslS7ReadRequest request, CancellationToken cancellationToken)
        {
            Requests.Add(request);
            return Task.FromResult(request.Address == "M0.0"
                ? HslReadResult.Success(true)
                : HslReadResult.Failure("HSL_READ_FAILED", "地址不存在", false));
        }

        public async ValueTask DisposeAsync()
        {
            if (IsConnected) await CloseAsync(CancellationToken.None);
        }
    }
}
