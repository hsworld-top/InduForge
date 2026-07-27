using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InstrumentMeters.Tests;

public sealed class InstrumentMeterDriverTests
{
    [Fact]
    public void TemperatureMetersUseDemoDefaults()
    {
        var dam = InstrumentSerialOptions.Parse(Profile("dam3601", new { portName = "COM3" }), InstrumentSerialKind.Dam3601);
        var yudian = InstrumentSerialOptions.Parse(Profile("yudian", new { portName = "COM4" }), InstrumentSerialKind.YuDian);
        var dtsu = InstrumentSerialOptions.Parse(Profile("delixi", new { portName = "COM5" }), InstrumentSerialKind.Dtsu6606);
        Assert.Equal(9600, dam.BaudRate); Assert.Equal(HslDataFormat.CDAB, dam.DataFormat);
        Assert.Equal(9600, yudian.BaudRate); Assert.Equal((byte)1, yudian.Station);
        Assert.Equal(2400, dtsu.BaudRate); Assert.Equal(HslSerialParity.Even, dtsu.Parity); Assert.Equal(HslDataFormat.ABCD, dtsu.DataFormat);
    }

    [Theory]
    [InlineData("dam3601", "temperature[001]", "float32", "temperature[1]")]
    [InlineData("yudian", "S=02;010", "int16", "s=2;10")]
    [InlineData("delixi", "x=3;0768", "float32", "x=3;0768")]
    public void TemperatureMetersNormalizeAddresses(string family, string address, string dataType, string expected)
    {
        var kind = family switch { "dam3601" => InstrumentSerialKind.Dam3601, "yudian" => InstrumentSerialKind.YuDian, _ => InstrumentSerialKind.Dtsu6606 };
        Assert.Equal(expected, InstrumentSerialAddress.Parse(Point(address, dataType), kind).Address);
    }

    [Fact]
    public void DltUsesDemoDefaults()
    {
        var serial2007 = DltOptions.Parse(Profile("dlt645", new { portName = "COM3" }), HslDltProtocol.Dlt645Serial);
        var serial1997 = DltOptions.Parse(Profile("dlt645", new { portName = "COM4" }), HslDltProtocol.Dlt645With1997Serial);
        var tcp698 = DltOptions.Parse(Profile("dlt698", new { host = "127.0.0.1" }), HslDltProtocol.Dlt698OverTcp);
        Assert.Equal(9600, serial2007.BaudRate); Assert.Equal(HslSerialParity.Even, serial2007.Parity); Assert.True(serial2007.CheckDataId);
        Assert.Equal(HslSerialParity.None, serial1997.Parity); Assert.Equal(2000, tcp698.Port); Assert.True(tcp698.UseSecurityRequest);
    }

    [Theory]
    [InlineData("00-00-00-00", HslDltProtocol.Dlt645Serial, "000000000000;00-00-00-00")]
    [InlineData("S=12;b6-11", HslDltProtocol.Dlt645With1997OverTcp, "s=000000000012;B6-11")]
    [InlineData("20-00-02-00", HslDltProtocol.Dlt698TcpNet, "20-00-02-00")]
    public void DltNormalizesAddresses(string address, HslDltProtocol protocol, string expected)
    {
        var actual = DltAddress.Parse(Point(address, protocol is HslDltProtocol.Dlt698TcpNet ? "float32" : "float64"), protocol).Address;
        if (!address.StartsWith("S=", StringComparison.OrdinalIgnoreCase)) expected = expected.Replace("000000000000;", string.Empty, StringComparison.Ordinal);
        Assert.Equal(expected, actual);
    }

    [Fact]
    public void Cjt188UsesDemoDefaults()
    {
        var serial = Cjt188Options.Parse(CjtProfile(new { portName = "COM3" }), HslInstrumentTransport.Serial);
        var tcp = Cjt188Options.Parse(CjtProfile(new { host = "127.0.0.1" }), HslInstrumentTransport.Tcp);

        Assert.Equal(9600, serial.BaudRate);
        Assert.Equal("78330015040963", serial.Station);
        Assert.Equal(0x19, serial.InstrumentType);
        Assert.False(serial.EnableCodeFE);
        Assert.Equal(502, tcp.Port);
    }

    [Theory]
    [InlineData("901f", "90-1F")]
    [InlineData("S=123;d1-2a", "s=00000000000123;D1-2A")]
    public void Cjt188NormalizesAddress(string value, string expected) =>
        Assert.Equal(expected, Cjt188Address.Parse(Point(value, "float64")).Address);

    [Fact]
    public async Task Cjt188ReusesClientAndIsolatesFailures()
    {
        var client = new FakeCjtClient();
        var driver = new Cjt188TcpDriver(new CjtFactory(client));
        await using var session = await driver.OpenSessionAsync(CjtProfile(new { host = "127.0.0.1" }), CancellationToken.None);

        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([Point("90-1F", "float64"), Point("bad", "float64")]), CancellationToken.None);

        Assert.Single(client.Requests);
        Assert.True(result.Values[0].Succeeded);
        Assert.Equal("CJT188_ADDRESS_INVALID", result.Values[1].ErrorCode);
    }

    [Fact]
    public void RkcUsesDemoDefaults()
    {
        var serial = RkcOptions.Parse(RkcProfile(new { portName = "COM3" }), HslInstrumentTransport.Serial);
        var tcp = RkcOptions.Parse(RkcProfile(new { host = "127.0.0.1" }), HslInstrumentTransport.Tcp);

        Assert.Equal(9600, serial.BaudRate);
        Assert.Equal((byte)1, serial.Station);
        Assert.Equal(2000, tcp.Port);
    }

    [Theory]
    [InlineData("m1", "M1")]
    [InlineData("S=02;pb", "s=2;PB")]
    public void RkcNormalizesAddress(string value, string expected) =>
        Assert.Equal(expected, RkcAddress.Parse(Point(value, "float64")).Address);

    [Fact]
    public void EcFanUsesDemoDefaultsAndFixedMetrics()
    {
        var options = EcFanOptions.Parse(Profile("ec-fan", new { portName = "COM3" }));
        Assert.Equal(9600, options.BaudRate);
        Assert.Equal((byte)1, options.Station);
        Assert.Equal(HslEcFanMetric.SpeedMaximum, EcFanAddress.Parse(Point("speedMaximum", "int32")));
    }

    [Fact]
    public async Task RkcReusesClientAndIsolatesFailures()
    {
        var client = new FakeRkcClient();
        var driver = new RkcTemperatureControllerTcpDriver(new RkcFactory(client));
        await using var session = await driver.OpenSessionAsync(RkcProfile(new { host = "127.0.0.1" }), CancellationToken.None);

        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([Point("M1", "float64"), Point("bad", "float64")]), CancellationToken.None);

        Assert.Single(client.Requests);
        Assert.True(result.Values[0].Succeeded);
        Assert.Equal("RKC_ADDRESS_INVALID", result.Values[1].ErrorCode);
    }

    private static ConnectionProfile CjtProfile(object config) => new("cjt188", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static ConnectionProfile RkcProfile(object config) => new("rkc", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static ConnectionProfile Profile(string family, object config) => new(family, JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string address, string dataType) => new(address, JsonSerializer.SerializeToElement(new { address }), dataType, 1, JsonSerializer.SerializeToElement(new { }));

    private sealed class CjtFactory(IHslCjt188Client client) : IHslCjt188ClientFactory
    {
        public IHslCjt188Client Create(HslCjt188ClientOptions options) => client;
    }

    private sealed class FakeCjtClient : IHslCjt188Client
    {
        public List<HslCjt188ReadRequest> Requests { get; } = [];
        public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslCjt188ReadRequest request, CancellationToken cancellationToken) { Requests.Add(request); return Task.FromResult(HslReadResult.Success(1d)); }
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }

    private sealed class RkcFactory(IHslRkcClient client) : IHslRkcClientFactory
    {
        public IHslRkcClient Create(HslRkcClientOptions options) => client;
    }

    private sealed class FakeRkcClient : IHslRkcClient
    {
        public List<HslRkcReadRequest> Requests { get; } = [];
        public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslRkcReadRequest request, CancellationToken cancellationToken) { Requests.Add(request); return Task.FromResult(HslReadResult.Success(1d)); }
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
