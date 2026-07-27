using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Xinje;

public sealed class XinjeTcpDriver : XinjeDriverBase { public XinjeTcpDriver() : base(XinjeDriverKind.Tcp, "xinje.tcp") { } internal XinjeTcpDriver(IHslXinjeClientFactory factory) : base(XinjeDriverKind.Tcp, "xinje.tcp", factory) { } }
public sealed class XinjeRtuOverTcpDriver : XinjeDriverBase { public XinjeRtuOverTcpDriver() : base(XinjeDriverKind.RtuOverTcp, "xinje.rtu-over-tcp") { } internal XinjeRtuOverTcpDriver(IHslXinjeClientFactory factory) : base(XinjeDriverKind.RtuOverTcp, "xinje.rtu-over-tcp", factory) { } }
public sealed class XinjeRtuDriver : XinjeDriverBase { public XinjeRtuDriver() : base(XinjeDriverKind.RtuSerial, "xinje.rtu") { } internal XinjeRtuDriver(IHslXinjeClientFactory factory) : base(XinjeDriverKind.RtuSerial, "xinje.rtu", factory) { } }
public sealed class XinjeInternalTcpDriver : XinjeDriverBase { public XinjeInternalTcpDriver() : base(XinjeDriverKind.InternalTcp, "xinje.internal-tcp") { } internal XinjeInternalTcpDriver(IHslXinjeClientFactory factory) : base(XinjeDriverKind.InternalTcp, "xinje.internal-tcp", factory) { } }

public abstract class XinjeDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly XinjeDriverKind _kind; private readonly IHslXinjeClientFactory _factory;
    private protected XinjeDriverBase(XinjeDriverKind kind, string driverId, IHslXinjeClientFactory? factory = null)
    {
        _kind = kind; _factory = factory ?? new HslXinjeClientFactory();
        Descriptor = new("xinje", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    }
    public DriverDescriptor Descriptor { get; }
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false); watch.Stop();
        return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = XinjeConnectionOptions.Parse(profile, _kind); var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded) { await client.DisposeAsync().ConfigureAwait(false); throw new XinjeDriverException(result.ErrorCode ?? "XINJE_CONNECT_FAILED", result.ErrorMessage ?? "无法连接信捷 PLC", result.Retryable); }
        var endpoint = options.PortName ?? $"{options.Host}:{options.Port}";
        var series = _kind == XinjeDriverKind.InternalTcp ? "Internal" : options.Series.ToString();
        return new XinjeConnectionSession(client, options, $"{endpoint} / {series} / Station={options.Station}");
    }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
