namespace InduForge.Collector.DevAgent;

internal enum AgentConnectionState
{
    NotRegistered,
    Disconnected,
    Connected,
}

internal sealed record AgentStatusSnapshot(
    AgentConnectionState ConnectionState,
    string CenterUrl,
    string AgentName,
    string CurrentTask,
    string LastMessage,
    DateTimeOffset UpdatedAt);
