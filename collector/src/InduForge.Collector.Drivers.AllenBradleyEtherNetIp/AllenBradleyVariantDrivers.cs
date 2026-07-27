using System.Diagnostics;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.AllenBradleyEtherNetIp;

public sealed class AllenBradleyConnectedCipDriver : AllenBradleyVariantDriverBase
{
    public AllenBradleyConnectedCipDriver() : base("allen-bradley.connected-cip", HslAllenBradleyProtocol.ConnectedCip, "Allen-Bradley Connected CIP", false) { }
}

public sealed class AllenBradleyMicroCipDriver : AllenBradleyVariantDriverBase
{
    public AllenBradleyMicroCipDriver() : base("allen-bradley.micro-cip", HslAllenBradleyProtocol.MicroCip, "Allen-Bradley MicroCIP", false) { }
}

public sealed class AllenBradleyPcccDriver : AllenBradleyVariantDriverBase
{
    public AllenBradleyPcccDriver() : base("allen-bradley.pccc", HslAllenBradleyProtocol.Pccc, "Allen-Bradley PCCC", true) { }
}

public sealed class AllenBradleySlcDriver : AllenBradleyVariantDriverBase
{
    public AllenBradleySlcDriver() : base("allen-bradley.slc", HslAllenBradleyProtocol.Slc, "Allen-Bradley SLC", true) { }
}

public sealed class AllenBradleyDf1SerialDriver : AllenBradleyVariantDriverBase
{
    public AllenBradleyDf1SerialDriver() : base("allen-bradley.df1-serial", HslAllenBradleyProtocol.Df1Serial, "Allen-Bradley DF1 Serial", true) { }
}

public abstract class AllenBradleyVariantDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly HslAllenBradleyProtocol _protocol;
    private readonly string _displayName;
    private readonly AllenBradleyAddressKind _addressKind;
    private readonly IHslAllenBradleyClientFactory _clientFactory;

    protected AllenBradleyVariantDriverBase(
        string driverId,
        HslAllenBradleyProtocol protocol,
        string displayName,
        bool legacyAddress)
        : this(
            driverId,
            protocol,
            displayName,
            legacyAddress ? AllenBradleyAddressKind.LegacyFile : AllenBradleyAddressKind.LogixTag,
            new HslAllenBradleyClientFactory())
    {
    }

    internal AllenBradleyVariantDriverBase(
        string driverId,
        HslAllenBradleyProtocol protocol,
        string displayName,
        AllenBradleyAddressKind addressKind,
        IHslAllenBradleyClientFactory clientFactory)
    {
        _protocol = protocol;
        _displayName = displayName;
        _addressKind = addressKind;
        _clientFactory = clientFactory;
        Descriptor = new DriverDescriptor(
            "allen-bradley",
            driverId,
            "1.0.0",
            [1],
            [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    }

    public DriverDescriptor Descriptor { get; }

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var stopwatch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        stopwatch.Stop();
        return new ConnectionTestResult(session.IsConnected, stopwatch.Elapsed, session.ServerName, NoDiagnostics);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = AllenBradleyVariantConnectionOptions.Parse(profile, _protocol);
        var client = _clientFactory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new AllenBradleyEtherNetIpDriverException(
                result.ErrorCode ?? "ALLEN_BRADLEY_CONNECT_FAILED",
                result.ErrorMessage ?? $"无法连接 {_displayName} 设备",
                result.Retryable);
        }
        return new AllenBradleyEtherNetIpConnectionSession(client, options.ServerName, _addressKind);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}
