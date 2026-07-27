using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MitsubishiMc3ETcp;

internal sealed class MitsubishiMc3ETcpConnectionSession : IIndustrialConnectionSession, IPointReaderSession
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslMelsecMcClient _client;

    public MitsubishiMc3ETcpConnectionSession(IHslMelsecMcClient client, string serverName)
    {
        _client = client;
        ServerName = serverName;
    }

    public bool IsConnected => _client.IsConnected;

    public string? ServerName { get; }

    public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
    {
        if (request.Points.Count == 0)
        {
            return new ReadResult([], NoDiagnostics);
        }

        var readAt = DateTimeOffset.UtcNow;
        var values = new PointReadValue[request.Points.Count];
        for (var index = 0; index < request.Points.Count; index++)
        {
            var point = request.Points[index];
            MitsubishiMc3ETcpAddress address;
            try
            {
                address = MitsubishiMc3ETcpAddress.Parse(point);
            }
            catch (MitsubishiMc3ETcpDriverException exception)
            {
                values[index] = Failure(point, exception.Code, exception.Message);
                continue;
            }

            // MC 会话始终复用同一 TCP 连接；单点读取失败只影响当前变量，不中断同批其他变量。
            var result = await _client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? Success(point, result.Value, readAt)
                : Failure(
                    point,
                    result.ErrorCode ?? "MELSEC_MC_READ_FAILED",
                    result.ErrorMessage ?? "Mitsubishi MC 3E TCP 变量读取失败");
        }

        return new ReadResult(values, NoDiagnostics);
    }

    public ValueTask DisposeAsync() => _client.DisposeAsync();

    private static PointReadValue Success(PointReadRequest point, object? value, DateTimeOffset readAt) => new(
        point.Key,
        true,
        value,
        point.DataType,
        "Good",
        readAt,
        readAt,
        null,
        null);

    private static PointReadValue Failure(PointReadRequest point, string errorCode, string errorMessage) => new(
        point.Key,
        false,
        null,
        point.DataType,
        "Bad",
        null,
        null,
        errorCode,
        errorMessage);
}
