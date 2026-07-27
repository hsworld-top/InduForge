using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins;

public sealed class OmronFinsTcpDriver : OmronFinsDriverBase
{
    public OmronFinsTcpDriver()
        : this(new HslOmronFinsClientFactory())
    {
    }

    internal OmronFinsTcpDriver(IHslOmronFinsClientFactory clientFactory)
        : base("omron.fins-tcp", HslOmronFinsTransport.Tcp, clientFactory)
    {
    }
}

public sealed class OmronFinsUdpDriver : OmronFinsDriverBase
{
    public OmronFinsUdpDriver()
        : this(new HslOmronFinsClientFactory())
    {
    }

    internal OmronFinsUdpDriver(IHslOmronFinsClientFactory clientFactory)
        : base("omron.fins-udp", HslOmronFinsTransport.Udp, clientFactory)
    {
    }
}

public abstract class OmronFinsDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly HslOmronFinsTransport _transport;
    private readonly IHslOmronFinsClientFactory _clientFactory;

    private protected OmronFinsDriverBase(
        string driverId,
        HslOmronFinsTransport transport,
        IHslOmronFinsClientFactory clientFactory)
    {
        _transport = transport;
        _clientFactory = clientFactory;
        Descriptor = new DriverDescriptor(
            ProtocolFamily: "omron",
            DriverId: driverId,
            DriverVersion: "1.0.0",
            SchemaVersions: [1],
            Operations:
            [
                DriverOperations.ConnectionTest,
                DriverOperations.ConnectionOpen,
                DriverOperations.ConnectionClose,
                DriverOperations.PointRead,
            ]);
    }

    public DriverDescriptor Descriptor { get; }

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
        var options = OmronFinsConnectionOptions.Parse(profile, _transport);
        var client = _clientFactory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new OmronFinsDriverException(
                result.ErrorCode ?? "OMRON_FINS_CONNECT_FAILED",
                result.ErrorMessage ?? "无法连接 Omron FINS 设备",
                result.Retryable);
        }

        var transportName = _transport == HslOmronFinsTransport.Tcp ? "TCP" : "UDP";
        return new OmronFinsConnectionSession(client, $"{options.Host}:{options.Port} ({transportName})");
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
