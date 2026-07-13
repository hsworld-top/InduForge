using InduForge.Collector.Contracts;
using InduForge.Collector.Drivers.OpcUa;

namespace InduForge.Collector.DevAgent;

internal sealed class OpcUaSelfTestService
{
    private readonly OpcUaDriver _driver = new();

    public async Task<string> RunAsync(string endpointUrl, CancellationToken cancellationToken)
    {
        var profile = new ConnectionProfile(
            "opcua",
            endpointUrl,
            "None",
            "None",
            new ConnectionAuthentication(AuthenticationType.Anonymous),
            TimeSpan.FromSeconds(10));
        var connection = await _driver.TestConnectionAsync(profile, cancellationToken).ConfigureAwait(false);
        var browse = await _driver.BrowseAsync(profile, new BrowseRequest("ns=0;i=85"), cancellationToken).ConfigureAwait(false);
        return $"连接成功，耗时 {connection.Elapsed.TotalMilliseconds:F0} ms，Objects 下发现 {browse.Nodes.Count} 个节点";
    }
}
