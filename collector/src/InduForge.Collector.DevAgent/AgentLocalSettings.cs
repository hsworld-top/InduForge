namespace InduForge.Collector.DevAgent;

internal sealed record AgentLocalSettings(string CenterUrl, string AgentName, string OpcUaEndpoint)
{
    public static AgentLocalSettings Default { get; } = new(
        "http://127.0.0.1:18102",
        Environment.MachineName,
        "opc.tcp://127.0.0.1:18540/induforge/sim");
}
