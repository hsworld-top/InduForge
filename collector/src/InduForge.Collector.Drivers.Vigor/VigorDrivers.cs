using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Vigor;
public sealed class VigorSerialOverTcpDriver : VigorDriverBase { public VigorSerialOverTcpDriver() : base(VigorDriverKind.SerialOverTcp, "vigor.serial-over-tcp") { } internal VigorSerialOverTcpDriver(IHslVigorClientFactory factory) : base(VigorDriverKind.SerialOverTcp, "vigor.serial-over-tcp", factory) { } }
public sealed class VigorSerialDriver : VigorDriverBase { public VigorSerialDriver() : base(VigorDriverKind.Serial, "vigor.serial") { } internal VigorSerialDriver(IHslVigorClientFactory factory) : base(VigorDriverKind.Serial, "vigor.serial", factory) { } }
public abstract class VigorDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly VigorDriverKind _kind; private readonly IHslVigorClientFactory _factory;
    private protected VigorDriverBase(VigorDriverKind kind, string id, IHslVigorClientFactory? factory = null) { _kind = kind; _factory = factory ?? new HslVigorClientFactory(); Descriptor = new("vigor", id, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]); }
    public DriverDescriptor Descriptor { get; }
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = VigorConnectionOptions.Parse(profile, _kind); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded) { await client.DisposeAsync().ConfigureAwait(false); throw new VigorDriverException(result.ErrorCode ?? "VIGOR_CONNECT_FAILED", result.ErrorMessage ?? "无法连接丰炜 PLC", result.Retryable); }
        return new VigorConnectionSession(client, $"{options.PortName ?? $"{options.Host}:{options.Port}"} / Station={options.Station}");
    }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false); }
}
internal sealed class VigorConnectionSession(IHslVigorClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
{
    public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
    public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
    {
        var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count];
        for (var i = 0; i < request.Points.Count; i++) { var point = request.Points[i]; VigorAddress address; try { address = VigorAddress.Parse(point); } catch (VigorDriverException ex) { values[i] = Fail(point, ex.Code, ex.Message); continue; }
            // 同批读取复用一条 VS 通信连接，单点错误不影响后续变量。
            var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false); values[i] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : Fail(point, result.ErrorCode ?? "VIGOR_READ_FAILED", result.ErrorMessage ?? "丰炜变量读取失败"); }
        return new(values, []);
    }
    public ValueTask DisposeAsync() => client.DisposeAsync(); private static PointReadValue Fail(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}
