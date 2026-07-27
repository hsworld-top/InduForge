using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusTcp;

public sealed class ModbusRtuOverTcpDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly ModbusNetworkDriverCore _core;

    public ModbusRtuOverTcpDriver() : this(new HslModbusTcpClientFactory()) { }

    internal ModbusRtuOverTcpDriver(IHslModbusTcpClientFactory clientFactory) =>
        _core = new("modbus.rtu-over-tcp", HslModbusNetworkProtocol.RtuOverTcp, "Modbus RTU over TCP", clientFactory);

    public DriverDescriptor Descriptor => _core.Descriptor;
    public Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) => _core.TestConnectionAsync(profile, cancellationToken);
    public Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) => _core.OpenSessionAsync(profile, cancellationToken);
    public Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) => _core.ReadAsync(profile, request, cancellationToken);
}

public sealed class ModbusAsciiOverTcpDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly ModbusNetworkDriverCore _core;

    public ModbusAsciiOverTcpDriver() : this(new HslModbusTcpClientFactory()) { }

    internal ModbusAsciiOverTcpDriver(IHslModbusTcpClientFactory clientFactory) =>
        _core = new("modbus.ascii-over-tcp", HslModbusNetworkProtocol.AsciiOverTcp, "Modbus ASCII over TCP", clientFactory);

    public DriverDescriptor Descriptor => _core.Descriptor;
    public Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) => _core.TestConnectionAsync(profile, cancellationToken);
    public Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) => _core.OpenSessionAsync(profile, cancellationToken);
    public Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) => _core.ReadAsync(profile, request, cancellationToken);
}

public sealed class ModbusUdpDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly ModbusNetworkDriverCore _core;

    public ModbusUdpDriver() : this(new HslModbusTcpClientFactory()) { }

    internal ModbusUdpDriver(IHslModbusTcpClientFactory clientFactory) =>
        _core = new("modbus.udp", HslModbusNetworkProtocol.Udp, "Modbus UDP", clientFactory);

    public DriverDescriptor Descriptor => _core.Descriptor;
    public Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) => _core.TestConnectionAsync(profile, cancellationToken);
    public Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) => _core.OpenSessionAsync(profile, cancellationToken);
    public Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) => _core.ReadAsync(profile, request, cancellationToken);
}

internal sealed class ModbusNetworkDriverCore(
    string driverId,
    HslModbusNetworkProtocol protocol,
    string displayName,
    IHslModbusTcpClientFactory clientFactory)
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];

    public DriverDescriptor Descriptor { get; } = new(
        "modbus",
        driverId,
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
        var options = ModbusTcpConnectionOptions.Parse(profile, protocol);
        var client = clientFactory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new ModbusTcpDriverException(
                result.ErrorCode ?? "MODBUS_CONNECT_FAILED",
                result.ErrorMessage ?? $"无法连接 {displayName} 设备",
                result.Retryable);
        }
        return new ModbusTcpConnectionSession(client, $"{options.Host}:{options.Port}");
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
