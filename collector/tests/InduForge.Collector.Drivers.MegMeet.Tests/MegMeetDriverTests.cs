using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
using Xunit;

namespace InduForge.Collector.Drivers.MegMeet.Tests;

public sealed class MegMeetDriverTests
{
    [Theory]
    [InlineData("tcp")]
    [InlineData("rtu-tcp")]
    public void UsesDemoTcpDefaults(string kindName)
    {
        var options = MegMeetConnectionOptions.Parse(Profile(new { host = "127.0.0.1" }), Kind(kindName));
        Assert.Equal(502, options.Port);
        Assert.Equal((byte)1, options.Station);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal(HslDataFormat.CDAB, options.ToHslOptions().DataFormat);
    }

    [Fact]
    public void UsesDemoSerialDefaults()
    {
        var options = MegMeetConnectionOptions.Parse(Profile(new { portName = "COM3" }), MegMeetDriverKind.RtuSerial);
        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(8, options.DataBits);
        Assert.Equal(HslSerialParity.None, options.Parity);
        Assert.Equal(HslSerialStopBits.One, options.StopBits);
    }

    [Theory]
    [InlineData("d100", "int16", "D100")]
    [InlineData("m100", "bool", "M100")]
    [InlineData("s=02;d100", "int16", "s=2;D100")]
    [InlineData("c200", "int32", "C200")]
    public void NormalizesDocumentedAddresses(string address, string type, string expected) =>
        Assert.Equal(expected, MegMeetAddress.Parse(Point(address, type)).ProtocolAddress);

    [Theory]
    [InlineData("D100", "bool", "MEGMEET_BOOL_AREA_INVALID")]
    [InlineData("X100", "int16", "MEGMEET_WORD_AREA_INVALID")]
    [InlineData("C200", "int16", "MEGMEET_COUNTER_TYPE_INVALID")]
    public void RejectsUnsupportedAddressAndTypeCombinations(string address, string type, string expectedCode)
    {
        var exception = Assert.Throws<MegMeetDriverException>(() => MegMeetAddress.Parse(Point(address, type)));
        Assert.Equal(expectedCode, exception.Code);
    }

    [Fact]
    public async Task SessionReusesClientAndIsolatesFailures()
    {
        var client = new FakeClient();
        var driver = new MegMeetTcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);
        var result = await ((IPointReaderSession)session).ReadAsync(
            new ReadRequest([Point("D100", "int16"), Point("D101", "int16"), Point("bad", "int16")]), CancellationToken.None);
        Assert.Equal(2, client.Requests.Count);
        Assert.True(result.Values[0].Succeeded);
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("MEGMEET_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    private static MegMeetDriverKind Kind(string name) => name == "tcp" ? MegMeetDriverKind.Tcp : MegMeetDriverKind.RtuOverTcp;
    private static ConnectionProfile Profile(object config) =>
        new("megmeet", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));
    private static PointReadRequest Point(string address, string type) =>
        new(address, JsonSerializer.SerializeToElement(new { address }), type, 1, JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslMegMeetClient client) : IHslMegMeetClientFactory
    {
        public IHslMegMeetClient Create(HslMegMeetClientOptions options) => client;
    }

    private sealed class FakeClient : IHslMegMeetClient
    {
        public List<HslMegMeetReadRequest> Requests { get; } = [];
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
        public Task<HslReadResult> ReadAsync(HslMegMeetReadRequest request, CancellationToken cancellationToken)
        {
            Requests.Add(request);
            return Task.FromResult(request.Address == "D100"
                ? HslReadResult.Success((short)1)
                : HslReadResult.Failure("READ_FAILED", "失败", false));
        }
        public async ValueTask DisposeAsync()
        {
            if (IsConnected) await CloseAsync(CancellationToken.None);
        }
    }
}
