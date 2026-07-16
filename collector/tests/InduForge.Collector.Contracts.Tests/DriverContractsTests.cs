using InduForge.Collector.Contracts;

namespace InduForge.Collector.Contracts.Tests;

public sealed class DriverContractsTests
{
    [Fact]
    public void DriverDescriptorUsesStablePublicNames()
    {
        var descriptor = new DriverDescriptor(
            "opcua", "opcua.standard", "1.0.0", [2],
            [DriverOperations.ConnectionTest, DriverOperations.DeviceBrowse, DriverOperations.PointRead]);

        Assert.Equal("opcua.standard", descriptor.DriverId);
        Assert.Contains("device.browse", descriptor.Operations);
        Assert.DoesNotContain(descriptor.Operations, operation => operation.Contains("sdk", StringComparison.OrdinalIgnoreCase));
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
