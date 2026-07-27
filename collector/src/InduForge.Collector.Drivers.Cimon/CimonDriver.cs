using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Cimon;

public sealed class CimonDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly IHslCimonClientFactory _factory;
    public CimonDriver() : this(new HslCimonClientFactory()) { }
    internal CimonDriver(IHslCimonClientFactory factory) => _factory = factory;
    public DriverDescriptor Descriptor { get; } = new("cimon", "cimon.hmi-protocol", "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var options = CimonOptions.Parse(profile); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken); if (!result.Succeeded) { await client.DisposeAsync(); throw new CimonDriverException(result.ErrorCode ?? "CIMON_CONNECT_FAILED", result.ErrorMessage ?? "无法连接 Cimon HMI", result.Retryable); } return new Session(client, $"{options.Host}:{options.Port} / FrameNo={options.FrameNumber}"); }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
    private sealed class Session(IHslCimonClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken) { var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count]; for (var index = 0; index < request.Points.Count; index++) { var point = request.Points[index]; try { var address = CimonAddress.Parse(point); var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken); values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : new(point.Key, false, null, point.DataType, "Bad", null, null, result.ErrorCode, result.ErrorMessage); } catch (CimonDriverException exception) { values[index] = new(point.Key, false, null, point.DataType, "Bad", null, null, exception.Code, exception.Message); } } return new(values, []); }
        public ValueTask DisposeAsync() => client.DisposeAsync();
    }
}
