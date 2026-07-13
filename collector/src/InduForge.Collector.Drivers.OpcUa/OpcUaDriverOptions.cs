namespace InduForge.Collector.Drivers.OpcUa;

public sealed record OpcUaDriverOptions(
    string ApplicationName = "InduForge Collector Dev Agent",
    uint SessionTimeoutMilliseconds = 30_000,
    int ConnectionTimeoutMilliseconds = 10_000);
