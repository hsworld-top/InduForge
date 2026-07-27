using System.Diagnostics;
using System.Globalization;
using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Yokogawa;

public sealed class YokogawaLinkTcpDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly IHslYokogawaClientFactory _factory;
    public YokogawaLinkTcpDriver() : this(new HslYokogawaClientFactory()) { }
    internal YokogawaLinkTcpDriver(IHslYokogawaClientFactory factory) { _factory = factory; Descriptor = new("yokogawa", "yokogawa.link-tcp", "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]); }
    public DriverDescriptor Descriptor { get; }
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = YokogawaOptions.Parse(profile); var client = _factory.Create(options); var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded) { await client.DisposeAsync(); throw new YokogawaDriverException(result.ErrorCode ?? "YOKOGAWA_CONNECT_FAILED", result.ErrorMessage ?? "无法连接横河 PLC"); }
        return new YokogawaSession(client, $"{options.Host}:{options.Port} / CPU={options.CpuNumber}");
    }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
}

internal static class YokogawaOptions
{
    public static HslYokogawaClientOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "yokogawa", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw new YokogawaDriverException("YOKOGAWA_CONFIG_INVALID", "横河 PLC 连接配置无效");
        var host = Text(profile.Config, "host")?.Trim(); if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)) throw new YokogawaDriverException("YOKOGAWA_HOST_INVALID", "横河设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? 12289; if (port is < 1 or > 65535) throw new YokogawaDriverException("YOKOGAWA_PORT_INVALID", "横河端口必须在 1 到 65535 之间");
        var cpu = Integer(profile.Config, "cpuNumber") ?? 1; if (cpu is < 0 or > 255) throw new YokogawaDriverException("YOKOGAWA_CPU_INVALID", "横河 CPU 编号必须在 0 到 255 之间");
        return new(host, port, checked((byte)cpu), Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000), HslDataFormat.CDAB);
    }
    private static int Timeout(JsonElement c, string n, int f) { var v = Integer(c, n) ?? f; if (v is < 100 or > 120000) throw new YokogawaDriverException("YOKOGAWA_TIMEOUT_INVALID", "横河超时必须在 100 到 120000 毫秒之间"); return v; }
    private static string? Text(JsonElement c, string n) => c.TryGetProperty(n, out var v) && v.ValueKind == JsonValueKind.String ? v.GetString() : null;
    private static int? Integer(JsonElement c, string n) => c.TryGetProperty(n, out var v) && v.TryGetInt32(out var r) ? r : null;
}

internal sealed partial record YokogawaAddress(string Address, HslValueType Type, int Count)
{
    public static YokogawaAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String) throw new YokogawaDriverException("YOKOGAWA_ADDRESS_INVALID", "横河地址必须包含 address 字段");
        var source = value.GetString()?.Trim() ?? string.Empty; var normalized = Normalize(source); var type = ParseType(point.DataType); if (point.ElementCount is < 1 or > 65535) throw new YokogawaDriverException("YOKOGAWA_ELEMENT_COUNT_INVALID", "横河元素数量必须在 1 到 65535 之间");
        if (normalized.StartsWith("Special:", StringComparison.Ordinal)) { if (type == HslValueType.Boolean) throw new YokogawaDriverException("YOKOGAWA_SPECIAL_TYPE_INVALID", "横河特殊模块地址不支持 bool 类型"); return new(normalized, type, point.ElementCount); }
        var body = normalized.Contains(';') ? normalized[(normalized.IndexOf(';') + 1)..] : normalized; var area = AreaRegex().Match(body).Value; var bit = area is "X" or "Y" or "I" or "E" or "M" or "T" or "C" or "L";
        if (type == HslValueType.Boolean && !bit) throw new YokogawaDriverException("YOKOGAWA_BOOL_AREA_INVALID", "横河 bool 变量必须使用继电器地址"); if (type != HslValueType.Boolean && bit) throw new YokogawaDriverException("YOKOGAWA_WORD_AREA_INVALID", "横河非 bool 变量必须使用寄存器地址"); return new(normalized, type, point.ElementCount);
    }
    public HslYokogawaReadRequest ToRequest() => new(Address, Type, Count);
    private static string Normalize(string s)
    {
        var special = SpecialRegex().Match(s); if (special.Success) return $"Special:{(special.Groups[1].Success ? $"cpu={byte.Parse(special.Groups[1].Value, CultureInfo.InvariantCulture)};" : "")}unit={byte.Parse(special.Groups[2].Value, CultureInfo.InvariantCulture)};slot={byte.Parse(special.Groups[3].Value, CultureInfo.InvariantCulture)};{uint.Parse(special.Groups[4].Value, CultureInfo.InvariantCulture)}";
        var normal = NormalRegex().Match(s); if (!normal.Success) throw new YokogawaDriverException("YOKOGAWA_ADDRESS_INVALID", "横河 PLC 设备地址无效"); return (normal.Groups[1].Success ? $"cpu={byte.Parse(normal.Groups[1].Value, CultureInfo.InvariantCulture)};" : "") + normal.Groups[2].Value.ToUpperInvariant();
    }
    private static HslValueType ParseType(string t) => t switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw new YokogawaDriverException("YOKOGAWA_DATA_TYPE_UNSUPPORTED", "横河数据类型不受支持") };
    [GeneratedRegex("^(?:cpu=([0-9]+);)?((?:X|Y|I|E|M|T|C|L|D|B|F|R|V|Z|W|TN|CN)[0-9]+)$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)] private static partial Regex NormalRegex();
    [GeneratedRegex("^Special:(?:cpu=([0-9]+);)?unit=([0-9]+);slot=([0-9]+);([0-9]+)$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)] private static partial Regex SpecialRegex();
    [GeneratedRegex("^[A-Z]+", RegexOptions.CultureInvariant)] private static partial Regex AreaRegex();
}

internal sealed class YokogawaSession(IHslYokogawaClient client, string name) : IIndustrialConnectionSession, IPointReaderSession
{
    public bool IsConnected => client.IsConnected; public string? ServerName { get; } = name;
    public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken token) { var at = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count]; for (var i = 0; i < request.Points.Count; i++) { var p = request.Points[i]; YokogawaAddress a; try { a = YokogawaAddress.Parse(p); } catch (YokogawaDriverException e) { values[i] = Fail(p, e.Code, e.Message); continue; } var r = await client.ReadAsync(a.ToRequest(), token); values[i] = r.Succeeded ? new(p.Key, true, r.Value, p.DataType, "Good", at, at, null, null) : Fail(p, r.ErrorCode ?? "YOKOGAWA_READ_FAILED", r.ErrorMessage ?? "横河变量读取失败"); } return new(values, []); }
    public ValueTask DisposeAsync() => client.DisposeAsync(); private static PointReadValue Fail(PointReadRequest p, string c, string m) => new(p.Key, false, null, p.DataType, "Bad", null, null, c, m);
}
internal sealed class YokogawaDriverException(string code, string message) : Exception(message) { public string Code { get; } = code; }
