using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InovanceModbusTcp;

public sealed class InovanceModbusTcpDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslInovanceModbusTcpClientFactory _clientFactory;

    public InovanceModbusTcpDriver() : this(new HslInovanceModbusTcpClientFactory()) { }
    internal InovanceModbusTcpDriver(IHslInovanceModbusTcpClientFactory clientFactory) => _clientFactory = clientFactory;

    public DriverDescriptor Descriptor { get; } = new(
        "inovance", "inovance.modbus-tcp", "1.0.0", [1],
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
        var options = InovanceModbusTcpConnectionOptions.Parse(profile);
        var client = _clientFactory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new InovanceModbusTcpDriverException(
                result.ErrorCode ?? "INOVANCE_MODBUS_TCP_CONNECT_FAILED",
                result.ErrorMessage ?? "无法连接汇川 PLC",
                result.Retryable);
        }
        return new InovanceModbusTcpConnectionSession(
            client, $"{options.Host}:{options.Port} / {options.Series} / Station={options.Station}");
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
