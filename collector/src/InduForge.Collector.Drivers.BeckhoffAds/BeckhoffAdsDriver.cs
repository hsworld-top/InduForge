using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.BeckhoffAds;

public sealed class BeckhoffAdsDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslBeckhoffAdsClientFactory _clientFactory;

    public BeckhoffAdsDriver()
        : this(new HslBeckhoffAdsClientFactory())
    {
    }

    internal BeckhoffAdsDriver(IHslBeckhoffAdsClientFactory clientFactory)
    {
        _clientFactory = clientFactory;
    }

    public DriverDescriptor Descriptor { get; } = new(
        ProtocolFamily: "beckhoff",
        DriverId: "beckhoff.ads-tcp",
        DriverVersion: "1.0.0",
        SchemaVersions: [1],
        Operations:
        [
            DriverOperations.ConnectionTest,
            DriverOperations.ConnectionOpen,
            DriverOperations.ConnectionClose,
            DriverOperations.PointRead,
        ]);

    public async Task<ConnectionTestResult> TestConnectionAsync(
        ConnectionProfile profile,
        CancellationToken cancellationToken)
    {
        var stopwatch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        stopwatch.Stop();
        return new ConnectionTestResult(
            session.IsConnected,
            stopwatch.Elapsed,
            session.ServerName,
            NoDiagnostics);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(
        ConnectionProfile profile,
        CancellationToken cancellationToken)
    {
        var options = BeckhoffAdsConnectionOptions.Parse(profile);
        var client = _clientFactory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new BeckhoffAdsDriverException(
                result.ErrorCode ?? "BECKHOFF_ADS_CONNECT_FAILED",
                result.ErrorMessage ?? "无法连接倍福 ADS 设备",
                result.Retryable);
        }

        return new BeckhoffAdsConnectionSession(client, $"{options.Host}:{options.Port}");
    }

    public async Task<ReadResult> ReadAsync(
        ConnectionProfile profile,
        ReadRequest request,
        CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
