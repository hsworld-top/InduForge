using System.Diagnostics;
using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InovanceModbusTcp;

public sealed class InovanceModbusSerialDriver : InovanceModbusVariantDriverBase
{
    public InovanceModbusSerialDriver() : base("inovance.modbus-serial", HslInovanceModbusVariantProtocol.Serial) { }
}

public sealed class InovanceModbusRtuOverTcpDriver : InovanceModbusVariantDriverBase
{
    public InovanceModbusRtuOverTcpDriver() : base("inovance.modbus-rtu-over-tcp", HslInovanceModbusVariantProtocol.SerialOverTcp) { }
}

public abstract class InovanceModbusVariantDriverBase : IIndustrialDriver, IConnectionSessionDriver, IPointReader
{
    private readonly HslInovanceModbusVariantProtocol _protocol;
    private readonly IHslInovanceModbusVariantClientFactory _factory;

    private protected InovanceModbusVariantDriverBase(string driverId, HslInovanceModbusVariantProtocol protocol, IHslInovanceModbusVariantClientFactory? factory = null)
    {
        _protocol = protocol;
        _factory = factory ?? new HslInovanceModbusVariantClientFactory();
        Descriptor = new("inovance", driverId, "1.0.0", [1], [DriverOperations.ConnectionTest, DriverOperations.ConnectionOpen, DriverOperations.ConnectionClose, DriverOperations.PointRead]);
    }

    public DriverDescriptor Descriptor { get; }

    public async Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var watch = Stopwatch.StartNew();
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        watch.Stop();
        return new(session.IsConnected, watch.Elapsed, session.ServerName, []);
    }

    public async Task<IIndustrialConnectionSession> OpenSessionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        var options = InovanceModbusVariantOptions.Parse(profile, _protocol);
        var client = _factory.Create(options.ToHslOptions());
        var result = await client.ConnectAsync(cancellationToken);
        if (!result.Succeeded)
        {
            await client.DisposeAsync();
            throw new InovanceModbusTcpDriverException(result.ErrorCode ?? "INOVANCE_CONNECT_FAILED", result.ErrorMessage ?? "无法连接汇川设备", result.Retryable);
        }
        return new Session(client, options.ServerName);
    }

    public async Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
    {
        await using var session = await OpenSessionAsync(profile, cancellationToken);
        return await ((IPointReaderSession)session).ReadAsync(request, cancellationToken);
    }

    private sealed class Session(IHslInovanceModbusVariantClient client, string serverName) : IIndustrialConnectionSession, IPointReaderSession
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
                    var address = InovanceModbusTcpAddress.Parse(point);
                    var result = await client.ReadAsync(new(address.ProtocolAddress, address.ValueType, address.ElementCount), cancellationToken);
                    values[index] = result.Succeeded
                        ? new(point.Key, true, result.Value, point.DataType, "Good", readAt, readAt, null, null)
                        : new(point.Key, false, null, point.DataType, "Bad", null, null, result.ErrorCode, result.ErrorMessage);
                }
                catch (InovanceModbusTcpDriverException exception)
                {
                    values[index] = new(point.Key, false, null, point.DataType, "Bad", null, null, exception.Code, exception.Message);
                }
            }
            return new(values, []);
        }

        public ValueTask DisposeAsync() => client.DisposeAsync();
    }
}

internal sealed record InovanceModbusVariantOptions(
    HslInovanceModbusVariantProtocol Protocol, string Host, int Port, string? PortName,
    int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, byte Station,
    HslInovanceSeries Series, bool AddressStartWithZero, HslDataFormat DataFormat, bool StringReverse)
{
    public static InovanceModbusVariantOptions Parse(ConnectionProfile profile, HslInovanceModbusVariantProtocol protocol)
    {
        if (!string.Equals(profile.ProtocolFamily, "inovance", StringComparison.OrdinalIgnoreCase) || profile.Config.ValueKind != JsonValueKind.Object)
            throw Invalid("INOVANCE_CONFIG_INVALID", "汇川连接配置无效");
        var serial = protocol == HslInovanceModbusVariantProtocol.Serial;
        var host = Text(profile.Config, "host")?.Trim() ?? string.Empty;
        if (!serial && (string.IsNullOrWhiteSpace(host) || host.Contains("://", StringComparison.Ordinal))) throw Invalid("INOVANCE_HOST_INVALID", "汇川设备 IP / 主机名无效");
        var port = Integer(profile.Config, "port") ?? 502;
        if (!serial && port is < 1 or > 65535) throw Invalid("INOVANCE_PORT_INVALID", "汇川端口无效");
        var portName = Text(profile.Config, "portName")?.Trim();
        if (serial && string.IsNullOrWhiteSpace(portName)) throw Invalid("INOVANCE_PORT_NAME_INVALID", "汇川串口名称不能为空");
        var station = Integer(profile.Config, "station") ?? 1;
        if (station is < 0 or > 255) throw Invalid("INOVANCE_STATION_INVALID", "汇川站号无效");
        return new(protocol, host, port, portName, Integer(profile.Config, "baudRate") ?? 9600, Integer(profile.Config, "dataBits") ?? 8,
            ParseParity(Text(profile.Config, "parity") ?? "none"), (Text(profile.Config, "stopBits") ?? "one") == "two" ? HslSerialStopBits.Two : HslSerialStopBits.One,
            Timeout(profile.Config, "connectTimeoutMs", 5000), Timeout(profile.Config, "receiveTimeoutMs", 10000), checked((byte)station),
            ParseSeries(Text(profile.Config, "series") ?? "AM"), Boolean(profile.Config, "addressStartWithZero", true), ParseDataFormat(Text(profile.Config, "dataFormat") ?? "CDAB"), Boolean(profile.Config, "stringReverse", false));
    }

    public HslInovanceModbusVariantClientOptions ToHslOptions() => new(Protocol, Host, Port, PortName, BaudRate, DataBits, Parity, StopBits, ConnectTimeoutMilliseconds, ReceiveTimeoutMilliseconds, Station, Series, AddressStartWithZero, DataFormat, StringReverse);
    public string ServerName => Protocol == HslInovanceModbusVariantProtocol.Serial ? PortName! : $"{Host}:{Port}";
    private static int Timeout(JsonElement config, string name, int fallback) { var value = Integer(config, name) ?? fallback; if (value is < 100 or > 120000) throw Invalid("INOVANCE_TIMEOUT_INVALID", "汇川超时配置无效"); return value; }
    private static HslInovanceSeries ParseSeries(string value) => value switch { "AM" => HslInovanceSeries.AM, "H3U" => HslInovanceSeries.H3U, "H5U" => HslInovanceSeries.H5U, "Easy" => HslInovanceSeries.Easy, _ => throw Invalid("INOVANCE_SERIES_INVALID", "汇川 PLC 系列无效") };
    private static HslDataFormat ParseDataFormat(string value) => value switch { "ABCD" => HslDataFormat.ABCD, "BADC" => HslDataFormat.BADC, "CDAB" => HslDataFormat.CDAB, "DCBA" => HslDataFormat.DCBA, _ => throw Invalid("INOVANCE_DATA_FORMAT_INVALID", "汇川数据格式无效") };
    private static HslSerialParity ParseParity(string value) => value switch { "none" => HslSerialParity.None, "odd" => HslSerialParity.Odd, "even" => HslSerialParity.Even, _ => throw Invalid("INOVANCE_PARITY_INVALID", "汇川串口校验方式无效") };
    private static string? Text(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int? Integer(JsonElement config, string name) => config.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result) ? result : null;
    private static bool Boolean(JsonElement config, string name, bool fallback) => config.TryGetProperty(name, out var value) && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : fallback;
    private static InovanceModbusTcpDriverException Invalid(string code, string message) => new(code, message, false);
}
