using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp;

public sealed class SiemensS7TcpDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslS7TcpClientFactory _clientFactory;

    public SiemensS7TcpDriver()
        : this(new HslS7TcpClientFactory())
    {
    }

    internal SiemensS7TcpDriver(IHslS7TcpClientFactory clientFactory)
    {
        _clientFactory = clientFactory;
    }

    public DriverDescriptor Descriptor { get; } = new(
        ProtocolFamily: "siemens",
        DriverId: "siemens.s7-tcp",
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
        var options = SiemensS7TcpConnectionOptions.Parse(profile);
        var client = _clientFactory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new SiemensS7TcpDriverException(
                result.ErrorCode ?? "S7_CONNECT_FAILED",
                result.ErrorMessage ?? "无法连接 Siemens S7 TCP 设备",
                result.Retryable);
        }

        return new SiemensS7TcpConnectionSession(client, $"{options.PlcType} {options.Host}:{options.Port}");
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
