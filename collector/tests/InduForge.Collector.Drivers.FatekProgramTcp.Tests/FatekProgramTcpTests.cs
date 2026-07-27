using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.FatekProgramTcp.Tests;

public sealed class FatekProgramTcpTests
{
    [Fact]
    public void UsesDemoDefaults()
    {
        var options = FatekProgramTcpConnectionOptions.Parse(Profile(new { host = "127.0.0.1" }));
        Assert.Equal(2000, options.Port); Assert.Equal((byte)1, options.Station);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds); Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Theory]
    [InlineData("M0", "bool")]
    [InlineData("X0", "bool")]
    [InlineData("Y0", "bool")]
    [InlineData("S0", "bool")]
    [InlineData("T0", "bool")]
    [InlineData("C0", "bool")]
    [InlineData("RT0", "int16")]
    [InlineData("RC0", "int16")]
    [InlineData("D0", "int16")]
    [InlineData("R0", "int16")]
    [InlineData("s=2;D100", "int16")]
    public void AcceptsDocumentedAddresses(string address, string dataType) => Assert.Equal(address, FatekProgramAddress.Parse(Point("p", address, dataType)).ProtocolAddress);

    [Fact]
    public void RejectsAreaTypeMismatch()
    {
        Assert.Equal("FATEK_PROGRAM_BOOL_AREA_INVALID", Assert.Throws<FatekProgramDriverException>(() => FatekProgramAddress.Parse(Point("p", "D0", "bool"))).Code);
        Assert.Equal("FATEK_PROGRAM_WORD_AREA_INVALID", Assert.Throws<FatekProgramDriverException>(() => FatekProgramAddress.Parse(Point("p", "M0", "int16"))).Code);
    }

    [Fact]
    public async Task SessionReusesClientAndIsolatesFailures()
    {
        var client = new FakeClient(); var driver = new FatekProgramTcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);
        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([Point("ok", "D0", "int16"), Point("failed", "D1", "int16"), Point("invalid", "M0", "int16")]), CancellationToken.None);
        Assert.Equal(2, client.Requests.Count); Assert.Equal((short)123, Assert.IsType<short>(result.Values[0].Value));
        Assert.False(result.Values[1].Succeeded); Assert.Equal("FATEK_PROGRAM_WORD_AREA_INVALID", result.Values[2].ErrorCode);
    }

    [Fact] public void ExposesDescriptor() => Assert.Equal("fatek.program-tcp", new FatekProgramTcpDriver().Descriptor.DriverId);
    private static ConnectionProfile Profile(object config) => new("fatek", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string key, string address, string type) => new(key, JsonSerializer.SerializeToElement(new { address }), type, 1, JsonSerializer.SerializeToElement(new { }));
    private sealed class FakeFactory(IHslFatekProgramClient client) : IHslFatekProgramTcpClientFactory { public IHslFatekProgramClient Create(HslFatekProgramTcpClientOptions options) => client; }
    private sealed class FakeClient : IHslFatekProgramClient
    {
        public List<HslFatekProgramReadRequest> Requests { get; } = []; public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslFatekProgramReadRequest request, CancellationToken cancellationToken) { Requests.Add(request); return Task.FromResult(request.Address == "D0" ? HslReadResult.Success((short)123) : HslReadResult.Failure("HSL_READ_FAILED", "地址不存在", false)); }
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
