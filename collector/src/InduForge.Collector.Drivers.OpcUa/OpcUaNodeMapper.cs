using InduForge.Collector.Contracts;
using Opc.Ua;
using Opc.Ua.Client;

namespace InduForge.Collector.Drivers.OpcUa;

internal static class OpcUaNodeMapper
{
    public static IndustrialNode Map(ReferenceDescription reference, NamespaceTable namespaceUris, string? dataType = null)
    {
        ArgumentNullException.ThrowIfNull(reference);
        ArgumentNullException.ThrowIfNull(namespaceUris);

        var nodeId = ExpandedNodeId.ToNodeId(reference.NodeId, namespaceUris)?.ToString()
            ?? reference.NodeId.ToString();

        return new IndustrialNode(
            nodeId,
            reference.BrowseName?.ToString() ?? string.Empty,
            reference.DisplayName?.Text ?? reference.BrowseName?.Name ?? nodeId,
            MapNodeClass(reference.NodeClass),
            DataType: dataType,
            HasChildren: reference.NodeClass is NodeClass.Object or NodeClass.View);
    }

    public static PointReadValue MapDataValue(PointReadRequest point, DataValue value)
    {
        ArgumentNullException.ThrowIfNull(point);
        ArgumentNullException.ThrowIfNull(value);

        var sourceTimestamp = NormalizeTimestamp(value.SourceTimestamp);
        var serverTimestamp = NormalizeTimestamp(value.ServerTimestamp);
        var succeeded = StatusCode.IsGood(value.StatusCode);
        var dataType = value.WrappedValue.TypeInfo is null
            ? point.DataType
            : OpcUaAddressMapper.MapDataType(value.WrappedValue.TypeInfo.BuiltInType) ?? point.DataType;

        return new PointReadValue(
            point.Key,
            succeeded,
            succeeded ? NormalizeValue(value.WrappedValue.Value) : null,
            dataType,
            succeeded ? "Good" : "Bad",
            sourceTimestamp,
            serverTimestamp,
            succeeded ? null : value.StatusCode.SymbolicId ?? "OPCUA_READ_BAD",
            succeeded ? null : "OPC UA 变量读取失败");
    }

    private static string MapNodeClass(NodeClass nodeClass) => nodeClass switch
    {
        NodeClass.Object => "object",
        NodeClass.Variable => "variable",
        NodeClass.Method => "method",
        _ => "other",
    };

    private static object? NormalizeValue(object? value) => value switch
    {
        null => null,
        NodeId nodeId => nodeId.ToString(),
        ExpandedNodeId expandedNodeId => expandedNodeId.ToString(),
        QualifiedName qualifiedName => qualifiedName.ToString(),
        LocalizedText localizedText => localizedText.Text,
        ExtensionObject extensionObject => extensionObject.Body?.ToString(),
        Array array => array.Cast<object?>().Select(NormalizeValue).ToArray(),
        _ => value,
    };

    private static DateTimeOffset? NormalizeTimestamp(DateTime timestamp)
    {
        if (timestamp == DateTime.MinValue)
        {
            return null;
        }

        return new DateTimeOffset(DateTime.SpecifyKind(timestamp, DateTimeKind.Utc));
    }
}
