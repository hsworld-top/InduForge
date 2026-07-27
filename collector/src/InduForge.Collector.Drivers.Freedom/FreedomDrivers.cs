using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Freedom;

public sealed class FreedomTcpDriver : FreedomDriverBase { public FreedomTcpDriver() : base("freedom.tcp", HslFreedomTransport.Tcp) { } internal FreedomTcpDriver(IHslFreedomClientFactory factory) : base("freedom.tcp", HslFreedomTransport.Tcp, factory) { } }
public sealed class FreedomUdpDriver : FreedomDriverBase { public FreedomUdpDriver() : base("freedom.udp", HslFreedomTransport.Udp) { } internal FreedomUdpDriver(IHslFreedomClientFactory factory) : base("freedom.udp", HslFreedomTransport.Udp, factory) { } }
public sealed class FreedomSerialDriver : FreedomDriverBase { public FreedomSerialDriver() : base("freedom.serial", HslFreedomTransport.Serial) { } internal FreedomSerialDriver(IHslFreedomClientFactory factory) : base("freedom.serial", HslFreedomTransport.Serial, factory) { } }

public abstract class FreedomDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslFreedomTransport _transport; private readonly IHslFreedomClientFactory _factory;
    private protected FreedomDriverBase(string driverId, HslFreedomTransport transport, IHslFreedomClientFactory? factory = null) { _transport = transport; _factory = factory ?? new HslFreedomClientFactory(); Descriptor = new("freedom", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]); }
    public DriverDescriptor Descriptor { get; }
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var options = FreedomOptions.Parse(profile, _transport); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken); if (!result.Succeeded) { await client.DisposeAsync(); throw new FreedomDriverException(result.ErrorCode ?? "FREEDOM_CONNECT_FAILED", result.ErrorMessage ?? "无法连接自由协议设备", result.Retryable); } return new Session(client, options.ServerName); }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
    private sealed class Session(IHslFreedomClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken) { var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count]; for (var index = 0; index < request.Points.Count; index++) { var point = request.Points[index]; try { var address = FreedomAddress.Parse(point); var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken); values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : new(point.Key, false, null, point.DataType, "Bad", null, null, result.ErrorCode, result.ErrorMessage); } catch (FreedomDriverException exception) { values[index] = new(point.Key, false, null, point.DataType, "Bad", null, null, exception.Code, exception.Message); } } return new(values, []); }
        public ValueTask DisposeAsync() => client.DisposeAsync();
    }
}
