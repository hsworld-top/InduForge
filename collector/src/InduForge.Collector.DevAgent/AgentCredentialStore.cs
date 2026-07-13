using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

namespace InduForge.Collector.DevAgent;

internal sealed class AgentCredentialStore
{
    private static readonly byte[] Entropy = Encoding.UTF8.GetBytes("InduForge.Collector.DevAgent.v1");
    private readonly string _filePath;

    public AgentCredentialStore(string? filePath = null)
    {
        _filePath = filePath ?? Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData), "InduForge", "CollectorDevAgent", "credentials.dat");
    }

    public AgentCredentials? Load()
    {
        if (!File.Exists(_filePath)) return null;
        var protectedBytes = File.ReadAllBytes(_filePath);
        var plainBytes = ProtectedData.Unprotect(protectedBytes, Entropy, DataProtectionScope.CurrentUser);
        return JsonSerializer.Deserialize<AgentCredentials>(plainBytes);
    }

    public void Save(AgentCredentials credentials)
    {
        var directory = Path.GetDirectoryName(_filePath) ?? throw new InvalidOperationException("凭据目录无效");
        Directory.CreateDirectory(directory);
        var plainBytes = JsonSerializer.SerializeToUtf8Bytes(credentials);
        var protectedBytes = ProtectedData.Protect(plainBytes, Entropy, DataProtectionScope.CurrentUser);
        File.WriteAllBytes(_filePath, protectedBytes);
    }

    public void Clear()
    {
        if (File.Exists(_filePath)) File.Delete(_filePath);
    }
}
