using System.Text.Json;
using InduForge.Collector.Contracts;
using InduForge.Collector.DevAgent;

namespace InduForge.Collector.DevAgent.Tests;

public sealed class CollectorTaskExecutorTests
{
    [Fact]
    public async Task ExecuteAsyncRejectsUnsupportedOperation()
    {
        var request = JsonSerializer.SerializeToElement(new
        {
            driverId = "test.driver",
            driverVersion = "1.0.0",
            schemaVersion = 1,
            connection = new { protocolFamily = "opcua", config = new { host = "127.0.0.1", port = 18540, endpointPath = "/induforge/sim", securityMode = "None", securityPolicy = "None", authenticationType = "anonymous" }, secrets = new { } },
            input = new { },
        });
        var task = new CollectorTaskEnvelope("task-1", "project-1", "agent-1", "point.write", "running", request, "2026-07-13 12:00:00");
        var result = await new CollectorTaskExecutor(CreateRegistry()).ExecuteAsync(task, CancellationToken.None);
        Assert.Equal("failed", result.Status);
        Assert.Equal("COLLECTOR_OPERATION_UNSUPPORTED", result.Error?.Code);
    }

    [Fact]
    public async Task ExecutePointReadUsesSavedStructuredAddresses()
    {
        var request = JsonSerializer.SerializeToElement(new
        {
            driverId = "test.driver",
            driverVersion = "1.0.0",
            schemaVersion = 1,
            connection = new { protocolFamily = "opcua", config = new { host = "127.0.0.1", port = 18540, endpointPath = "/induforge/sim", securityMode = "None", securityPolicy = "None", authenticationType = "anonymous" }, secrets = new { } },
            input = new { points = new[] { new { pointId = "point-1", address = new { nodeId = "ns=2;s=Temperature" }, dataType = "float32", elementCount = 1, readOptions = new { } } } },
        });
        var task = new CollectorTaskEnvelope("task-1", "project-1", "agent-1", "point.read", "running", request, "2026-07-13 12:00:00");

        var result = await new CollectorTaskExecutor(CreateRegistry()).ExecuteAsync(task, CancellationToken.None);

        Assert.Equal("succeeded", result.Status);
        var readResult = Assert.IsType<ReadResult>(result.Result);
        Assert.Equal("ns=2;s=Temperature", Assert.Single(readResult.Values).NodeId);
    }

    private static DriverRegistry CreateRegistry() => new([() => new TestDriver()]);

    [Theory]
    [InlineData("127.0.0.1", 4840, "/factory/server", "opc.tcp://127.0.0.1:4840/factory/server")]
    [InlineData("plc.local", 4840, "factory/server", "opc.tcp://plc.local:4840/factory/server")]
    [InlineData("2001:db8::1", 4840, "/", "opc.tcp://[2001:db8::1]:4840/")]
    public void OpcUaEndpointBuilderCombinesStructuredConfiguration(string host, int port, string path, string expected)
    {
        Assert.Equal(expected, OpcUaEndpointBuilder.Build(host, port, path));
    }

    [Theory]
    [InlineData("opc.tcp://127.0.0.1", 4840)]
    [InlineData("", 4840)]
    [InlineData("127.0.0.1", 0)]
    public void OpcUaEndpointBuilderRejectsInvalidConfiguration(string host, int port)
    {
        Assert.Throws<CollectorTaskExecutionException>(() => OpcUaEndpointBuilder.Build(host, port, "/"));
    }

    private sealed class TestDriver : IIndustrialDriver, IPointReader
    {
        public DriverDescriptor Descriptor { get; } = new(
            "opcua", "test.driver", "1.0.0", [1],
            [DriverOperations.ConnectionTest, DriverOperations.PointRead]);

        public Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) =>
            Task.FromResult(new ConnectionTestResult(true, TimeSpan.Zero, "test", []));

        public Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) =>
            Task.FromResult(new ReadResult(request.NodeIds.Select(nodeId => new IndustrialDataValue(nodeId, 12.5, "float64", "Good", DateTimeOffset.UtcNow, DateTimeOffset.UtcNow)).ToArray(), []));
    }

    [Fact]
    public void CredentialStoreRoundTripsForCurrentUser()
    {
        var path = Path.Combine(Path.GetTempPath(), $"induforge-collector-{Guid.NewGuid():N}.dat");
        try
        {
            var store = new AgentCredentialStore(path);
            var expected = new AgentCredentials("http://127.0.0.1:18102", "agent-1", "secret-token", "tenant-1", "dev-machine");
            store.Save(expected);
            Assert.Equal(expected, store.Load());
            Assert.DoesNotContain("secret-token", File.ReadAllText(path));
            store.Clear();
            Assert.Null(store.Load());
        }
        finally
        {
            if (File.Exists(path)) File.Delete(path);
        }
    }

    [Fact]
    public void CredentialStoreDefaultPathUsesExecutableDataDirectory()
    {
        var baseDirectory = Path.Combine(Path.GetTempPath(), $"induforge-collector-{Guid.NewGuid():N}");
        Assert.Equal(Path.Combine(baseDirectory, "data", "credentials.dat"), AgentCredentialStore.GetDefaultPath(baseDirectory));
    }

    [Fact]
    public void CredentialStoreCreatesDataDirectoryOnStartup()
    {
        var baseDirectory = Path.Combine(Path.GetTempPath(), $"induforge-collector-{Guid.NewGuid():N}");
        try
        {
            _ = new AgentCredentialStore(Path.Combine(baseDirectory, "data", "credentials.dat"));
            Assert.True(Directory.Exists(Path.Combine(baseDirectory, "data")));
        }
        finally
        {
            if (Directory.Exists(baseDirectory)) Directory.Delete(baseDirectory, true);
        }
    }
}

