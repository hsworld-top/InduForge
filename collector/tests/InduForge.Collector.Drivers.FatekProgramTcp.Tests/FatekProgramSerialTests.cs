using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.FatekProgramTcp.Tests;

public sealed class FatekProgramSerialTests
{
    [Fact]
    public void UsesDemoDefaults()
    {
        var options = FatekProgramSerialConnectionOptions.Parse(Profile(new { portName = "COM3" }));

        Assert.Equal("COM3", options.PortName);
        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(7, options.DataBits);
        Assert.Equal(HslSerialParity.Even, options.Parity);
        Assert.Equal(HslSerialStopBits.Two, options.StopBits);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal((byte)1, options.Station);
    }

    [Fact]
    public async Task SessionReusesClientAndIsolatesFailures()
    {
        var client = new FakeClient();
        var driver = new FatekProgramSerialDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { portName = "COM3" }), CancellationToken.None);

        var result = await ((IPointReaderSession)session).ReadAsync(
            new ReadRequest([
                Point("ok", "D0", "int16"),
                Point("failed", "D1", "int16"),
                Point("invalid", "M0", "int16"),
            ]),
            CancellationToken.None);

        Assert.Equal(2, client.Requests.Count);
        Assert.Equal((short)123, Assert.IsType<short>(result.Values[0].Value));
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("FATEK_PROGRAM_WORD_AREA_INVALID", result.Values[2].ErrorCode);
    }

    [Fact]
    public void ExposesDescriptor() =>
        Assert.Equal("fatek.program-serial", new FatekProgramSerialDriver().Descriptor.DriverId);

    private static ConnectionProfile Profile(object config) =>
        new("fatek", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest Point(string key, string address, string type) =>
        new(key, JsonSerializer.SerializeToElement(new { address }), type, 1, JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslFatekProgramClient client) : IHslFatekProgramSerialClientFactory
    {
        public IHslFatekProgramClient Create(HslFatekProgramSerialClientOptions options) => client;
    }

    private sealed class FakeClient : IHslFatekProgramClient
    {
        public List<HslFatekProgramReadRequest> Requests { get; } = [];
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

        public Task<HslReadResult> ReadAsync(HslFatekProgramReadRequest request, CancellationToken cancellationToken)
        {
            Requests.Add(request);
            return Task.FromResult(request.Address == "D0"
                ? HslReadResult.Success((short)123)
                : HslReadResult.Failure("HSL_READ_FAILED", "地址不存在", false));
        }

        public async ValueTask DisposeAsync()
        {
            if (IsConnected) await CloseAsync(CancellationToken.None);
        }
    }
}
