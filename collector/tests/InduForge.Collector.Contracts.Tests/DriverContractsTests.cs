using InduForge.Collector.Contracts;

namespace InduForge.Collector.Contracts.Tests;

public sealed class DriverContractsTests
{
    [Fact]
    public void OpcUaCapabilityUsesStablePublicNames()
    {
        var capability = new ProtocolCapability(
            "opcua",
            "1.0",
            [DriverOperations.ConnectionTest, DriverOperations.OpcUaBrowse, DriverOperations.OpcUaRead]);

        Assert.Equal("opcua", capability.ProtocolType);
        Assert.Contains("opcua.browse", capability.Operations);
        Assert.DoesNotContain(capability.Operations, operation => operation.Contains("sdk", StringComparison.OrdinalIgnoreCase));
    }

    [Fact]
    public void BrowseRequestDefaultsToDirectChildren()
    {
        var request = new BrowseRequest("ns=0;i=85");

        Assert.Equal("ns=0;i=85", request.ParentNodeId);
        Assert.Equal(1, request.MaxDepth);
    }

    [Fact]
    public void ConnectionProfileDoesNotExposeSecretsThroughStringRepresentation()
    {
        var profile = new ConnectionProfile(
            "opcua",
            "opc.tcp://127.0.0.1:18540/induforge/sim",
            "None",
            "None",
            new ConnectionAuthentication(AuthenticationType.Username, "operator", "secret"));

        Assert.DoesNotContain("secret", profile.ToString(), StringComparison.Ordinal);
    }
}
