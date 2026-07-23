using System.Text.Json;

namespace InduForge.Collector.Contracts;

public sealed record ConnectionProfile(
    string ProtocolFamily,
    JsonElement Config,
    JsonElement Secrets)
{
    // 连接对象可能进入诊断上下文，禁止默认 record 字符串展开敏感配置。
    public override string ToString() => $"ConnectionProfile {{ ProtocolFamily = {ProtocolFamily} }}";
}

public sealed record BrowseRequest(string ParentNodeId, int MaxDepth = 1);

public sealed record PointReadRequest(
    string Key,
    JsonElement Address,
    string DataType,
    int ElementCount,
    JsonElement ReadOptions);

public sealed record ReadRequest(IReadOnlyList<PointReadRequest> Points);

public sealed record PointWriteValue(string NodeId, object? Value);

public sealed record WriteRequest(IReadOnlyList<PointWriteValue> Values);

public sealed record SubscriptionPreviewRequest(IReadOnlyList<string> NodeIds, TimeSpan Duration);
