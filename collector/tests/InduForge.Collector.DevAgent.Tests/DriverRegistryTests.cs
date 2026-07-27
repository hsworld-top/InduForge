using InduForge.Collector.Contracts;
using InduForge.Collector.DevAgent;

namespace InduForge.Collector.DevAgent.Tests;

public sealed class DriverRegistryTests
{
    [Fact]
    public void RegistryBuildsOperationsFromImplementedInterfaces()
    {
        var registry = new DriverRegistry([() => new BrowserAndReaderDriver()]);

        var descriptor = registry.Describe("test.driver");

        Assert.Equal(["connection.test", "device.browse", "point.read"], descriptor.Operations);
    }

    [Fact]
    public void RegistryAddsConnectionSessionOperationsFromSessionDriver()
    {
        var registry = new DriverRegistry([() => new SessionDriver()]);

        var descriptor = registry.Describe("session.driver");

        Assert.Equal(
            ["connection.test", "connection.open", "connection.close"],
            descriptor.Operations);
    }
    [Fact]
    public void RegistryRejectsDuplicateDriverId()
    {
        Assert.Throws<InvalidOperationException>(() => new DriverRegistry([
            () => new BrowserAndReaderDriver(),
            () => new BrowserAndReaderDriver(),
        ]));
    }

    [Fact]
    public void DefaultOpcUaDriverUsesStructuredConnectionSchemaVersion()
    {
        var descriptor = DriverRegistry.CreateDefault().Describe("opcua.standard");
        Assert.Equal([1], descriptor.SchemaVersions);
    }

    [Fact]
    public void DefaultRegistryLoadsAllenBradleyEtherNetIpManifest()
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe("allen-bradley.ethernet-ip");

        Assert.Equal("allen-bradley", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.True(registry.UsesTransport("allen-bradley.ethernet-ip", "tcp"));
    }

    [Theory]
    [InlineData("allen-bradley.connected-cip", "tcp")]
    [InlineData("allen-bradley.micro-cip", "tcp")]
    [InlineData("allen-bradley.pccc", "tcp")]
    [InlineData("allen-bradley.slc", "tcp")]
    [InlineData("allen-bradley.df1-serial", "serial")]
    public void DefaultRegistryLoadsAllenBradleyVariantManifests(string driverId, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);

        Assert.Equal("allen-bradley", descriptor.ProtocolFamily);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    [Fact]
    public void DefaultRegistryLoadsModbusTcpManifest()
    {
        var descriptor = DriverRegistry.CreateDefault().Describe("modbus.tcp");

        Assert.Equal("modbus", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Fact]
    public void DefaultRegistryLoadsModbusRtuManifest()
    {
        var descriptor = DriverRegistry.CreateDefault().Describe("modbus.rtu");

        Assert.Equal("modbus", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Theory]
    [InlineData("modbus.rtu-over-tcp", "tcp")]
    [InlineData("modbus.ascii-over-tcp", "tcp")]
    [InlineData("modbus.ascii", "serial")]
    [InlineData("modbus.udp", "udp")]
    public void DefaultRegistryLoadsModbusVariantManifests(string driverId, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);

        Assert.Equal("modbus", descriptor.ProtocolFamily);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    [Fact]
    public void DefaultRegistryLoadsMitsubishiMc3ETcpManifest()
    {
        var descriptor = DriverRegistry.CreateDefault().Describe("mitsubishi.mc-3e-tcp");

        Assert.Equal("mitsubishi", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Theory]
    [InlineData("mitsubishi.a1e-ascii-tcp", "tcp")]
    [InlineData("mitsubishi.a1e-binary-tcp", "tcp")]
    [InlineData("mitsubishi.mc-ascii-tcp", "tcp")]
    [InlineData("mitsubishi.mc-ascii-udp", "udp")]
    [InlineData("mitsubishi.mc-binary-udp", "udp")]
    [InlineData("mitsubishi.mc-r-binary-tcp", "tcp")]
    [InlineData("mitsubishi.cip", "tcp")]
    public void DefaultRegistryLoadsMitsubishiNetworkVariantManifests(string driverId, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);

        Assert.Equal("melsec", descriptor.ProtocolFamily);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    [Theory]
    [InlineData("mitsubishi.a3c-serial", "serial")]
    [InlineData("mitsubishi.a3c-serial-over-tcp", "tcp")]
    [InlineData("mitsubishi.fx-links-serial", "serial")]
    [InlineData("mitsubishi.fx-links-over-tcp", "tcp")]
    [InlineData("mitsubishi.fx-serial", "serial")]
    [InlineData("mitsubishi.fx-serial-over-tcp", "tcp")]
    public void DefaultRegistryLoadsMitsubishiSerialVariantManifests(string driverId, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);

        Assert.Equal("melsec", descriptor.ProtocolFamily);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    [Fact]
    public void DefaultRegistryLoadsBeckhoffAdsManifest()
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe("beckhoff.ads-tcp");

        Assert.Equal("beckhoff", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
        Assert.True(registry.UsesTransport("beckhoff.ads-tcp", "tcp"));
    }

    [Fact]
    public void DefaultRegistryLoadsIec104Manifest()
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe("iec.60870-5-104");

        Assert.Equal("iec", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
        Assert.True(registry.UsesTransport("iec.60870-5-104", "tcp"));
    }

    [Fact]
    public void DefaultRegistryLoadsPanasonicMewtocolTcpManifest()
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe("panasonic.mewtocol-tcp");
        Assert.Equal("panasonic", descriptor.ProtocolFamily);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
        Assert.True(registry.UsesTransport("panasonic.mewtocol-tcp", "tcp"));
    }

    [Fact]
    public void DefaultRegistryLoadsLsisFastEnetManifest()
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe("lsis.fast-enet");

        Assert.Equal("lsis", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
        Assert.True(registry.UsesTransport("lsis.fast-enet", "tcp"));
    }

    [Theory]
    [InlineData("lsis.cnet", "serial")]
    [InlineData("lsis.cnet-over-tcp", "tcp")]
    [InlineData("lsis.cpu-serial", "serial")]
    public void DefaultRegistryLoadsLsisSerialVariantManifests(string driverId, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);

        Assert.Equal("lsis", descriptor.ProtocolFamily);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    [Fact]
    public void DefaultRegistryLoadsGeSrtpTcpManifest()
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe("ge.srtp-tcp");

        Assert.Equal("ge", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
        Assert.True(registry.UsesTransport("ge.srtp-tcp", "tcp"));
    }

    [Theory]
    [InlineData("inovance.modbus-tcp", "tcp")]
    [InlineData("inovance.modbus-serial", "serial")]
    [InlineData("inovance.modbus-rtu-over-tcp", "tcp")]
    public void DefaultRegistryLoadsInovanceModbusManifests(string driverId, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);

        Assert.Equal("inovance", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    [Theory]
    [InlineData("inovance.connected-cip", "tcp")]
    [InlineData("inovance.easy-net", "tcp")]
    [InlineData("inovance.computer-link", "serial")]
    public void DefaultRegistryLoadsInovanceSpecialManifests(string driverId, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);

        Assert.Equal("inovance", descriptor.ProtocolFamily);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    [Fact]
    public void DefaultRegistryLoadsFatekProgramTcpManifest()
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe("fatek.program-tcp");
        Assert.Equal("fatek", descriptor.ProtocolFamily);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
        Assert.True(registry.UsesTransport("fatek.program-tcp", "tcp"));
    }

    [Theory]
    [InlineData("panasonic.mewtocol-tcp", "tcp")]
    [InlineData("panasonic.mewtocol-serial", "serial")]
    [InlineData("panasonic.mc-binary-tcp", "tcp")]
    public void DefaultRegistryLoadsPanasonicManifests(string driverId, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);
        Assert.Equal("panasonic", descriptor.ProtocolFamily);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    [Theory]
    [InlineData("keyence.mc-ascii-tcp", "tcp")]
    [InlineData("keyence.nano-serial", "serial")]
    [InlineData("keyence.nano-serial-over-tcp", "tcp")]
    public void DefaultRegistryLoadsKeyenceVariantManifests(string driverId, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);

        Assert.Equal("keyence", descriptor.ProtocolFamily);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    [Theory]
    [InlineData("omron.fins-tcp", "tcp")]
    [InlineData("omron.fins-udp", "udp")]
    [InlineData("omron.cip", "tcp")]
    [InlineData("omron.connected-cip", "tcp")]
    [InlineData("omron.hostlink", "serial")]
    [InlineData("omron.hostlink-over-tcp", "tcp")]
    [InlineData("omron.hostlink-cmode", "serial")]
    [InlineData("omron.hostlink-cmode-over-tcp", "tcp")]
    public void DefaultRegistryLoadsOmronFinsManifests(string driverId, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);

        Assert.Equal("omron", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    [Fact]
    public void DefaultRegistryLoadsSiemensS7TcpManifest()
    {
        var descriptor = DriverRegistry.CreateDefault().Describe("siemens.s7-tcp");

        Assert.Equal("siemens", descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
    }

    [Theory]
    [InlineData("cjt188.serial", "cjt188", "serial")]
    [InlineData("cjt188.tcp", "cjt188", "tcp")]
    [InlineData("rkc.temperature-controller-serial", "rkc", "serial")]
    [InlineData("rkc.temperature-controller-tcp", "rkc", "tcp")]
    [InlineData("robot.estun-tcp", "estun", "tcp")]
    [InlineData("robot.fanuc-interface", "fanuc", "tcp")]
    [InlineData("dam3601.serial", "dam3601", "serial")]
    [InlineData("dcs.nanjing-auto", "dcs", "tcp")]
    [InlineData("ec-fan.machine-serial", "ec-fan", "serial")]
    [InlineData("mqtt.rpc-device", "mqtt-rpc", "mqtt")]
    [InlineData("siemens.s7-plus", "siemens", "tcp")]
    [InlineData("yudian.ai-bus", "yudian", "serial")]
    [InlineData("delixi.dtsu6606", "delixi", "serial")]
    [InlineData("dlt645.2007-serial", "dlt645", "serial")]
    [InlineData("dlt645.2007-over-tcp", "dlt645", "tcp")]
    [InlineData("dlt645.1997-serial", "dlt645", "serial")]
    [InlineData("dlt645.1997-over-tcp", "dlt645", "tcp")]
    [InlineData("dlt698.serial", "dlt698", "serial")]
    [InlineData("dlt698.over-tcp", "dlt698", "tcp")]
    [InlineData("dlt698.tcp-net", "dlt698", "tcp")]
    public void DefaultRegistryLoadsInstrumentMeterManifests(string driverId, string protocolFamily, string transport)
    {
        var registry = DriverRegistry.CreateDefault();
        var descriptor = registry.Describe(driverId);

        Assert.Equal(protocolFamily, descriptor.ProtocolFamily);
        Assert.Equal("1.0.0", descriptor.DriverVersion);
        Assert.Equal([1], descriptor.SchemaVersions);
        Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
        Assert.Contains(DriverOperations.ConnectionClose, descriptor.Operations);
        Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
        Assert.True(registry.UsesTransport(driverId, transport));
    }

    private sealed class SessionDriver : IIndustrialDriver, IConnectionSessionDriver
    {
        public DriverDescriptor Descriptor { get; } = new(
            "test",
            "session.driver",
            "1.0.0",
            [1],
            [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose]);

        public Task<ConnectionTestResult> TestConnectionAsync(
            ConnectionProfile profile,
            CancellationToken cancellationToken) =>
            Task.FromResult(new ConnectionTestResult(true, TimeSpan.Zero, "test", []));

        public Task<IIndustrialConnectionSession> OpenSessionAsync(
            ConnectionProfile profile,
            CancellationToken cancellationToken) =>
            Task.FromResult<IIndustrialConnectionSession>(new EmptySession());
    }

    private sealed class EmptySession : IIndustrialConnectionSession
    {
        public bool IsConnected => true;

        public string? ServerName => "test";

        public ValueTask DisposeAsync() => ValueTask.CompletedTask;
    }
    private sealed class BrowserAndReaderDriver : IIndustrialDriver, IDeviceBrowser, IPointReader
    {
        public DriverDescriptor Descriptor { get; } = new(
            "test", "test.driver", "1.0.0", [1],
            [DriverOperations.ConnectionTest, DriverOperations.DeviceBrowse, DriverOperations.PointRead]);

        public Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) =>
            Task.FromResult(new ConnectionTestResult(true, TimeSpan.Zero, "test", []));

        public Task<BrowseResult> BrowseAsync(ConnectionProfile profile, BrowseRequest request, CancellationToken cancellationToken) =>
            Task.FromResult(new BrowseResult([], []));

        public Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) =>
            Task.FromResult(new ReadResult([], []));
    }
}
