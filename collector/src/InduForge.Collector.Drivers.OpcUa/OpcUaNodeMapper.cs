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

    public static IndustrialDataValue MapDataValue(string nodeId, DataValue value)
    {
        ArgumentException.ThrowIfNullOrWhiteSpace(nodeId);
        ArgumentNullException.ThrowIfNull(value);

        var sourceTimestamp = NormalizeTimestamp(value.SourceTimestamp);
        var serverTimestamp = NormalizeTimestamp(value.ServerTimestamp);

        return new IndustrialDataValue(
            nodeId,
            NormalizeValue(value.WrappedValue.Value),
            value.WrappedValue.TypeInfo is null ? null : OpcUaAddressMapper.MapDataType(value.WrappedValue.TypeInfo.BuiltInType),
            MapQuality(value.StatusCode),
            sourceTimestamp,
            serverTimestamp);
    }

    private static string MapQuality(StatusCode statusCode)
    {
        if (StatusCode.IsGood(statusCode))
        {
            return "Good";
        }

        if (StatusCode.IsUncertain(statusCode))
        {
            return statusCode.SymbolicId ?? "Uncertain";
        }

        return statusCode.SymbolicId ?? "Bad";
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

    private static DateTimeOffset NormalizeTimestamp(DateTime timestamp)
    {
        if (timestamp == DateTime.MinValue)
        {
            return DateTimeOffset.MinValue;
        }

        return new DateTimeOffset(DateTime.SpecifyKind(timestamp, DateTimeKind.Utc));
    }
}
