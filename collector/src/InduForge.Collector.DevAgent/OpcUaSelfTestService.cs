using System.Text.Json;
using InduForge.Collector.Contracts;
using InduForge.Collector.Drivers.OpcUa;

namespace InduForge.Collector.DevAgent;

internal sealed class OpcUaSelfTestService
{
    private readonly OpcUaDriver _driver = new();

    public async Task<string> RunAsync(string endpointUrl, CancellationToken cancellationToken)
    {
        if (!Uri.TryCreate(endpointUrl, UriKind.Absolute, out var endpoint))
        {
            throw new ArgumentException("OPC UA Endpoint 地址无效", nameof(endpointUrl));
        }
        var profile = new ConnectionProfile(
            "opcua",
            JsonSerializer.SerializeToElement(new
            {
                host = endpoint.Host,
                port = endpoint.Port,
                endpointPath = endpoint.AbsolutePath,
                securityMode = "None",
                securityPolicy = "None",
                authenticationType = "anonymous",
                timeoutMs = 10000,
            }),
            JsonSerializer.SerializeToElement(new { }));
        var connection = await _driver.TestConnectionAsync(profile, cancellationToken).ConfigureAwait(false);
        var browse = await _driver.BrowseAsync(profile, new BrowseRequest("ns=0;i=85"), cancellationToken).ConfigureAwait(false);
        return $"连接成功，耗时 {connection.Elapsed.TotalMilliseconds:F0} ms，Objects 下发现 {browse.Nodes.Count} 个节点";
    }
}
