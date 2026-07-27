using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.FatekProgramTcp;

public sealed class FatekProgramTcpDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly IHslFatekProgramTcpClientFactory _factory;
    public FatekProgramTcpDriver() : this(new HslFatekProgramTcpClientFactory()) { }
    internal FatekProgramTcpDriver(IHslFatekProgramTcpClientFactory factory) => _factory = factory;
    public DriverDescriptor Descriptor { get; } = new("fatek", "fatek.program-tcp", "1.0.0", [1],
        [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        watch.Stop();
        return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = FatekProgramTcpConnectionOptions.Parse(profile);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new FatekProgramDriverException(result.ErrorCode ?? "FATEK_PROGRAM_TCP_CONNECT_FAILED", result.ErrorMessage ?? "无法连接永宏 PLC", result.Retryable);
        }
        return new FatekProgramConnectionSession(client, $"{options.Host}:{options.Port} / Station={options.Station}");
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
