using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Fuji;

public sealed class FujiCommandSettingTypeTcpDriver : FujiDriverBase { public FujiCommandSettingTypeTcpDriver() : base(FujiDriverKind.CommandSettingTypeTcp, "fuji.command-setting-tcp") { } internal FujiCommandSettingTypeTcpDriver(IHslFujiClientFactory factory) : base(FujiDriverKind.CommandSettingTypeTcp, "fuji.command-setting-tcp", factory) { } }
public sealed class FujiSphTcpDriver : FujiDriverBase { public FujiSphTcpDriver() : base(FujiDriverKind.SphTcp, "fuji.sph-tcp") { } internal FujiSphTcpDriver(IHslFujiClientFactory factory) : base(FujiDriverKind.SphTcp, "fuji.sph-tcp", factory) { } }
public sealed class FujiSpbOverTcpDriver : FujiDriverBase { public FujiSpbOverTcpDriver() : base(FujiDriverKind.SpbOverTcp, "fuji.spb-over-tcp") { } internal FujiSpbOverTcpDriver(IHslFujiClientFactory factory) : base(FujiDriverKind.SpbOverTcp, "fuji.spb-over-tcp", factory) { } }
public sealed class FujiSpbDriver : FujiDriverBase { public FujiSpbDriver() : base(FujiDriverKind.SpbSerial, "fuji.spb") { } internal FujiSpbDriver(IHslFujiClientFactory factory) : base(FujiDriverKind.SpbSerial, "fuji.spb", factory) { } }

public abstract class FujiDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly FujiDriverKind _kind; private readonly IHslFujiClientFactory _factory;
    private protected FujiDriverBase(FujiDriverKind kind, string driverId, IHslFujiClientFactory? factory = null)
    {
        _kind = kind; _factory = factory ?? new HslFujiClientFactory();
        Descriptor = new("fuji", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    }
    public DriverDescriptor Descriptor { get; }
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false); watch.Stop();
        return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = FujiConnectionOptions.Parse(profile, _kind); var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded) { await client.DisposeAsync().ConfigureAwait(false); throw new FujiDriverException(result.ErrorCode ?? "FUJI_CONNECT_FAILED", result.ErrorMessage ?? "无法连接富士 PLC", result.Retryable); }
        var endpoint = options.PortName ?? $"{options.Host}:{options.Port}";
        var identity = _kind == FujiDriverKind.SphTcp ? $"ConnectionId={options.ConnectionId}" : _kind is FujiDriverKind.SpbOverTcp or FujiDriverKind.SpbSerial ? $"Station={options.Station}" : $"DataSwap={options.DataSwap}";
        return new FujiConnectionSession(client, _kind, $"{endpoint} / {identity}");
    }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
