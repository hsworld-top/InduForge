using System.Diagnostics;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OpcUa;

public sealed class OpcUaDriver : IIndustrialDriver, IConnectionSessionDriver, IDeviceBrowser, IPointReader
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IOpcUaConnectionFactory _connectionFactory;

    public OpcUaDriver()
        : this(new OpcUaConnectionFactory(new OpcUaDriverOptions()))
    {
    }

    internal OpcUaDriver(IOpcUaConnectionFactory connectionFactory)
    {
        _connectionFactory = connectionFactory;
    }

    public DriverDescriptor Descriptor { get; } = new(
        ProtocolFamily: "opcua",
        DriverId: "opcua.standard",
        DriverVersion: "1.0.0",
        SchemaVersions: [1],
        Operations:
        [
            DriverOperations.ConnectionTest,
            DriverOperations.ConnectionOpen,
            DriverOperations.ConnectionClose,
            DriverOperations.DeviceBrowse,
            DriverOperations.PointRead,
        ]);

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(
        ConnectionProfile profile,
        CancellationToken cancellationToken)
    {
        var session = await _connectionFactory.ConnectAsync(profile, cancellationToken).ConfigureAwait(false);
        return new OpcUaConnectionSession(session);
    }

    public async Task<ConnectionTestResult> TestConnectionAsync(
        ConnectionProfile profile,
        CancellationToken cancellationToken)
    {
        var stopwatch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        stopwatch.Stop();

        return new ConnectionTestResult(
            Connected: session.IsConnected,
            Elapsed: stopwatch.Elapsed,
            ServerName: session.ServerName,
            Diagnostics: NoDiagnostics);
    }

    public async Task<BrowseResult> BrowseAsync(
        ConnectionProfile profile,
        BrowseRequest request,
        CancellationToken cancellationToken)
    {
        // 请求能力校验必须先于网络连接，避免无效请求被连接状态掩盖。
        if (request.MaxDepth != 1)
        {
            throw new OpcUaDriverException(
                "OPCUA_BROWSE_DEPTH_UNSUPPORTED",
                "当前版本只支持浏览指定父节点的直接子节点",
                retryable: false);
        }

        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IDeviceBrowserSession)session).BrowseAsync(request, cancellationToken).ConfigureAwait(false);
    }

    public async Task<ReadResult> ReadAsync(
        ConnectionProfile profile,
        ReadRequest request,
        CancellationToken cancellationToken)
    {
        if (request.Points.Count == 0)
        {
            return new ReadResult([], NoDiagnostics);
        }

        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
