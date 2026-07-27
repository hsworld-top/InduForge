using System.Diagnostics;
using System.Globalization;
using System.Text.Json;
using System.Text.RegularExpressions;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ToyoPuc;

public sealed class ToyoPucDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly IHslSpecializedNetworkClientFactory _factory;
    public ToyoPucDriver() : this(new HslSpecializedNetworkClientFactory()) { }
    internal ToyoPucDriver(IHslSpecializedNetworkClientFactory factory) => _factory = factory;
    public DriverDescriptor Descriptor { get; } = new("toyo", "toyo.puc", "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var options = ToyoPucOptions.Parse(profile); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken); if (!result.Succeeded) { await client.DisposeAsync(); throw new ToyoPucDriverException(result.ErrorCode ?? "TOYO_PUC_CONNECT_FAILED", result.ErrorMessage ?? "无法连接东洋 PUC", result.Retryable); } return new Session(client, $"{options.Host}:{options.Port}"); }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
    private sealed class Session(IHslSpecializedNetworkClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken) { var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count]; for (var index = 0; index < request.Points.Count; index++) { var point = request.Points[index]; try { var address = ToyoPucAddress.Parse(point); var result = await client.ReadAsync(address.ToRequest(), cancellationToken); values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : Fail(point, result.ErrorCode ?? "TOYO_PUC_READ_FAILED", result.ErrorMessage ?? "东洋变量读取失败"); } catch (ToyoPucDriverException exception) { values[index] = Fail(point, exception.Code, exception.Message); } } return new(values, []); }
        public ValueTask DisposeAsync() => client.DisposeAsync(); private static PointReadValue Fail(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}

internal sealed record ToyoPucOptions(string Host, int Port, int ConnectTimeoutMs, int ReceiveTimeoutMs)
{
    public HslSpecializedNetworkClientOptions ToHslOptions() => new(HslSpecializedNetworkProtocol.ToyoPuc, Host, Port, ConnectTimeoutMs, ReceiveTimeoutMs);
    public static ToyoPucOptions Parse(ConnectionProfile profile) { if (!string.Equals(profile.ProtocolFamily, "toyo", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("CONFIG", "东洋 PUC 连接配置无效"); var host = Text(profile.Config, "host")?.Trim(); if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)) throw Invalid("HOST", "东洋设备 IP / 主机名无效"); var port = Integer(profile.Config, "port") ?? 6000; if (port is < 1 or > 65535) throw Invalid("PORT", "东洋端口必须在 1 到 65535 之间"); return new(host, port, Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000)); }
    private static int Timeout(JsonElement config,string name,int fallback){var value=Integer(config,name)??fallback;if(value is <100 or >120000)throw Invalid("TIMEOUT","东洋超时必须在 100 到 120000 毫秒之间");return value;} private static string? Text(JsonElement config,string name)=>config.TryGetProperty(name,out var value)&&value.ValueKind==JsonValueKind.String?value.GetString():null;private static int? Integer(JsonElement config,string name)=>config.TryGetProperty(name,out var value)&&value.TryGetInt32(out var result)?result:null;private static ToyoPucDriverException Invalid(string part,string message)=>new($"TOYO_PUC_{part}_INVALID",message,false);
}

internal sealed partial record ToyoPucAddress(string Address,HslValueType DataType,int ElementCount)
{
    public static ToyoPucAddress Parse(PointReadRequest point){if(point.Address.ValueKind!=JsonValueKind.Object||!point.Address.TryGetProperty("address",out var value)||value.ValueKind!=JsonValueKind.String)throw Invalid();var match=AddressRegex().Match(value.GetString()?.Trim()??string.Empty);if(!match.Success||point.ElementCount is <1 or >65535)throw Invalid();var prefix="";if(match.Groups[1].Success){var program=int.Parse(match.Groups[1].Value,CultureInfo.InvariantCulture);if(program is <0 or >255)throw Invalid();prefix=$"prg={program};";}var offset=ulong.Parse(match.Groups[3].Value,NumberStyles.HexNumber,CultureInfo.InvariantCulture);return new(prefix+match.Groups[2].Value.ToUpperInvariant()+offset.ToString("X",CultureInfo.InvariantCulture),Type(point.DataType),point.ElementCount);}
    public HslSpecializedNetworkReadRequest ToRequest()=>new(Address,DataType,ElementCount);
    private static HslValueType Type(string value)=>value switch{"bool"=>HslValueType.Boolean,"int8"=>HslValueType.Signed8,"uint8"=>HslValueType.Unsigned8,"int16"=>HslValueType.Signed16,"uint16"=>HslValueType.Unsigned16,"int32"=>HslValueType.Signed32,"uint32"=>HslValueType.Unsigned32,"int64"=>HslValueType.Signed64,"uint64"=>HslValueType.Unsigned64,"float32"=>HslValueType.SinglePrecision,"float64"=>HslValueType.DoublePrecision,"string"=>HslValueType.Text,"bytes"=>HslValueType.Binary,_=>throw new ToyoPucDriverException("TOYO_PUC_DATA_TYPE_UNSUPPORTED","东洋数据类型不受支持",false)};
    private static ToyoPucDriverException Invalid()=>new("TOYO_PUC_ADDRESS_INVALID","东洋 PUC 软元件地址无效",false);
    [GeneratedRegex("^(?:prg=([0-9]{1,3});)?(EK|EV|ET|EC|EL|EX|EY|EM|ES|EN|GX|GY|GM|EB|K|V|T|C|L|X|Y|M|S|N|R|D|B|H|U)([0-9A-F]+)$",RegexOptions.IgnoreCase|RegexOptions.CultureInvariant)]private static partial Regex AddressRegex();
}
internal sealed class ToyoPucDriverException(string code,string message,bool retryable):Exception(message){public string Code{get;}=code;public bool Retryable{get;}=retryable;}
