using System.Diagnostics;
using System.Globalization;
using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OrientalMotor;

public sealed class OrientalMotorEipDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly IHslSpecializedNetworkClientFactory _factory;
    public OrientalMotorEipDriver() : this(new HslSpecializedNetworkClientFactory()) { }
    internal OrientalMotorEipDriver(IHslSpecializedNetworkClientFactory factory) => _factory = factory;
    public DriverDescriptor Descriptor { get; } = new("oriental-motor", "oriental-motor.eip", "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var options = OrientalMotorOptions.Parse(profile); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken); if (!result.Succeeded) { await client.DisposeAsync(); throw new OrientalMotorDriverException(result.ErrorCode ?? "ORIENTAL_MOTOR_CONNECT_FAILED", result.ErrorMessage ?? "无法连接东方马达控制器", result.Retryable); } return new Session(client, $"{options.Host}:{options.Port}"); }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
    private sealed class Session(IHslSpecializedNetworkClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken) { var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count]; for (var index = 0; index < request.Points.Count; index++) { var point = request.Points[index]; try { var address = OrientalMotorAddress.Parse(point); var result = await client.ReadAsync(address.ToRequest(), cancellationToken); values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : Fail(point, result.ErrorCode ?? "ORIENTAL_MOTOR_READ_FAILED", result.ErrorMessage ?? "东方马达变量读取失败"); } catch (OrientalMotorDriverException exception) { values[index] = Fail(point, exception.Code, exception.Message); } } return new(values, []); }
        public ValueTask DisposeAsync() => client.DisposeAsync(); private static PointReadValue Fail(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}

internal sealed record OrientalMotorOptions(string Host, int Port, uint RunIdleHeader, uint RpiTimeMs, int ActualTimeoutSeconds, int ConnectTimeoutMs, int ReceiveTimeoutMs)
{
    public HslSpecializedNetworkClientOptions ToHslOptions() => new(HslSpecializedNetworkProtocol.OrientalMotorEip, Host, Port, ConnectTimeoutMs, ReceiveTimeoutMs, RunIdleHeader, RpiTimeMs, ActualTimeoutSeconds);
    public static OrientalMotorOptions Parse(ConnectionProfile profile) { if (!string.Equals(profile.ProtocolFamily, "oriental-motor", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("CONFIG", "东方马达连接配置无效"); var host = Text(profile.Config, "host")?.Trim(); if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)) throw Invalid("HOST", "东方马达设备 IP / 主机名无效"); var port = Integer(profile.Config, "port") ?? 44818; if (port is < 1 or > 65535) throw Invalid("PORT", "东方马达端口必须在 1 到 65535 之间"); var runIdle = Integer(profile.Config, "runIdleHeader") ?? 1; if (runIdle < 0) throw Invalid("RUN_IDLE", "Run/Idle Header 不能小于 0"); var rpi = Integer(profile.Config, "rpiTimeMs") ?? 100; if (rpi is < 1 or > 60000) throw Invalid("RPI", "RPI 时间必须在 1 到 60000 毫秒之间"); var actual = Integer(profile.Config, "actualTimeoutSeconds") ?? 2; if (!new[] { 1, 2, 4, 8, 16, 32, 64, 128, 256 }.Contains(actual)) throw Invalid("ACTUAL_TIMEOUT", "EIP 实际超时必须是 1 到 256 的二次幂"); return new(host, port, checked((uint)runIdle), checked((uint)rpi), actual, Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000)); }
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("TIMEOUT", "东方马达超时必须在 100 到 120000 毫秒之间"); return value; }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static OrientalMotorDriverException Invalid(string part, string message) => new($"ORIENTAL_MOTOR_{part}_INVALID", message, false);
}

internal sealed partial record OrientalMotorAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static OrientalMotorAddress Parse(PointReadRequest point) { if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String) throw Invalid(); var match = AddressRegex().Match(value.GetString()?.Trim() ?? string.Empty); if (!match.Success || point.ElementCount is < 1 or > 65535) throw Invalid(); return new($"{match.Groups[1].Value.ToLowerInvariant()}[{long.Parse(match.Groups[2].Value, CultureInfo.InvariantCulture).ToString(CultureInfo.InvariantCulture)}]", Type(point.DataType), point.ElementCount); }
    public HslSpecializedNetworkReadRequest ToRequest() => new(Address, DataType, ElementCount);
    private static HslValueType Type(string value) => value switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw new OrientalMotorDriverException("ORIENTAL_MOTOR_DATA_TYPE_UNSUPPORTED", "东方马达数据类型不受支持", false) };
    private static OrientalMotorDriverException Invalid() => new("ORIENTAL_MOTOR_ADDRESS_INVALID", "东方马达地址必须使用 input[n] 或 output[n]", false);
    [GeneratedRegex("^(input|output)\\[([0-9]+)\\]$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)] private static partial Regex AddressRegex();
}
internal sealed class OrientalMotorDriverException(string code, string message, bool retryable) : Exception(message) { public string Code { get; } = code; public bool Retryable { get; } = retryable; }
