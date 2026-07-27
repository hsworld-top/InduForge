using System.Diagnostics;
using System.Globalization;
using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Yamatake;

public sealed class YamatakeDigitronSerialDriver : YamatakeDriverBase { public YamatakeDigitronSerialDriver() : base("yamatake.digitron-serial", HslYamatakeTransport.Serial) { } internal YamatakeDigitronSerialDriver(IHslYamatakeClientFactory factory) : base("yamatake.digitron-serial", HslYamatakeTransport.Serial, factory) { } }
public sealed class YamatakeDigitronTcpDriver : YamatakeDriverBase { public YamatakeDigitronTcpDriver() : base("yamatake.digitron-tcp", HslYamatakeTransport.Tcp) { } internal YamatakeDigitronTcpDriver(IHslYamatakeClientFactory factory) : base("yamatake.digitron-tcp", HslYamatakeTransport.Tcp, factory) { } }

public abstract class YamatakeDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslYamatakeTransport _transport;
    private readonly IHslYamatakeClientFactory _factory;
    private protected YamatakeDriverBase(string driverId, HslYamatakeTransport transport, IHslYamatakeClientFactory? factory = null) { _transport = transport; _factory = factory ?? new HslYamatakeClientFactory(); Descriptor = new("yamatake", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]); }
    public DriverDescriptor Descriptor { get; }
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var options = YamatakeOptions.Parse(profile, _transport); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken); if (!result.Succeeded) { await client.DisposeAsync(); throw new YamatakeDriverException(result.ErrorCode ?? "YAMATAKE_CONNECT_FAILED", result.ErrorMessage ?? "无法连接山武 Digitron", result.Retryable); } return new Session(client, options.ServerName); }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
    private sealed class Session(IHslYamatakeClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken) { var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count]; for (var index = 0; index < request.Points.Count; index++) { var point = request.Points[index]; try { var address = YamatakeAddress.Parse(point); var result = await client.ReadAsync(address.ToRequest(), cancellationToken); values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : Fail(point, result.ErrorCode ?? "YAMATAKE_READ_FAILED", result.ErrorMessage ?? "山武变量读取失败"); } catch (YamatakeDriverException exception) { values[index] = Fail(point, exception.Code, exception.Message); } } return new(values, []); }
        public ValueTask DisposeAsync() => client.DisposeAsync();
        private static PointReadValue Fail(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}

internal sealed record YamatakeOptions(HslYamatakeTransport Transport, string Host, int Port, string? PortName, int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits, byte Station, int ConnectTimeoutMs, int ReceiveTimeoutMs)
{
    public string ServerName => Transport == HslYamatakeTransport.Serial ? $"{PortName} / Station={Station}" : $"{Host}:{Port} / Station={Station}";
    public HslYamatakeClientOptions ToHslOptions() => new(Transport, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits, Station, ConnectTimeoutMs, ReceiveTimeoutMs);
    public static YamatakeOptions Parse(ConnectionProfile profile, HslYamatakeTransport transport)
    {
        if (!string.Equals(profile.ProtocolFamily, "yamatake", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("CONFIG", "山武 Digitron 连接配置无效");
        var station = Integer(profile.Config, "station") ?? 1; if (station is < 0 or > 255) throw Invalid("STATION", "山武站号必须在 0 到 255 之间");
        var receive = Timeout(profile.Config, "receiveTimeoutMs", 10000);
        if (transport == HslYamatakeTransport.Serial)
        {
            var portName = Text(profile.Config, "portName")?.Trim(); if (string.IsNullOrWhiteSpace(portName)) throw Invalid("PORT_NAME", "山武串口名称不能为空");
            var baudRate = Integer(profile.Config, "baudRate") ?? 9600; var dataBits = Integer(profile.Config, "dataBits") ?? 8;
            var parity = EnumText(profile.Config, "parity", "none") switch { "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, "none" => HslSerialParity.None, _ => throw Invalid("PARITY", "山武串口校验方式无效") };
            var stopBits = Integer(profile.Config, "stopBits") ?? 1; if (stopBits is < 1 or > 2) throw Invalid("STOP_BITS", "山武串口停止位无效");
            return new(transport, "", 0, portName, baudRate, dataBits, parity, stopBits == 2 ? HslSerialStopBits.Two : HslSerialStopBits.One, checked((byte)station), 5000, receive);
        }
        var host = Text(profile.Config, "host")?.Trim(); if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)) throw Invalid("HOST", "山武设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? 2000; if (port is < 1 or > 65535) throw Invalid("PORT", "山武端口必须在 1 到 65535 之间");
        return new(transport, host, port, null, 9600, 8, HslSerialParity.None, HslSerialStopBits.One, checked((byte)station), Timeout(profile.Config, "connectTimeoutMs", 5000), receive);
    }
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("TIMEOUT", "山武超时必须在 100 到 120000 毫秒之间"); return value; }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static string EnumText(JsonElement config, string name, string fallback) => Text(config, name)?.Trim().ToLowerInvariant() ?? fallback;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static YamatakeDriverException Invalid(string part, string message) => new($"YAMATAKE_{part}_INVALID", message, false);
}

internal sealed partial record YamatakeAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static YamatakeAddress Parse(PointReadRequest point) { if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String) throw Invalid(); var source = value.GetString()?.Trim() ?? string.Empty; var match = AddressRegex().Match(source); if (!match.Success || point.ElementCount is < 1 or > 65535) throw Invalid(); var prefix = match.Groups[1].Success ? $"s={byte.Parse(match.Groups[1].Value, CultureInfo.InvariantCulture)};" : ""; var address = prefix + long.Parse(match.Groups[2].Value, CultureInfo.InvariantCulture).ToString(CultureInfo.InvariantCulture); return new(address, Type(point.DataType), point.ElementCount); }
    public HslYamatakeReadRequest ToRequest() => new(Address, DataType, ElementCount);
    private static HslValueType Type(string value) => value switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw new YamatakeDriverException("YAMATAKE_DATA_TYPE_UNSUPPORTED", "山武数据类型不受支持", false) };
    private static YamatakeDriverException Invalid() => new("YAMATAKE_ADDRESS_INVALID", "山武 Digitron 地址必须是数字，可选使用 s=<站号>; 前缀", false);
    [GeneratedRegex("^(?:s=([0-9]{1,3});)?([0-9]+)$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)] private static partial Regex AddressRegex();
}

internal sealed class YamatakeDriverException(string code, string message, bool retryable) : Exception(message) { public string Code { get; } = code; public bool Retryable { get; } = retryable; }
