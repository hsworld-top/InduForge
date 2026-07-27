using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp;

public sealed class SiemensPpiDriver : SiemensVariantDriverBase
{
    public SiemensPpiDriver() : base("siemens.ppi", HslSiemensVariantProtocol.PpiSerial) { }
    internal SiemensPpiDriver(IHslSiemensVariantClientFactory factory) : base("siemens.ppi", HslSiemensVariantProtocol.PpiSerial, factory) { }
}

public sealed class SiemensPpiOverTcpDriver : SiemensVariantDriverBase
{
    public SiemensPpiOverTcpDriver() : base("siemens.ppi-over-tcp", HslSiemensVariantProtocol.PpiOverTcp) { }
    internal SiemensPpiOverTcpDriver(IHslSiemensVariantClientFactory factory) : base("siemens.ppi-over-tcp", HslSiemensVariantProtocol.PpiOverTcp, factory) { }
}

public sealed class SiemensMpiDriver : SiemensVariantDriverBase
{
    public SiemensMpiDriver() : base("siemens.mpi", HslSiemensVariantProtocol.MpiSerial) { }
    internal SiemensMpiDriver(IHslSiemensVariantClientFactory factory) : base("siemens.mpi", HslSiemensVariantProtocol.MpiSerial, factory) { }
}

public sealed class SiemensFetchWriteDriver : SiemensVariantDriverBase
{
    public SiemensFetchWriteDriver() : base("siemens.fetch-write", HslSiemensVariantProtocol.FetchWrite) { }
    internal SiemensFetchWriteDriver(IHslSiemensVariantClientFactory factory) : base("siemens.fetch-write", HslSiemensVariantProtocol.FetchWrite, factory) { }
}

public sealed class SiemensWebApiDriver : SiemensVariantDriverBase
{
    public SiemensWebApiDriver() : base("siemens.web-api", HslSiemensVariantProtocol.WebApi) { }
    internal SiemensWebApiDriver(IHslSiemensVariantClientFactory factory) : base("siemens.web-api", HslSiemensVariantProtocol.WebApi, factory) { }
}

public sealed class SiemensS7PlusDriver : SiemensVariantDriverBase
{
    public SiemensS7PlusDriver() : base("siemens.s7-plus", HslSiemensVariantProtocol.S7Plus) { }
    internal SiemensS7PlusDriver(IHslSiemensVariantClientFactory factory) : base("siemens.s7-plus", HslSiemensVariantProtocol.S7Plus, factory) { }
}

public abstract class SiemensVariantDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslSiemensVariantProtocol _protocol;
    private readonly IHslSiemensVariantClientFactory _factory;

    private protected SiemensVariantDriverBase(
        string driverId,
        HslSiemensVariantProtocol protocol,
        IHslSiemensVariantClientFactory? factory = null)
    {
        _protocol = protocol;
        _factory = factory ?? new HslSiemensVariantClientFactory();
        Descriptor = new(
            "siemens",
            driverId,
            "1.0.0",
            [1],
            [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    }

    public DriverDescriptor Descriptor { get; }

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        watch.Stop();
        return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = SiemensVariantOptions.Parse(profile, _protocol);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new SiemensVariantDriverException(
                result.ErrorCode ?? "SIEMENS_CONNECT_FAILED",
                result.ErrorMessage ?? "无法连接 Siemens 设备",
                result.Retryable);
        }

        return new Session(client, options.ServerName, _protocol);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }

    private sealed class Session(
        IHslSiemensVariantClient client,
        string serverName,
        HslSiemensVariantProtocol protocol) : IIndustrialConnectionSession, IPointReaderSession
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
                SiemensVariantAddress address;
                try
                {
                    address = SiemensVariantAddress.Parse(point, protocol);
                }
                catch (SiemensVariantDriverException exception)
                {
                    values[index] = Failure(point, exception.Code, exception.Message);
                    continue;
                }

                // 所有 Siemens 变体复用已打开连接；单点地址或读取失败只影响当前变量。
                var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
                values[index] = result.Succeeded
                    ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                    : Failure(point, result.ErrorCode ?? "SIEMENS_READ_FAILED", result.ErrorMessage ?? "Siemens 变量读取失败");
            }
            return new(values, []);
        }

        public ValueTask DisposeAsync() => client.DisposeAsync();

        private static PointReadValue Failure(PointReadRequest point, string code, string message) =>
            new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}
