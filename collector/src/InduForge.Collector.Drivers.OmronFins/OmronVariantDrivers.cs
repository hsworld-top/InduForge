using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins;

public sealed class OmronCipDriver : OmronVariantDriverBase { public OmronCipDriver() : base("omron.cip", HslOmronVariantProtocol.Cip, false) { } }
public sealed class OmronConnectedCipDriver : OmronVariantDriverBase { public OmronConnectedCipDriver() : base("omron.connected-cip", HslOmronVariantProtocol.ConnectedCip, false) { } }
public sealed class OmronHostLinkDriver : OmronVariantDriverBase { public OmronHostLinkDriver() : base("omron.hostlink", HslOmronVariantProtocol.HostLink, true) { } }
public sealed class OmronHostLinkOverTcpDriver : OmronVariantDriverBase { public OmronHostLinkOverTcpDriver() : base("omron.hostlink-over-tcp", HslOmronVariantProtocol.HostLinkOverTcp, true) { } }
public sealed class OmronHostLinkCModeDriver : OmronVariantDriverBase { public OmronHostLinkCModeDriver() : base("omron.hostlink-cmode", HslOmronVariantProtocol.HostLinkCMode, true) { } }
public sealed class OmronHostLinkCModeOverTcpDriver : OmronVariantDriverBase { public OmronHostLinkCModeOverTcpDriver() : base("omron.hostlink-cmode-over-tcp", HslOmronVariantProtocol.HostLinkCModeOverTcp, true) { } }

public abstract class OmronVariantDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslOmronVariantProtocol _protocol;
    private readonly bool _cMode;
    private readonly IHslOmronVariantClientFactory _factory;
    private protected OmronVariantDriverBase(string driverId, HslOmronVariantProtocol protocol, bool cMode, IHslOmronVariantClientFactory? factory = null)
    {
        _protocol = protocol; _cMode = cMode; _factory = factory ?? new HslOmronVariantClientFactory();
        Descriptor = new("omron", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    }
    public DriverDescriptor Descriptor { get; }
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = OmronVariantOptions.Parse(profile, _protocol); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded) { await client.DisposeAsync(); throw new OmronFinsDriverException(result.ErrorCode ?? "OMRON_CONNECT_FAILED", result.ErrorMessage ?? "无法连接欧姆龙设备", result.Retryable); }
        return new Session(client, options.ServerName, _cMode);
    }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
    private sealed class Session(IHslOmronVariantClient client, string serverName, bool cMode) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
        {
            var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count];
            for (var index = 0; index < request.Points.Count; index++) { var point = request.Points[index]; try { var address = OmronVariantAddress.Parse(point, cMode); var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken); values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : new(point.Key, false, null, point.DataType, "Bad", null, null, result.ErrorCode, result.ErrorMessage); } catch (OmronFinsDriverException exception) { values[index] = new(point.Key, false, null, point.DataType, "Bad", null, null, exception.Code, exception.Message); } }
            return new(values, []);
        }
        public ValueTask DisposeAsync() => client.DisposeAsync();
    }
}
