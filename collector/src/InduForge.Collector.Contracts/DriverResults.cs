namespace InduForge.Collector.Contracts;

public sealed record DriverDiagnostic(string Level, string Code, string Message);

public sealed record ConnectionTestResult(
    bool Connected,
    TimeSpan Elapsed,
    string? ServerName,
    IReadOnlyList<DriverDiagnostic> Diagnostics);

public sealed record IndustrialNode(
    string NodeId,
    string BrowseName,
    string DisplayName,
    string NodeClass,
    string? DataType,
    bool HasChildren);

public sealed record BrowseResult(
    IReadOnlyList<IndustrialNode> Nodes,
    IReadOnlyList<DriverDiagnostic> Diagnostics);

public sealed record IndustrialDataValue(
    string NodeId,
    object? Value,
    string? DataType,
    string Quality,
    DateTimeOffset SourceTimestamp,
    DateTimeOffset ServerTimestamp);

public sealed record ReadResult(
    IReadOnlyList<IndustrialDataValue> Values,
    IReadOnlyList<DriverDiagnostic> Diagnostics);
