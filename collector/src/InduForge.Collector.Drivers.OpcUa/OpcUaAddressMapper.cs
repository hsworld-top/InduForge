using System.Text.Json;
using Opc.Ua;

namespace InduForge.Collector.Drivers.OpcUa;

public static class OpcUaAddressMapper
{
    public static string ParseNodeId(JsonElement address)
    {
        if (address.ValueKind != JsonValueKind.Object ||
            !address.TryGetProperty("nodeId", out var nodeIdElement) ||
            nodeIdElement.ValueKind != JsonValueKind.String ||
            string.IsNullOrWhiteSpace(nodeIdElement.GetString()))
        {
            throw new OpcUaDriverException("OPCUA_NODE_ID_REQUIRED", "OPC UA 地址必须包含 nodeId", retryable: false);
        }

        try
        {
            return NodeId.Parse(nodeIdElement.GetString()).ToString();
        }
        catch (Exception exception) when (exception is FormatException or ServiceResultException)
        {
            throw new OpcUaDriverException("OPCUA_NODE_ID_INVALID", "OPC UA NodeId 无效", retryable: false, exception);
        }
    }

    public static string ToAddressText(JsonElement address) => ParseNodeId(address);

    public static string? MapDataType(BuiltInType builtInType) => builtInType switch
    {
        BuiltInType.Boolean => "bool",
        BuiltInType.SByte => "int8",
        BuiltInType.Byte => "uint8",
        BuiltInType.Int16 => "int16",
        BuiltInType.UInt16 => "uint16",
        BuiltInType.Int32 => "int32",
        BuiltInType.UInt32 => "uint32",
        BuiltInType.Int64 => "int64",
        BuiltInType.UInt64 => "uint64",
        BuiltInType.Float => "float32",
        BuiltInType.Double => "float64",
        BuiltInType.String => "string",
        BuiltInType.ByteString => "bytes",
        BuiltInType.DateTime => "datetime",
        _ => null,
    };
}
