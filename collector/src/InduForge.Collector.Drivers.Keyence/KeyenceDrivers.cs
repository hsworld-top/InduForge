using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Keyence;

public sealed class KeyenceMc3ETcpDriver : KeyenceTcpDriverBase
{
    public KeyenceMc3ETcpDriver() : base(KeyenceDriverKind.Mc3E, "keyence.mc-3e-tcp") { }
    internal KeyenceMc3ETcpDriver(IHslKeyenceTcpClientFactory factory) : base(KeyenceDriverKind.Mc3E, "keyence.mc-3e-tcp", factory) { }
}

public sealed class KeyenceKvOldTcpDriver : KeyenceTcpDriverBase
{
    public KeyenceKvOldTcpDriver() : base(KeyenceDriverKind.KvOld, "keyence.kv-old-tcp") { }
    internal KeyenceKvOldTcpDriver(IHslKeyenceTcpClientFactory factory) : base(KeyenceDriverKind.KvOld, "keyence.kv-old-tcp", factory) { }
}

public sealed class KeyenceNanoTcpDriver : KeyenceTcpDriverBase
{
    public KeyenceNanoTcpDriver() : base(KeyenceDriverKind.Nano, "keyence.nano-tcp") { }
    internal KeyenceNanoTcpDriver(IHslKeyenceTcpClientFactory factory) : base(KeyenceDriverKind.Nano, "keyence.nano-tcp", factory) { }
}

public sealed class KeyenceMcAsciiTcpDriver : KeyenceTcpDriverBase
{
    public KeyenceMcAsciiTcpDriver() : base(KeyenceDriverKind.McAscii, "keyence.mc-ascii-tcp") { }
}

public sealed class KeyenceNanoSerialDriver : KeyenceTcpDriverBase
{
    public KeyenceNanoSerialDriver() : base(KeyenceDriverKind.NanoSerial, "keyence.nano-serial") { }
}

public sealed class KeyenceNanoSerialOverTcpDriver : KeyenceTcpDriverBase
{
    public KeyenceNanoSerialOverTcpDriver() : base(KeyenceDriverKind.NanoSerialOverTcp, "keyence.nano-serial-over-tcp") { }
}

public abstract class KeyenceTcpDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly KeyenceDriverKind _kind;
    private readonly IHslKeyenceTcpClientFactory _factory;

    private protected KeyenceTcpDriverBase(KeyenceDriverKind kind, string driverId, IHslKeyenceTcpClientFactory? factory = null)
    {
        _kind = kind;
        _factory = factory ?? new HslKeyenceTcpClientFactory();
        Descriptor = new("keyence", driverId, "1.0.0", [1],
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
        var options = KeyenceConnectionOptions.Parse(profile, _kind);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new KeyenceDriverException(result.ErrorCode ?? "KEYENCE_CONNECT_FAILED", result.ErrorMessage ?? "无法连接基恩士设备", result.Retryable);
        }
        var suffix = _kind switch
        {
            KeyenceDriverKind.Mc3E or KeyenceDriverKind.McAscii => options.Protocol == HslKeyenceProtocol.McAscii ? "MC ASCII" : "MC Binary",
            KeyenceDriverKind.KvOld => "KV Old",
            KeyenceDriverKind.Nano or KeyenceDriverKind.NanoSerial or KeyenceDriverKind.NanoSerialOverTcp => options.UseStation ? $"Nano / Station={options.Station}" : "Nano",
            _ => "Keyence",
        };
        var endpoint = _kind == KeyenceDriverKind.NanoSerial ? options.PortName : $"{options.Host}:{options.Port}";
        return new KeyenceConnectionSession(client, _kind, $"{endpoint} / {suffix}");
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
