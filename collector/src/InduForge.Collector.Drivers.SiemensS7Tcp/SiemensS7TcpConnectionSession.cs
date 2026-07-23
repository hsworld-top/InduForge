using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp;

internal sealed class SiemensS7TcpConnectionSession : IIndustrialConnectionSession, IPointReaderSession
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslS7TcpClient _client;

    public SiemensS7TcpConnectionSession(IHslS7TcpClient client, string serverName)
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
            SiemensS7TcpAddress address;
            try
            {
                address = SiemensS7TcpAddress.Parse(point);
            }
            catch (SiemensS7TcpDriverException exception)
            {
                values[index] = Failure(point, exception.Code, exception.Message);
                continue;
            }

            // S7 会话始终复用同一 TCP 连接；单点读取失败只影响当前变量，不中断同批其他变量。
            var result = await _client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? Success(point, result.Value, readAt)
                : Failure(
                    point,
                    result.ErrorCode ?? "S7_READ_FAILED",
                    result.ErrorMessage ?? "Siemens S7 TCP 变量读取失败");
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
