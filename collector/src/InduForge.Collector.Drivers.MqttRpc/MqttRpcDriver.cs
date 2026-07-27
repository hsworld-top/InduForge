using System.Diagnostics;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MqttRpc;

public sealed class MqttRpcDeviceDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly IHslMqttRpcClientFactory _factory;
    public MqttRpcDeviceDriver() : this(new HslMqttRpcClientFactory()) { }
    internal MqttRpcDeviceDriver(IHslMqttRpcClientFactory factory) => _factory = factory;
    public DriverDescriptor Descriptor { get; } = new("mqtt-rpc", "mqtt.rpc-device", "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile,CancellationToken cancellationToken){var watch=Stopwatch.StartNew();await using var session=await OpenSessionAsync(profile,cancellationToken);watch.Stop();return new(session.IsConnected,watch.Elapsed,session.ServerName,[]);}
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile,CancellationToken cancellationToken){var options=MqttRpcOptions.Parse(profile);var client=_factory.Create(options.ToHslOptions());var result=await client.ConnectAsync(cancellationToken);if(!result.Succeeded){await client.DisposeAsync();throw new MqttRpcDriverException(result.ErrorCode??"MQTT_RPC_CONNECT_FAILED",result.ErrorMessage??"MQTT RPC 连接失败",result.Retryable);}return new Session(client,$"{options.Host}:{options.Port} / {options.DeviceTopic}");}
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile,ReadRequest request,CancellationToken cancellationToken){await using var session=await OpenSessionAsync(profile,cancellationToken);return await((IPointReaderSession)session).ReadAsync(request,cancellationToken);}
    private sealed class Session(IHslMqttRpcClient client,string serverName):IIndustrialConnectionSession,IPointReaderSession{public bool IsConnected=>client.IsConnected;public string? ServerName{get;}=serverName;public async Task<ReadResult> ReadAsync(ReadRequest request,CancellationToken cancellationToken){var readAt=DateTimeOffset.UtcNow;var values=new PointReadValue[request.Points.Count];for(var index=0;index<request.Points.Count;index++){var point=request.Points[index];try{var address=MqttRpcAddress.Parse(point);var result=await client.ReadAsync(address.ToRequest(),cancellationToken);values[index]=result.Succeeded?new(point.Key,true,result.Value,point.DataType,"Good",readAt,readAt,null,null):Fail(point,result.ErrorCode??"MQTT_RPC_READ_FAILED",result.ErrorMessage??"MQTT RPC 读取失败");}catch(MqttRpcDriverException exception){values[index]=Fail(point,exception.Code,exception.Message);}}return new(values,[]);}public ValueTask DisposeAsync()=>client.DisposeAsync();private static PointReadValue Fail(PointReadRequest point,string code,string message)=>new(point.Key,false,null,point.DataType,"Bad",null,null,code,message);}
}

internal sealed record MqttRpcOptions(string Host,int Port,string ClientId,string? UserName,string? Password,string DeviceTopic,bool UseRsa)
{
 public HslMqttRpcClientOptions ToHslOptions()=>new(Host,Port,ClientId,UserName,Password,DeviceTopic,UseRsa);
 public static MqttRpcOptions Parse(ConnectionProfile profile){if(!string.Equals(profile.ProtocolFamily,"mqtt-rpc",StringComparison.OrdinalIgnoreCase)||profile.Config.ValueKind!=JsonValueKind.Object)throw Invalid("CONFIG","MQTT RPC 连接配置无效");var host=Text(profile.Config,"host")?.Trim();if(string.IsNullOrWhiteSpace(host)||host.Contains("://",StringComparison.Ordinal))throw Invalid("HOST","MQTT Broker IP / 主机名无效");var port=Integer(profile.Config,"port")??1883;if(port is<1 or>65535)throw Invalid("PORT","MQTT Broker 端口无效");return new(host,port,Text(profile.Config,"clientId")?.Trim()??string.Empty,Text(profile.Config,"userName"),Text(profile.Config,"password"),Text(profile.Config,"deviceTopic")?.Trim()??string.Empty,Boolean(profile.Config,"useRsa"));}
 private static string? Text(JsonElement c,string n)=>c.TryGetProperty(n,out var v)&&v.ValueKind==JsonValueKind.String?v.GetString():null;private static int? Integer(JsonElement c,string n)=>c.TryGetProperty(n,out var v)&&v.TryGetInt32(out var r)?r:null;private static bool Boolean(JsonElement c,string n)=>c.TryGetProperty(n,out var v)&&v.ValueKind is JsonValueKind.True or JsonValueKind.False&&v.GetBoolean();private static MqttRpcDriverException Invalid(string p,string m)=>new($"MQTT_RPC_{p}_INVALID",m,false);
}

internal sealed record MqttRpcAddress(string Address,HslValueType DataType,int ElementCount)
{
 public static MqttRpcAddress Parse(PointReadRequest point){if(point.Address.ValueKind!=JsonValueKind.Object||!point.Address.TryGetProperty("address",out var value)||value.ValueKind!=JsonValueKind.String||point.ElementCount is<1 or>65535)throw Invalid();var address=value.GetString()?.Trim()??string.Empty;if(address.Length is<1 or>256||address.Any(char.IsWhiteSpace))throw Invalid();return new(address,Type(point.DataType),point.ElementCount);}public HslMqttRpcReadRequest ToRequest()=>new(Address,DataType,ElementCount);
 private static HslValueType Type(string value)=>value switch{"bool"=>HslValueType.Boolean,"int16"=>HslValueType.Signed16,"uint16"=>HslValueType.Unsigned16,"int32"=>HslValueType.Signed32,"uint32"=>HslValueType.Unsigned32,"int64"=>HslValueType.Signed64,"uint64"=>HslValueType.Unsigned64,"float32"=>HslValueType.SinglePrecision,"float64"=>HslValueType.DoublePrecision,"string"=>HslValueType.Text,"bytes"=>HslValueType.Binary,_=>throw new MqttRpcDriverException("MQTT_RPC_DATA_TYPE_UNSUPPORTED","MQTT RPC 数据类型不受支持",false)};private static MqttRpcDriverException Invalid()=>new("MQTT_RPC_ADDRESS_INVALID","MQTT RPC 设备地址无效",false);
}
internal sealed class MqttRpcDriverException(string code,string message,bool retryable):Exception(message){public string Code{get;}=code;public bool Retryable{get;}=retryable;}
