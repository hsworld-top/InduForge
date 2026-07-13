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
    string AgentId,
    string CurrentTask,
    string LastMessage,
    DateTimeOffset? LastHeartbeatAt,
    DateTimeOffset UpdatedAt);

internal sealed record AgentRegistrationFormValue(string CenterUrl, string AgentName, string RegistrationCode);
