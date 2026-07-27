using System.Diagnostics;
using System.Globalization;
using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InstrumentMeters;

public sealed class Dam3601SerialDriver : InstrumentSerialDriverBase
{
    public Dam3601SerialDriver() : this(new HslInstrumentSerialClientFactory()) { }
    internal Dam3601SerialDriver(IHslInstrumentSerialClientFactory factory) : base(InstrumentSerialKind.Dam3601, factory) { }
    public override DriverDescriptor Descriptor { get; } = DescriptorFor("dam3601", "dam3601.serial");
}

public sealed class YuDianAiBusDriver : InstrumentSerialDriverBase
{
    public YuDianAiBusDriver() : this(new HslInstrumentSerialClientFactory()) { }
    internal YuDianAiBusDriver(IHslInstrumentSerialClientFactory factory) : base(InstrumentSerialKind.YuDian, factory) { }
    public override DriverDescriptor Descriptor { get; } = DescriptorFor("yudian", "yudian.ai-bus");
}

public sealed class DelixiDtsu6606Driver : InstrumentSerialDriverBase
{
    public DelixiDtsu6606Driver() : this(new HslInstrumentSerialClientFactory()) { }
    internal DelixiDtsu6606Driver(IHslInstrumentSerialClientFactory factory) : base(InstrumentSerialKind.Dtsu6606, factory) { }
    public override DriverDescriptor Descriptor { get; } = DescriptorFor("delixi", "delixi.dtsu6606");
}

public enum InstrumentSerialKind { Dam3601, YuDian, Dtsu6606 }

public abstract class InstrumentSerialDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly InstrumentSerialKind _kind;
    private readonly IHslInstrumentSerialClientFactory _factory;

    protected InstrumentSerialDriverBase(InstrumentSerialKind kind, IHslInstrumentSerialClientFactory factory)
    {
        _kind = kind;
        _factory = factory;
    }

    public abstract DriverDescriptor Descriptor { get; }
    protected static DriverDescriptor DescriptorFor(string family, string driverId) => new(family, driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        watch.Stop();
        return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = InstrumentSerialOptions.Parse(profile, _kind);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded)
        {
            await client.DisposeAsync();
            throw new InstrumentDriverException(result.ErrorCode ?? "INSTRUMENT_CONNECT_FAILED", result.ErrorMessage ?? "无法连接串口仪表", result.Retryable);
        }
        return new Session(client, options.PortName, _kind);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken);
    }

    private sealed class Session(IHslInstrumentSerialClient client, string portName, InstrumentSerialKind kind) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected;
        public string? ServerName { get; } = portName;

        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
        {
            var readAt = DateTimeOffset.UtcNow;
            var values = new PointReadValue[request.Points.Count];
            for (var index = 0; index < request.Points.Count; index++)
            {
                var point = request.Points[index];
                try
                {
                    var parsed = InstrumentSerialAddress.Parse(point, kind);
                    var result = await client.ReadAsync(parsed.ToRequest(), cancellationToken);
                    values[index] = result.Succeeded
                        ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                        : Fail(point, result.ErrorCode ?? "INSTRUMENT_READ_FAILED", result.ErrorMessage ?? "串口仪表变量读取失败");
                }
                catch (InstrumentDriverException exception)
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

internal sealed record InstrumentSerialOptions(string PortName, int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits, byte Station, int ReceiveTimeoutMs, HslDataFormat DataFormat, bool StringReverse, bool AddressStartWithZero, InstrumentSerialKind Kind)
{
    public HslInstrumentSerialClientOptions ToHslOptions() => new(Kind switch { InstrumentSerialKind.Dam3601 => HslInstrumentSerialProtocol.Dam3601, InstrumentSerialKind.YuDian => HslInstrumentSerialProtocol.YuDianAiBus, _ => HslInstrumentSerialProtocol.Dtsu6606 }, PortName, BaudRate, DataBits, Parity, StopBits, Station, ReceiveTimeoutMs, DataFormat, StringReverse, AddressStartWithZero);

    public static InstrumentSerialOptions Parse(ConnectionProfile profile, InstrumentSerialKind kind)
    {
        var family = kind switch { InstrumentSerialKind.Dam3601 => "dam3601", InstrumentSerialKind.YuDian => "yudian", _ => "delixi" };
        if (!string.Equals(profile.ProtocolFamily, family, StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid(kind, "CONFIG", "串口仪表连接配置无效");
        var portName = Text(profile.Config, "portName")?.Trim();
        if (string.IsNullOrWhiteSpace(portName)) throw Invalid(kind, "PORT_NAME", "串口名称不能为空");
        var defaultBaudRate = kind == InstrumentSerialKind.Dtsu6606 ? 2400 : 9600;
        var defaultParity = kind == InstrumentSerialKind.Dtsu6606 ? HslSerialParity.Even : HslSerialParity.None;
        var defaultFormat = kind == InstrumentSerialKind.Dam3601 ? HslDataFormat.CDAB : HslDataFormat.ABCD;
        var baudRate = Integer(profile.Config, "baudRate") ?? defaultBaudRate;
        var dataBits = Integer(profile.Config, "dataBits") ?? 8;
        var station = Integer(profile.Config, "station") ?? 1;
        var timeout = Integer(profile.Config, "receiveTimeoutMs") ?? 10000;
        if (baudRate <= 0 || dataBits is < 5 or > 8 || station is < 0 or > 255 || timeout is < 100 or > 120000) throw Invalid(kind, "PARAMETER", "串口仪表参数无效");
        return new(portName, baudRate, dataBits, ParseParity(Text(profile.Config, "parity"), defaultParity), ParseStopBits(Integer(profile.Config, "stopBits") ?? 1), (byte)station, timeout, ParseDataFormat(Text(profile.Config, "dataFormat"), defaultFormat), Boolean(profile.Config, "stringReverse"), Boolean(profile.Config, "addressStartWithZero"), kind);
    }

    private static HslSerialParity ParseParity(string? value, HslSerialParity fallback) => value?.Trim().ToLowerInvariant() switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, null or "" => fallback, _ => throw new InstrumentDriverException("INSTRUMENT_PARITY_INVALID", "校验方式无效", false) };
    private static HslSerialStopBits ParseStopBits(int value) => value switch { 1 => HslSerialStopBits.One, 2 => HslSerialStopBits.Two, _ => throw new InstrumentDriverException("INSTRUMENT_STOP_BITS_INVALID", "停止位只支持 1 或 2", false) };
    private static HslDataFormat ParseDataFormat(string? value, HslDataFormat fallback) => value?.Trim().ToUpperInvariant() switch { "ABCD" => HslDataFormat.ABCD, "BADC" => HslDataFormat.BADC, "CDAB" => HslDataFormat.CDAB, "DCBA" => HslDataFormat.DCBA, null or "" => fallback, _ => throw new InstrumentDriverException("INSTRUMENT_DATA_FORMAT_INVALID", "数据格式无效", false) };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.TryGetInt32(out var result) ? result : null;
    private static bool Boolean(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False && value.GetBoolean();
    private static InstrumentDriverException Invalid(InstrumentSerialKind kind, string part, string message) => new($"{kind.ToString().ToUpperInvariant()}_{part}_INVALID", message, false);
}

internal sealed partial record InstrumentSerialAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static InstrumentSerialAddress Parse(PointReadRequest point, InstrumentSerialKind kind) => kind switch
    {
        InstrumentSerialKind.Dam3601 => ParseDam(point),
        InstrumentSerialKind.YuDian => ParseYuDian(point),
        _ => ParseDtsu(point),
    };
    public HslInstrumentSerialReadRequest ToRequest() => new(Address, DataType, ElementCount);

    private static InstrumentSerialAddress ParseDam(PointReadRequest point)
    {
        var source = AddressText(point);
        var match = DamPattern().Match(source);
        if (!match.Success || point.DataType != "float32" || point.ElementCount is < 1 or > 128 || !int.TryParse(match.Groups[2].Value, out var channel) || channel + point.ElementCount > 128) throw Invalid("DAM3601", "DAM3601 地址应为 temperature[0] 到 temperature[127]，数据类型固定为 float32");
        var prefix = match.Groups[1].Success ? $"s={byte.Parse(match.Groups[1].Value, CultureInfo.InvariantCulture)};" : string.Empty;
        return new(prefix + $"temperature[{channel}]", HslValueType.SinglePrecision, point.ElementCount);
    }

    private static InstrumentSerialAddress ParseYuDian(PointReadRequest point)
    {
        var match = NumericPattern().Match(AddressText(point));
        if (!match.Success || point.ElementCount is < 1 or > 4) throw Invalid("YUDIAN", "宇电 AIBus 地址必须是 0 到 255 的参数号，可选使用 s=<站号>; 前缀");
        var parameter = byte.Parse(match.Groups[2].Value, CultureInfo.InvariantCulture);
        var prefix = match.Groups[1].Success ? $"s={byte.Parse(match.Groups[1].Value, CultureInfo.InvariantCulture)};" : string.Empty;
        var type = point.DataType switch { "int16" => HslValueType.Signed16, "float64" => HslValueType.DoublePrecision, _ => throw Invalid("YUDIAN", "宇电 AIBus 数据类型仅支持 int16 和 float64") };
        return new(prefix + parameter.ToString(CultureInfo.InvariantCulture), type, point.ElementCount);
    }

    private static InstrumentSerialAddress ParseDtsu(PointReadRequest point)
    {
        var address = AddressText(point);
        if (address.Length > 128 || address.Any(char.IsWhiteSpace) || point.ElementCount is < 1 or > 65535) throw Invalid("DTSU6606", "DTSU6606 地址无效");
        return new(address, Type(point.DataType), point.ElementCount);
    }

    private static string AddressText(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String || string.IsNullOrWhiteSpace(value.GetString())) throw Invalid("INSTRUMENT", "仪表地址不能为空");
        return value.GetString()!.Trim();
    }
    private static HslValueType Type(string value) => value switch { "bool" => HslValueType.Boolean, "int8" => HslValueType.Signed8, "uint8" => HslValueType.Unsigned8, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "bytes" => HslValueType.Binary, _ => throw Invalid("DTSU6606", "DTSU6606 数据类型不受支持") };
    private static InstrumentDriverException Invalid(string code, string message) => new($"{code}_ADDRESS_INVALID", message, false);

    [GeneratedRegex("^(?:s=([0-9]{1,3});)?temperature\\[([0-9]{1,3})\\]$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)] private static partial Regex DamPattern();
    [GeneratedRegex("^(?:s=([0-9]{1,3});)?([0-9]{1,3})$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)] private static partial Regex NumericPattern();
}
