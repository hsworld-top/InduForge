using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Keyence;

internal sealed class KeyenceConnectionSession(IHslKeyenceTcpClient client, KeyenceDriverKind kind, string serverName)
    : IIndustrialConnectionSession, IPointReaderSession
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
            KeyenceAddress address;
            try { address = KeyenceAddress.Parse(point, kind); }
            catch (KeyenceDriverException exception) { values[index] = Failure(point, exception.Code, exception.Message); continue; }
            // 同一连接内串行执行读取，保证设备报文有序；单点失败只影响当前变量。
            var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                : Failure(point, result.ErrorCode ?? "KEYENCE_READ_FAILED", result.ErrorMessage ?? "基恩士变量读取失败");
        }
        return new(values, []);
    }

    public ValueTask DisposeAsync() => client.DisposeAsync();
    private static PointReadValue Failure(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}
