using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.PanasonicMewtocolTcp;

internal sealed class PanasonicMewtocolTcpConnectionSession : IIndustrialConnectionSession, IPointReaderSession
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslPanasonicMewtocolClient _client;

    public PanasonicMewtocolTcpConnectionSession(IHslPanasonicMewtocolClient client, string serverName)
    {
        _client = client;
        ServerName = serverName;
    }

    public bool IsConnected => _client.IsConnected;
    public string? ServerName { get; }

    public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
    {
        var readAt = DateTimeOffset.UtcNow;
        var values = new PointReadValue[request.Points.Count];
        for (var index = 0; index < request.Points.Count; index++)
        {
            var point = request.Points[index];
            PanasonicMewtocolTcpAddress address;
            try { address = PanasonicMewtocolTcpAddress.Parse(point); }
            catch (PanasonicMewtocolTcpDriverException exception)
            {
                values[index] = Failure(point, exception.Code, exception.Message);
                continue;
            }

            // 复用同一 TCP 会话逐点读取，单地址失败不影响同批其他变量。
            var result = await _client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? new PointReadValue(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                : Failure(point, result.ErrorCode ?? "PANASONIC_MEWTOCOL_READ_FAILED", result.ErrorMessage ?? "松下 Mewtocol 地址读取失败");
        }
        return new ReadResult(values, NoDiagnostics);
    }

    public ValueTask DisposeAsync() => _client.DisposeAsync();
    private static PointReadValue Failure(PointReadRequest point, string code, string message) =>
        new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}
