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
            connection = new { protocolFamily = "opcua", config = new { endpointUrl = "opc.tcp://127.0.0.1:18540/induforge/sim", securityMode = "None", securityPolicy = "None", authenticationType = "anonymous" }, secrets = new { } },
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
            connection = new { protocolFamily = "opcua", config = new { endpointUrl = "opc.tcp://127.0.0.1:18540/induforge/sim", securityMode = "None", securityPolicy = "None", authenticationType = "anonymous" }, secrets = new { } },
            input = new { points = new[] { new { pointId = "point-1", address = new { nodeId = "ns=2;s=Temperature" }, dataType = "float32", elementCount = 1, readOptions = new { } } } },
        });
        var task = new CollectorTaskEnvelope("task-1", "project-1", "agent-1", "point.read", "running", request, "2026-07-13 12:00:00");

        var result = await new CollectorTaskExecutor(CreateRegistry()).ExecuteAsync(task, CancellationToken.None);

        Assert.Equal("succeeded", result.Status);
        var readResult = Assert.IsType<ReadResult>(result.Result);
        Assert.Equal("ns=2;s=Temperature", Assert.Single(readResult.Values).NodeId);
    }

    private static DriverRegistry CreateRegistry() => new([() => new TestDriver()]);

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
}

