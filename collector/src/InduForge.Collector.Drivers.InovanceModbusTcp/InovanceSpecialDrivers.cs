using System.Diagnostics;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InovanceModbusTcp;

public sealed class InovanceConnectedCipDriver : InovanceSpecialDriverBase
{
    public InovanceConnectedCipDriver() : base("inovance.connected-cip", HslInovanceSpecialProtocol.ConnectedCip) { }
}

public sealed class InovanceEasyNetDriver : InovanceSpecialDriverBase
{
    public InovanceEasyNetDriver() : base("inovance.easy-net", HslInovanceSpecialProtocol.EasyNet) { }
}

public sealed class InovanceComputerLinkDriver : InovanceSpecialDriverBase
{
    public InovanceComputerLinkDriver() : base("inovance.computer-link", HslInovanceSpecialProtocol.ComputerLink) { }
}

public abstract class InovanceSpecialDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslInovanceSpecialProtocol _protocol;
    private readonly IHslInovanceSpecialClientFactory _factory;

    private protected InovanceSpecialDriverBase(string driverId, HslInovanceSpecialProtocol protocol, IHslInovanceSpecialClientFactory? factory = null)
    {
        _protocol = protocol;
        _factory = factory ?? new HslInovanceSpecialClientFactory();
        Descriptor = new("inovance", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
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
        var options = InovanceSpecialOptions.Parse(profile, _protocol);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded)
        {
            await client.DisposeAsync();
            throw new InovanceModbusTcpDriverException(result.ErrorCode ?? "INOVANCE_CONNECT_FAILED", result.ErrorMessage ?? "无法连接汇川设备", result.Retryable);
        }
        return new Session(client, options.ServerName, _protocol);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken);
    }

    private sealed class Session(IHslInovanceSpecialClient client, string serverName, HslInovanceSpecialProtocol protocol) : IIndustrialConnectionSession, IPointReaderSession
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
                    var address = InovanceSpecialAddress.Parse(point, protocol);
                    var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken);
                    values[index] = result.Succeeded
                        ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                        : new(point.Key, false, null, point.DataType, "Bad", null, null, result.ErrorCode, result.ErrorMessage);
                }
                catch (InovanceModbusTcpDriverException exception)
                {
                    values[index] = new(point.Key, false, null, point.DataType, "Bad", null, null, exception.Code, exception.Message);
                }
            }
            return new(values, []);
        }

        public ValueTask DisposeAsync() => client.DisposeAsync();
    }
}

internal sealed record InovanceSpecialAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static InovanceSpecialAddress Parse(PointReadRequest point, HslInovanceSpecialProtocol protocol)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("INOVANCE_ADDRESS_INVALID", "汇川地址必须包含 address 字段");
        var address = value.GetString()?.Trim() ?? string.Empty;
        var maximumLength = protocol == HslInovanceSpecialProtocol.ConnectedCip ? 512 : 128;
        if (address.Length is < 1 || address.Length > maximumLength || address.Any(char.IsWhiteSpace))
            throw Invalid("INOVANCE_ADDRESS_INVALID", "汇川设备地址无效");
        if (point.ElementCount is < 1 or > ushort.MaxValue) throw Invalid("INOVANCE_ELEMENT_COUNT_INVALID", "汇川元素数量无效");
        if (point.DataType == "datetime" && (protocol != HslInovanceSpecialProtocol.ConnectedCip || point.ElementCount != 1))
            throw Invalid("INOVANCE_DATA_TYPE_UNSUPPORTED", "当前汇川协议不支持该 datetime 配置");
        var dataType = point.DataType switch
        {
            "bool" => HslValueType.Boolean,
            "int16" => HslValueType.Signed16,
            "uint16" => HslValueType.Unsigned16,
            "int32" => HslValueType.Signed32,
            "uint32" => HslValueType.Unsigned32,
            "int64" => HslValueType.Signed64,
            "uint64" => HslValueType.Unsigned64,
            "float32" => HslValueType.SinglePrecision,
            "float64" => HslValueType.DoublePrecision,
            "string" => HslValueType.Text,
            "datetime" when protocol == HslInovanceSpecialProtocol.ConnectedCip => HslValueType.DateTime,
            _ => throw Invalid("INOVANCE_DATA_TYPE_UNSUPPORTED", "当前汇川协议不支持该数据类型"),
        };
        return new(address, dataType, point.ElementCount);
    }

    public HslInovanceSpecialReadRequest ToHslRequest() => new(Address, DataType, ElementCount);
    private static InovanceModbusTcpDriverException Invalid(string code, string message) => new(code, message, false);
}

internal sealed record InovanceSpecialOptions(
    HslInovanceSpecialProtocol Protocol, string Host, int Port, string? PortName,
    int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, byte Station,
    byte WaitingTime, bool SumCheck, int Format, uint ToConnectionId, uint OtConnectionId)
{
    public static InovanceSpecialOptions Parse(ConnectionProfile profile, HslInovanceSpecialProtocol protocol)
    {
        if (!string.Equals(profile.ProtocolFamily, "inovance", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("INOVANCE_CONFIG_INVALID", "汇川连接配置无效");
        var serial = protocol == HslInovanceSpecialProtocol.ComputerLink;
        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty;
        if (!serial && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))) throw Invalid("INOVANCE_HOST_INVALID", "汇川设备 IP / 主机名无效");
        var defaultPort = protocol == HslInovanceSpecialProtocol.ConnectedCip ? 44818 : 12939;
        var port = Integer(profile.Config, "port") ?? defaultPort;
        if (!serial && port is < 1 or > 65535) throw Invalid("INOVANCE_PORT_INVALID", "汇川端口无效");
        var portName = Text(profile.Config, "portName")?.Trim();
        if (serial && string.IsNullOrWhiteSpace(portName)) throw Invalid("INOVANCE_PORT_NAME_INVALID", "汇川 ComputerLink 串口名称不能为空");
        var station = Integer(profile.Config, "station") ?? 0;
        var waitingTime = Integer(profile.Config, "waitingTime") ?? 0;
        if (station is < 0 or > 255 || waitingTime is < 0 or > 255) throw Invalid("INOVANCE_CONFIG_INVALID", "汇川站号或等待时间无效");
        var format = Integer(profile.Config, "format") ?? 4;
        if (format is not (1 or 4)) throw Invalid("INOVANCE_FORMAT_INVALID", "汇川 ComputerLink 格式无效");
        return new(protocol, host, port, portName, Integer(profile.Config, "baudRate") ?? 9600, Integer(profile.Config, "dataBits") ?? 7,
            ParseParity(Text(profile.Config, "parity") ?? "even"), (Text(profile.Config, "stopBits") ?? "two") == "two" ? HslSerialStopBits.Two : HslSerialStopBits.One,
            Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000), checked((byte)station), checked((byte)waitingTime),
            Boolean(profile.Config, "sumCheck", true), format, Unsigned(profile.Config, "toConnectionId"), Unsigned(profile.Config, "otConnectionId"));
    }

    public HslInovanceSpecialClientOptions ToHslOptions() => new(Protocol, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, Station, WaitingTime, SumCheck, Format, ToConnectionId, OtConnectionId);
    public string ServerName => Protocol == HslInovanceSpecialProtocol.ComputerLink ? PortName! : $"{Host}:{Port}";
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("INOVANCE_TIMEOUT_INVALID", "汇川超时配置无效"); return value; }
    private static HslSerialParity ParseParity(string value) => value switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, _ => throw Invalid("INOVANCE_PARITY_INVALID", "汇川串口校验方式无效") };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;
    private static uint Unsigned(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetUInt32(out var result) ? result : 0;
    private static bool Boolean(JsonElement config, string name, bool fallback) => config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : fallback;
    private static InovanceModbusTcpDriverException Invalid(string code, string message) => new(code, message, false);
}
