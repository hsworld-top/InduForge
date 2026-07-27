using System.Diagnostics;
using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.PanasonicMcBinary;

public sealed class PanasonicMcBinaryDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly IHslPanasonicMcClientFactory _clientFactory;
    public PanasonicMcBinaryDriver() : this(new HslPanasonicMcClientFactory()) { }
    internal PanasonicMcBinaryDriver(IHslPanasonicMcClientFactory clientFactory) => _clientFactory = clientFactory;
    public DriverDescriptor Descriptor { get; } = new("panasonic", "panasonic.mc-binary-tcp", "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var stopwatch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        stopwatch.Stop();
        return new(session.IsConnected, stopwatch.Elapsed, session.ServerName, []);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = PanasonicMcOptions.Parse(profile);
        var client = _clientFactory.Create(options);
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new PanasonicMcException(result.ErrorCode ?? "PANASONIC_MC_CONNECT_FAILED", result.ErrorMessage ?? "无法连接松下 MC Binary 设备", result.Retryable);
        }
        return new PanasonicMcSession(client, $"{options.Host}:{options.Port}");
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }
}

internal static class PanasonicMcOptions
{
    public static HslPanasonicMcClientOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "panasonic", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("PANASONIC_MC_CONFIG_INVALID", "松下 MC Binary 连接配置无效");
        var host = Text(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)) throw Invalid("PANASONIC_MC_HOST_INVALID", "松下 MC Binary 设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? 5000;
        if (port is < 1 or > 65535) throw Invalid("PANASONIC_MC_PORT_INVALID", "松下 MC Binary 端口必须在 1 到 65535 之间");
        var networkStation = Integer(profile.Config, "networkStationNumber") ?? 0;
        if (networkStation is < 0 or > 255) throw Invalid("PANASONIC_MC_NETWORK_STATION_INVALID", "松下 MC Binary 网络站号必须在 0 到 255 之间");
        var targetIo = Integer(profile.Config, "targetIoStation") ?? 1023;
        if (targetIo is < 0 or > 65535) throw Invalid("PANASONIC_MC_TARGET_IO_INVALID", "松下 MC Binary 目标 IO 站号必须在 0 到 65535 之间");
        return new(host, port, Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000), checked((byte)networkStation), checked((ushort)targetIo), HslDataFormat.DCBA);
    }
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("PANASONIC_MC_TIMEOUT_INVALID", "松下 MC Binary 超时必须在 100 到 120000 毫秒之间"); return value; }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static PanasonicMcException Invalid(string code, string message) => new(code, message, false);
}

internal sealed partial record PanasonicMcAddress(string Address, HslValueType ValueType, int ElementCount)
{
    public static PanasonicMcAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
            throw Invalid("PANASONIC_MC_ADDRESS_INVALID", "松下 MC Binary 地址必须包含 address 字段");
        var address = value.GetString()?.Trim().ToUpperInvariant() ?? string.Empty;
        var match = AddressRegex().Match(address);
        if (!match.Success || address.Length > 128) throw Invalid("PANASONIC_MC_ADDRESS_INVALID", "松下 MC Binary 设备地址无效");
        if (point.ElementCount is < 1 or > ushort.MaxValue) throw Invalid("PANASONIC_MC_ELEMENT_COUNT_INVALID", "松下 MC Binary 元素数量必须在 1 到 65535 之间");
        var valueType = ParseType(point.DataType);
        var area = match.Groups[1].Value;
        var bitArea = area is "X" or "Y" or "R" or "TS" or "CS" or "L";
        if (valueType == HslValueType.Boolean && !bitArea) throw Invalid("PANASONIC_MC_BOOL_AREA_INVALID", "松下 MC Binary bool 变量必须使用继电器或触点地址");
        if (valueType != HslValueType.Boolean && bitArea) throw Invalid("PANASONIC_MC_WORD_AREA_INVALID", "松下 MC Binary 非 bool 变量必须使用寄存器地址");
        return new(address, valueType, point.ElementCount);
    }
    public HslPanasonicMcReadRequest ToRequest() => new(Address, ValueType, ElementCount);
    private static HslValueType ParseType(string value) => value switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw Invalid("PANASONIC_MC_DATA_TYPE_UNSUPPORTED", "松下 MC Binary 数据类型不受支持") };
    private static PanasonicMcException Invalid(string code, string message) => new(code, message, false);
    [GeneratedRegex("^(X|Y|R|TS|CS|L|TN|CN|DT|D|LD|SD)[0-9]+(?:\\.[0-9]+)?$", RegexOptions.CultureInvariant)] private static partial Regex AddressRegex();
}

internal sealed class PanasonicMcSession(IHslPanasonicMcClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
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
            PanasonicMcAddress address;
            try { address = PanasonicMcAddress.Parse(point); }
            catch (PanasonicMcException exception) { values[index] = Failure(point, exception.Code, exception.Message); continue; }
            var result = await client.ReadAsync(address.ToRequest(), cancellationToken).ConfigureAwait(false);
            values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : Failure(point, result.ErrorCode ?? "PANASONIC_MC_READ_FAILED", result.ErrorMessage ?? "松下 MC Binary 地址读取失败");
        }
        return new(values, []);
    }
    public ValueTask DisposeAsync() => client.DisposeAsync();
    private static PointReadValue Failure(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
}

internal sealed class PanasonicMcException(string code, string message, bool retryable) : IndustrialDriverException(code, message, retryable);
