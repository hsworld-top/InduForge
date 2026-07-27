using InduForge.Collector.DevAgent;

namespace InduForge.Collector.DevAgent.Tests;

public sealed class SerialPortCatalogTests
{
    [Fact]
    public void ListReturnsStableDistinctPortNames()
    {
        var catalog = new SerialPortCatalog(() => ["COM10", "com2", " COM2 ", ""]);

        var ports = catalog.List();

        Assert.Equal(["com2", "COM10"], ports);
    }

    [Fact]
    public void ListReturnsEmptyWhenEnumerationFails()
    {
        var catalog = new SerialPortCatalog(() => throw new InvalidOperationException("failed"));

        Assert.Empty(catalog.List());
    }
}
