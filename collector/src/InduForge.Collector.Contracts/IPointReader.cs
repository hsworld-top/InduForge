namespace InduForge.Collector.Contracts;

public interface IPointReader
{
    Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken);
}
