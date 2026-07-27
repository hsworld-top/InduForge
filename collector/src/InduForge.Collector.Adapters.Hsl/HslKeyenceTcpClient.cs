using System.IO.Ports;
using System.Text;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.Keyence;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslKeyenceTcpClient : IHslKeyenceTcpClient
{
    private readonly DeviceCommunication _device;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslKeyenceTcpClient(HslKeyenceTcpClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _device = CreateDevice(options);
        _device.ByteTransform.DataFormat = MapDataFormat(options.DataFormat);
        _device.ByteTransform.IsStringReverseByteWord = options.StringReverse;
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
                _ => throw new InvalidOperationException("未知的基恩士通信设备类型"),
            };
            _connected = result.IsSuccess;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_KEYENCE_CONNECT_FAILED", FormatError(result), retryable: true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_KEYENCE_CONNECT_FAILED", exception.Message, retryable: true);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected)
            {
                _connected = false;
                return HslOperationResult.Success();
            }
            OperateResult result;
            if (_device is DeviceTcpNet tcp)
            {
                result = await tcp.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            }
            else if (_device is DeviceSerialPort serial)
            {
                await Task.Run(serial.Close, cancellationToken).ConfigureAwait(false);
                result = OperateResult.CreateSuccessResult();
            }
            else
            {
                throw new InvalidOperationException("未知的基恩士通信设备类型");
            }
            _connected = false;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_KEYENCE_CLOSE_FAILED", FormatError(result), retryable: false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_KEYENCE_CLOSE_FAILED", exception.Message, retryable: false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslKeyenceTcpReadRequest request, CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(request);
        if (request.ElementCount < 1)
            return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", retryable: false);

        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected)
                return HslReadResult.Failure("HSL_NOT_CONNECTED", "基恩士通信连接尚未建立", retryable: true);
            return await ReadCoreAsync(request, cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            return HslReadResult.Failure("HSL_KEYENCE_READ_FAILED", exception.Message, retryable: true);
        }
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

    private static DeviceCommunication CreateDevice(HslKeyenceTcpClientOptions options)
    {
        DeviceCommunication device = options.Protocol switch
        {
            HslKeyenceProtocol.McBinary => new KeyenceMcNet(options.Host, options.Port)
            {
                NetworkNumber = options.NetworkNumber,
                NetworkStationNumber = options.NetworkStationNumber,
                TargetIOStation = options.TargetIoStation,
                EnableWriteBitToWordRegister = options.EnableWriteBitToWordRegister,
            },
            HslKeyenceProtocol.McAscii => new KeyenceMcAsciiNet(options.Host, options.Port)
            {
                NetworkNumber = options.NetworkNumber,
                NetworkStationNumber = options.NetworkStationNumber,
                TargetIOStation = options.TargetIoStation,
                EnableWriteBitToWordRegister = options.EnableWriteBitToWordRegister,
            },
            HslKeyenceProtocol.KvOld => new KeyenceKvOld(options.Host, options.Port),
            HslKeyenceProtocol.Nano => new KeyenceNanoSerialOverTcp(options.Host, options.Port)
            {
                Station = options.Station,
                UseStation = options.UseStation,
            },
            HslKeyenceProtocol.NanoSerialOverTcp => new KeyenceNanoSerialOverTcp(options.Host, options.Port)
            {
                Station = options.Station,
                UseStation = options.UseStation,
            },
            HslKeyenceProtocol.NanoSerial => CreateNanoSerial(options),
            _ => throw new ArgumentOutOfRangeException(nameof(options), options.Protocol, "未知的基恩士协议类型"),
        };
        device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
        if (device is DeviceTcpNet tcp) tcp.ConnectTimeOut = options.ConnectTimeoutMilliseconds;
        return device;
    }

    private static KeyenceNanoSerial CreateNanoSerial(HslKeyenceTcpClientOptions options)
    {
        if (string.IsNullOrWhiteSpace(options.PortName))
            throw new ArgumentException("基恩士 Nano 串口名称不能为空", nameof(options));

        var device = new KeyenceNanoSerial { Station = options.Station, UseStation = options.UseStation };
        device.SerialPortInni(
            options.PortName,
            options.BaudRate,
            options.DataBits,
            options.StopBits == HslSerialStopBits.Two ? StopBits.Two : StopBits.One,
            options.Parity switch
            {
                HslSerialParity.Odd => Parity.Odd,
                HslSerialParity.Even => Parity.Even,
                _ => Parity.None,
            });
        return device;
    }

    private async Task<HslReadResult> ReadCoreAsync(HslKeyenceTcpReadRequest request, CancellationToken cancellationToken)
    {
        var count = checked((ushort)request.ElementCount);
        return request.DataType switch
        {
            HslValueType.Boolean => request.ElementCount == 1
                ? Map(await _device.ReadBoolAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _device.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed8 => await ReadRawAsync(request, bytes => request.ElementCount == 1 ? unchecked((sbyte)bytes[0]) : bytes.Select(value => unchecked((sbyte)value)).ToArray(), cancellationToken).ConfigureAwait(false),
            HslValueType.Unsigned8 => await ReadRawAsync(request, bytes => request.ElementCount == 1 ? bytes[0] : bytes, cancellationToken).ConfigureAwait(false),
            HslValueType.Signed16 => request.ElementCount == 1 ? Map(await _device.ReadInt16Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)) : Map(await _device.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned16 => request.ElementCount == 1 ? Map(await _device.ReadUInt16Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)) : Map(await _device.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed32 => request.ElementCount == 1 ? Map(await _device.ReadInt32Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)) : Map(await _device.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned32 => request.ElementCount == 1 ? Map(await _device.ReadUInt32Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)) : Map(await _device.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed64 => request.ElementCount == 1 ? Map(await _device.ReadInt64Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)) : Map(await _device.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned64 => request.ElementCount == 1 ? Map(await _device.ReadUInt64Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)) : Map(await _device.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.SinglePrecision => request.ElementCount == 1 ? Map(await _device.ReadFloatAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)) : Map(await _device.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.DoublePrecision => request.ElementCount == 1 ? Map(await _device.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)) : Map(await _device.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Text => Map(await _device.ReadStringAsync(request.Address, count, Encoding.UTF8).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Binary => await ReadRawAsync(request, bytes => bytes, cancellationToken).ConfigureAwait(false),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "基恩士读取数据类型不受支持", retryable: false),
        };
    }

    private async Task<HslReadResult> ReadRawAsync(HslKeyenceTcpReadRequest request, Func<byte[], object> convert, CancellationToken cancellationToken)
    {
        var wordCount = checked((ushort)Math.Ceiling(request.ElementCount / 2d));
        var result = await _device.ReadAsync(request.Address, wordCount).WaitAsync(cancellationToken).ConfigureAwait(false);
        return result.IsSuccess
            ? HslReadResult.Success(convert(result.Content.Take(request.ElementCount).ToArray()))
            : HslReadResult.Failure("HSL_KEYENCE_READ_FAILED", FormatError(result), retryable: true);
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess
        ? HslReadResult.Success(result.Content)
        : HslReadResult.Failure("HSL_KEYENCE_READ_FAILED", FormatError(result), retryable: true);

    private static DataFormat MapDataFormat(HslDataFormat format) => format switch
    {
        HslDataFormat.ABCD => DataFormat.ABCD,
        HslDataFormat.BADC => DataFormat.BADC,
        HslDataFormat.CDAB => DataFormat.CDAB,
        HslDataFormat.DCBA => DataFormat.DCBA,
        _ => throw new ArgumentOutOfRangeException(nameof(format), format, "未知的数据格式"),
    };

    private static string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message)
        ? $"基恩士通信操作失败，错误码 {result.ErrorCode}"
        : $"{result.Message}（通信错误码 {result.ErrorCode}）";

    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
