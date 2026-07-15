namespace InduForge.Collector.Contracts;

public interface IPointWriter
{
    Task<WriteResult> WriteAsync(ConnectionProfile profile, WriteRequest request, CancellationToken cancellationToken);
}
