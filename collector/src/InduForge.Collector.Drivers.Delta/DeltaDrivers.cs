using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Delta;

public sealed class DeltaTcpDriver : DeltaDriverBase { public DeltaTcpDriver() : base(DeltaDriverKind.Tcp, "delta.tcp") { } internal DeltaTcpDriver(IHslDeltaClientFactory factory) : base(DeltaDriverKind.Tcp, "delta.tcp", factory) { } }
public sealed class DeltaRtuOverTcpDriver : DeltaDriverBase { public DeltaRtuOverTcpDriver() : base(DeltaDriverKind.RtuOverTcp, "delta.rtu-over-tcp") { } internal DeltaRtuOverTcpDriver(IHslDeltaClientFactory factory) : base(DeltaDriverKind.RtuOverTcp, "delta.rtu-over-tcp", factory) { } }
public sealed class DeltaAsciiOverTcpDriver : DeltaDriverBase { public DeltaAsciiOverTcpDriver() : base(DeltaDriverKind.AsciiOverTcp, "delta.ascii-over-tcp") { } internal DeltaAsciiOverTcpDriver(IHslDeltaClientFactory factory) : base(DeltaDriverKind.AsciiOverTcp, "delta.ascii-over-tcp", factory) { } }
public sealed class DeltaRtuDriver : DeltaDriverBase { public DeltaRtuDriver() : base(DeltaDriverKind.RtuSerial, "delta.rtu") { } internal DeltaRtuDriver(IHslDeltaClientFactory factory) : base(DeltaDriverKind.RtuSerial, "delta.rtu", factory) { } }
public sealed class DeltaAsciiDriver : DeltaDriverBase { public DeltaAsciiDriver() : base(DeltaDriverKind.AsciiSerial, "delta.ascii") { } internal DeltaAsciiDriver(IHslDeltaClientFactory factory) : base(DeltaDriverKind.AsciiSerial, "delta.ascii", factory) { } }

public abstract class DeltaDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly DeltaDriverKind _kind;
    private readonly IHslDeltaClientFactory _factory;
    private protected DeltaDriverBase(DeltaDriverKind kind, string driverId, IHslDeltaClientFactory? factory = null)
    {
        _kind = kind;
        _factory = factory ?? new HslDeltaClientFactory();
        Descriptor = new("delta", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    }
    public DriverDescriptor Descriptor { get; }
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        watch.Stop();
        return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = DeltaConnectionOptions.Parse(profile, _kind);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new DeltaDriverException(result.ErrorCode ?? "DELTA_CONNECT_FAILED", result.ErrorMessage ?? "无法连接台达 PLC", result.Retryable);
        }
        var endpoint = options.PortName ?? $"{options.Host}:{options.Port}";
        return new DeltaConnectionSession(client, $"{endpoint} / {options.Series} / Station={options.Station}");
    }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
