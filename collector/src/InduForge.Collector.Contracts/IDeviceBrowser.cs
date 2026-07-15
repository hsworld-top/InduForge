namespace InduForge.Collector.Contracts;

public interface IDeviceBrowser
{
    Task<BrowseResult> BrowseAsync(ConnectionProfile profile, BrowseRequest request, CancellationToken cancellationToken);
}
