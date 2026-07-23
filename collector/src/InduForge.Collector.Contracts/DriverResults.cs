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

public sealed record BrowseBranchResult(
    string ParentNodeId,
    IReadOnlyList<IndustrialNode> Nodes,
    IReadOnlyList<DriverDiagnostic> Diagnostics);

public sealed record BrowseBatchResult(
    IReadOnlyList<BrowseBranchResult> Branches,
    IReadOnlyList<DriverDiagnostic> Diagnostics);

public sealed record IndustrialDataValue(
    string NodeId,
    object? Value,
    string? DataType,
    string Quality,
    DateTimeOffset SourceTimestamp,
    DateTimeOffset ServerTimestamp);

public sealed record PointReadValue(
    string Key,
    bool Succeeded,
    object? Value,
    string? DataType,
    string Quality,
    DateTimeOffset? SourceTimestamp,
    DateTimeOffset? ServerTimestamp,
    string? ErrorCode,
    string? ErrorMessage);

public sealed record ReadResult(
    IReadOnlyList<PointReadValue> Values,
    IReadOnlyList<DriverDiagnostic> Diagnostics);

public sealed record PointWriteResult(string NodeId, bool Succeeded, string? ErrorCode, string? ErrorMessage);

public sealed record WriteResult(
    IReadOnlyList<PointWriteResult> Results,
    IReadOnlyList<DriverDiagnostic> Diagnostics);

public sealed record SubscriptionPreviewResult(
    IReadOnlyList<IndustrialDataValue> Values,
    IReadOnlyList<DriverDiagnostic> Diagnostics);

public sealed record ConnectionSessionResult(
    bool Connected,
    string? ServerName,
    DateTimeOffset ConnectedAt);

public sealed record ConnectionSessionCloseResult(bool Closed);
