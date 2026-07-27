using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Iec104;

internal sealed class Iec104ConnectionSession : IIndustrialConnectionSession, IPointReaderSession
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslIec104Client _client;

    public Iec104ConnectionSession(IHslIec104Client client, string serverName)
    {
        _client = client;
        ServerName = serverName;
    }

    public bool IsConnected => _client.IsConnected;

    public string? ServerName { get; }

    public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
    {
        if (request.Points.Count == 0) return new ReadResult([], NoDiagnostics);

        var values = new PointReadValue[request.Points.Count];
        var validIndexes = new List<int>(request.Points.Count);
        var validAddresses = new List<Iec104Address>(request.Points.Count);
        for (var index = 0; index < request.Points.Count; index++)
        {
            try
            {
                validAddresses.Add(Iec104Address.Parse(request.Points[index]));
                validIndexes.Add(index);
            }
            catch (Iec104DriverException exception)
            {
                values[index] = Failure(request.Points[index], exception.Code, exception.Message);
            }
        }

        if (validAddresses.Count > 0)
        {
            // IEC 104 通过一次总召唤刷新整批缓存，不能按点循环发送网络请求。
            var readAt = DateTimeOffset.UtcNow;
            var results = await _client.ReadAsync(
                validAddresses.Select(address => address.ToHslRequest()).ToArray(),
                cancellationToken).ConfigureAwait(false);
            for (var resultIndex = 0; resultIndex < validIndexes.Count; resultIndex++)
            {
                var pointIndex = validIndexes[resultIndex];
                var point = request.Points[pointIndex];
                if (resultIndex >= results.Count)
                {
                    values[pointIndex] = Failure(point, "IEC104_RESULT_MISSING", "IEC 104 适配器未返回该信息对象结果");
                    continue;
                }

                var result = results[resultIndex];
                values[pointIndex] = result.Succeeded
                    ? Success(point, result.Value, result.Quality ?? byte.MaxValue, result.SourceTimestamp, readAt)
                    : Failure(
                        point,
                        result.ErrorCode ?? "IEC104_READ_FAILED",
                        result.ErrorMessage ?? "IEC 104 信息对象读取失败");
            }
        }

        return new ReadResult(values, NoDiagnostics);
    }

    public ValueTask DisposeAsync() => _client.DisposeAsync();

    private static PointReadValue Success(
        PointReadRequest point,
        object? value,
        byte quality,
        DateTimeOffset? sourceTimestamp,
        DateTimeOffset readAt) => new(
        point.Key,
        true,
        value,
        point.DataType,
        quality == 0 ? "Good" : "Bad",
        sourceTimestamp ?? readAt,
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
