using System.Text.Json;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OpcUa.Tests;

internal static class OpcUaTestProfile
{
    public static ConnectionProfile Create() => new(
        "opcua",
        JsonSerializer.SerializeToElement(new
        {
            host = "127.0.0.1",
            port = 18540,
            endpointPath = "/induforge/sim",
            securityMode = "None",
            securityPolicy = "None",
            authenticationType = "anonymous",
            timeoutMs = 10000,
        }),
        JsonSerializer.SerializeToElement(new { }));
}
