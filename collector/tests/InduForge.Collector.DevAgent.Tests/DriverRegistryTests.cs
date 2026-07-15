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
    public void RegistryRejectsDuplicateDriverId()
    {
        Assert.Throws<InvalidOperationException>(() => new DriverRegistry([
            () => new BrowserAndReaderDriver(),
            () => new BrowserAndReaderDriver(),
        ]));
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
