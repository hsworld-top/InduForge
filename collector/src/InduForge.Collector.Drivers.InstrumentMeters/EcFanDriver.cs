using System.Diagnostics;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InstrumentMeters;

public sealed class EcFanMachineDriver : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly IHslEcFanClientFactory _factory;
    public EcFanMachineDriver() : this(new HslEcFanClientFactory()) { }
    internal EcFanMachineDriver(IHslEcFanClientFactory factory) => _factory = factory;
    public DriverDescriptor Descriptor { get; } = new("ec-fan", "ec-fan.machine-serial", "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var options = EcFanOptions.Parse(profile); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken); if (!result.Succeeded) { await client.DisposeAsync(); throw new InstrumentDriverException(result.ErrorCode ?? "EC_FAN_CONNECT_FAILED", result.ErrorMessage ?? "EC 风机串口打开失败", result.Retryable); } return new Session(client, $"{options.PortName} / Station={options.Station}"); }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
    private sealed class Session(IHslEcFanClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken) { var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count]; for (var index = 0; index < request.Points.Count; index++) { var point = request.Points[index]; try { var metric = EcFanAddress.Parse(point); var result = await client.ReadAsync(metric, cancellationToken); values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : Fail(point, result.ErrorCode ?? "EC_FAN_READ_FAILED", result.ErrorMessage ?? "EC 风机读取失败"); } catch (InstrumentDriverException exception) { values[index] = Fail(point, exception.Code, exception.Message); } } return new(values, []); }
        public ValueTask DisposeAsync() => client.DisposeAsync(); private static PointReadValue Fail(PointReadRequest point, string code, string message) => new(point.Key, false, null, point.DataType, "Bad", null, null, code, message);
    }
}

internal sealed record EcFanOptions(string PortName, int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits, byte Station, int ReceiveTimeoutMs)
{
    public HslEcFanClientOptions ToHslOptions() => new(PortName, BaudRate, DataBits, Parity, StopBits, Station, ReceiveTimeoutMs);
    public static EcFanOptions Parse(ConnectionProfile profile) { if (!string.Equals(profile.ProtocolFamily, "ec-fan", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("CONFIG", "EC 风机连接配置无效"); var portName = Text(profile.Config, "portName")?.Trim(); if (string.IsNullOrWhiteSpace(portName)) throw Invalid("PORT_NAME", "EC 风机串口名称不能为空"); var station = Integer(profile.Config, "station") ?? 1; var timeout = Integer(profile.Config, "receiveTimeoutMs") ?? 10000; if (station is < 0 or > 255 || timeout is < 100 or > 120000) throw Invalid("PARAMETER", "EC 风机站号或超时无效"); return new(portName, Integer(profile.Config, "baudRate") ?? 9600, Integer(profile.Config, "dataBits") ?? 8, ParseParity(Text(profile.Config, "parity")), (Integer(profile.Config, "stopBits") ?? 1) == 2 ? HslSerialStopBits.Two : HslSerialStopBits.One, (byte)station, timeout); }
    private static HslSerialParity ParseParity(string? value) => value?.ToLowerInvariant() switch { "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, _ => HslSerialParity.None }; private static string? Text(JsonElement c,string n)=>c.TryGetProperty(n,out var v)&&v.ValueKind==JsonValueKind.String?v.GetString():null;private static int? Integer(JsonElement c,string n)=>c.TryGetProperty(n,out var v)&&v.TryGetInt32(out var r)?r:null;private static InstrumentDriverException Invalid(string p,string m)=>new($"EC_FAN_{p}_INVALID",m,false);
}

internal static class EcFanAddress
{
    public static HslEcFanMetric Parse(PointReadRequest point)
    {
        if (point.DataType != "int32" || point.ElementCount != 1 || point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String) throw Invalid();
        return value.GetString()?.Trim() switch { "speedEmergency" => HslEcFanMetric.SpeedEmergency, "speedMinimum" => HslEcFanMetric.SpeedMinimum, "speedMaximum" => HslEcFanMetric.SpeedMaximum, _ => throw Invalid() };
    }
    private static InstrumentDriverException Invalid() => new("EC_FAN_ADDRESS_INVALID", "EC 风机地址仅支持 speedEmergency、speedMinimum、speedMaximum，数据类型必须为 int32", false);
}
