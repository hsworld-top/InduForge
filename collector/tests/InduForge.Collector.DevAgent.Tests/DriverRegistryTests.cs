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
