using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Delta;

internal sealed class DeltaConnectionSession(IHslDeltaClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
{
    public bool IsConnected => client.IsConnected;
    public string? ServerName { get; } = serverName;
    public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
    {
        var readAt = DateTimeOffset.UtcNow;
        var values = new PointReadValue[request.Points.Count];
        for (var index = 0; index < request.Points.Count; index++)
        {
            var point = request.Points[index];
            DeltaAddress address;
            try { address = DeltaAddress.Parse(point); }
            catch (DeltaDriverException exception) { values[index] = Failure(point, exception.Code, exception.Message); continue; }
            // 所有帧格式都复用当前会话，单点异常不关闭连接，也不阻断同批其他变量。
            var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                : Failure(point, result.ErrorCode ?? "DELTA_READ_FAILED", result.ErrorMessage ?? "台达变量读取失败");
        }
        return new(values, []);
    }
    public ValueTask DisposeAsync() => client.DisposeAsync();
    private static PointReadValue Failure(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}
