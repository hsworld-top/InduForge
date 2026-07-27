using System.IO.Ports;
using System.Text;
using HslCommunication;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.Melsec;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslMelsecSerialClient : IHslMelsecSerialClient
{
    private readonly DeviceCommunication _device;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslMelsecSerialClient(HslMelsecSerialClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = CreateDevice(options);
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();
            var result = _device switch
            {
                DeviceTcpNet tcp => await tcp.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false),
                DeviceSerialPort serial => await Task.Run(serial.Open, cancellationToken).ConfigureAwait(false),
                _ => throw new InvalidOperationException("未知的三菱串行通信设备类型"),
            };
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_MELSEC_SERIAL_CONNECT_FAILED", exception.Message, true);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); }
            OperateResult result;
            if (_device is DeviceTcpNet tcp) result = await tcp.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            else if (_device is DeviceSerialPort serial)
            {
                await Task.Run(serial.Close, cancellationToken).ConfigureAwait(false);
                result = OperateResult.CreateSuccessResult();
            }
            else throw new InvalidOperationException("未知的三菱串行通信设备类型");
            _connected = false;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CLOSE", result, false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_MELSEC_SERIAL_CLOSE_FAILED", exception.Message, false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslMelsecSerialReadRequest request, CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "三菱串行通信连接尚未建立", true);
            var count = checked((ushort)request.ElementCount);
            return request.DataType switch
            {
                HslValueType.Boolean => request.ElementCount == 1 ? Map(await _device.ReadBoolAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Signed8 => await ReadRawAsync(request, bytes => request.ElementCount == 1 ? unchecked((sbyte)bytes[0]) : bytes.Select(value => unchecked((sbyte)value)).ToArray(), cancellationToken),
                HslValueType.Unsigned8 => await ReadRawAsync(request, bytes => request.ElementCount == 1 ? bytes[0] : bytes, cancellationToken),
                HslValueType.Signed16 => request.ElementCount == 1 ? Map(await _device.ReadInt16Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Unsigned16 => request.ElementCount == 1 ? Map(await _device.ReadUInt16Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Signed32 => request.ElementCount == 1 ? Map(await _device.ReadInt32Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Unsigned32 => request.ElementCount == 1 ? Map(await _device.ReadUInt32Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Signed64 => request.ElementCount == 1 ? Map(await _device.ReadInt64Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Unsigned64 => request.ElementCount == 1 ? Map(await _device.ReadUInt64Async(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.SinglePrecision => request.ElementCount == 1 ? Map(await _device.ReadFloatAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.DoublePrecision => request.ElementCount == 1 ? Map(await _device.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken)) : Map(await _device.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken)),
                HslValueType.Text => Map(await _device.ReadStringAsync(request.Address, count, Encoding.ASCII).WaitAsync(cancellationToken)),
                HslValueType.Binary => await ReadRawAsync(request, bytes => bytes, cancellationToken),
                _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "当前三菱串行协议不支持该数据类型", false),
            };
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_MELSEC_SERIAL_READ_FAILED", exception.Message, true); }
        finally { _operationGate.Release(); }
    }

    public async ValueTask DisposeAsync()
    {
        if (_disposed) return;
        await CloseAsync(CancellationToken.None).ConfigureAwait(false);
        _disposed = true;
        _device.Dispose();
        _operationGate.Dispose();
    }

    private async Task<HslReadResult> ReadRawAsync(HslMelsecSerialReadRequest request, Func<byte[], object> convert, CancellationToken cancellationToken)
    {
        var wordCount = checked((ushort)Math.Ceiling(request.ElementCount / 2d));
        var result = await _device.ReadAsync(request.Address, wordCount).WaitAsync(cancellationToken);
        return result.IsSuccess ? HslReadResult.Success(convert(result.Content.Take(request.ElementCount).ToArray())) : HslReadResult.Failure("HSL_MELSEC_SERIAL_READ_FAILED", FormatError(result), true);
    }

    private static DeviceCommunication CreateDevice(HslMelsecSerialClientOptions options)
    {
        DeviceCommunication device = options.Protocol switch
        {
            HslMelsecSerialProtocol.A3CSerial => CreateA3CSerial(options),
            HslMelsecSerialProtocol.A3CSerialOverTcp => new MelsecA3CNetOverTcp(options.Host, options.Port)
            {
                Station = options.Station,
                SumCheck = options.SumCheck,
                Format = options.Format,
                ConnectTimeOut = options.ConnectTimeoutMilliseconds,
                ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
            },
            HslMelsecSerialProtocol.FxLinksSerial => CreateFxLinksSerial(options),
            HslMelsecSerialProtocol.FxLinksOverTcp => new MelsecFxLinksOverTcp(options.Host, options.Port)
            {
                Station = options.Station,
                WaittingTime = options.WaitingTime,
                SumCheck = options.SumCheck,
                Format = options.Format,
                ConnectTimeOut = options.ConnectTimeoutMilliseconds,
                ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
            },
            HslMelsecSerialProtocol.FxSerial => CreateFxSerial(options),
            HslMelsecSerialProtocol.FxSerialOverTcp => new MelsecFxSerialOverTcp(options.Host, options.Port)
            {
                IsNewVersion = options.IsNewVersion,
                UseGOT = options.UseGot,
                ConnectTimeOut = options.ConnectTimeoutMilliseconds,
                ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
            },
            _ => throw new ArgumentOutOfRangeException(nameof(options), "不支持的三菱串行协议"),
        };
        return device;
    }

    private static MelsecA3CNet CreateA3CSerial(HslMelsecSerialClientOptions options)
    {
        var device = new MelsecA3CNet
        {
            Station = options.Station,
            SumCheck = options.SumCheck,
            Format = options.Format,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
        };
        InitializeSerialPort(device, options);
        return device;
    }

    private static MelsecFxLinks CreateFxLinksSerial(HslMelsecSerialClientOptions options)
    {
        var device = new MelsecFxLinks
        {
            Station = options.Station,
            WaittingTime = options.WaitingTime,
            SumCheck = options.SumCheck,
            Format = options.Format,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
        };
        InitializeSerialPort(device, options);
        return device;
    }

    private static MelsecFxSerial CreateFxSerial(HslMelsecSerialClientOptions options)
    {
        var device = new MelsecFxSerial
        {
            IsNewVersion = options.IsNewVersion,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
        };
        InitializeSerialPort(device, options);
        return device;
    }

    private static void InitializeSerialPort(DeviceSerialPort device, HslMelsecSerialClientOptions options)
    {
        if (string.IsNullOrWhiteSpace(options.PortName)) throw new ArgumentException("三菱串口名称不能为空", nameof(options));
        device.SerialPortInni(options.PortName, options.BaudRate, options.DataBits, MapStopBits(options.StopBits), MapParity(options.Parity));
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess ? HslReadResult.Success(result.Content) : HslReadResult.Failure("HSL_MELSEC_SERIAL_READ_FAILED", FormatError(result), true);
    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure($"HSL_MELSEC_SERIAL_{operation}_FAILED", FormatError(result), retryable);
    private static StopBits MapStopBits(HslSerialStopBits value) => value == HslSerialStopBits.Two ? StopBits.Two : StopBits.One;
    private static Parity MapParity(HslSerialParity value) => value switch { HslSerialParity.None => Parity.None, HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None };
    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"三菱串行通信失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
