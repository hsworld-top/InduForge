using System.Diagnostics;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Dcs;

public sealed class DcsNanJingAutoDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly IHslSpecializedNetworkClientFactory _factory;

    public DcsNanJingAutoDriver() : this(new HslSpecializedNetworkClientFactory()) { }

    internal DcsNanJingAutoDriver(IHslSpecializedNetworkClientFactory factory) => _factory = factory;

    public DriverDescriptor Descriptor { get; } = new(
        "dcs",
        "dcs.nanjing-auto",
        "1.0.0",
        [1],
        [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        watch.Stop();
        return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = DcsNanJingAutoOptions.Parse(profile);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken).ConfigureAwait(false);
        if (!result.Succeeded)
        {
            await client.DisposeAsync().ConfigureAwait(false);
            throw new DcsDriverException(result.ErrorCode ?? "DCS_NANJING_AUTO_CONNECT_FAILED", result.ErrorMessage ?? "南京自动化 DCS 连接失败", result.Retryable);
        }
        return new Session(client, options.ServerName);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken).ConfigureAwait(false);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken).ConfigureAwait(false);
    }

    private sealed class Session(IHslSpecializedNetworkClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
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
                    var address = DcsNanJingAutoAddress.Parse(point);
                    var result = await client.ReadAsync(address.ToRequest(), cancellationToken).ConfigureAwait(false);
                    values[index] = result.Succeeded
                        ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                        : Fail(point, result.ErrorCode ?? "DCS_NANJING_AUTO_READ_FAILED", result.ErrorMessage ?? "南京自动化 DCS 变量读取失败");
                }
                catch (DcsDriverException exception)
                {
                    values[index] = Fail(point, exception.Code, exception.Message);
                }
            }
            return new(values, []);
        }

        public ValueTask DisposeAsync() => client.DisposeAsync();

        private static PointReadValue Fail(PointReadRequest point, string code, string message) =>
            new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}

internal sealed record DcsNanJingAutoOptions(
    string Host,
    int Port,
    byte Station,
    bool AddressStartWithZero,
    HslDataFormat DataFormat,
    bool IsStringReverse,
    int ConnectTimeoutMs,
    int ReceiveTimeoutMs)
{
    public string ServerName => $"{Host}:{Port}";

    public HslSpecializedNetworkClientOptions ToHslOptions() => new(
        HslSpecializedNetworkProtocol.DcsNanJingAuto,
        Host,
        Port,
        ConnectTimeoutMs,
        ReceiveTimeoutMs,
        Station: Station,
        AddressStartWithZero: AddressStartWithZero,
        DataFormat: DataFormat,
        IsStringReverse: IsStringReverse);

    public static DcsNanJingAutoOptions Parse(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolFamily, "dcs", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object)
        {
            throw Invalid("CONFIG", "南京自动化 DCS 连接配置无效");
        }
        var host = Text(profile.Config, "host")?.Trim();
        if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))
        {
            throw Invalid("HOST", "南京自动化 DCS 设备 IP / 主机名无效");
        }
        var port = Integer(profile.Config, "port") ?? 502;
        var station = Integer(profile.Config, "station") ?? 1;
        if (port is < 1 or > 65535 || station is < 0 or > 255)
        {
            throw Invalid("PARAMETER", "南京自动化 DCS 端口或站号无效");
        }
        return new(
            host,
            port,
            (byte)station,
            Boolean(profile.Config, "addressStartWithZero", true),
            ParseDataFormat(Text(profile.Config, "dataFormat")),
            Boolean(profile.Config, "stringReverse"),
            Timeout(profile.Config, "connectTimeoutMs", 5000),
            Timeout(profile.Config, "receiveTimeoutMs", 10000));
    }

    private static HslDataFormat ParseDataFormat(string? value) => value?.Trim().ToUpperInvariant() switch
    {
        null or "" or "ABCD" => HslDataFormat.ABCD,
        "BADC" => HslDataFormat.BADC,
        "CDAB" => HslDataFormat.CDAB,
        "DCBA" => HslDataFormat.DCBA,
        _ => throw Invalid("DATA_FORMAT", "南京自动化 DCS 数据格式无效"),
    };

    private static int Timeout(JsonElement config, string name, int fallback)
    {
        var value = Integer(config, name) ?? fallback;
        if (value is < 100 or > 120000) throw Invalid("TIMEOUT", "南京自动化 DCS 超时必须在 100 到 120000 毫秒之间");
        return value;
    }

    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static bool Boolean(JsonElement config, string name, bool fallback = false) => config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : fallback;
    private static DcsDriverException Invalid(string part, string message) => new($"DCS_NANJING_AUTO_{part}_INVALID", message, false);
}

internal sealed record DcsNanJingAutoAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static DcsNanJingAutoAddress Parse(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("address", out var value) ||
            value.ValueKind != JsonValueKind.String ||
            point.ElementCount is < 1 or > 65535)
        {
            throw Invalid();
        }
        var address = value.GetString()?.Trim() ?? string.Empty;
        if (address.Length is < 1 or > 128 || address.Any(char.IsWhiteSpace)) throw Invalid();
        return new(address, ParseDataType(point.DataType), point.ElementCount);
    }

    public HslSpecializedNetworkReadRequest ToRequest() => new(Address, DataType, ElementCount);

    private static HslValueType ParseDataType(string value) => value switch
    {
        "bool" => HslValueType.Boolean,
        "int8" => HslValueType.Signed8,
        "uint8" => HslValueType.Unsigned8,
        "int16" => HslValueType.Signed16,
        "uint16" => HslValueType.Unsigned16,
        "int32" => HslValueType.Signed32,
        "uint32" => HslValueType.Unsigned32,
        "int64" => HslValueType.Signed64,
        "uint64" => HslValueType.Unsigned64,
        "float32" => HslValueType.SinglePrecision,
        "float64" => HslValueType.DoublePrecision,
        "string" => HslValueType.Text,
        "bytes" => HslValueType.Binary,
        _ => throw new DcsDriverException("DCS_NANJING_AUTO_DATA_TYPE_UNSUPPORTED", "南京自动化 DCS 数据类型不受支持", false),
    };

    private static DcsDriverException Invalid() => new("DCS_NANJING_AUTO_ADDRESS_INVALID", "南京自动化 DCS 地址无效", false);
}

internal sealed class DcsDriverException(string code, string message, bool retryable) : Exception(message)
{
    public string Code { get; } = code;
    public bool Retryable { get; } = retryable;
}
