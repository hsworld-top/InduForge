using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Xinje;
internal sealed class XinjeConnectionSession(IHslXinjeClient client, XinjeConnectionOptions options, string serverName) : IIndustrialConnectionSession, IPointReaderSession
{
    public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
    public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
    {
        var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count];
        for (var index = 0; index < request.Points.Count; index++)
        {
            var point = request.Points[index]; XinjeAddress address;
            try { address = XinjeAddress.Parse(point, options); } catch (XinjeDriverException exception) { values[index] = Failure(point, exception.Code, exception.Message); continue; }
            // 当前连接内保持报文有序，读取失败只标记当前变量。
            var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : Failure(point, result.ErrorCode ?? "XINJE_READ_FAILED", result.ErrorMessage ?? "信捷变量读取失败");
        }
        return new(values, []);
    }
    public ValueTask DisposeAsync() => client.DisposeAsync();
    private static PointReadValue Failure(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}
