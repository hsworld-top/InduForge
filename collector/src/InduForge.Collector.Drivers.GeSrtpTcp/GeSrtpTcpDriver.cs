using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.GeSrtpTcp;

public sealed class GeSrtpTcpDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslGeSrtpClientFactory _clientFactory;

    public GeSrtpTcpDriver() : this(new HslGeSrtpClientFactory()) { }
    internal GeSrtpTcpDriver(IHslGeSrtpClientFactory clientFactory) => _clientFactory = clientFactory;

    public DriverDescriptor Descriptor { get; } = new(
        "ge", "ge.srtp-tcp", "1.0.0", [1],
        [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var stopwatch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        stopwatch.Stop();
        return new ConnectionTestResult(session.IsConnected, stopwatch.Elapsed, session.ServerName, NoDiagnostics);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = GeSrtpTcpConnectionOptions.Parse(profile);
        var client = _clientFactory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new GeSrtpTcpDriverException(
                result.ErrorCode ?? "GE_SRTP_CONNECT_FAILED",
                result.ErrorMessage ?? "无法连接 GE SRTP 设备",
                result.Retryable);
        }
        return new GeSrtpTcpConnectionSession(client, $"{options.Host}:{options.Port}");
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
