using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusRtu;

public sealed class ModbusAsciiDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslModbusRtuClientFactory _clientFactory;

    public ModbusAsciiDriver() : this(new HslModbusRtuClientFactory()) { }

    internal ModbusAsciiDriver(IHslModbusRtuClientFactory clientFactory) => _clientFactory = clientFactory;

    public DriverDescriptor Descriptor { get; } = new(
        "modbus",
        "modbus.ascii",
        "1.0.0",
        [1],
        [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var stopwatch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        stopwatch.Stop();
        return new(session.IsConnected, stopwatch.Elapsed, session.ServerName, NoDiagnostics);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = ModbusRtuConnectionOptions.Parse(profile, HslModbusSerialProtocol.Ascii);
        var client = _clientFactory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new ModbusRtuDriverException(
                result.ErrorCode ?? "MODBUS_CONNECT_FAILED",
                result.ErrorMessage ?? "无法连接 Modbus ASCII 设备",
                result.Retryable);
        }
        return new ModbusRtuConnectionSession(client, options.PortName);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
