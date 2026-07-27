using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.BeckhoffAds;

internal sealed class BeckhoffAdsConnectionSession : IIndustrialConnectionSession, IPointReaderSession
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslBeckhoffAdsClient _client;

    public BeckhoffAdsConnectionSession(IHslBeckhoffAdsClient client, string serverName)
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
            BeckhoffAdsAddress address;
            try
            {
                address = BeckhoffAdsAddress.Parse(point);
            }
            catch (BeckhoffAdsDriverException exception)
            {
                values[index] = Failure(point, exception.Code, exception.Message);
                continue;
            }

            // 同一连接内逐点读取并隔离单点错误，避免一个无效变量中断整批调试任务。
            var result = await _client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? Success(point, result.Value, readAt)
                : Failure(
                    point,
                    result.ErrorCode ?? "BECKHOFF_ADS_READ_FAILED",
                    result.ErrorMessage ?? "倍福 ADS 地址读取失败");
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
