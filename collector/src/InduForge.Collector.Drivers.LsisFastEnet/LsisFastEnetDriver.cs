using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.LsisFastEnet;
public sealed class LsisFastEnetDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly IHslLsisFastEnetClientFactory factory;
    public LsisFastEnetDriver() : this(new HslLsisFastEnetClientFactory()) { }
    internal LsisFastEnetDriver(IHslLsisFastEnetClientFactory factory) => this.factory = factory;
    public DriverDescriptor Descriptor { get; } = new("lsis", "lsis.fast-enet", "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var options = LsisFastEnetConnectionOptions.Parse(profile); var client = factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken); if (!result.Succeeded) { await client.DisposeAsync(); throw new LsisFastEnetDriverException(result.ErrorCode ?? "LSIS_FAST_ENET_CONNECT_FAILED", result.ErrorMessage ?? "无法连接 LSIS Fast Enet 设备", result.Retryable); } return new LsisFastEnetConnectionSession(client, $"{options.Host}:{options.Port} / {options.CpuType}"); }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
}
