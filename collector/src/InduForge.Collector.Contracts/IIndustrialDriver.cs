namespace InduForge.Collector.Contracts;

public interface IIndustrialDriver
{
    ProtocolCapability Capability { get; }

    Task<ConnectionTestResult> TestConnectionAsync(
        ConnectionProfile profile,
        CancellationToken cancellationToken);

    Task<BrowseResult> BrowseAsync(
        ConnectionProfile profile,
        BrowseRequest request,
        CancellationToken cancellationToken);

    Task<ReadResult> ReadAsync(
        ConnectionProfile profile,
        ReadRequest request,
        CancellationToken cancellationToken);
}
