namespace InduForge.Collector.Adapters.Hsl;

public enum HslDataFormat
{
    ABCD,
    BADC,
    CDAB,
    DCBA,
}

public enum HslValueType
{
    Boolean,
    Signed8,
    Unsigned8,
    Signed16,
    Unsigned16,
    Signed32,
    Unsigned32,
    Signed64,
    Unsigned64,
    SinglePrecision,
    DoublePrecision,
    Text,
    Binary,
    DateTime,
}

public enum HslSiemensPlc
{
    S1200,
    S300,
    S400,
    S1500,
    S200Smart,
    S200,
}

public enum HslModbusReadArea
{
    Coil,
    DiscreteInput,
    Register,
}

public sealed record HslModbusTcpClientOptions(
    string Host,
    int Port,
    int TimeoutMilliseconds,
    HslDataFormat DataFormat);

public sealed record HslReadRequest(
    string Address,
    HslModbusReadArea Area,
    HslValueType DataType,
    int ElementCount);

public sealed record HslS7TcpClientOptions(
    string Host,
    int Port,
    int TimeoutMilliseconds,
    HslSiemensPlc PlcType,
    byte Rack,
    byte Slot,
    int? LocalTsap,
    int? RemoteTsap);

public sealed record HslS7ReadRequest(
    string Address,
    HslValueType DataType,
    int ElementCount);

public sealed record HslOperationResult(
    bool Succeeded,
    string? ErrorCode,
    string? ErrorMessage,
    bool Retryable)
{
    public static HslOperationResult Success() => new(true, null, null, false);

    public static HslOperationResult Failure(string errorCode, string errorMessage, bool retryable) =>
        new(false, errorCode, errorMessage, retryable);
}

public sealed record HslReadResult(
    bool Succeeded,
    object? Value,
    string? ErrorCode,
    string? ErrorMessage,
    bool Retryable)
{
    public static HslReadResult Success(object? value) => new(true, value, null, null, false);

    public static HslReadResult Failure(string errorCode, string errorMessage, bool retryable) =>
        new(false, null, errorCode, errorMessage, retryable);
}

public interface IHslModbusTcpClient : IAsyncDisposable
{
    bool IsConnected { get; }

    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);

    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);

    Task<HslReadResult> ReadAsync(HslReadRequest request, CancellationToken cancellationToken);
}

public interface IHslModbusTcpClientFactory
{
    IHslModbusTcpClient Create(HslModbusTcpClientOptions options);
}

public interface IHslS7TcpClient : IAsyncDisposable
{
    bool IsConnected { get; }

    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);

    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);

    Task<HslReadResult> ReadAsync(HslS7ReadRequest request, CancellationToken cancellationToken);
}

public interface IHslS7TcpClientFactory
{
    IHslS7TcpClient Create(HslS7TcpClientOptions options);
}

public sealed class HslModbusTcpClientFactory : IHslModbusTcpClientFactory
{
    public IHslModbusTcpClient Create(HslModbusTcpClientOptions options) => new HslModbusTcpClient(options);
}
