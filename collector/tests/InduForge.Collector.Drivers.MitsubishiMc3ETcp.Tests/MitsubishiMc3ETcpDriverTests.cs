using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MitsubishiMc3ETcp.Tests;

public sealed class MitsubishiMc3ETcpDriverTests
{
    [Theory]
    [InlineData("mitsubishi.a1e-ascii-tcp")]
    [InlineData("mitsubishi.a1e-binary-tcp")]
    [InlineData("mitsubishi.mc-ascii-tcp")]
    [InlineData("mitsubishi.mc-ascii-udp")]
    [InlineData("mitsubishi.mc-binary-udp")]
    [InlineData("mitsubishi.mc-r-binary-tcp")]
    [InlineData("mitsubishi.cip")]
    public void NetworkVariantDriversExposeDescriptor(string driverId)
    {
        var descriptor = CreateNetworkVariantDriver(driverId).Descriptor;

        Assert.Equal("melsec", descriptor.ProtocolFamily);
        Assert.Equal(driverId, descriptor.DriverId);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Theory]
    [InlineData(HslMelsecNetworkProtocol.A1EAsciiTcp, 6000, false)]
    [InlineData(HslMelsecNetworkProtocol.A1EBinaryTcp, 6000, false)]
    [InlineData(HslMelsecNetworkProtocol.McAsciiTcp, 6000, false)]
    [InlineData(HslMelsecNetworkProtocol.McAsciiUdp, 6000, false)]
    [InlineData(HslMelsecNetworkProtocol.McBinaryUdp, 6000, true)]
    [InlineData(HslMelsecNetworkProtocol.McRBinaryTcp, 6000, false)]
    [InlineData(HslMelsecNetworkProtocol.CipTcp, 44818, false)]
    public void NetworkVariantsUseDemoDefaults(
        HslMelsecNetworkProtocol protocol,
        int expectedPort,
        bool expectedWriteBitToWordRegister)
    {
        var options = MitsubishiNetworkVariantOptions.Parse(CreateNetworkProfile(), protocol);

        Assert.Equal(expectedPort, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal((byte)0, options.NetworkNumber);
        Assert.Equal((byte)0, options.NetworkStationNumber);
        Assert.Equal((ushort)0x03FF, options.TargetIoStation);
        Assert.Equal(expectedWriteBitToWordRegister, options.EnableWriteBitToWordRegister);
        Assert.False(options.StringReverse);
    }

    [Theory]
    [InlineData(HslMelsecNetworkProtocol.A1EAsciiTcp)]
    [InlineData(HslMelsecNetworkProtocol.A1EBinaryTcp)]
    [InlineData(HslMelsecNetworkProtocol.McAsciiTcp)]
    [InlineData(HslMelsecNetworkProtocol.McAsciiUdp)]
    [InlineData(HslMelsecNetworkProtocol.McBinaryUdp)]
    [InlineData(HslMelsecNetworkProtocol.McRBinaryTcp)]
    public void NativeNetworkVariantsAcceptDemoAddress(HslMelsecNetworkProtocol protocol)
    {
        var address = MitsubishiNetworkVariantAddress.Parse(CreatePoint("point", "D100", "int16"), protocol);

        Assert.Equal("D100", address.Address);
    }

    [Fact]
    public void MitsubishiCipPreservesTagCase()
    {
        var address = MitsubishiNetworkVariantAddress.Parse(CreatePoint("point", "PumpA.Speed", "float32"), HslMelsecNetworkProtocol.CipTcp);

        Assert.Equal("PumpA.Speed", address.Address);
    }

    [Fact]
    public void MitsubishiCipRejectsDateTimeArrays()
    {
        var point = new PointReadRequest(
            "point",
            JsonSerializer.SerializeToElement(new { address = "ClockTag" }),
            "datetime",
            2,
            JsonSerializer.SerializeToElement(new { }));

        var exception = Assert.Throws<MitsubishiMc3ETcpDriverException>(() =>
            MitsubishiNetworkVariantAddress.Parse(point, HslMelsecNetworkProtocol.CipTcp));

        Assert.Equal("MELSEC_CIP_ELEMENT_COUNT_INVALID", exception.Code);
    }

    [Theory]
    [InlineData("mitsubishi.a3c-serial")]
    [InlineData("mitsubishi.a3c-serial-over-tcp")]
    [InlineData("mitsubishi.fx-links-serial")]
    [InlineData("mitsubishi.fx-links-over-tcp")]
    [InlineData("mitsubishi.fx-serial")]
    [InlineData("mitsubishi.fx-serial-over-tcp")]
    public void SerialVariantDriversExposeDescriptor(string driverId)
    {
        var descriptor = CreateSerialVariantDriver(driverId).Descriptor;

        Assert.Equal("melsec", descriptor.ProtocolFamily);
        Assert.Equal(driverId, descriptor.DriverId);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Fact]
    public void A3CSerialUsesDemoDefaults()
    {
        var options = MitsubishiSerialVariantOptions.Parse(CreateSerialProfile(), HslMelsecSerialProtocol.A3CSerial);

        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(8, options.DataBits);
        Assert.Equal(HslSerialParity.None, options.Parity);
        Assert.Equal(HslSerialStopBits.One, options.StopBits);
        Assert.Equal((byte)0, options.Station);
        Assert.True(options.SumCheck);
        Assert.Equal(1, options.Format);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Fact]
    public void FxLinksSerialUsesDemoDefaults()
    {
        var options = MitsubishiSerialVariantOptions.Parse(CreateSerialProfile(), HslMelsecSerialProtocol.FxLinksSerial);

        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(7, options.DataBits);
        Assert.Equal(HslSerialParity.Even, options.Parity);
        Assert.Equal(HslSerialStopBits.Two, options.StopBits);
        Assert.Equal((byte)0, options.Station);
        Assert.Equal((byte)0, options.WaitingTime);
        Assert.True(options.SumCheck);
        Assert.Equal(1, options.Format);
    }

    [Fact]
    public void FxSerialUsesDemoDefaults()
    {
        var options = MitsubishiSerialVariantOptions.Parse(CreateSerialProfile(), HslMelsecSerialProtocol.FxSerial);

        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(7, options.DataBits);
        Assert.Equal(HslSerialParity.Even, options.Parity);
        Assert.Equal(HslSerialStopBits.One, options.StopBits);
        Assert.True(options.IsNewVersion);
    }

    [Theory]
    [InlineData(HslMelsecSerialProtocol.A3CSerialOverTcp, 6000)]
    [InlineData(HslMelsecSerialProtocol.FxLinksOverTcp, 2000)]
    [InlineData(HslMelsecSerialProtocol.FxSerialOverTcp, 5014)]
    public void SerialOverTcpVariantsUseDemoDefaults(HslMelsecSerialProtocol protocol, int expectedPort)
    {
        var options = MitsubishiSerialVariantOptions.Parse(CreateSerialOverTcpProfile(), protocol);

        Assert.Equal(expectedPort, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.True(options.IsNewVersion);
        Assert.False(options.UseGot);
    }

    [Fact]
    public void SerialVariantsAcceptDemoAddress()
    {
        var address = MitsubishiSerialVariantAddress.Parse(CreatePoint("point", "D100", "int16"));

        Assert.Equal("D100", address.Address);
    }

    [Fact]
    public async Task SessionReusesConnectionAndIsolatesPointFailures()
    {
        var client = new FakeClient
        {
            ReadHandler = request => request.Address == "D100"
                ? HslReadResult.Success((short)25)
                : HslReadResult.Failure("HSL_MELSEC_MC_READ_FAILED", "地址不存在", retryable: false),
        };
        var driver = new MitsubishiMc3ETcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);

        var result = await reader.ReadAsync(new ReadRequest([
            CreatePoint("valid", "D100", "int16"),
            CreatePoint("failed", "D101", "int16"),
            new PointReadRequest("invalid", JsonSerializer.SerializeToElement(new { address = "" }), "int16", 1, JsonSerializer.SerializeToElement(new { })),
        ]), CancellationToken.None);

        Assert.Equal(2, client.ReadRequests.Count);
        Assert.Equal((short)25, Assert.IsType<short>(result.Values[0].Value));
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("MELSEC_MC_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    [Fact]
    public async Task DisposeClosesConnectedClient()
    {
        var client = new FakeClient();
        var driver = new MitsubishiMc3ETcpDriver(new FakeFactory(client));
        await using (var session = await driver.OpenSessionAsync(CreateProfile(), CancellationToken.None))
        {
            Assert.True(session.IsConnected);
        }
        Assert.Equal(1, client.CloseCount);
    }

    private static ConnectionProfile CreateProfile() => new(
        "mitsubishi",
        JsonSerializer.SerializeToElement(new { host = "127.0.0.1", port = 6000 }),
        JsonSerializer.SerializeToElement(new { }));

    private static ConnectionProfile CreateNetworkProfile() => new(
        "melsec",
        JsonSerializer.SerializeToElement(new { host = "127.0.0.1" }),
        JsonSerializer.SerializeToElement(new { }));

    private static ConnectionProfile CreateSerialProfile() => new(
        "melsec",
        JsonSerializer.SerializeToElement(new { portName = "COM3" }),
        JsonSerializer.SerializeToElement(new { }));

    private static ConnectionProfile CreateSerialOverTcpProfile() => new(
        "melsec",
        JsonSerializer.SerializeToElement(new { host = "127.0.0.1" }),
        JsonSerializer.SerializeToElement(new { }));

    private static IIndustrialDriver CreateNetworkVariantDriver(string driverId) => driverId switch
    {
        "mitsubishi.a1e-ascii-tcp" => new MitsubishiA1EAsciiTcpDriver(),
        "mitsubishi.a1e-binary-tcp" => new MitsubishiA1EBinaryTcpDriver(),
        "mitsubishi.mc-ascii-tcp" => new MitsubishiMcAsciiTcpDriver(),
        "mitsubishi.mc-ascii-udp" => new MitsubishiMcAsciiUdpDriver(),
        "mitsubishi.mc-binary-udp" => new MitsubishiMcBinaryUdpDriver(),
        "mitsubishi.mc-r-binary-tcp" => new MitsubishiMcRBinaryTcpDriver(),
        "mitsubishi.cip" => new MitsubishiCipDriver(),
        _ => throw new ArgumentOutOfRangeException(nameof(driverId), driverId, null),
    };

    private static IIndustrialDriver CreateSerialVariantDriver(string driverId) => driverId switch
    {
        "mitsubishi.a3c-serial" => new MitsubishiA3CSerialDriver(),
        "mitsubishi.a3c-serial-over-tcp" => new MitsubishiA3CSerialOverTcpDriver(),
        "mitsubishi.fx-links-serial" => new MitsubishiFxLinksSerialDriver(),
        "mitsubishi.fx-links-over-tcp" => new MitsubishiFxLinksOverTcpDriver(),
        "mitsubishi.fx-serial" => new MitsubishiFxSerialDriver(),
        "mitsubishi.fx-serial-over-tcp" => new MitsubishiFxSerialOverTcpDriver(),
        _ => throw new ArgumentOutOfRangeException(nameof(driverId), driverId, null),
    };

    private static PointReadRequest CreatePoint(string key, string address, string dataType) => new(
        key,
        JsonSerializer.SerializeToElement(new { address }),
        dataType,
        1,
        JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslMelsecMcClient client) : IHslMelsecMcClientFactory
    {
        public IHslMelsecMcClient Create(HslMelsecMcClientOptions options) => client;
    }

    private sealed class FakeClient : IHslMelsecMcClient
    {
        public Func<HslMelsecMcReadRequest, HslReadResult> ReadHandler { get; init; } = _ => HslReadResult.Success((short)1);
        public List<HslMelsecMcReadRequest> ReadRequests { get; } = [];
        public int CloseCount { get; private set; }
        public bool IsConnected { get; private set; }

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

        public Task<HslReadResult> ReadAsync(HslMelsecMcReadRequest request, CancellationToken cancellationToken)
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
