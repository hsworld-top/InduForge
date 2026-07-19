namespace InduForge.Collector.Contracts;

public interface IConnectionSessionDriver
{
    Task<IIndustrialConnectionSession> OpenSessionAsync(
        ConnectionProfile profile,
        CancellationToken cancellationToken);
}

public interface IIndustrialConnectionSession : IAsyncDisposable
{
    bool IsConnected { get; }

    string? ServerName { get; }
}

public interface IDeviceBrowserSession
{
    Task<BrowseResult> BrowseAsync(BrowseRequest request, CancellationToken cancellationToken);
}

public interface IPointReaderSession
{
    Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken);
}
