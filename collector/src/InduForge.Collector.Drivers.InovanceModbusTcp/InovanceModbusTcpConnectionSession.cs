using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InovanceModbusTcp;

internal sealed class InovanceModbusTcpConnectionSession(IHslInovanceModbusTcpClient client, string serverName)
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
            InovanceModbusTcpAddress address;
            try { address = InovanceModbusTcpAddress.Parse(point); }
            catch (InovanceModbusTcpDriverException exception)
            {
                values[index] = Failure(point, exception.Code, exception.Message);
                continue;
            }

            // 汇川地址由客户端按 PLC 系列转换；同批变量复用长连接并隔离单点失败。
            var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? new PointReadValue(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                : Failure(point, result.ErrorCode ?? "INOVANCE_MODBUS_TCP_READ_FAILED", result.ErrorMessage ?? "汇川 PLC 地址读取失败");
        }
        return new ReadResult(values, NoDiagnostics);
    }

    public ValueTask DisposeAsync() => client.DisposeAsync();

    private static PointReadValue Failure(PointReadRequest point, string code, string message) =>
        new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}
