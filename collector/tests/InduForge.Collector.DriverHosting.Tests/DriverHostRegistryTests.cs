using InduForge.Collector.Contracts;
using InduForge.Collector.DriverHosting;

namespace InduForge.Collector.DriverHosting.Tests;

public sealed class DriverHostRegistryTests
{
    [Fact]
    public void RejectsDuplicateFactory()
    {
        Assert.Throws<DriverRegistrationException>(() => new DriverHostRegistry([() => new TestDriver(), () => new TestDriver()]));
    }

    [Fact]
    public void RejectsDescriptorOperationMismatch()
    {
        Assert.Throws<DriverRegistrationException>(() => new DriverHostRegistry([() => new TestDriver([DriverOperations.ConnectionTest]) ]));
    }

    [Theory]
    [InlineData("other", "test.driver", "1.0.0", 1)]
    [InlineData("test", "other.driver", "1.0.0", 1)]
    [InlineData("test", "test.driver", "2.0.0", 1)]
    [InlineData("test", "test.driver", "1.0.0", 2)]
    public void RejectsDescriptorAndManifestMismatch(string protocolFamily, string driverId, string version, int schemaVersion)
    {
        var manifests = new Dictionary<string, DriverManifest>
        {
            ["test.driver"] = Manifest(protocolFamily, driverId, version, schemaVersion),
        };

        Assert.Throws<DriverRegistrationException>(() => new DriverHostRegistry([() => new TestDriver()], manifests));
    }

    [Fact]
    public void RejectsManifestOperationMismatch()
    {
        var manifests = new Dictionary<string, DriverManifest>
        {
            ["test.driver"] = Manifest(operations: [DriverOperations.ConnectionTest]),
        };

        Assert.Throws<DriverRegistrationException>(() => new DriverHostRegistry([() => new TestDriver()], manifests));
    }

    [Fact]
    public void RejectsUnsupportedPlatform()
    {
        var manifests = new Dictionary<string, DriverManifest>
        {
            ["test.driver"] = Manifest(platforms: new Dictionary<string, IReadOnlyList<string>>
            {
                ["runtime"] = ["linux-x64"],
            }),
        };

        Assert.Throws<DriverRegistrationException>(() => new DriverHostRegistry(
            [() => new TestDriver()],
            manifests,
            new DriverHostPlatform("runtime", "osx-arm64")));
    }

    private static DriverManifest Manifest(
        string protocolFamily = "test",
        string driverId = "test.driver",
        string version = "1.0.0",
        int schemaVersion = 1,
        IReadOnlyList<string>? operations = null,
        IReadOnlyDictionary<string, IReadOnlyList<string>>? platforms = null) => new(
        protocolFamily,
        driverId,
        version,
        schemaVersion,
        operations ?? [DriverOperations.ConnectionTest, DriverOperations.PointRead],
        ["tcp"],
        platforms);

    private sealed class TestDriver(IReadOnlyList<string>? operations = null) : IIndustrialDriver, IPointReader
    {
        public DriverDescriptor Descriptor { get; } = new(
            "test",
            "test.driver",
            "1.0.0",
            [1],
            operations ?? [DriverOperations.ConnectionTest, DriverOperations.PointRead]);

        public Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) =>
            Task.FromResult(new ConnectionTestResult(true, TimeSpan.Zero, null, []));

        public Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) =>
            Task.FromResult(new ReadResult([], []));
    }
}
