using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InovanceModbusTcp.Tests;

public sealed class InovanceModbusTcpTests
{
    [Fact]
    public void UsesDemoDefaults()
    {
        var options = InovanceModbusTcpConnectionOptions.Parse(Profile(new { host = "192.168.1.10" }));
        Assert.Equal(502, options.Port);
        Assert.Equal((byte)1, options.Station);
        Assert.Equal(HslInovanceSeries.AM, options.Series);
        Assert.True(options.AddressStartWithZero);
        Assert.Equal(HslDataFormat.CDAB, options.DataFormat);
        Assert.False(options.StringReverse);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Fact]
    public void SerialUsesDemoDefaults()
    {
        var options = InovanceModbusVariantOptions.Parse(Profile(new { portName = "COM3" }), HslInovanceModbusVariantProtocol.Serial);

        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(8, options.DataBits);
        Assert.Equal(HslSerialParity.None, options.Parity);
        Assert.Equal(HslSerialStopBits.One, options.StopBits);
        Assert.Equal((byte)1, options.Station);
        Assert.Equal(HslInovanceSeries.AM, options.Series);
        Assert.Equal(HslDataFormat.CDAB, options.DataFormat);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Fact]
    public void SerialOverTcpUsesDemoDefaults()
    {
        var options = InovanceModbusVariantOptions.Parse(Profile(new { host = "192.168.1.10" }), HslInovanceModbusVariantProtocol.SerialOverTcp);

        Assert.Equal(502, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal((byte)1, options.Station);
    }

    [Theory]
    [InlineData(HslInovanceSpecialProtocol.ConnectedCip, 44818)]
    [InlineData(HslInovanceSpecialProtocol.EasyNet, 12939)]
    public void SpecialNetworkProtocolsUseDemoDefaults(HslInovanceSpecialProtocol protocol, int expectedPort)
    {
        var options = InovanceSpecialOptions.Parse(Profile(new { host = "192.168.1.10" }), protocol);

        Assert.Equal(expectedPort, options.Port);
        Assert.Equal(5000, options.ConnectTimeoutMilliseconds);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
        Assert.Equal(0u, options.ToConnectionId);
        Assert.Equal(0u, options.OtConnectionId);
    }

    [Fact]
    public void ComputerLinkUsesDemoDefaults()
    {
        var options = InovanceSpecialOptions.Parse(Profile(new { portName = "COM3" }), HslInovanceSpecialProtocol.ComputerLink);

        Assert.Equal(9600, options.BaudRate);
        Assert.Equal(7, options.DataBits);
        Assert.Equal(HslSerialParity.Even, options.Parity);
        Assert.Equal(HslSerialStopBits.Two, options.StopBits);
        Assert.Equal((byte)0, options.Station);
        Assert.Equal((byte)0, options.WaitingTime);
        Assert.True(options.SumCheck);
        Assert.Equal(4, options.Format);
        Assert.Equal(10000, options.ReceiveTimeoutMilliseconds);
    }

    [Theory]
    [InlineData(HslInovanceSpecialProtocol.ConnectedCip, "A1")]
    [InlineData(HslInovanceSpecialProtocol.EasyNet, "W100")]
    [InlineData(HslInovanceSpecialProtocol.ComputerLink, "D100")]
    public void SpecialProtocolsAcceptDemoAddresses(HslInovanceSpecialProtocol protocol, string address)
    {
        var result = InovanceSpecialAddress.Parse(Point("point", address, "int16", 1), protocol);

        Assert.Equal(address, result.Address);
    }

    [Theory]
    [InlineData("Q0")]
    [InlineData("I0")]
    [InlineData("M0")]
    [InlineData("SM0")]
    [InlineData("SD0")]
    [InlineData("D0")]
    [InlineData("R0")]
    [InlineData("X0")]
    [InlineData("Y0")]
    [InlineData("B0")]
    public void AcceptsNativeSeriesAddresses(string address)
    {
        Assert.Equal(address, InovanceModbusTcpAddress.Parse(Point("point", address.ToLowerInvariant(), "int16", 1)).ProtocolAddress);
    }

    [Fact]
    public void RejectsReadLengthBeyondModbusLimit()
    {
        var exception = Assert.Throws<InovanceModbusTcpDriverException>(() =>
            InovanceModbusTcpAddress.Parse(Point("point", "D0", "float64", 32)));
        Assert.Equal("INOVANCE_MODBUS_TCP_READ_LENGTH_INVALID", exception.Code);
    }

    [Fact]
    public async Task SessionReusesClientAndIsolatesFailures()
    {
        var client = new FakeClient();
        var driver = new InovanceModbusTcpDriver(new FakeFactory(client));
        await using var session = await driver.OpenSessionAsync(Profile(new { host = "127.0.0.1" }), CancellationToken.None);
        var result = await ((IPointReaderSession)session).ReadAsync(new ReadRequest([
            Point("valid", "D0", "int16", 1),
            Point("failed", "D1", "int16", 1),
            Point("invalid", "D 2", "int16", 1),
        ]), CancellationToken.None);

        Assert.Equal(2, client.Requests.Count);
        Assert.Equal((short)123, Assert.IsType<short>(result.Values[0].Value));
        Assert.False(result.Values[1].Succeeded);
        Assert.Equal("INOVANCE_MODBUS_TCP_ADDRESS_INVALID", result.Values[2].ErrorCode);
    }

    [Fact]
    public void ExposesDescriptor()
    {
        var descriptor = new InovanceModbusTcpDriver().Descriptor;
        Assert.Equal("inovance.modbus-tcp", descriptor.DriverId);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Theory]
    [InlineData("inovance.modbus-serial")]
    [InlineData("inovance.modbus-rtu-over-tcp")]
    public void VariantDriversExposeDescriptor(string driverId)
    {
        var descriptor = driverId == "inovance.modbus-serial"
            ? new InovanceModbusSerialDriver().Descriptor
            : new InovanceModbusRtuOverTcpDriver().Descriptor;

        Assert.Equal(driverId, descriptor.DriverId);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Theory]
    [InlineData("inovance.connected-cip")]
    [InlineData("inovance.easy-net")]
    [InlineData("inovance.computer-link")]
    public void SpecialDriversExposeDescriptor(string driverId)
    {
        var descriptor = driverId switch
        {
            "inovance.connected-cip" => new InovanceConnectedCipDriver().Descriptor,
            "inovance.easy-net" => new InovanceEasyNetDriver().Descriptor,
            _ => new InovanceComputerLinkDriver().Descriptor,
        };

        Assert.Equal(driverId, descriptor.DriverId);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    private static ConnectionProfile Profile(object config) =>
        new("inovance", JsonSerializer.SerializeToElement(config), JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest Point(string key, string address, string dataType, int elementCount) =>
        new(key, JsonSerializer.SerializeToElement(new { address }), dataType, elementCount, JsonSerializer.SerializeToElement(new { }));

    private sealed class FakeFactory(IHslInovanceModbusTcpClient client) : IHslInovanceModbusTcpClientFactory
    {
        public IHslInovanceModbusTcpClient Create(HslInovanceModbusTcpClientOptions options) => client;
    }

    private sealed class FakeClient : IHslInovanceModbusTcpClient
    {
        public List<HslInovanceModbusTcpReadRequest> Requests { get; } = [];
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
        public Task<HslReadResult> ReadAsync(HslInovanceModbusTcpReadRequest request, CancellationToken cancellationToken)
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
