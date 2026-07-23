using System.Text.Json;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Contracts.Tests;

public sealed class DriverContractsTests
{
    [Fact]
    public void DriverDescriptorUsesStablePublicNames()
    {
        var descriptor = new DriverDescriptor(
            "opcua", "opcua.standard", "1.0.0", [2],
            [DriverOperations.ConnectionTest, DriverOperations.DeviceBrowse, DriverOperations.PointRead]);

        Assert.Equal("opcua.standard", descriptor.DriverId);
        Assert.Contains("device.browse", descriptor.Operations);
        Assert.DoesNotContain(descriptor.Operations, operation => operation.Contains("sdk", StringComparison.OrdinalIgnoreCase));
    }

    [Fact]
    public void BrowseRequestDefaultsToDirectChildren()
    {
        var request = new BrowseRequest("ns=0;i=85");

        Assert.Equal("ns=0;i=85", request.ParentNodeId);
        Assert.Equal(1, request.MaxDepth);
    }

    [Fact]
    public void ConnectionProfileDoesNotExposeSecretsThroughStringRepresentation()
    {
        var profile = new ConnectionProfile(
            "opcua",
            JsonSerializer.SerializeToElement(new { host = "127.0.0.1", port = 18540 }),
            JsonSerializer.SerializeToElement(new { username = "operator", password = "secret" }));

        Assert.DoesNotContain("secret", profile.ToString(), StringComparison.Ordinal);
    }

    [Fact]
    public void ConnectionProfilePreservesProtocolSpecificConfiguration()
    {
        var profile = new ConnectionProfile(
            "modbus",
            JsonSerializer.SerializeToElement(new
            {
                host = "127.0.0.1",
                port = 502,
                dataFormat = "CDAB",
            }),
            JsonSerializer.SerializeToElement(new { }));

        Assert.Equal("modbus", profile.ProtocolFamily);
        Assert.Equal("127.0.0.1", profile.Config.GetProperty("host").GetString());
        Assert.Equal(502, profile.Config.GetProperty("port").GetInt32());
        Assert.Equal("CDAB", profile.Config.GetProperty("dataFormat").GetString());
    }

    [Fact]
    public void ReadRequestPreservesStructuredPointConfiguration()
    {
        var request = new ReadRequest([
            new PointReadRequest(
                "point-1",
                JsonSerializer.SerializeToElement(new
                {
                    station = 1,
                    area = "holdingRegister",
                    address = 10,
                }),
                "int16",
                2,
                JsonSerializer.SerializeToElement(new { byteOrder = "ABCD" })),
        ]);

        var point = Assert.Single(request.Points);
        Assert.Equal("point-1", point.Key);
        Assert.Equal("holdingRegister", point.Address.GetProperty("area").GetString());
        Assert.Equal("int16", point.DataType);
        Assert.Equal(2, point.ElementCount);
        Assert.Equal("ABCD", point.ReadOptions.GetProperty("byteOrder").GetString());
    }
}
