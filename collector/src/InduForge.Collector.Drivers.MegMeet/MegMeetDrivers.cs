using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MegMeet;

public sealed class MegMeetTcpDriver : MegMeetDriverBase
{
    public MegMeetTcpDriver() : base(MegMeetDriverKind.Tcp, "megmeet.tcp") { }
    internal MegMeetTcpDriver(IHslMegMeetClientFactory factory) : base(MegMeetDriverKind.Tcp, "megmeet.tcp", factory) { }
}

public sealed class MegMeetRtuOverTcpDriver : MegMeetDriverBase
{
    public MegMeetRtuOverTcpDriver() : base(MegMeetDriverKind.RtuOverTcp, "megmeet.rtu-over-tcp") { }
    internal MegMeetRtuOverTcpDriver(IHslMegMeetClientFactory factory) : base(MegMeetDriverKind.RtuOverTcp, "megmeet.rtu-over-tcp", factory) { }
}

public sealed class MegMeetRtuDriver : MegMeetDriverBase
{
    public MegMeetRtuDriver() : base(MegMeetDriverKind.RtuSerial, "megmeet.rtu") { }
    internal MegMeetRtuDriver(IHslMegMeetClientFactory factory) : base(MegMeetDriverKind.RtuSerial, "megmeet.rtu", factory) { }
}

public abstract class MegMeetDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly MegMeetDriverKind _kind;
    private readonly IHslMegMeetClientFactory _factory;

    private protected MegMeetDriverBase(MegMeetDriverKind kind, string driverId, IHslMegMeetClientFactory? factory = null)
    {
        _kind = kind;
        _factory = factory ?? new HslMegMeetClientFactory();
        Descriptor = new("megmeet", driverId, "1.0.0", [1],
            [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
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
        var options = MegMeetConnectionOptions.Parse(profile, _kind);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new MegMeetDriverException(result.ErrorCode ?? "MEGMEET_CONNECT_FAILED", result.ErrorMessage ?? "无法连接麦格米特 PLC", result.Retryable);
        }
        var endpoint = options.PortName ?? $"{options.Host}:{options.Port}";
        return new MegMeetConnectionSession(client, $"{endpoint} / Station={options.Station}");
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
