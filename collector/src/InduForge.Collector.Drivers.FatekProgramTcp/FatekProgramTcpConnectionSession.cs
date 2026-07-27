using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.FatekProgramTcp;

internal sealed class FatekProgramConnectionSession(IHslFatekProgramClient client, string serverName)
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
            FatekProgramAddress address;
            try { address = FatekProgramAddress.Parse(point); }
            catch (FatekProgramDriverException exception) { values[index] = Failure(point, exception.Code, exception.Message); continue; }
            // 同一连接内连续读取全部变量，单点失败只标记当前变量，不中断同批后续读取。
            var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                : Failure(point, result.ErrorCode ?? "FATEK_PROGRAM_READ_FAILED", result.ErrorMessage ?? "永宏 PLC 地址读取失败");
        }
        return new(values, []);
    }

    public ValueTask DisposeAsync() => client.DisposeAsync();
    private static PointReadValue Failure(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}
