namespace InduForge.Collector.Contracts;

public static class DriverOperations
{
    public const string ConnectionTest = "connection.test";
    public const string OpcUaBrowse = "opcua.browse";
    public const string OpcUaRead = "opcua.read";
}

public sealed record ProtocolCapability(
    string ProtocolType,
    string CapabilityVersion,
    IReadOnlyList<string> Operations);
