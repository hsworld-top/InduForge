using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OpcUa.Tests;

public sealed class OpcUaDriverValidationTests
{
    private static readonly ConnectionProfile ValidProfile = OpcUaTestProfile.Create();

    [Fact]
    public async Task RejectsNestedBrowseUntilRecursiveContractExists()
    {
        var driver = new OpcUaDriver();

        var exception = await Assert.ThrowsAsync<OpcUaDriverException>(() =>
            driver.BrowseAsync(ValidProfile, new BrowseRequest("ns=0;i=85", MaxDepth: 2), CancellationToken.None));

        Assert.Equal("OPCUA_BROWSE_DEPTH_UNSUPPORTED", exception.Code);
        Assert.False(exception.Retryable);
    }

    [Fact]
    public async Task ReturnsEmptyReadWithoutOpeningConnection()
    {
        var driver = new OpcUaDriver();

        var result = await driver.ReadAsync(ValidProfile, new ReadRequest([]), CancellationToken.None);

        Assert.Empty(result.Values);
    }
}
