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

public enum HslSiemensVariantProtocol
{
    PpiSerial,
    PpiOverTcp,
    MpiSerial,
    FetchWrite,
    WebApi,
    S7Plus,
}

public enum HslModbusReadArea
{
    Coil,
    DiscreteInput,
    Register,
}

public enum HslModbusNetworkProtocol
{
    Tcp,
    RtuOverTcp,
    AsciiOverTcp,
    Udp,
}

public enum HslModbusSerialProtocol
{
    Rtu,
    Ascii,
}

public enum HslSerialParity
{
    None,
    Odd,
    Even,
}

public enum HslSerialStopBits
{
    One,
    Two,
}

public enum HslOmronFinsTransport
{
    Tcp,
    Udp,
}

public enum HslOmronPlcType
{
    CSCJ,
    CV,
}

public enum HslOmronVariantProtocol
{
    Cip,
    ConnectedCip,
    HostLink,
    HostLinkOverTcp,
    HostLinkCMode,
    HostLinkCModeOverTcp,
}

public enum HslAllenBradleyProtocol
{
    EtherNetIp,
    ConnectedCip,
    MicroCip,
    Pccc,
    Slc,
    Df1Serial,
}

public enum HslAllenBradleyCheckType
{
    Bcc,
    Crc16,
}

public enum HslIec104InformationType
{
    SinglePoint,
    DoublePoint,
    NormalizedMeasured,
    ScaledMeasured,
    ShortFloatMeasured,
    BitString32,
    IntegratedTotal,
}

public enum HslLsisCpuType
{
    XGK,
    XGI,
    XGR,
    XgbMk,
    XgbIec,
    XGB,
}

public enum HslLsisSerialProtocol
{
    CnetSerial,
    CnetOverTcp,
    CpuSerial,
}

public enum HslInovanceSeries
{
    AM,
    H3U,
    H5U,
    Easy,
}

public enum HslInovanceModbusVariantProtocol
{
    Serial,
    SerialOverTcp,
}

public enum HslInovanceSpecialProtocol
{
    ConnectedCip,
    EasyNet,
    ComputerLink,
}

public enum HslMelsecNetworkProtocol
{
    A1EAsciiTcp,
    A1EBinaryTcp,
    McAsciiTcp,
    McAsciiUdp,
    McBinaryUdp,
    McRBinaryTcp,
    CipTcp,
}

public enum HslMelsecSerialProtocol
{
    A3CSerial,
    A3CSerialOverTcp,
    FxLinksSerial,
    FxLinksOverTcp,
    FxSerial,
    FxSerialOverTcp,
}

public enum HslKeyenceProtocol
{
    McBinary,
    McAscii,
    KvOld,
    Nano,
    NanoSerial,
    NanoSerialOverTcp,
}

public enum HslDeltaTransport
{
    Tcp,
    RtuOverTcp,
    AsciiOverTcp,
    RtuSerial,
    AsciiSerial,
}

public enum HslDeltaSeries
{
    Dvp,
    AS,
}

public enum HslXinjeTransport
{
    Tcp,
    RtuOverTcp,
    RtuSerial,
    InternalTcp,
}

public enum HslXinjeSeries
{
    XC,
    XD,
    XL,
}

public enum HslMegMeetTransport
{
    Tcp,
    RtuOverTcp,
    RtuSerial,
}

public enum HslFujiProtocol
{
    CommandSettingTypeTcp,
    SphTcp,
    SpbOverTcp,
    SpbSerial,
}

public enum HslVigorTransport
{
    SerialOverTcp,
    Serial,
}

public enum HslFreedomTransport
{
    Tcp,
    Udp,
    Serial,
}

public enum HslYamatakeTransport
{
    Serial,
    Tcp,
}

public enum HslInstrumentTransport
{
    Serial,
    Tcp,
}

public enum HslInstrumentSerialProtocol
{
    Dam3601,
    YuDianAiBus,
    Dtsu6606,
}

public enum HslDltProtocol
{
    Dlt645Serial,
    Dlt645OverTcp,
    Dlt645With1997Serial,
    Dlt645With1997OverTcp,
    Dlt698Serial,
    Dlt698OverTcp,
    Dlt698TcpNet,
}

public enum HslYaskawaTransport
{
    Tcp,
    Udp,
}

public enum HslSpecializedNetworkProtocol
{
    OrientalMotorEip,
    ToyoPuc,
    TurckReader,
    DcsNanJingAuto,
    EstunRobot,
    FanucRobot,
}

public sealed record HslModbusTcpClientOptions(
    string Host,
    int Port,
    int TimeoutMilliseconds,
    HslDataFormat DataFormat,
    HslModbusNetworkProtocol Protocol = HslModbusNetworkProtocol.Tcp,
    int ReceiveTimeoutMilliseconds = 10000);

public sealed record HslModbusRtuClientOptions(
    string PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int TimeoutMilliseconds,
    HslDataFormat DataFormat,
    HslModbusSerialProtocol Protocol = HslModbusSerialProtocol.Rtu);

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

public sealed record HslSiemensVariantClientOptions(
    HslSiemensVariantProtocol Protocol,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    byte Station,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    bool HandshakeCheck,
    string? UserName,
    string? Password,
    bool UseHttps);

public sealed record HslMelsecMcClientOptions(
    string Host,
    int Port,
    int TimeoutMilliseconds,
    byte NetworkNumber,
    byte NetworkStationNumber,
    ushort TargetIoStation);

public sealed record HslMelsecMcReadRequest(
    string Address,
    HslValueType DataType,
    int ElementCount);

public sealed record HslMelsecNetworkClientOptions(
    HslMelsecNetworkProtocol Protocol,
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte NetworkNumber,
    byte NetworkStationNumber,
    ushort TargetIoStation,
    bool EnableWriteBitToWordRegister,
    bool StringReverse);

public sealed record HslMelsecNetworkReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslMelsecSerialClientOptions(
    HslMelsecSerialProtocol Protocol,
    string? Host,
    int Port,
    string? PortName,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    byte Station,
    byte WaitingTime,
    bool SumCheck,
    int Format,
    bool IsNewVersion,
    bool UseGot);

public sealed record HslMelsecSerialReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslOmronFinsClientOptions(
    HslOmronFinsTransport Transport,
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslOmronPlcType PlcType,
    int ReadSplits,
    byte Gct,
    byte Sid,
    HslDataFormat DataFormat,
    bool StringReverseByteWord,
    bool ReceiveUntilEmpty);

public sealed record HslOmronFinsReadRequest(
    string Address,
    HslValueType DataType,
    int ElementCount);

public sealed record HslOmronVariantClientOptions(
    HslOmronVariantProtocol Protocol,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte UnitNumber,
    HslDataFormat DataFormat,
    HslOmronPlcType PlcType = HslOmronPlcType.CSCJ);

public sealed record HslOmronVariantReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslAllenBradleyClientOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte Slot,
    string? MessageRouter,
    bool ContextCheck,
    bool ReadArrayUseSegment,
    HslDataFormat DataFormat,
    HslAllenBradleyProtocol Protocol = HslAllenBradleyProtocol.EtherNetIp,
    string? PortName = null,
    int BaudRate = 9600,
    int DataBits = 8,
    HslSerialParity Parity = HslSerialParity.None,
    HslSerialStopBits StopBits = HslSerialStopBits.One,
    byte Station = 1,
    byte DestinationNode = 1,
    byte SourceNode = 2,
    HslAllenBradleyCheckType CheckType = HslAllenBradleyCheckType.Crc16,
    IReadOnlyList<byte>? PortSlot = null);

public sealed record HslAllenBradleyReadRequest(
    string Address,
    HslValueType DataType,
    int ElementCount);

public sealed record HslBeckhoffAdsClientOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    bool UseAutoAmsNetId,
    int AmsPort,
    string? TargetAmsNetId,
    string? SenderAmsNetId,
    bool UseTagCache,
    HslDataFormat DataFormat);

public sealed record HslBeckhoffAdsReadRequest(
    string Address,
    HslValueType DataType,
    int ElementCount);

public sealed record HslIec104ClientOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    ushort CommonAddress,
    byte InterrogationCode,
    byte InterrogationReason,
    int InterrogationTimeoutMilliseconds);

public sealed record HslIec104ReadRequest(
    HslIec104InformationType InformationType,
    int InformationObjectAddress);

public sealed record HslPanasonicMewtocolClientOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte Station,
    HslDataFormat DataFormat,
    HslPanasonicMewtocolTransport Transport = HslPanasonicMewtocolTransport.Tcp,
    string? PortName = null,
    int BaudRate = 9600,
    int DataBits = 8,
    HslSerialParity Parity = HslSerialParity.None,
    HslSerialStopBits StopBits = HslSerialStopBits.One);

public enum HslPanasonicMewtocolTransport
{
    Tcp,
    Serial,
}

public sealed record HslPanasonicMewtocolReadRequest(
    string Address,
    HslValueType DataType,
    int ElementCount);

public sealed record HslPanasonicMcClientOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte NetworkStationNumber,
    ushort TargetIoStation,
    HslDataFormat DataFormat);

public sealed record HslPanasonicMcReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslLsisFastEnetClientOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslLsisCpuType CpuType,
    string CompanyId,
    byte BaseNumber,
    byte SlotNumber,
    HslDataFormat DataFormat);

public sealed record HslLsisFastEnetReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslLsisSerialClientOptions(
    HslLsisSerialProtocol Protocol, string? Host, int Port, string? PortName,
    int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds,
    int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits, byte Station);

public sealed record HslLsisSerialReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslGeSrtpClientOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat);

public sealed record HslGeSrtpReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslInovanceModbusTcpClientOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte Station,
    HslInovanceSeries Series,
    bool AddressStartWithZero,
    HslDataFormat DataFormat,
    bool StringReverse);

public sealed record HslInovanceModbusTcpReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslInovanceModbusVariantClientOptions(
    HslInovanceModbusVariantProtocol Protocol,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte Station,
    HslInovanceSeries Series,
    bool AddressStartWithZero,
    HslDataFormat DataFormat,
    bool StringReverse);

public sealed record HslInovanceModbusVariantReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslInovanceSpecialClientOptions(
    HslInovanceSpecialProtocol Protocol,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte Station,
    byte WaitingTime,
    bool SumCheck,
    int Format,
    uint ToConnectionId,
    uint OtConnectionId);

public sealed record HslInovanceSpecialReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslFatekProgramTcpClientOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte Station);

public sealed record HslFatekProgramSerialClientOptions(
    string PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int ReceiveTimeoutMilliseconds,
    byte Station);

public sealed record HslFatekProgramReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslFreedomClientOptions(
    HslFreedomTransport Transport,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    bool RtsEnable,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat,
    bool StringReverse);

public sealed record HslFreedomReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslCimonClientOptions(
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    byte FrameNumber);

public sealed record HslCimonReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslYamatakeClientOptions(
    HslYamatakeTransport Transport,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    byte Station,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds);

public sealed record HslYamatakeReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslCjt188ClientOptions(
    HslInstrumentTransport Transport,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    string Station,
    byte InstrumentType,
    bool EnableCodeFE,
    bool StationMatch,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds);

public sealed record HslCjt188ReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslRkcClientOptions(
    HslInstrumentTransport Transport,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    byte Station,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds);

public sealed record HslRkcReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslInstrumentSerialClientOptions(
    HslInstrumentSerialProtocol Protocol,
    string PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    byte Station,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat,
    bool StringReverse,
    bool AddressStartWithZero);

public sealed record HslInstrumentSerialReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslDltClientOptions(
    HslDltProtocol Protocol,
    string Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    string Station,
    string Password,
    string OpCode,
    bool EnableCodeFE,
    bool CheckDataId,
    bool UseSecurityRequest,
    byte ClientAddress,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds);

public sealed record HslDltReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslYaskawaClientOptions(
    HslYaskawaTransport Transport,
    string Host,
    int Port,
    byte CpuFrom,
    byte CpuTo,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat);

public sealed record HslYaskawaReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslSpecializedNetworkClientOptions(
    HslSpecializedNetworkProtocol Protocol,
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    uint RunIdleHeader = 1,
    uint RpiTimeMilliseconds = 100,
    int ActualTimeoutSeconds = 2,
    byte Station = 1,
    bool AddressStartWithZero = true,
    HslDataFormat DataFormat = HslDataFormat.ABCD,
    bool IsStringReverse = false,
    int DataRetainTimeMilliseconds = 100);

public sealed record HslSpecializedNetworkReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslKeyenceTcpClientOptions(
    HslKeyenceProtocol Protocol,
    string Host,
    int Port,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat,
    bool StringReverse,
    byte NetworkNumber = 0,
    byte NetworkStationNumber = 0,
    ushort TargetIoStation = 1023,
    bool EnableWriteBitToWordRegister = false,
    byte Station = 0,
    bool UseStation = false,
    string? PortName = null,
    int BaudRate = 9600,
    int DataBits = 8,
    HslSerialParity Parity = HslSerialParity.None,
    HslSerialStopBits StopBits = HslSerialStopBits.One);

public sealed record HslKeyenceTcpReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslDeltaClientOptions(
    HslDeltaTransport Transport,
    HslDeltaSeries Series,
    byte Station,
    string? Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat);

public sealed record HslDeltaReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslXinjeClientOptions(
    HslXinjeTransport Transport,
    HslXinjeSeries Series,
    byte Station,
    string? Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat);

public sealed record HslXinjeReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslMegMeetClientOptions(
    HslMegMeetTransport Transport,
    byte Station,
    string? Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat);

public sealed record HslMegMeetReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslFujiClientOptions(
    HslFujiProtocol Protocol,
    string? Host,
    int Port,
    string? PortName,
    int BaudRate,
    int DataBits,
    HslSerialParity Parity,
    HslSerialStopBits StopBits,
    byte Station,
    byte ConnectionId,
    bool DataSwap,
    int ConnectTimeoutMilliseconds,
    int ReceiveTimeoutMilliseconds,
    HslDataFormat DataFormat);

public sealed record HslFujiReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslVigorClientOptions(
    HslVigorTransport Transport, byte Station, string? Host, int Port, string? PortName,
    int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits,
    int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, HslDataFormat DataFormat);

public sealed record HslVigorReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslYokogawaClientOptions(string Host, int Port, byte CpuNumber, int ConnectTimeoutMilliseconds, int ReceiveTimeoutMilliseconds, HslDataFormat DataFormat);
public sealed record HslYokogawaReadRequest(string Address, HslValueType DataType, int ElementCount);

public sealed record HslIec104PointReadResult(
    HslIec104ReadRequest Request,
    bool Succeeded,
    object? Value,
    byte? Quality,
    DateTimeOffset? SourceTimestamp,
    string? ErrorCode,
    string? ErrorMessage,
    bool Retryable)
{
    public static HslIec104PointReadResult Success(
        HslIec104ReadRequest request,
        object? value,
        byte quality,
        DateTimeOffset? sourceTimestamp) =>
        new(request, true, value, quality, sourceTimestamp, null, null, false);

    public static HslIec104PointReadResult Failure(
        HslIec104ReadRequest request,
        string errorCode,
        string errorMessage,
        bool retryable) =>
        new(request, false, null, null, null, errorCode, errorMessage, retryable);
}

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

public interface IHslModbusRtuClient : IAsyncDisposable
{
    bool IsConnected { get; }

    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);

    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);

    Task<HslReadResult> ReadAsync(HslReadRequest request, CancellationToken cancellationToken);
}

public interface IHslModbusRtuClientFactory
{
    IHslModbusRtuClient Create(HslModbusRtuClientOptions options);
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

public interface IHslSiemensVariantClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslS7ReadRequest request, CancellationToken cancellationToken);
}

public interface IHslSiemensVariantClientFactory
{
    IHslSiemensVariantClient Create(HslSiemensVariantClientOptions options);
}

public interface IHslMelsecMcClient : IAsyncDisposable
{
    bool IsConnected { get; }

    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);

    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);

    Task<HslReadResult> ReadAsync(HslMelsecMcReadRequest request, CancellationToken cancellationToken);
}

public interface IHslMelsecMcClientFactory
{
    IHslMelsecMcClient Create(HslMelsecMcClientOptions options);
}

public interface IHslMelsecNetworkClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslMelsecNetworkReadRequest request, CancellationToken cancellationToken);
}

public interface IHslMelsecNetworkClientFactory
{
    IHslMelsecNetworkClient Create(HslMelsecNetworkClientOptions options);
}

public interface IHslMelsecSerialClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslMelsecSerialReadRequest request, CancellationToken cancellationToken);
}

public interface IHslMelsecSerialClientFactory
{
    IHslMelsecSerialClient Create(HslMelsecSerialClientOptions options);
}

public interface IHslOmronFinsClient : IAsyncDisposable
{
    bool IsConnected { get; }

    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);

    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);

    Task<HslReadResult> ReadAsync(HslOmronFinsReadRequest request, CancellationToken cancellationToken);
}

public interface IHslOmronFinsClientFactory
{
    IHslOmronFinsClient Create(HslOmronFinsClientOptions options);
}

public interface IHslOmronVariantClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslOmronVariantReadRequest request, CancellationToken cancellationToken);
}

public interface IHslOmronVariantClientFactory
{
    IHslOmronVariantClient Create(HslOmronVariantClientOptions options);
}

public interface IHslAllenBradleyClient : IAsyncDisposable
{
    bool IsConnected { get; }

    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);

    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);

    Task<HslReadResult> ReadAsync(HslAllenBradleyReadRequest request, CancellationToken cancellationToken);
}

public interface IHslAllenBradleyClientFactory
{
    IHslAllenBradleyClient Create(HslAllenBradleyClientOptions options);
}

public interface IHslBeckhoffAdsClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslBeckhoffAdsReadRequest request, CancellationToken cancellationToken);
}

public interface IHslBeckhoffAdsClientFactory
{
    IHslBeckhoffAdsClient Create(HslBeckhoffAdsClientOptions options);
}

public interface IHslIec104Client : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<IReadOnlyList<HslIec104PointReadResult>> ReadAsync(
        IReadOnlyList<HslIec104ReadRequest> requests,
        CancellationToken cancellationToken);
}

public interface IHslIec104ClientFactory
{
    IHslIec104Client Create(HslIec104ClientOptions options);
}

public interface IHslPanasonicMewtocolClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslPanasonicMewtocolReadRequest request, CancellationToken cancellationToken);
}

public interface IHslPanasonicMewtocolClientFactory
{
    IHslPanasonicMewtocolClient Create(HslPanasonicMewtocolClientOptions options);
}

public interface IHslPanasonicMcClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslPanasonicMcReadRequest request, CancellationToken cancellationToken);
}

public interface IHslPanasonicMcClientFactory
{
    IHslPanasonicMcClient Create(HslPanasonicMcClientOptions options);
}

public interface IHslLsisFastEnetClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslLsisFastEnetReadRequest request, CancellationToken cancellationToken);
}

public interface IHslLsisFastEnetClientFactory
{
    IHslLsisFastEnetClient Create(HslLsisFastEnetClientOptions options);
}

public interface IHslLsisSerialClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslLsisSerialReadRequest request, CancellationToken cancellationToken);
}

public interface IHslLsisSerialClientFactory
{
    IHslLsisSerialClient Create(HslLsisSerialClientOptions options);
}

public interface IHslGeSrtpClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslGeSrtpReadRequest request, CancellationToken cancellationToken);
}

public interface IHslGeSrtpClientFactory
{
    IHslGeSrtpClient Create(HslGeSrtpClientOptions options);
}

public interface IHslInovanceModbusTcpClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslInovanceModbusTcpReadRequest request, CancellationToken cancellationToken);
}

public interface IHslInovanceModbusTcpClientFactory
{
    IHslInovanceModbusTcpClient Create(HslInovanceModbusTcpClientOptions options);
}

public interface IHslInovanceModbusVariantClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslInovanceModbusVariantReadRequest request, CancellationToken cancellationToken);
}

public interface IHslInovanceModbusVariantClientFactory
{
    IHslInovanceModbusVariantClient Create(HslInovanceModbusVariantClientOptions options);
}

public interface IHslInovanceSpecialClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslInovanceSpecialReadRequest request, CancellationToken cancellationToken);
}

public interface IHslInovanceSpecialClientFactory
{
    IHslInovanceSpecialClient Create(HslInovanceSpecialClientOptions options);
}

public interface IHslFatekProgramClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslFatekProgramReadRequest request, CancellationToken cancellationToken);
}

public interface IHslFatekProgramTcpClientFactory
{
    IHslFatekProgramClient Create(HslFatekProgramTcpClientOptions options);
}

public interface IHslFatekProgramSerialClientFactory
{
    IHslFatekProgramClient Create(HslFatekProgramSerialClientOptions options);
}

public interface IHslFreedomClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslFreedomReadRequest request, CancellationToken cancellationToken);
}

public interface IHslFreedomClientFactory
{
    IHslFreedomClient Create(HslFreedomClientOptions options);
}

public interface IHslCimonClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslCimonReadRequest request, CancellationToken cancellationToken);
}

public interface IHslCimonClientFactory
{
    IHslCimonClient Create(HslCimonClientOptions options);
}

public interface IHslYamatakeClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslYamatakeReadRequest request, CancellationToken cancellationToken);
}

public interface IHslYamatakeClientFactory
{
    IHslYamatakeClient Create(HslYamatakeClientOptions options);
}

public interface IHslCjt188Client : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslCjt188ReadRequest request, CancellationToken cancellationToken);
}

public interface IHslCjt188ClientFactory
{
    IHslCjt188Client Create(HslCjt188ClientOptions options);
}

public interface IHslRkcClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslRkcReadRequest request, CancellationToken cancellationToken);
}

public interface IHslRkcClientFactory
{
    IHslRkcClient Create(HslRkcClientOptions options);
}

public interface IHslInstrumentSerialClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslInstrumentSerialReadRequest request, CancellationToken cancellationToken);
}

public interface IHslInstrumentSerialClientFactory
{
    IHslInstrumentSerialClient Create(HslInstrumentSerialClientOptions options);
}

public interface IHslDltClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslDltReadRequest request, CancellationToken cancellationToken);
}

public interface IHslDltClientFactory
{
    IHslDltClient Create(HslDltClientOptions options);
}

public interface IHslYaskawaClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslYaskawaReadRequest request, CancellationToken cancellationToken);
}

public interface IHslYaskawaClientFactory
{
    IHslYaskawaClient Create(HslYaskawaClientOptions options);
}

public interface IHslSpecializedNetworkClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslSpecializedNetworkReadRequest request, CancellationToken cancellationToken);
}

public interface IHslSpecializedNetworkClientFactory
{
    IHslSpecializedNetworkClient Create(HslSpecializedNetworkClientOptions options);
}

public interface IHslKeyenceTcpClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslKeyenceTcpReadRequest request, CancellationToken cancellationToken);
}

public interface IHslKeyenceTcpClientFactory
{
    IHslKeyenceTcpClient Create(HslKeyenceTcpClientOptions options);
}

public interface IHslDeltaClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslDeltaReadRequest request, CancellationToken cancellationToken);
}

public interface IHslDeltaClientFactory
{
    IHslDeltaClient Create(HslDeltaClientOptions options);
}

public interface IHslXinjeClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslXinjeReadRequest request, CancellationToken cancellationToken);
}

public interface IHslXinjeClientFactory
{
    IHslXinjeClient Create(HslXinjeClientOptions options);
}

public interface IHslMegMeetClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslMegMeetReadRequest request, CancellationToken cancellationToken);
}

public interface IHslMegMeetClientFactory
{
    IHslMegMeetClient Create(HslMegMeetClientOptions options);
}

public interface IHslFujiClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslFujiReadRequest request, CancellationToken cancellationToken);
}

public interface IHslFujiClientFactory
{
    IHslFujiClient Create(HslFujiClientOptions options);
}

public interface IHslVigorClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslVigorReadRequest request, CancellationToken cancellationToken);
}

public interface IHslVigorClientFactory
{
    IHslVigorClient Create(HslVigorClientOptions options);
}

public interface IHslYokogawaClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslYokogawaReadRequest request, CancellationToken cancellationToken);
}
public interface IHslYokogawaClientFactory { IHslYokogawaClient Create(HslYokogawaClientOptions options); }

public sealed class HslS7TcpClientFactory : IHslS7TcpClientFactory
{
    public IHslS7TcpClient Create(HslS7TcpClientOptions options) => new HslS7TcpClient(options);
}

public sealed class HslSiemensVariantClientFactory : IHslSiemensVariantClientFactory
{
    public IHslSiemensVariantClient Create(HslSiemensVariantClientOptions options) => new HslSiemensVariantClient(options);
}

public sealed class HslModbusTcpClientFactory : IHslModbusTcpClientFactory
{
    public IHslModbusTcpClient Create(HslModbusTcpClientOptions options) => new HslModbusTcpClient(options);
}

public sealed class HslModbusRtuClientFactory : IHslModbusRtuClientFactory
{
    public IHslModbusRtuClient Create(HslModbusRtuClientOptions options) => new HslModbusRtuClient(options);
}

public sealed class HslMelsecMcClientFactory : IHslMelsecMcClientFactory
{
    public IHslMelsecMcClient Create(HslMelsecMcClientOptions options) => new HslMelsecMcClient(options);
}

public sealed class HslMelsecNetworkClientFactory : IHslMelsecNetworkClientFactory
{
    public IHslMelsecNetworkClient Create(HslMelsecNetworkClientOptions options) => new HslMelsecNetworkClient(options);
}

public sealed class HslMelsecSerialClientFactory : IHslMelsecSerialClientFactory
{
    public IHslMelsecSerialClient Create(HslMelsecSerialClientOptions options) => new HslMelsecSerialClient(options);
}

public sealed class HslOmronFinsClientFactory : IHslOmronFinsClientFactory
{
    public IHslOmronFinsClient Create(HslOmronFinsClientOptions options) => new HslOmronFinsClient(options);
}

public sealed class HslOmronVariantClientFactory : IHslOmronVariantClientFactory
{
    public IHslOmronVariantClient Create(HslOmronVariantClientOptions options) => new HslOmronVariantClient(options);
}

public sealed class HslAllenBradleyClientFactory : IHslAllenBradleyClientFactory
{
    public IHslAllenBradleyClient Create(HslAllenBradleyClientOptions options) => new HslAllenBradleyClient(options);
}

public sealed class HslBeckhoffAdsClientFactory : IHslBeckhoffAdsClientFactory
{
    public IHslBeckhoffAdsClient Create(HslBeckhoffAdsClientOptions options) => new HslBeckhoffAdsClient(options);
}

public sealed class HslIec104ClientFactory : IHslIec104ClientFactory
{
    public IHslIec104Client Create(HslIec104ClientOptions options) => new HslIec104Client(options);
}

public sealed class HslPanasonicMewtocolClientFactory : IHslPanasonicMewtocolClientFactory
{
    public IHslPanasonicMewtocolClient Create(HslPanasonicMewtocolClientOptions options) => new HslPanasonicMewtocolClient(options);
}

public sealed class HslPanasonicMcClientFactory : IHslPanasonicMcClientFactory
{
    public IHslPanasonicMcClient Create(HslPanasonicMcClientOptions options) => new HslPanasonicMcClient(options);
}

public sealed class HslLsisFastEnetClientFactory : IHslLsisFastEnetClientFactory
{
    public IHslLsisFastEnetClient Create(HslLsisFastEnetClientOptions options) => new HslLsisFastEnetClient(options);
}

public sealed class HslLsisSerialClientFactory : IHslLsisSerialClientFactory
{
    public IHslLsisSerialClient Create(HslLsisSerialClientOptions options) => new HslLsisSerialClient(options);
}

public sealed class HslGeSrtpClientFactory : IHslGeSrtpClientFactory
{
    public IHslGeSrtpClient Create(HslGeSrtpClientOptions options) => new HslGeSrtpClient(options);
}

public sealed class HslInovanceModbusTcpClientFactory : IHslInovanceModbusTcpClientFactory
{
    public IHslInovanceModbusTcpClient Create(HslInovanceModbusTcpClientOptions options) => new HslInovanceModbusTcpClient(options);
}

public sealed class HslInovanceModbusVariantClientFactory : IHslInovanceModbusVariantClientFactory
{
    public IHslInovanceModbusVariantClient Create(HslInovanceModbusVariantClientOptions options) => new HslInovanceModbusVariantClient(options);
}

public sealed class HslInovanceSpecialClientFactory : IHslInovanceSpecialClientFactory
{
    public IHslInovanceSpecialClient Create(HslInovanceSpecialClientOptions options) => new HslInovanceSpecialClient(options);
}

public sealed class HslFatekProgramTcpClientFactory : IHslFatekProgramTcpClientFactory
{
    public IHslFatekProgramClient Create(HslFatekProgramTcpClientOptions options) => new HslFatekProgramTcpClient(options);
}

public sealed class HslFatekProgramSerialClientFactory : IHslFatekProgramSerialClientFactory
{
    public IHslFatekProgramClient Create(HslFatekProgramSerialClientOptions options) => new HslFatekProgramSerialClient(options);
}

public sealed class HslFreedomClientFactory : IHslFreedomClientFactory
{
    public IHslFreedomClient Create(HslFreedomClientOptions options) => new HslFreedomClient(options);
}

public sealed class HslCimonClientFactory : IHslCimonClientFactory
{
    public IHslCimonClient Create(HslCimonClientOptions options) => new HslCimonClient(options);
}

public sealed class HslYamatakeClientFactory : IHslYamatakeClientFactory
{
    public IHslYamatakeClient Create(HslYamatakeClientOptions options) => new HslYamatakeClient(options);
}

public sealed class HslCjt188ClientFactory : IHslCjt188ClientFactory
{
    public IHslCjt188Client Create(HslCjt188ClientOptions options) => new HslCjt188Client(options);
}

public sealed class HslRkcClientFactory : IHslRkcClientFactory
{
    public IHslRkcClient Create(HslRkcClientOptions options) => new HslRkcClient(options);
}

public sealed class HslInstrumentSerialClientFactory : IHslInstrumentSerialClientFactory
{
    public IHslInstrumentSerialClient Create(HslInstrumentSerialClientOptions options) => new HslInstrumentSerialClient(options);
}

public sealed class HslDltClientFactory : IHslDltClientFactory
{
    public IHslDltClient Create(HslDltClientOptions options) => new HslDltClient(options);
}

public sealed class HslYaskawaClientFactory : IHslYaskawaClientFactory
{
    public IHslYaskawaClient Create(HslYaskawaClientOptions options) => new HslYaskawaClient(options);
}

public sealed class HslSpecializedNetworkClientFactory : IHslSpecializedNetworkClientFactory
{
    public IHslSpecializedNetworkClient Create(HslSpecializedNetworkClientOptions options) => new HslSpecializedNetworkClient(options);
}

public sealed class HslKeyenceTcpClientFactory : IHslKeyenceTcpClientFactory
{
    public IHslKeyenceTcpClient Create(HslKeyenceTcpClientOptions options) => new HslKeyenceTcpClient(options);
}

public sealed class HslDeltaClientFactory : IHslDeltaClientFactory
{
    public IHslDeltaClient Create(HslDeltaClientOptions options) => new HslDeltaClient(options);
}

public sealed class HslXinjeClientFactory : IHslXinjeClientFactory
{
    public IHslXinjeClient Create(HslXinjeClientOptions options) => new HslXinjeClient(options);
}

public sealed class HslMegMeetClientFactory : IHslMegMeetClientFactory
{
    public IHslMegMeetClient Create(HslMegMeetClientOptions options) => new HslMegMeetClient(options);
}

public sealed class HslFujiClientFactory : IHslFujiClientFactory
{
    public IHslFujiClient Create(HslFujiClientOptions options) => new HslFujiClient(options);
}

public sealed class HslVigorClientFactory : IHslVigorClientFactory
{
    public IHslVigorClient Create(HslVigorClientOptions options) => new HslVigorClient(options);
}

public sealed class HslYokogawaClientFactory : IHslYokogawaClientFactory
{
    public IHslYokogawaClient Create(HslYokogawaClientOptions options) => new HslYokogawaClient(options);
}

public enum HslEcFanMetric { SpeedEmergency, SpeedMinimum, SpeedMaximum }
public sealed record HslEcFanClientOptions(string PortName, int BaudRate, int DataBits, HslSerialParity Parity, HslSerialStopBits StopBits, byte Station, int ReceiveTimeoutMilliseconds);
public interface IHslEcFanClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslEcFanMetric metric, CancellationToken cancellationToken);
}
public interface IHslEcFanClientFactory { IHslEcFanClient Create(HslEcFanClientOptions options); }
public sealed class HslEcFanClientFactory : IHslEcFanClientFactory { public IHslEcFanClient Create(HslEcFanClientOptions options) => new HslEcFanClient(options); }

public sealed record HslMqttRpcClientOptions(string Host, int Port, string ClientId, string? UserName, string? Password, string DeviceTopic, bool UseRsa);
public sealed record HslMqttRpcReadRequest(string Address, HslValueType DataType, int ElementCount);
public interface IHslMqttRpcClient : IAsyncDisposable
{
    bool IsConnected { get; }
    Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken);
    Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken);
    Task<HslReadResult> ReadAsync(HslMqttRpcReadRequest request, CancellationToken cancellationToken);
}
public interface IHslMqttRpcClientFactory { IHslMqttRpcClient Create(HslMqttRpcClientOptions options); }
public sealed class HslMqttRpcClientFactory : IHslMqttRpcClientFactory { public IHslMqttRpcClient Create(HslMqttRpcClientOptions options) => new HslMqttRpcClient(options); }
