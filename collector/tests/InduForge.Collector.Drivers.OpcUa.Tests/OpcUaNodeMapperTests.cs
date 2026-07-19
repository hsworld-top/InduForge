using InduForge.Collector.Contracts;
using Opc.Ua;

namespace InduForge.Collector.Drivers.OpcUa.Tests;

public sealed class OpcUaNodeMapperTests
{
    [Fact]
    public void MapsVariableReferenceToStableNodeModel()
    {
        var namespaces = new NamespaceTable();
        namespaces.Append("urn:induforge:test");
        var reference = new ReferenceDescription
        {
            NodeId = new ExpandedNodeId("Pressure", 1),
            BrowseName = new QualifiedName("Pressure", 1),
            DisplayName = new LocalizedText("压力"),
            NodeClass = NodeClass.Variable,
        };

        var result = OpcUaNodeMapper.Map(reference, namespaces);

        Assert.Equal("ns=1;s=Pressure", result.NodeId);
        Assert.Equal("1:Pressure", result.BrowseName);
        Assert.Equal("压力", result.DisplayName);
        Assert.Equal("variable", result.NodeClass);
        Assert.Null(result.DataType);
        Assert.False(result.HasChildren);
    }

    [Fact]
    public void KeepsBrowseDataTypeOnVariableNode()
    {
        var result = OpcUaNodeMapper.Map(
            new ReferenceDescription
            {
                NodeId = new ExpandedNodeId("Temperature", 2),
                BrowseName = new QualifiedName("Temperature", 2),
                DisplayName = new LocalizedText("温度"),
                NodeClass = NodeClass.Variable,
            },
            new NamespaceTable(),
            "float32");

        Assert.Equal("float32", result.DataType);
    }

    [Fact]
    public void MapsObjectReferenceAsExpandable()
    {
        var result = OpcUaNodeMapper.Map(
            new ReferenceDescription
            {
                NodeId = new ExpandedNodeId(ObjectIds.ObjectsFolder),
                BrowseName = new QualifiedName("Objects", 0),
                DisplayName = new LocalizedText("Objects"),
                NodeClass = NodeClass.Object,
            },
            new NamespaceTable());

        Assert.Equal("object", result.NodeClass);
        Assert.True(result.HasChildren);
    }

    [Fact]
    public void MapsDataValueWithoutTimestampToMinimumTimestamp()
    {
        var value = new DataValue(new Variant(12.5), StatusCodes.Good, DateTime.MinValue, DateTime.MinValue);

        var result = OpcUaNodeMapper.MapDataValue("ns=2;s=Pressure", value);

        Assert.Equal(12.5, result.Value);
        Assert.Equal("float64", result.DataType);
        Assert.Equal("Good", result.Quality);
        Assert.Equal(DateTimeOffset.MinValue, result.SourceTimestamp);
    }
}
