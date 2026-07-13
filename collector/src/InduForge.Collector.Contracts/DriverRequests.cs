namespace InduForge.Collector.Contracts;

public enum AuthenticationType
{
    Anonymous,
    Username,
    Certificate,
}

public sealed class ConnectionAuthentication
{
    public ConnectionAuthentication(
        AuthenticationType type,
        string? username = null,
        string? password = null,
        string? certificatePath = null,
        string? privateKeyPath = null)
    {
        Type = type;
        Username = username;
        Password = password;
        CertificatePath = certificatePath;
        PrivateKeyPath = privateKeyPath;
    }

    public AuthenticationType Type { get; }

    public string? Username { get; }

    public string? Password { get; }

    public string? CertificatePath { get; }

    public string? PrivateKeyPath { get; }

    // 认证对象可能进入诊断上下文，字符串表示只保留非敏感身份信息。
    public override string ToString() => $"ConnectionAuthentication {{ Type = {Type}, Username = {Username ?? "<null>"} }}";
}

public sealed record ConnectionProfile(
    string ProtocolType,
    string EndpointUrl,
    string SecurityMode,
    string SecurityPolicy,
    ConnectionAuthentication Authentication,
    TimeSpan? Timeout = null);

public sealed record BrowseRequest(string ParentNodeId, int MaxDepth = 1);

public sealed record ReadRequest(IReadOnlyList<string> NodeIds);
