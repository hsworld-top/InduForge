using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.GeSrtpTcp;

internal sealed class GeSrtpTcpConnectionSession(IHslGeSrtpClient client, string serverName)
    : IIndustrialConnectionSession, IPointReaderSession
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];

    public bool IsConnected => client.IsConnected;
    public string? ServerName { get; } = serverName;

    public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
    {
        var readAt = DateTimeOffset.UtcNow;
        var values = new PointReadValue[request.Points.Count];
        for (var index = 0; index < request.Points.Count; index++)
        {
            var point = request.Points[index];
            GeSrtpTcpAddress address;
            try { address = GeSrtpTcpAddress.Parse(point); }
            catch (GeSrtpTcpDriverException exception)
            {
                values[index] = Failure(point, exception.Code, exception.Message);
                continue;
            }

            // 复用同一 SRTP 长连接逐点读取；单地址失败只标记当前变量，不中断同批任务。
            var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? new PointReadValue(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                : Failure(point, result.ErrorCode ?? "GE_SRTP_READ_FAILED", result.ErrorMessage ?? "GE SRTP 地址读取失败");
        }
        return new ReadResult(values, NoDiagnostics);
    }

    public ValueTask DisposeAsync() => client.DisposeAsync();

    private static PointReadValue Failure(PointReadRequest point, string code, string message) =>
        new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}
