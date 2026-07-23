using System.Text.Json;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OpcUa.Tests;

public sealed class OpcUaConnectionProfileTests
{
    [Theory]
    [InlineData("127.0.0.1", 4840, "/factory/server", "opc.tcp://127.0.0.1:4840/factory/server")]
    [InlineData("plc.local", 4840, "factory/server", "opc.tcp://plc.local:4840/factory/server")]
    [InlineData("2001:db8::1", 4840, "/", "opc.tcp://[2001:db8::1]:4840/")]
    public void ParseCombinesStructuredConfiguration(string host, int port, string path, string expected)
    {
        var profile = new ConnectionProfile(
            "opcua",
            JsonSerializer.SerializeToElement(new
            {
                host,
                port,
                endpointPath = path,
                securityMode = "None",
                securityPolicy = "None",
                authenticationType = "anonymous",
                timeoutMs = 30000,
            }),
            JsonSerializer.SerializeToElement(new { }));

        var parsed = OpcUaConnectionProfile.Parse(profile, 30000);

        Assert.Equal(expected, parsed.EndpointUrl);
    }

    [Theory]
    [InlineData("opc.tcp://127.0.0.1", 4840)]
    [InlineData("", 4840)]
    [InlineData("127.0.0.1", 0)]
    public void ParseRejectsInvalidConfiguration(string host, int port)
    {
        var profile = new ConnectionProfile(
            "opcua",
            JsonSerializer.SerializeToElement(new
            {
                host,
                port,
                endpointPath = "/",
                securityMode = "None",
                securityPolicy = "None",
                authenticationType = "anonymous",
            }),
            JsonSerializer.SerializeToElement(new { }));

        Assert.Throws<OpcUaDriverException>(() => OpcUaConnectionProfile.Parse(profile, 30000));
    }
}
