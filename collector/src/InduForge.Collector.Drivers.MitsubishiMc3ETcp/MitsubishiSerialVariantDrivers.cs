using System.Diagnostics;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MitsubishiMc3ETcp;

public sealed class MitsubishiA3CSerialDriver : MitsubishiSerialVariantDriverBase
{
    public MitsubishiA3CSerialDriver() : base("mitsubishi.a3c-serial", HslMelsecSerialProtocol.A3CSerial) { }
}

public sealed class MitsubishiA3CSerialOverTcpDriver : MitsubishiSerialVariantDriverBase
{
    public MitsubishiA3CSerialOverTcpDriver() : base("mitsubishi.a3c-serial-over-tcp", HslMelsecSerialProtocol.A3CSerialOverTcp) { }
}

public sealed class MitsubishiFxLinksSerialDriver : MitsubishiSerialVariantDriverBase
{
    public MitsubishiFxLinksSerialDriver() : base("mitsubishi.fx-links-serial", HslMelsecSerialProtocol.FxLinksSerial) { }
}

public sealed class MitsubishiFxLinksOverTcpDriver : MitsubishiSerialVariantDriverBase
{
    public MitsubishiFxLinksOverTcpDriver() : base("mitsubishi.fx-links-over-tcp", HslMelsecSerialProtocol.FxLinksOverTcp) { }
}

public sealed class MitsubishiFxSerialDriver : MitsubishiSerialVariantDriverBase
{
    public MitsubishiFxSerialDriver() : base("mitsubishi.fx-serial", HslMelsecSerialProtocol.FxSerial) { }
}

public sealed class MitsubishiFxSerialOverTcpDriver : MitsubishiSerialVariantDriverBase
{
    public MitsubishiFxSerialOverTcpDriver() : base("mitsubishi.fx-serial-over-tcp", HslMelsecSerialProtocol.FxSerialOverTcp) { }
}

public abstract class MitsubishiSerialVariantDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslMelsecSerialProtocol _protocol;
    private readonly IHslMelsecSerialClientFactory _factory;

    private protected MitsubishiSerialVariantDriverBase(
        string driverId,
        HslMelsecSerialProtocol protocol,
        IHslMelsecSerialClientFactory? factory = null)
    {
        _protocol = protocol;
        _factory = factory ?? new HslMelsecSerialClientFactory();
        Descriptor = new(
            "melsec",
            driverId,
            "1.0.0",
            [1],
            [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    }

    public DriverDescriptor Descriptor { get; }

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        watch.Stop();
        return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = MitsubishiSerialVariantOptions.Parse(profile, _protocol);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded)
        {
            await client.DisposeAsync();
            throw new MitsubishiMc3ETcpDriverException(
                result.ErrorCode ?? "MELSEC_SERIAL_CONNECT_FAILED",
                result.ErrorMessage ?? "无法连接三菱串行设备",
                result.Retryable);
        }
        return new Session(client, options.ServerName);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken);
    }

    private sealed class Session(IHslMelsecSerialClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected;
        public string? ServerName { get; } = serverName;

        // 单个变量地址或通信失败只影响当前结果，长连接继续服务同一批次中的其他变量。
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
        {
            var readAt = DateTimeOffset.UtcNow;
            var values = new PointReadValue[request.Points.Count];
            for (var index = 0; index < request.Points.Count; index++)
            {
                var point = request.Points[index];
                try
                {
                    var address = MitsubishiSerialVariantAddress.Parse(point);
                    var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken);
                    values[index] = result.Succeeded
                        ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                        : new(point.Key, false, null, point.DataType, "Bad", null, null, result.ErrorCode, result.ErrorMessage);
                }
                catch (MitsubishiMc3ETcpDriverException exception)
                {
                    values[index] = new(point.Key, false, null, point.DataType, "Bad", null, null, exception.Code, exception.Message);
                }
            }
            return new(values, []);
        }

        public ValueTask DisposeAsync() => client.DisposeAsync();
    }
}

internal sealed record MitsubishiSerialVariantAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static MitsubishiSerialVariantAddress Parse(PointReadRequest point)
    {
        var parsed = MitsubishiMc3ETcpAddress.Parse(point);
        return new(parsed.HslAddress, parsed.ValueType, parsed.ElementCount);
    }

    public HslMelsecSerialReadRequest ToHslRequest() => new(Address, DataType, ElementCount);
}

internal sealed record MitsubishiSerialVariantOptions(
    HslMelsecSerialProtocol Protocol,
    string Host,
    int Port,
    string? PortName,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    byte Station,
    byte WaitingTime,
    bool SumCheck,
    int Format,
    bool IsNewVersion,
    bool UseGot)
{
    public static MitsubishiSerialVariantOptions Parse(ConnectionProfile profile, HslMelsecSerialProtocol protocol)
    {
        if (!string.Equals(profile.ProtocolFamily, "melsec", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("MELSEC_SERIAL_CONFIG_INVALID", "三菱串行连接配置无效");

        var serial = protocol is HslMelsecSerialProtocol.A3CSerial or HslMelsecSerialProtocol.FxLinksSerial or HslMelsecSerialProtocol.FxSerial;
        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty;
        if (!serial && (host.Length == 0 || host.Contains("://", StringComparison.Ordinal)))
            throw Invalid("MELSEC_SERIAL_HOST_INVALID", "三菱串口透传设备 IP / 主机名无效");

        var port = Integer(profile.Config, "port") ?? DefaultPort(protocol);
        if (!serial && port is < 1 or > 65535)
            throw Invalid("MELSEC_SERIAL_PORT_INVALID", "三菱串口透传端口无效");

        var portName = Text(profile.Config, "portName")?.Trim();
        if (serial && string.IsNullOrWhiteSpace(portName))
            throw Invalid("MELSEC_SERIAL_PORT_NAME_INVALID", "三菱串口名称不能为空");

        var defaults = DefaultSerialSettings(protocol);
        var baudRate = Integer(profile.Config, "baudRate") ?? 9600;
        var dataBits = Integer(profile.Config, "dataBits") ?? defaults.DataBits;
        var stopBitsValue = Integer(profile.Config, "stopBits") ?? defaults.StopBits;
        if (baudRate <= 0 || dataBits is < 5 or > 8 || stopBitsValue is < 1 or > 2)
            throw Invalid("MELSEC_SERIAL_CONFIG_INVALID", "三菱串口参数无效");

        return new(
            protocol,
            host,
            port,
            portName,
            Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000),
            baudRate,
            dataBits,
            ParseParity(Text(profile.Config, "parity") ?? defaults.Parity),
            stopBitsValue == 2 ? HslSerialStopBits.Two : HslSerialStopBits.One,
            Byte(profile.Config, "station", 0),
            Byte(profile.Config, "waitingTime", 0),
            Boolean(profile.Config, "sumCheck", true),
            Integer(profile.Config, "format") ?? 1,
            Boolean(profile.Config, "isNewVersion", true),
            Boolean(profile.Config, "useGot", false));
    }

    public HslMelsecSerialClientOptions ToHslOptions() => new(
        Protocol,
        Host,
        Port,
        PortName,
        ConnectTimeoutMilliseconds,
        ReceiveTimeoutMilliseconds,
        BaudRate,
        DataBits,
        Parity,
        StopBits,
        Station,
        WaitingTime,
        SumCheck,
        Format,
        IsNewVersion,
        UseGot);

    public string ServerName => Protocol is HslMelsecSerialProtocol.A3CSerial or HslMelsecSerialProtocol.FxLinksSerial or HslMelsecSerialProtocol.FxSerial
        ? PortName!
        : $"{Host}:{Port}";

    private static int DefaultPort(HslMelsecSerialProtocol protocol) => protocol switch
    {
        HslMelsecSerialProtocol.A3CSerialOverTcp => 6000,
        HslMelsecSerialProtocol.FxLinksOverTcp => 2000,
        HslMelsecSerialProtocol.FxSerialOverTcp => 5014,
        _ => 0,
    };

    private static (int DataBits, string Parity, int StopBits) DefaultSerialSettings(HslMelsecSerialProtocol protocol) => protocol switch
    {
        HslMelsecSerialProtocol.A3CSerial => (8, "none", 1),
        HslMelsecSerialProtocol.FxLinksSerial => (7, "even", 2),
        HslMelsecSerialProtocol.FxSerial => (7, "even", 1),
        _ => (8, "none", 1),
    };

    private static int Timeout(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 100 or > 120000) throw Invalid("MELSEC_SERIAL_TIMEOUT_INVALID", "三菱通信超时配置无效");
        return value;
    }

    private static byte Byte(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 0 or > 255) throw Invalid("MELSEC_SERIAL_CONFIG_INVALID", $"三菱 {name} 无效");
        return checked((byte)value);
    }

    private static HslSerialParity ParseParity(string value) => value switch
    {
        "none" => HslSerialParity.None,
        "odd" => HslSerialParity.Odd,
        "even" => HslSerialParity.Even,
        _ => throw Invalid("MELSEC_SERIAL_PARITY_INVALID", "三菱串口校验方式无效"),
    };

    private static string? Text(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;

    private static int? Integer(JsonElement config, string name) =>
        config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;

    private static bool Boolean(JsonElement config, string name, bool fallback) =>
        config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : fallback;

    private static MitsubishiMc3ETcpDriverException Invalid(string code, string message) => new(code, message, false);
}
