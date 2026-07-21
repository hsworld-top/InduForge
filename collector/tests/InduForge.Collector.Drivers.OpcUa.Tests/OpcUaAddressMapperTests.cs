using System.Text.Json;
using Opc.Ua;

namespace InduForge.Collector.Drivers.OpcUa.Tests;

public sealed class OpcUaAddressMapperTests
{
    [Fact]
    public void ParsesStructuredAddress()
    {
        var address = JsonSerializer.SerializeToElement(new { nodeId = "ns=2;s=Temperature" });

        Assert.Equal("ns=2;s=Temperature", OpcUaAddressMapper.ParseNodeId(address));
        Assert.Equal("ns=2;s=Temperature", OpcUaAddressMapper.ToAddressText(address));
    }

    [Fact]
    public void MapsOpcUaTypeToPlatformType()
    {
        Assert.Equal("float64", OpcUaAddressMapper.MapDataType(BuiltInType.Double));
        Assert.Equal("datetime", OpcUaAddressMapper.MapDataType(BuiltInType.DateTime));
    }

    [Fact]
    public void RejectsNodeIdWithoutIdentifierType()
    {
        var address = JsonSerializer.SerializeToElement(new { nodeId = "111" });

        var exception = Assert.Throws<OpcUaDriverException>(() => OpcUaAddressMapper.ParseNodeId(address));

        Assert.Equal("OPCUA_NODE_ID_INVALID", exception.Code);
    }
}
