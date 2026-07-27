using System.Diagnostics;
using System.Globalization;
using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InstrumentMeters;

public sealed class Cjt188SerialDriver : Cjt188DriverBase
{
    public Cjt188SerialDriver() : base(HslInstrumentTransport.Serial, new HslCjt188ClientFactory()) { }
    internal Cjt188SerialDriver(IHslCjt188ClientFactory factory) : base(HslInstrumentTransport.Serial, factory) { }
    public override DriverDescriptor Descriptor { get; } = DescriptorFor("cjt188.serial");
}

public sealed class Cjt188TcpDriver : Cjt188DriverBase
{
    public Cjt188TcpDriver() : base(HslInstrumentTransport.Tcp, new HslCjt188ClientFactory()) { }
    internal Cjt188TcpDriver(IHslCjt188ClientFactory factory) : base(HslInstrumentTransport.Tcp, factory) { }
    public override DriverDescriptor Descriptor { get; } = DescriptorFor("cjt188.tcp");
}

public abstract class Cjt188DriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslInstrumentTransport _transport;
    private readonly IHslCjt188ClientFactory _factory;

    protected Cjt188DriverBase(HslInstrumentTransport transport, IHslCjt188ClientFactory factory)
    {
        _transport = transport;
        _factory = factory;
    }

    public abstract DriverDescriptor Descriptor { get; }

    protected static DriverDescriptor DescriptorFor(string driverId) => new(
        "cjt188",
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
        var options = Cjt188Options.Parse(profile, _transport);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded)
        {
            await client.DisposeAsync();
            throw new InstrumentDriverException(result.ErrorCode ?? "CJT188_CONNECT_FAILED", result.ErrorMessage ?? "无法连接 CJT188 仪表", result.Retryable);
        }

        return new Session(client, options.ServerName);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken);
    }

    private sealed class Session(IHslCjt188Client client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected;
        public string? ServerName { get; } = serverName;

        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
        {
            var readAt = DateTimeOffset.UtcNow;
            var values = new PointReadValue[request.Points.Count];

            // 一个坏地址不能中断同一长连接中的其他变量读取。
            for (var index = 0; index < request.Points.Count; index++)
            {
                var point = request.Points[index];
                try
                {
                    var address = Cjt188Address.Parse(point);
                    var result = await client.ReadAsync(address.ToRequest(), cancellationToken);
                    values[index] = result.Succeeded
                        ? new PointReadValue(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                        : Fail(point, result.ErrorCode ?? "CJT188_READ_FAILED", result.ErrorMessage ?? "CJT188 变量读取失败");
                }
                catch (InstrumentDriverException exception)
                {
                    values[index] = Fail(point, exception.Code, exception.Message);
                }
            }

            return new ReadResult(values, []);
        }

        public ValueTask DisposeAsync() => client.DisposeAsync();

        private static PointReadValue Fail(PointReadRequest point, string code, string message) =>
            new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}

internal sealed record Cjt188Options(
    HslInstrumentTransport Transport,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    string Station,
    byte InstrumentType,
    bool EnableCodeFE,
    bool StationMatch,
    int ConnectTimeoutMs,
    int ReceiveTimeoutMs)
{
    public string ServerName => Transport == HslInstrumentTransport.Serial ? PortName! : $"{Host}:{Port}";

    public HslCjt188ClientOptions ToHslOptions() => new(
        Transport, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits, Station,
        InstrumentType, EnableCodeFE, StationMatch, ConnectTimeoutMs, ReceiveTimeoutMs);

    public static Cjt188Options Parse(ConnectionProfile profile, HslInstrumentTransport transport)
    {
        if (!string.Equals(profile.ProtocolFamily, "cjt188", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("CONFIG", "CJT188 连接配置无效");
        }

        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty;
        var portName = Text(profile.Config, "portName")?.Trim();
        if (transport == HslInstrumentTransport.Tcp && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)))
        {
            throw Invalid("HOST", "CJT188 设备 IP / 主机名无效");
        }
        if (transport == HslInstrumentTransport.Serial && string.IsNullOrWhiteSpace(portName))
        {
            throw Invalid("PORT_NAME", "CJT188 串口名称不能为空");
        }

        var port = Integer(profile.Config, "port") ?? 502;
        if (port is < 1 or > 65535) throw Invalid("PORT", "CJT188 端口必须在 1 到 65535 之间");

        var station = NormalizeStation(Text(profile.Config, "station") ?? "78330015040963");
        var instrumentTypeText = (Text(profile.Config, "instrumentType") ?? "19").Trim();
        if (!byte.TryParse(instrumentTypeText, NumberStyles.HexNumber, CultureInfo.InvariantCulture, out var instrumentType))
        {
            throw Invalid("INSTRUMENT_TYPE", "CJT188 仪表类型必须是两位十六进制数");
        }

        var baudRate = Integer(profile.Config, "baudRate") ?? 9600;
        var dataBits = Integer(profile.Config, "dataBits") ?? 8;
        var parity = ParseParity(Text(profile.Config, "parity"));
        var stopBits = ParseStopBits(Integer(profile.Config, "stopBits") ?? 1);
        if (baudRate <= 0) throw Invalid("BAUD_RATE", "CJT188 波特率必须大于 0");
        if (dataBits is < 5 or > 8) throw Invalid("DATA_BITS", "CJT188 数据位必须在 5 到 8 之间");

        var connectTimeoutMs = Integer(profile.Config, "connectTimeoutMs") ?? 5000;
        var receiveTimeoutMs = Integer(profile.Config, "receiveTimeoutMs") ?? 10000;
        if (connectTimeoutMs is < 100 or > 120000 || receiveTimeoutMs is < 100 or > 120000)
        {
            throw Invalid("TIMEOUT", "CJT188 超时时间必须在 100 到 120000 毫秒之间");
        }

        return new Cjt188Options(
            transport, host, port, portName, baudRate, dataBits, parity, stopBits, station, instrumentType,
            Boolean(profile.Config, "enableCodeFE"), Boolean(profile.Config, "stationMatch"), connectTimeoutMs, receiveTimeoutMs);
    }

    private static string NormalizeStation(string value)
    {
        var station = value.Trim().ToUpperInvariant();
        if (!Regex.IsMatch(station, "^[0-9A]{1,14}$", RegexOptions.CultureInvariant))
        {
            throw Invalid("STATION", "CJT188 表地址必须是最多 14 位数字，可使用 A 作为高位通配符");
        }
        return station.PadLeft(14, '0');
    }

    private static HslSerialParity ParseParity(string? value) => value?.Trim().ToLowerInvariant() switch
    {
        "odd" => HslSerialParity.Odd,
        "even" => HslSerialParity.Even,
        null or "" or "none" => HslSerialParity.None,
        _ => throw Invalid("PARITY", "CJT188 校验方式无效"),
    };

    private static HslSerialStopBits ParseStopBits(int value) => value switch
    {
        1 => HslSerialStopBits.One,
        2 => HslSerialStopBits.Two,
        _ => throw Invalid("STOP_BITS", "CJT188 停止位只支持 1 或 2"),
    };

    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static bool Boolean(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False && value.GetBoolean();
    private static InstrumentDriverException Invalid(string part, string message) => new($"CJT188_{part}_INVALID", message, false);
}

internal sealed partial record Cjt188Address(string Address, HslValueType DataType, int ElementCount)
{
    public static Cjt188Address Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
        {
            throw Invalid();
        }

        var match = AddressRegex().Match(value.GetString()?.Trim() ?? string.Empty);
        if (!match.Success || point.ElementCount is < 1 or > 255) throw Invalid();

        var prefix = match.Groups[1].Success ? $"s={NormalizeStation(match.Groups[1].Value)};" : string.Empty;
        var address = $"{match.Groups[2].Value.ToUpperInvariant()}-{match.Groups[3].Value.ToUpperInvariant()}";
        return new Cjt188Address(prefix + address, Type(point.DataType), point.ElementCount);
    }

    public HslCjt188ReadRequest ToRequest() => new(Address, DataType, ElementCount);

    private static HslValueType Type(string value) => value switch
    {
        "float32" => HslValueType.SinglePrecision,
        "float64" => HslValueType.DoublePrecision,
        "string" => HslValueType.Text,
        "bytes" => HslValueType.Binary,
        _ => throw new InstrumentDriverException("CJT188_DATA_TYPE_UNSUPPORTED", "CJT188 数据类型仅支持 float32、float64、string 和 bytes", false),
    };

    private static string NormalizeStation(string value) => value.Trim().ToUpperInvariant().PadLeft(14, '0');
    private static InstrumentDriverException Invalid() => new("CJT188_ADDRESS_INVALID", "CJT188 数据标识应为 90-1F，可选使用 s=<表地址>; 前缀", false);

    [GeneratedRegex("^(?:s=([0-9Aa]{1,14});)?([0-9A-Fa-f]{2})[- ]?([0-9A-Fa-f]{2})$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)]
    private static partial Regex AddressRegex();
}
