using System.Text.Json;
using InduForge.Collector.DevAgent;

namespace InduForge.Collector.DevAgent.Tests;

public sealed class CollectorTaskExecutorTests
{
    [Fact]
    public async Task ExecuteAsyncRejectsUnsupportedOperation()
    {
        var request = JsonSerializer.SerializeToElement(new
        {
            connection = new { protocolType = "opcua", endpointUrl = "opc.tcp://127.0.0.1:18540/induforge/sim", securityMode = "None", securityPolicy = "None", authentication = new { type = "anonymous" } },
            input = new { },
        });
        var task = new CollectorTaskEnvelope("task-1", "project-1", "agent-1", "opcua.write", "running", request, "2026-07-13 12:00:00");
        var result = await new CollectorTaskExecutor().ExecuteAsync(task, CancellationToken.None);
        Assert.Equal("failed", result.Status);
        Assert.Equal("COLLECTOR_OPERATION_UNSUPPORTED", result.Error?.Code);
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

