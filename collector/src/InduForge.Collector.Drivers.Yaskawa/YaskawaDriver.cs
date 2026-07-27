using System.Diagnostics;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Yaskawa;

public sealed class YaskawaMemobusTcpDriver : YaskawaDriverBase { public YaskawaMemobusTcpDriver() : base("yaskawa.memobus-tcp", HslYaskawaTransport.Tcp) { } internal YaskawaMemobusTcpDriver(IHslYaskawaClientFactory factory) : base("yaskawa.memobus-tcp", HslYaskawaTransport.Tcp, factory) { } }
public sealed class YaskawaMemobusUdpDriver : YaskawaDriverBase { public YaskawaMemobusUdpDriver() : base("yaskawa.memobus-udp", HslYaskawaTransport.Udp) { } internal YaskawaMemobusUdpDriver(IHslYaskawaClientFactory factory) : base("yaskawa.memobus-udp", HslYaskawaTransport.Udp, factory) { } }

public abstract class YaskawaDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslYaskawaTransport _transport; private readonly IHslYaskawaClientFactory _factory;
    private protected YaskawaDriverBase(string driverId, HslYaskawaTransport transport, IHslYaskawaClientFactory? factory = null) { _transport = transport; _factory = factory ?? new HslYaskawaClientFactory(); Descriptor = new("yaskawa", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]); }
    public DriverDescriptor Descriptor { get; }
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var options = YaskawaOptions.Parse(profile, _transport); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken); if (!result.Succeeded) { await client.DisposeAsync(); throw new YaskawaDriverException(result.ErrorCode ?? "YASKAWA_CONNECT_FAILED", result.ErrorMessage ?? "无法连接安川 Memobus", result.Retryable); } return new Session(client, options.ServerName); }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
    private sealed class Session(IHslYaskawaClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken) { var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count]; for (var index = 0; index < request.Points.Count; index++) { var point = request.Points[index]; try { var address = YaskawaAddress.Parse(point); var result = await client.ReadAsync(address.ToRequest(), cancellationToken); values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : Fail(point, result.ErrorCode ?? "YASKAWA_READ_FAILED", result.ErrorMessage ?? "安川变量读取失败"); } catch (YaskawaDriverException exception) { values[index] = Fail(point, exception.Code, exception.Message); } } return new(values, []); }
        public ValueTask DisposeAsync() => client.DisposeAsync(); private static PointReadValue Fail(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}

internal sealed record YaskawaOptions(HslYaskawaTransport Transport, string Host, int Port, byte CpuFrom, byte CpuTo, int ConnectTimeoutMs, int ReceiveTimeoutMs, HslDataFormat DataFormat)
{
    public string ServerName => $"{Host}:{Port} / CPU {CpuFrom}->{CpuTo}";
    public HslYaskawaClientOptions ToHslOptions() => new(Transport, Host, Port, CpuFrom, CpuTo, ConnectTimeoutMs, ReceiveTimeoutMs, DataFormat);
    public static YaskawaOptions Parse(ConnectionProfile profile, HslYaskawaTransport transport) { if (!string.Equals(profile.ProtocolFamily, "yaskawa", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("CONFIG", "安川 Memobus 连接配置无效"); var host = Text(profile.Config, "host")?.Trim(); if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)) throw Invalid("HOST", "安川设备 IP / 主机名无效"); var port = Integer(profile.Config, "port") ?? 9999; if (port is < 1 or > 65535) throw Invalid("PORT", "安川端口必须在 1 到 65535 之间"); var cpuFrom = Integer(profile.Config, "cpuFrom") ?? 1; var cpuTo = Integer(profile.Config, "cpuTo") ?? 2; if (cpuFrom is < 0 or > 255 || cpuTo is < 0 or > 255) throw Invalid("CPU", "安川 CPU 编号必须在 0 到 255 之间"); return new(transport, host, port, checked((byte)cpuFrom), checked((byte)cpuTo), Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000), Format(profile.Config)); }
    private static HslDataFormat Format(JsonElement config) => (Text(config, "dataFormat") ?? "CDAB").ToUpperInvariant() switch { "ABCD" => HslDataFormat.ABCD, "BADC" => HslDataFormat.BADC, "CDAB" => HslDataFormat.CDAB, "DCBA" => HslDataFormat.DCBA, _ => throw Invalid("DATA_FORMAT", "安川数据格式无效") };
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("TIMEOUT", "安川超时必须在 100 到 120000 毫秒之间"); return value; }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static YaskawaDriverException Invalid(string part, string message) => new($"YASKAWA_{part}_INVALID", message, false);
}

internal sealed record YaskawaAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static YaskawaAddress Parse(PointReadRequest point) { if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String) throw Invalid(); var address = value.GetString()?.Trim() ?? string.Empty; if (address.Length is < 1 or > 128 || address.Any(char.IsWhiteSpace) || point.ElementCount is < 1 or > 65535) throw Invalid(); return new(address, Type(point.DataType), point.ElementCount); }
    public HslYaskawaReadRequest ToRequest() => new(Address, DataType, ElementCount);
    private static HslValueType Type(string value) => value switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw new YaskawaDriverException("YASKAWA_DATA_TYPE_UNSUPPORTED", "安川数据类型不受支持", false) };
    private static YaskawaDriverException Invalid() => new("YASKAWA_ADDRESS_INVALID", "安川 Memobus 地址无效", false);
}

internal sealed class YaskawaDriverException(string code, string message, bool retryable) : Exception(message) { public string Code { get; } = code; public bool Retryable { get; } = retryable; }
