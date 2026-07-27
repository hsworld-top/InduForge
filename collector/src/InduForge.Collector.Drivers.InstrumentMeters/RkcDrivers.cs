using System.Diagnostics;
using System.Globalization;
using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InstrumentMeters;

public sealed class RkcTemperatureControllerSerialDriver : RkcDriverBase
{
    public RkcTemperatureControllerSerialDriver() : base(HslInstrumentTransport.Serial, new HslRkcClientFactory()) { }
    internal RkcTemperatureControllerSerialDriver(IHslRkcClientFactory factory) : base(HslInstrumentTransport.Serial, factory) { }
    public override DriverDescriptor Descriptor { get; } = DescriptorFor("rkc.temperature-controller-serial");
}

public sealed class RkcTemperatureControllerTcpDriver : RkcDriverBase
{
    public RkcTemperatureControllerTcpDriver() : base(HslInstrumentTransport.Tcp, new HslRkcClientFactory()) { }
    internal RkcTemperatureControllerTcpDriver(IHslRkcClientFactory factory) : base(HslInstrumentTransport.Tcp, factory) { }
    public override DriverDescriptor Descriptor { get; } = DescriptorFor("rkc.temperature-controller-tcp");
}

public abstract class RkcDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslInstrumentTransport _transport;
    private readonly IHslRkcClientFactory _factory;

    protected RkcDriverBase(HslInstrumentTransport transport, IHslRkcClientFactory factory)
    {
        _transport = transport;
        _factory = factory;
    }

    public abstract DriverDescriptor Descriptor { get; }

    protected static DriverDescriptor DescriptorFor(string driverId) => new(
        "rkc",
        driverId,
        "1.0.0",
        [1],
        [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        watch.Stop();
        return new ConnectionTestResult(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = RkcOptions.Parse(profile, _transport);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded)
        {
            await client.DisposeAsync();
            throw new InstrumentDriverException(result.ErrorCode ?? "RKC_CONNECT_FAILED", result.ErrorMessage ?? "无法连接 RKC 温控器", result.Retryable);
        }
        return new Session(client, options.ServerName);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken);
    }

    private sealed class Session(IHslRkcClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
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
                try
                {
                    var address = RkcAddress.Parse(point);
                    var result = await client.ReadAsync(address.ToRequest(), cancellationToken);
                    values[index] = result.Succeeded
                        ? new PointReadValue(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                        : Fail(point, result.ErrorCode ?? "RKC_READ_FAILED", result.ErrorMessage ?? "RKC 温控器变量读取失败");
                }
                catch (InstrumentDriverException exception)
                {
                    values[index] = Fail(point, exception.Code, exception.Message);
                }
            }
            return new ReadResult(values, []);
        }

        public ValueTask DisposeAsync() => client.DisposeAsync();
        private static PointReadValue Fail(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}

internal sealed record RkcOptions(
    HslInstrumentTransport Transport,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    byte Station,
    int ConnectTimeoutMs,
    int ReceiveTimeoutMs)
{
    public string ServerName => Transport == HslInstrumentTransport.Serial ? PortName! : $"{Host}:{Port}";
    public HslRkcClientOptions ToHslOptions() => new(Transport, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits, Station, ConnectTimeoutMs, ReceiveTimeoutMs);

    public static RkcOptions Parse(ConnectionProfile profile, HslInstrumentTransport transport)
    {
        if (!string.Equals(profile.ProtocolFamily, "rkc", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("CONFIG", "RKC 温控器连接配置无效");
        }

        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty;
        var portName = Text(profile.Config, "portName")?.Trim();
        if (transport == HslInstrumentTransport.Tcp && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))) throw Invalid("HOST", "RKC 温控器 IP / 主机名无效");
        if (transport == HslInstrumentTransport.Serial && string.IsNullOrWhiteSpace(portName)) throw Invalid("PORT_NAME", "RKC 温控器串口名称不能为空");

        var port = Integer(profile.Config, "port") ?? 2000;
        var station = Integer(profile.Config, "station") ?? 1;
        var baudRate = Integer(profile.Config, "baudRate") ?? 9600;
        var dataBits = Integer(profile.Config, "dataBits") ?? 8;
        var connectTimeoutMs = Integer(profile.Config, "connectTimeoutMs") ?? 5000;
        var receiveTimeoutMs = Integer(profile.Config, "receiveTimeoutMs") ?? 10000;
        if (port is < 1 or > 65535) throw Invalid("PORT", "RKC 温控器端口必须在 1 到 65535 之间");
        if (station is < 0 or > 99) throw Invalid("STATION", "RKC 温控器站号必须在 0 到 99 之间");
        if (baudRate <= 0 || dataBits is < 5 or > 8) throw Invalid("SERIAL", "RKC 温控器串口参数无效");
        if (connectTimeoutMs is < 100 or > 120000 || receiveTimeoutMs is < 100 or > 120000) throw Invalid("TIMEOUT", "RKC 温控器超时时间必须在 100 到 120000 毫秒之间");

        return new RkcOptions(
            transport, host, port, portName, baudRate, dataBits, ParseParity(Text(profile.Config, "parity")),
            ParseStopBits(Integer(profile.Config, "stopBits") ?? 1), (byte)station, connectTimeoutMs, receiveTimeoutMs);
    }

    private static HslSerialParity ParseParity(string? value) => value?.Trim().ToLowerInvariant() switch
    {
        "odd" => HslSerialParity.Odd,
        "even" => HslSerialParity.Even,
        null or "" or "none" => HslSerialParity.None,
        _ => throw Invalid("PARITY", "RKC 温控器校验方式无效"),
    };
    private static HslSerialStopBits ParseStopBits(int value) => value switch { 1 => HslSerialStopBits.One, 2 => HslSerialStopBits.Two, _ => throw Invalid("STOP_BITS", "RKC 温控器停止位只支持 1 或 2") };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static InstrumentDriverException Invalid(string part, string message) => new($"RKC_{part}_INVALID", message, false);
}

internal sealed partial record RkcAddress(string Address)
{
    public static RkcAddress Parse(PointReadRequest point)
    {
        if (point.DataType != "float64" || point.ElementCount != 1 || point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
        {
            throw Invalid();
        }

        var match = AddressRegex().Match(value.GetString()?.Trim() ?? string.Empty);
        if (!match.Success) throw Invalid();
        var prefix = match.Groups[1].Success
            ? $"s={byte.Parse(match.Groups[1].Value, CultureInfo.InvariantCulture)};"
            : string.Empty;
        return new RkcAddress(prefix + match.Groups[2].Value.ToUpperInvariant());
    }

    public HslRkcReadRequest ToRequest() => new(Address, HslValueType.DoublePrecision, 1);
    private static InstrumentDriverException Invalid() => new("RKC_ADDRESS_INVALID", "RKC 地址应为 M1、AA、ER 等两位代码，可选使用 s=<站号>; 前缀，数据类型固定为 float64", false);

    [GeneratedRegex("^(?:s=([0-9]{1,2});)?([A-Za-z][A-Za-z0-9])$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)]
    private static partial Regex AddressRegex();
}

internal sealed class InstrumentDriverException(string code, string message, bool retryable) : Exception(message)
{
    public string Code { get; } = code;
    public bool Retryable { get; } = retryable;
}
