using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Iec104;

public sealed class Iec104Driver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslIec104ClientFactory _clientFactory;

    public Iec104Driver()
        : this(new HslIec104ClientFactory())
    {
    }

    internal Iec104Driver(IHslIec104ClientFactory clientFactory)
    {
        _clientFactory = clientFactory;
    }

    public DriverDescriptor Descriptor { get; } = new(
        ProtocolFamily: "iec",
        DriverId: "iec.60870-5-104",
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
        return new ConnectionTestResult(session.IsConnected, stopwatch.Elapsed, session.ServerName, NoDiagnostics);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(
        ConnectionProfile profile,
        CancellationToken cancellationToken)
    {
        var options = Iec104ConnectionOptions.Parse(profile);
        var client = _clientFactory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new Iec104DriverException(
                result.ErrorCode ?? "IEC104_CONNECT_FAILED",
                result.ErrorMessage ?? "无法连接 IEC 104 设备",
                result.Retryable);
        }

        return new Iec104ConnectionSession(client, $"{options.Host}:{options.Port} / CA={options.CommonAddress}");
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
