using System.Diagnostics;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MitsubishiMc3ETcp;

public sealed class MitsubishiA1EAsciiTcpDriver : MitsubishiNetworkVariantDriverBase { public MitsubishiA1EAsciiTcpDriver() : base("mitsubishi.a1e-ascii-tcp", HslMelsecNetworkProtocol.A1EAsciiTcp) { } }
public sealed class MitsubishiA1EBinaryTcpDriver : MitsubishiNetworkVariantDriverBase { public MitsubishiA1EBinaryTcpDriver() : base("mitsubishi.a1e-binary-tcp", HslMelsecNetworkProtocol.A1EBinaryTcp) { } }
public sealed class MitsubishiMcAsciiTcpDriver : MitsubishiNetworkVariantDriverBase { public MitsubishiMcAsciiTcpDriver() : base("mitsubishi.mc-ascii-tcp", HslMelsecNetworkProtocol.McAsciiTcp) { } }
public sealed class MitsubishiMcAsciiUdpDriver : MitsubishiNetworkVariantDriverBase { public MitsubishiMcAsciiUdpDriver() : base("mitsubishi.mc-ascii-udp", HslMelsecNetworkProtocol.McAsciiUdp) { } }
public sealed class MitsubishiMcBinaryUdpDriver : MitsubishiNetworkVariantDriverBase { public MitsubishiMcBinaryUdpDriver() : base("mitsubishi.mc-binary-udp", HslMelsecNetworkProtocol.McBinaryUdp) { } }
public sealed class MitsubishiMcRBinaryTcpDriver : MitsubishiNetworkVariantDriverBase { public MitsubishiMcRBinaryTcpDriver() : base("mitsubishi.mc-r-binary-tcp", HslMelsecNetworkProtocol.McRBinaryTcp) { } }
public sealed class MitsubishiCipDriver : MitsubishiNetworkVariantDriverBase { public MitsubishiCipDriver() : base("mitsubishi.cip", HslMelsecNetworkProtocol.CipTcp) { } }

public abstract class MitsubishiNetworkVariantDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslMelsecNetworkProtocol _protocol;
    private readonly IHslMelsecNetworkClientFactory _factory;
    private protected MitsubishiNetworkVariantDriverBase(string driverId, HslMelsecNetworkProtocol protocol, IHslMelsecNetworkClientFactory? factory = null)
    {
        _protocol = protocol; _factory = factory ?? new HslMelsecNetworkClientFactory();
        Descriptor = new("melsec", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    }
    public DriverDescriptor Descriptor { get; }
    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) { var watch = Stopwatch.StartNew(); await using var session = await OpenSessionAsync(profile, cancellationToken); watch.Stop(); return new(session.IsConnected, watch.Elapsed, session.ServerName, []); }
    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = MitsubishiNetworkVariantOptions.Parse(profile, _protocol); var client = _factory.Create(options.ToHslOptions()); var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded) { await client.DisposeAsync(); throw new MitsubishiMc3ETcpDriverException(result.ErrorCode ?? "MELSEC_CONNECT_FAILED", result.ErrorMessage ?? "无法连接三菱设备", result.Retryable); }
        return new Session(client, options.ServerName, _protocol);
    }
    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) { await using var session = await OpenSessionAsync(profile, cancellationToken); return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken); }
    private sealed class Session(IHslMelsecNetworkClient client, string serverName, HslMelsecNetworkProtocol protocol) : IIndustrialConnectionSession, IPointReaderSession
    {
        public bool IsConnected => client.IsConnected; public string? ServerName { get; } = serverName;
        public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
        {
            var readAt = DateTimeOffset.UtcNow; var values = new PointReadValue[request.Points.Count];
            for (var index = 0; index < request.Points.Count; index++) { var point = request.Points[index]; try { var address = MitsubishiNetworkVariantAddress.Parse(point, protocol); var result = await client.ReadAsync(address.ToHslRequest(), cancellationToken); values[index] = result.Succeeded ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null) : new(point.Key, false, null, point.DataType, "Bad", null, null, result.ErrorCode, result.ErrorMessage); } catch (MitsubishiMc3ETcpDriverException exception) { values[index] = new(point.Key, false, null, point.DataType, "Bad", null, null, exception.Code, exception.Message); } }
            return new(values, []);
        }
        public ValueTask DisposeAsync() => client.DisposeAsync();
    }
}

internal sealed record MitsubishiNetworkVariantAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static MitsubishiNetworkVariantAddress Parse(PointReadRequest point, HslMelsecNetworkProtocol protocol)
    {
        if (protocol != HslMelsecNetworkProtocol.CipTcp) { var parsed = MitsubishiMc3ETcpAddress.Parse(point); return new(parsed.HslAddress, parsed.ValueType, parsed.ElementCount); }
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String) throw Invalid("MELSEC_CIP_ADDRESS_INVALID", "三菱 CIP 地址必须包含 address 字段");
        var address = value.GetString()?.Trim() ?? string.Empty; if (address.Length is < 1 or > 512 || address.Any(char.IsWhiteSpace)) throw Invalid("MELSEC_CIP_ADDRESS_INVALID", "三菱 CIP 标签地址无效");
        if (point.ElementCount is < 1 or > ushort.MaxValue) throw Invalid("MELSEC_CIP_ELEMENT_COUNT_INVALID", "三菱 CIP 元素数量无效");
        if (point.DataType == "datetime" && point.ElementCount != 1) throw Invalid("MELSEC_CIP_ELEMENT_COUNT_INVALID", "三菱 CIP datetime 标签只支持单元素读取");
        var dataType = point.DataType switch { "bool" => HslValueType.Boolean, "int16" => HslValueType.Signed16, "uint16" => HslValueType.Unsigned16, "int32" => HslValueType.Signed32, "uint32" => HslValueType.Unsigned32, "int64" => HslValueType.Signed64, "uint64" => HslValueType.Unsigned64, "float32" => HslValueType.SinglePrecision, "float64" => HslValueType.DoublePrecision, "string" => HslValueType.Text, "datetime" => HslValueType.DateTime, _ => throw Invalid("MELSEC_CIP_DATA_TYPE_UNSUPPORTED", "三菱 CIP 数据类型不受支持") };
        return new(address, dataType, point.ElementCount);
    }
    public HslMelsecNetworkReadRequest ToHslRequest() => new(Address, DataType, ElementCount);
    private static MitsubishiMc3ETcpDriverException Invalid(string code, string message) => new(code, message, false);
}

internal sealed record MitsubishiNetworkVariantOptions(HslMelsecNetworkProtocol Protocol, string Host, int Port, int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, byte NetworkNumber, byte NetworkStationNumber, ushort TargetIoStation, bool EnableWriteBitToWordRegister, bool StringReverse)
{
    public static MitsubishiNetworkVariantOptions Parse(ConnectionProfile profile, HslMelsecNetworkProtocol protocol)
    {
        if (!string.Equals(profile.ProtocolFamily, "melsec", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object) throw Invalid("MELSEC_CONFIG_INVALID", "三菱网络连接配置无效");
        var host = Text(profile.Config, "host")?.Trim(); if (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal)) throw Invalid("MELSEC_HOST_INVALID", "三菱设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? (protocol == HslMelsecNetworkProtocol.CipTcp ? 44818 : 6000); if (port is < 1 or > 65535) throw Invalid("MELSEC_PORT_INVALID", "三菱网络端口无效");
        return new(protocol, host, port, Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000), Byte(profile.Config, "networkNumber", 0), Byte(profile.Config, "networkStationNumber", 0), UShort(profile.Config, "targetIoStation", 0x03FF), Boolean(profile.Config, "enableWriteBitToWordRegister", protocol == HslMelsecNetworkProtocol.McBinaryUdp), Boolean(profile.Config, "stringReverse", false));
    }
    public HslMelsecNetworkClientOptions ToHslOptions() => new(Protocol, Host, Port, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, NetworkNumber, NetworkStationNumber, TargetIoStation, EnableWriteBitToWordRegister, StringReverse);
    public string ServerName => $"{Host}:{Port}";
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("MELSEC_TIMEOUT_INVALID", "三菱网络超时配置无效"); return value; }
    private static byte Byte(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 0 or > 255) throw Invalid("MELSEC_CONFIG_INVALID", $"三菱 {name} 无效"); return checked((byte)value); }
    private static ushort UShort(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 0 or > 65535) throw Invalid("MELSEC_CONFIG_INVALID", $"三菱 {name} 无效"); return checked((ushort)value); }
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;
    private static bool Boolean(JsonElement config, string name, bool fallback) => config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : fallback;
    private static MitsubishiMc3ETcpDriverException Invalid(string code, string message) => new(code, message, false);
}
