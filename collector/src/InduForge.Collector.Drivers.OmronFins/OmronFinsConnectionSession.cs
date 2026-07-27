using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins;

internal sealed class OmronFinsConnectionSession : IIndustrialConnectionSession, IPointReaderSession
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslOmronFinsClient _client;

    public OmronFinsConnectionSession(IHslOmronFinsClient client, string serverName)
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
            OmronFinsAddress address;
            try
            {
                address = OmronFinsAddress.Parse(point);
            }
            catch (OmronFinsDriverException exception)
            {
                values[index] = Failure(point, exception.Code, exception.Message);
                continue;
            }

            // 同一 FINS 客户端串行复用连接或 UDP 配置，单点异常只标记当前变量。
            var result = await _client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? Success(point, result.Value, readAt)
                : Failure(
                    point,
                    result.ErrorCode ?? "OMRON_FINS_READ_FAILED",
                    result.ErrorMessage ?? "Omron FINS 变量读取失败");
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
