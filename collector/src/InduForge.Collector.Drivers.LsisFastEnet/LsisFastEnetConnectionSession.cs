using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.LsisFastEnet;
internal sealed class LsisFastEnetConnectionSession(IHslLsisFastEnetClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
{
    public bool IsConnected => client.IsConnected;
    public string? ServerName { get; } = serverName;
    public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
    {
        var now = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count];
        for (var index = 0; index < request.Points.Count; index++)
        {
            var point = request.Points[index]; LsisFastEnetAddress address;
            try { address = LsisFastEnetAddress.Parse(point); } catch (LsisFastEnetDriverException error) { values[index] = Fail(point, error.Code, error.Message); continue; }
            // 同一 Fast Enet 会话逐点读取，避免重复握手并隔离单点错误。
            var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", now, now, null, null) : Fail(point, result.ErrorCode ?? "LSIS_FAST_ENET_READ_FAILED", result.ErrorMessage ?? "LSIS 地址读取失败");
        }
        return new(values, []);
    }
    public ValueTask DisposeAsync() => client.DisposeAsync();
    private static PointReadValue Fail(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}
