using System.Diagnostics;
using System.Globalization;
using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InstrumentMeters;

public sealed class Dlt645SerialDriver : DltDriverBase { public Dlt645SerialDriver() : this(new HslDltClientFactory()) { } internal Dlt645SerialDriver(IHslDltClientFactory factory) : base(HslDltProtocol.Dlt645Serial, factory) { } public override DriverDescriptor Descriptor { get; } = DescriptorFor("dlt645", "dlt645.2007-serial"); }
public sealed class Dlt645OverTcpDriver : DltDriverBase { public Dlt645OverTcpDriver() : this(new HslDltClientFactory()) { } internal Dlt645OverTcpDriver(IHslDltClientFactory factory) : base(HslDltProtocol.Dlt645OverTcp, factory) { } public override DriverDescriptor Descriptor { get; } = DescriptorFor("dlt645", "dlt645.2007-over-tcp"); }
public sealed class Dlt645With1997SerialDriver : DltDriverBase { public Dlt645With1997SerialDriver() : this(new HslDltClientFactory()) { } internal Dlt645With1997SerialDriver(IHslDltClientFactory factory) : base(HslDltProtocol.Dlt645With1997Serial, factory) { } public override DriverDescriptor Descriptor { get; } = DescriptorFor("dlt645", "dlt645.1997-serial"); }
public sealed class Dlt645With1997OverTcpDriver : DltDriverBase { public Dlt645With1997OverTcpDriver() : this(new HslDltClientFactory()) { } internal Dlt645With1997OverTcpDriver(IHslDltClientFactory factory) : base(HslDltProtocol.Dlt645With1997OverTcp, factory) { } public override DriverDescriptor Descriptor { get; } = DescriptorFor("dlt645", "dlt645.1997-over-tcp"); }
public sealed class Dlt698SerialDriver : DltDriverBase { public Dlt698SerialDriver() : this(new HslDltClientFactory()) { } internal Dlt698SerialDriver(IHslDltClientFactory factory) : base(HslDltProtocol.Dlt698Serial, factory) { } public override DriverDescriptor Descriptor { get; } = DescriptorFor("dlt698", "dlt698.serial"); }
public sealed class Dlt698OverTcpDriver : DltDriverBase { public Dlt698OverTcpDriver() : this(new HslDltClientFactory()) { } internal Dlt698OverTcpDriver(IHslDltClientFactory factory) : base(HslDltProtocol.Dlt698OverTcp, factory) { } public override DriverDescriptor Descriptor { get; } = DescriptorFor("dlt698", "dlt698.over-tcp"); }
public sealed class Dlt698TcpNetDriver : DltDriverBase { public Dlt698TcpNetDriver() : this(new HslDltClientFactory()) { } internal Dlt698TcpNetDriver(IHslDltClientFactory factory) : base(HslDltProtocol.Dlt698TcpNet, factory) { } public override DriverDescriptor Descriptor { get; } = DescriptorFor("dlt698", "dlt698.tcp-net"); }

public abstract class DltDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslDltProtocol _protocol;
    private readonly IHslDltClientFactory _factory;
    protected DltDriverBase(HslDltProtocol protocol, IHslDltClientFactory factory) { _protocol = protocol; _factory = factory; }
    public abstract DriverDescriptor Descriptor { get; }
    protected static DriverDescriptor DescriptorFor(string family, string driverId) => new(family, driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = DltOptions.Parse(profile, _protocol); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded) { await client.DisposeAsync(); throw new InstrumentDriverException(result.ErrorCode ?? "DLT_CONNECT_FAILED", result.ErrorMessage ?? "无法连接 DLT 仪表", result.Retryable); }
        return new Session(client, options.ServerName, _protocol);
    }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }

    private sealed class Session(IHslDltClient client, string serverName, HslDltProtocol protocol) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
        {
            var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count];
            // DLT 仪表通常挂在同一总线上，单个数据标识错误不能影响其他表计变量。
            for (var index = 0; index < request.Points.Count; index++)
            {
                var point = request.Points[index];
                try { var parsed = DltAddress.Parse(point, protocol); var result = await client.ReadAsync(parsed.ToRequest(), cancellationToken); values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : Fail(point, result.ErrorCode ?? "DLT_READ_FAILED", result.ErrorMessage ?? "DLT 变量读取失败"); }
                catch (InstrumentDriverException exception) { values[index] = Fail(point, exception.Code, exception.Message); }
            }
            return new(values, []);
        }
        public ValueTask DisposeAsync() => client.DisposeAsync();
        private static PointReadValue Fail(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}

internal sealed record DltOptions(HslDltProtocol Protocol, string Host, int Port, string? PortName, int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits, string Station, string Password, string OpCode, bool EnableCodeFE, bool CheckDataId, bool UseSecurityRequest, byte ClientAddress, int ConnectTimeoutMs, int ReceiveTimeoutMs)
{
    public string ServerName => IsSerial(Protocol) ? PortName! : $"{Host}:{Port}";
    public HslDltClientOptions ToHslOptions() => new(Protocol, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits, Station, Password, OpCode, EnableCodeFE, CheckDataId, UseSecurityRequest, ClientAddress, ConnectTimeoutMs, ReceiveTimeoutMs);
    public static DltOptions Parse(ConnectionProfile profile, HslDltProtocol protocol)
    {
        var family = Is698(protocol) ? "dlt698" : "dlt645";
        if (!string.Equals(profile.ProtocolFamily, family, StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("CONFIG", "DLT 连接配置无效");
        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty; var portName = Text(profile.Config, "portName")?.Trim();
        if (!IsSerial(protocol) && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))) throw Invalid("HOST", "DLT 设备 IP / 主机名无效");
        if (IsSerial(protocol) && string.IsNullOrWhiteSpace(portName)) throw Invalid("PORT_NAME", "DLT 串口名称不能为空");
        var port = Integer(profile.Config, "port") ?? 2000; if (port is < 1 or > 65535) throw Invalid("PORT", "DLT 端口必须在 1 到 65535 之间");
        var station = NormalizeStation(Text(profile.Config, "station") ?? "1");
        var password = NormalizeCode(Text(profile.Config, "password") ?? "00000000", "密码"); var opCode = NormalizeCode(Text(profile.Config, "opCode") ?? "00000000", "操作者代码");
        var baudRate = Integer(profile.Config, "baudRate") ?? 9600; var dataBits = Integer(profile.Config, "dataBits") ?? 8;
        var parityFallback = protocol is HslDltProtocol.Dlt645With1997Serial ? HslSerialParity.None : HslSerialParity.Even;
        var connectTimeout = Integer(profile.Config, "connectTimeoutMs") ?? 5000; var receiveTimeout = Integer(profile.Config, "receiveTimeoutMs") ?? 10000;
        var clientAddress = Integer(profile.Config, "clientAddress") ?? 0;
        if (baudRate <= 0 || dataBits is < 5 or > 8 || clientAddress is < 0 or > 255 || connectTimeout is < 100 or > 120000 || receiveTimeout is < 100 or > 120000) throw Invalid("PARAMETER", "DLT 连接参数无效");
        return new(protocol, host, port, portName, baudRate, dataBits, ParseParity(Text(profile.Config, "parity"), parityFallback), ParseStopBits(Integer(profile.Config, "stopBits") ?? 1), station, password, opCode, Boolean(profile.Config, "enableCodeFE"), Boolean(profile.Config, "checkDataId", true), Boolean(profile.Config, "useSecurityRequest", true), (byte)clientAddress, connectTimeout, receiveTimeout);
    }
    private static string NormalizeStation(string value) { var station = value.Trim().ToUpperInvariant(); if (!Regex.IsMatch(station, "^[0-9A]{1,12}$", RegexOptions.CultureInvariant)) throw Invalid("STATION", "DLT 表地址必须是最多 12 位数字，可使用 A 作为通配符"); return station.PadLeft(12, '0'); }
    private static string NormalizeCode(string value, string name) { var code = value.Trim().ToUpperInvariant(); if (!Regex.IsMatch(code, "^[0-9A-F]{8}$", RegexOptions.CultureInvariant)) throw Invalid("CODE", $"DLT {name}必须是 8 位十六进制数"); return code; }
    private static bool IsSerial(HslDltProtocol protocol) => protocol is HslDltProtocol.Dlt645Serial or HslDltProtocol.Dlt645With1997Serial or HslDltProtocol.Dlt698Serial;
    private static bool Is698(HslDltProtocol protocol) => protocol is HslDltProtocol.Dlt698Serial or HslDltProtocol.Dlt698OverTcp or HslDltProtocol.Dlt698TcpNet;
    private static HslSerialParity ParseParity(string? value, HslSerialParity fallback) => value?.Trim().ToLowerInvariant() switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, null or "" => fallback, _ => throw Invalid("PARITY", "DLT 校验方式无效") };
    private static HslSerialStopBits ParseStopBits(int value) => value switch { 1 => HslSerialStopBits.One, 2 => HslSerialStopBits.Two, _ => throw Invalid("STOP_BITS", "DLT 停止位只支持 1 或 2") };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static bool Boolean(JsonElement config, string name, bool fallback = false) => config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : fallback;
    private static InstrumentDriverException Invalid(string part, string message) => new($"DLT_{part}_INVALID", message, false);
}

internal sealed partial record DltAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static DltAddress Parse(PointReadRequest point, HslDltProtocol protocol)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String || point.ElementCount is < 1 or > 65535) throw Invalid();
        var source = value.GetString()?.Trim() ?? string.Empty; var byteCount = protocol is HslDltProtocol.Dlt645With1997Serial or HslDltProtocol.Dlt645With1997OverTcp ? 2 : 4;
        var match = AddressPattern().Match(source); if (!match.Success || match.Groups[2].Value.Split('-').Length != byteCount) throw Invalid();
        var prefix = match.Groups[1].Success ? $"s={NormalizeStation(match.Groups[1].Value)};" : string.Empty;
        var type = Is698(protocol) ? Type698(point.DataType) : Type645(point.DataType);
        return new(prefix + match.Groups[2].Value.ToUpperInvariant(), type, point.ElementCount);
    }
    public HslDltReadRequest ToRequest() => new(Address, DataType, ElementCount);
    private static string NormalizeStation(string value) => value.Trim().ToUpperInvariant().PadLeft(12, '0');
    private static bool Is698(HslDltProtocol protocol) => protocol is HslDltProtocol.Dlt698Serial or HslDltProtocol.Dlt698OverTcp or HslDltProtocol.Dlt698TcpNet;
    private static HslValueType Type645(string value) => value switch { "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw new InstrumentDriverException("DLT645_DATA_TYPE_UNSUPPORTED", "DLT645 数据类型仅支持 float64、string 和 bytes", false) };
    private static HslValueType Type698(string value) => value switch { "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw new InstrumentDriverException("DLT698_DATA_TYPE_UNSUPPORTED", "DLT698 数据类型不受支持", false) };
    private static InstrumentDriverException Invalid() => new("DLT_ADDRESS_INVALID", "DLT645-2007/698 地址应为 00-00-00-00，DLT645-1997 地址应为 B6-11，可选使用 s=<表地址>; 前缀", false);
    [GeneratedRegex("^(?:s=([0-9Aa]{1,12});)?((?:[0-9A-Fa-f]{2}-){1,3}[0-9A-Fa-f]{2})$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)] private static partial Regex AddressPattern();
}
