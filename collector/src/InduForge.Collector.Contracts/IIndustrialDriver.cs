namespace InduForge.Collector.Contracts;

public interface IIndustrialDriver
{
    DriverDescriptor Descriptor { get; }

    Task<ConnectionTestResult> TestConnectionAsync(
        ConnectionProfile profile,
        CancellationToken cancellationToken);
}
