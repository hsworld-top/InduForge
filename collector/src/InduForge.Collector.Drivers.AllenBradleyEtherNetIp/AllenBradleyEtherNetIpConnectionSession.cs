using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.AllenBradleyEtherNetIp;

internal sealed class AllenBradleyEtherNetIpConnectionSession : IIndustrialConnectionSession, IPointReaderSession
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslAllenBradleyClient _client;
    private readonly AllenBradleyAddressKind _addressKind;

    public AllenBradleyEtherNetIpConnectionSession(
        IHslAllenBradleyClient client,
        string serverName,
        AllenBradleyAddressKind addressKind = AllenBradleyAddressKind.LogixTag)
    {
        _client = client;
        ServerName = serverName;
        _addressKind = addressKind;
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
            AllenBradleyEtherNetIpAddress address;
            try
            {
                address = AllenBradleyEtherNetIpAddress.Parse(point, _addressKind);
            }
            catch (AllenBradleyEtherNetIpDriverException exception)
            {
                values[index] = Failure(point, exception.Code, exception.Message);
                continue;
            }

            // Logix 标签按同一 EtherNet/IP 会话逐点读取，保留单标签失败隔离。
            var result = await _client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded
                ? Success(point, result.Value, readAt)
                : Failure(
                    point,
                    result.ErrorCode ?? "ALLEN_BRADLEY_READ_FAILED",
                    result.ErrorMessage ?? "Allen-Bradley 标签读取失败");
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
