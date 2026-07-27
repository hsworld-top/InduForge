using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MegMeet;

internal sealed class MegMeetConnectionSession(IHslMegMeetClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
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
            MegMeetAddress address;
            try { address = MegMeetAddress.Parse(point); }
            catch (MegMeetDriverException exception) { values[index] = Failure(point, exception.Code, exception.Message); continue; }

            // 同一连接内串行读取并隔离单点失败，避免单个非法地址中断整批调试任务。
            var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                : Failure(point, result.ErrorCode ?? "MEGMEET_READ_FAILED", result.ErrorMessage ?? "麦格米特变量读取失败");
        }
        return new(values, []);
    }

    public ValueTask DisposeAsync() => client.DisposeAsync();
    private static PointReadValue Failure(PointReadRequest point, string code, string message) =>
        new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}
