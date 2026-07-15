namespace InduForge.Collector.Contracts;

public sealed record DriverDescriptor(
    string ProtocolFamily,
    string DriverId,
    string DriverVersion,
    IReadOnlyList<int> SchemaVersions,
    IReadOnlyList<string> Operations);
