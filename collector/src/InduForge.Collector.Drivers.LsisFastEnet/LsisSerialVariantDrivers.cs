using System.Diagnostics;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.LsisFastEnet;

public sealed class LsisCnetDriver : LsisSerialVariantDriverBase { public LsisCnetDriver() : base("lsis.cnet", HslLsisSerialProtocol.CnetSerial) { } }
public sealed class LsisCnetOverTcpDriver : LsisSerialVariantDriverBase { public LsisCnetOverTcpDriver() : base("lsis.cnet-over-tcp", HslLsisSerialProtocol.CnetOverTcp) { } }
public sealed class LsisCpuSerialDriver : LsisSerialVariantDriverBase { public LsisCpuSerialDriver() : base("lsis.cpu-serial", HslLsisSerialProtocol.CpuSerial) { } }

public abstract class LsisSerialVariantDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslLsisSerialProtocol _protocol;
    private readonly IHslLsisSerialClientFactory _factory;

    private protected LsisSerialVariantDriverBase(string driverId, HslLsisSerialProtocol protocol, IHslLsisSerialClientFactory? factory = null)
    {
        _protocol = protocol;
        _factory = factory ?? new HslLsisSerialClientFactory();
        Descriptor = new("lsis", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
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
        var options = LsisSerialVariantOptions.Parse(profile, _protocol);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded)
        {
            await client.DisposeAsync();
            throw new LsisFastEnetDriverException(result.ErrorCode ?? "LSIS_CONNECT_FAILED", result.ErrorMessage ?? "无法连接 LSIS 设备", result.Retryable);
        }
        return new Session(client, options.ServerName);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken);
    }

    private sealed class Session(IHslLsisSerialClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected;
        public string? ServerName { get; } = serverName;

        // 地址或单点通信异常仅返回当前点失败，避免中断同一长连接中的其他变量读取。
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
        {
            var readAt = DateTimeOffset.UtcNow;
            var values = new PointReadValue[request.Points.Count];
            for (var index = 0; index < request.Points.Count; index++)
            {
                var point = request.Points[index];
                try
                {
                    var address = LsisSerialVariantAddress.Parse(point);
                    var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken);
                    values[index] = result.Succeeded
                        ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                        : new(point.Key, false, null, point.DataType, "Bad", null, null, result.ErrorCode, result.ErrorMessage);
                }
                catch (LsisFastEnetDriverException exception)
                {
                    values[index] = new(point.Key, false, null, point.DataType, "Bad", null, null, exception.Code, exception.Message);
                }
            }
            return new(values, []);
        }

        public ValueTask DisposeAsync() => client.DisposeAsync();
    }
}

internal sealed record LsisSerialVariantAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static LsisSerialVariantAddress Parse(PointReadRequest point)
    {
        var parsed = LsisFastEnetAddress.Parse(point);
        return new(parsed.ProtocolAddress, parsed.ValueType, parsed.ElementCount);
    }

    public HslLsisSerialReadRequest ToHslRequest() => new(Address, DataType, ElementCount);
}

internal sealed record LsisSerialVariantOptions(
    HslLsisSerialProtocol Protocol, string Host, int Port, string? PortName,
    int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds,
    int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits, byte Station)
{
    public static LsisSerialVariantOptions Parse(ConnectionProfile profile, HslLsisSerialProtocol protocol)
    {
        if (!string.Equals(profile.ProtocolFamily, "lsis", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("LSIS_CONFIG_INVALID", "LSIS 连接配置无效");
        var tcp = protocol == HslLsisSerialProtocol.CnetOverTcp;
        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty;
        if (tcp && (host.Length == 0 || host.Contains("://", StringComparison.Ordinal))) throw Invalid("LSIS_HOST_INVALID", "LSIS 设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? 2000;
        if (tcp && port is < 1 or > 65535) throw Invalid("LSIS_PORT_INVALID", "LSIS 端口无效");
        var portName = Text(profile.Config, "portName")?.Trim();
        if (!tcp && string.IsNullOrWhiteSpace(portName)) throw Invalid("LSIS_PORT_NAME_INVALID", "LSIS 串口名称不能为空");
        var baudRate = Integer(profile.Config, "baudRate") ?? 9600;
        var dataBits = Integer(profile.Config, "dataBits") ?? 8;
        var stopBits = Integer(profile.Config, "stopBits") ?? 1;
        if (baudRate <= 0 || dataBits is < 5 or > 8 || stopBits is < 1 or > 2) throw Invalid("LSIS_SERIAL_CONFIG_INVALID", "LSIS 串口参数无效");
        return new(protocol, host, port, portName, Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000), baudRate, dataBits, ParseParity(Text(profile.Config, "parity") ?? "none"), stopBits == 2 ? HslSerialStopBits.Two : HslSerialStopBits.One, Byte(profile.Config, "station", 1));
    }

    public HslLsisSerialClientOptions ToHslOptions() => new(Protocol, Host, Port, PortName, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, BaudRate, DataBits, Parity, StopBits, Station);
    public string ServerName => Protocol == HslLsisSerialProtocol.CnetOverTcp ? $"{Host}:{Port}" : PortName!;

    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("LSIS_TIMEOUT_INVALID", "LSIS 超时配置无效"); return value; }
    private static byte Byte(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 0 or > 255) throw Invalid("LSIS_CONFIG_INVALID", $"LSIS {name} 无效"); return checked((byte)value); }
    private static HslSerialParity ParseParity(string value) => value switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, _ => throw Invalid("LSIS_PARITY_INVALID", "LSIS 串口校验方式无效") };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;
    private static LsisFastEnetDriverException Invalid(string code, string message) => new(code, message, false);
}
