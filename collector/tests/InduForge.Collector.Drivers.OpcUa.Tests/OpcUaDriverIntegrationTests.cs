using InduForge.Collector.Contracts;
using Opc.Ua;
using CollectorBrowseRequest = InduForge.Collector.Contracts.BrowseRequest;
using CollectorReadRequest = InduForge.Collector.Contracts.ReadRequest;
using Xunit.Sdk;

namespace InduForge.Collector.Drivers.OpcUa.Tests;

public sealed class OpcUaDriverIntegrationTests
{
    private static readonly ConnectionProfile Profile = new(
        "opcua",
        "opc.tcp://127.0.0.1:18540/induforge/sim",
        "None",
        "None",
        new ConnectionAuthentication(AuthenticationType.Anonymous),
        TimeSpan.FromSeconds(10));

    [Fact]
    [Trait("Category", "Integration")]
    public async Task ConnectBrowseAndReadAgainstIndustrialSimulator()
    {
        if (!string.Equals(Environment.GetEnvironmentVariable("IF_RUN_OPCUA_INTEGRATION"), "true", StringComparison.OrdinalIgnoreCase))
        {
            return;
        }

        var driver = new OpcUaDriver();
        var connection = await driver.TestConnectionAsync(Profile, CancellationToken.None);
        Assert.True(connection.Connected);

        var variableNode = await FindFirstVariableAsync(driver, ObjectIds.ObjectsFolder.ToString());
        var read = await driver.ReadAsync(Profile, new CollectorReadRequest([variableNode.NodeId]), CancellationToken.None);

        var value = Assert.Single(read.Values);
        Assert.Equal(variableNode.NodeId, value.NodeId);
        Assert.False(string.IsNullOrWhiteSpace(value.Quality));
    }

    private static async Task<IndustrialNode> FindFirstVariableAsync(OpcUaDriver driver, string parentNodeId)
    {
        var queue = new Queue<(string NodeId, int Depth)>();
        queue.Enqueue((parentNodeId, 0));

        while (queue.Count > 0)
        {
            var current = queue.Dequeue();
            var browse = await driver.BrowseAsync(Profile, new CollectorBrowseRequest(current.NodeId), CancellationToken.None);
            var variable = browse.Nodes.FirstOrDefault(node => node.NodeClass == "variable");
            if (variable is not null)
            {
                return variable;
            }

            if (current.Depth >= 4)
            {
                continue;
            }

            foreach (var node in browse.Nodes.Where(node => node.HasChildren))
            {
                queue.Enqueue((node.NodeId, current.Depth + 1));
            }
        }

        throw new XunitException("模拟器地址空间中未找到可读取变量节点");
    }
}
