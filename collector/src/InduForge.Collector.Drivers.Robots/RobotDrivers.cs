using System.Diagnostics;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Robots;

public sealed class EstunRobotDriver : RobotDriverBase
{
    public EstunRobotDriver() : this(new HslSpecializedNetworkClientFactory()) { }
    internal EstunRobotDriver(IHslSpecializedNetworkClientFactory factory) : base(HslSpecializedNetworkProtocol.EstunRobot, factory) { }
    public override DriverDescriptor Descriptor { get; } = DescriptorFor("estun", "robot.estun-tcp");
}

public sealed class FanucRobotDriver : RobotDriverBase
{
    public FanucRobotDriver() : this(new HslSpecializedNetworkClientFactory()) { }
    internal FanucRobotDriver(IHslSpecializedNetworkClientFactory factory) : base(HslSpecializedNetworkProtocol.FanucRobot, factory) { }
    public override DriverDescriptor Descriptor { get; } = DescriptorFor("fanuc", "robot.fanuc-interface");
}

public abstract class RobotDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslSpecializedNetworkProtocol _protocol;
    private readonly IHslSpecializedNetworkClientFactory _factory;

    protected RobotDriverBase(HslSpecializedNetworkProtocol protocol, IHslSpecializedNetworkClientFactory factory)
    {
        _protocol = protocol;
        _factory = factory;
    }

    public abstract DriverDescriptor Descriptor { get; }
    protected static DriverDescriptor DescriptorFor(string family, string driverId) => new(family, driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        watch.Stop();
        return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = RobotOptions.Parse(profile, _protocol);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new RobotDriverException(result.ErrorCode ?? "ROBOT_CONNECT_FAILED", result.ErrorMessage ?? "机器人连接失败", result.Retryable);
        }
        return new Session(client, options.ServerName, _protocol);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }

    private sealed class Session(IHslSpecializedNetworkClient client, string serverName, HslSpecializedNetworkProtocol protocol) : IIndustrialConnectionSession, IPointReaderSession
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
                    var address = RobotAddress.Parse(point, protocol);
                    var result = await client.ReadAsync(address.ToRequest(), cancellationToken).ConfigureAwait(false);
                    values[index] = result.Succeeded
                        ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                        : Fail(point, result.ErrorCode ?? "ROBOT_READ_FAILED", result.ErrorMessage ?? "机器人变量读取失败");
                }
                catch (RobotDriverException exception)
                {
                    values[index] = Fail(point, exception.Code, exception.Message);
                }
            }
            return new(values, []);
        }

        public ValueTask DisposeAsync() => client.DisposeAsync();
        private static PointReadValue Fail(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}

internal sealed record RobotOptions(HslSpecializedNetworkProtocol Protocol, string Host, int Port, byte Station, HslDataFormat DataFormat, int RetainTimeMs, int ConnectTimeoutMs, int ReceiveTimeoutMs)
{
    public string ServerName => $"{Host}:{Port}";
    public HslSpecializedNetworkClientOptions ToHslOptions() => new(Protocol, Host, Port, ConnectTimeoutMs, ReceiveTimeoutMs, Station: Station, DataFormat: DataFormat, DataRetainTimeMilliseconds: RetainTimeMs);

    public static RobotOptions Parse(ConnectionProfile profile, HslSpecializedNetworkProtocol protocol)
    {
        var family = protocol == HslSpecializedNetworkProtocol.EstunRobot ? "estun" : "fanuc";
        if (!string.Equals(profile.ProtocolFamily, family, StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("CONFIG", "机器人连接配置无效");
        var host = Text(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)) throw Invalid("HOST", "机器人设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? (protocol == HslSpecializedNetworkProtocol.EstunRobot ? 502 : 60008);
        var station = Integer(profile.Config, "station") ?? 1;
        var retain = Integer(profile.Config, "dataRetainTimeMs") ?? 100;
        if (port is < 1 or > 65535 || station is < 0 or > 255 || retain is < 0 or > 60000) throw Invalid("PARAMETER", "机器人连接参数无效");
        return new(protocol, host, port, (byte)station, protocol == HslSpecializedNetworkProtocol.EstunRobot ? HslDataFormat.CDAB : HslDataFormat.ABCD, retain, Timeout(profile.Config, "connectTimeoutMs", protocol == HslSpecializedNetworkProtocol.EstunRobot ? 2000 : 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000));
    }

    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("TIMEOUT", "机器人超时必须在 100 到 120000 毫秒之间"); return value; }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static RobotDriverException Invalid(string part, string message) => new($"ROBOT_{part}_INVALID", message, false);
}

internal sealed record RobotAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static RobotAddress Parse(PointReadRequest point, HslSpecializedNetworkProtocol protocol)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String || point.ElementCount is < 1 or > 65535) throw Invalid();
        var address = value.GetString()?.Trim() ?? string.Empty;
        if (address.Length is < 1 or > 128 || address.Any(char.IsWhiteSpace)) throw Invalid();
        return new(address, ParseDataType(point.DataType, protocol), point.ElementCount);
    }

    public HslSpecializedNetworkReadRequest ToRequest() => new(Address, DataType, ElementCount);
    private static HslValueType ParseDataType(string value, HslSpecializedNetworkProtocol protocol) => value switch
    {
        "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8,
        "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32,
        "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64,
        "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text,
        "bytes" => HslValueType.Binary, _ => throw new RobotDriverException("ROBOT_DATA_TYPE_UNSUPPORTED", $"{protocol} 数据类型不受支持", false),
    };
    private static RobotDriverException Invalid() => new("ROBOT_ADDRESS_INVALID", "机器人变量地址无效", false);
}

internal sealed class RobotDriverException(string code, string message, bool retryable) : Exception(message)
{
    public string Code { get; } = code;
    public bool Retryable { get; } = retryable;
}
