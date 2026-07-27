using System.IO.Ports;
using System.Text.RegularExpressions;
using HslCommunication;
using HslCommunication.Core.Device;
using HslCommunication.Instrument.Delixi;
using HslCommunication.Instrument.Temperature;

namespace InduForge.Collector.Adapters.Hsl;

public sealed partial class HslInstrumentSerialClient : IHslInstrumentSerialClient
{
    private readonly HslInstrumentSerialProtocol _protocol;
    private readonly DeviceSerialPort _device;
    private readonly bool _addressStartWithZero;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslInstrumentSerialClient(HslInstrumentSerialClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        if (string.IsNullOrWhiteSpace(options.PortName)) throw new ArgumentException("仪表串口名称不能为空", nameof(options));

        _protocol = options.Protocol;
        _addressStartWithZero = options.AddressStartWithZero;
        _device = CreateDevice(options);
        _device.SerialPortInni(options.PortName, options.BaudRate, options.DataBits, StopBitsOf(options.StopBits), ParityOf(options.Parity));
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();
            var result = await Task.Run(_device.Open, cancellationToken).ConfigureAwait(false);
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_INSTRUMENT_SERIAL_CONNECT_FAILED", exception.Message, true);
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
            await Task.Run(_device.Close, cancellationToken).ConfigureAwait(false);
            _connected = false;
            return HslOperationResult.Success();
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_INSTRUMENT_SERIAL_CLOSE_FAILED", exception.Message, false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslInstrumentSerialReadRequest request, CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", "仪表串口连接尚未建立", true);
            return _protocol == HslInstrumentSerialProtocol.Dam3601
                ? await ReadDam3601Async(request, cancellationToken).ConfigureAwait(false)
                : await HslTypedDeviceReader.ReadAsync(_device, request.Address, request.DataType, request.ElementCount, "HSL_INSTRUMENT_SERIAL_READ_FAILED", DisplayName(), cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure("HSL_INSTRUMENT_SERIAL_READ_FAILED", exception.Message, true); }
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

    private async Task<HslReadResult> ReadDam3601Async(HslInstrumentSerialReadRequest request, CancellationToken cancellationToken)
    {
        if (request.DataType != HslValueType.SinglePrecision || request.ElementCount is < 1 or > 128)
        {
            return HslReadResult.Failure("HSL_DAM3601_READ_SHAPE_INVALID", "DAM3601 仅支持读取 float32 温度通道", false);
        }
        var match = DamAddressPattern().Match(request.Address);
        if (!match.Success || !int.TryParse(match.Groups[2].Value, out var channel) || channel + request.ElementCount > 128)
        {
            return HslReadResult.Failure("HSL_DAM3601_ADDRESS_INVALID", "DAM3601 温度通道必须在 0 到 127 之间", false);
        }

        var prefix = match.Groups[1].Success ? $"s={match.Groups[1].Value};" : string.Empty;
        var register = channel + (_addressStartWithZero ? 0 : 1);
        var result = await _device.ReadInt16Async($"{prefix}x=4;{register}", checked((ushort)request.ElementCount)).WaitAsync(cancellationToken).ConfigureAwait(false);
        if (!result.IsSuccess) return HslReadResult.Failure("HSL_DAM3601_READ_FAILED", Format(result), true);
        var values = result.Content.Select(TransformDamTemperature).ToArray();
        return HslReadResult.Success(values.Length == 1 ? values[0] : values);
    }

    private static float TransformDamTemperature(short value) =>
        (value & 0x800) > 0 ? (((value & 0xFFF) ^ 0xFFF) + 1) * -0.0625f : (value & 0x7FF) * 0.0625f;

    private static DeviceSerialPort CreateDevice(HslInstrumentSerialClientOptions options) => options.Protocol switch
    {
        HslInstrumentSerialProtocol.Dam3601 => ConfigureModbus(new DAM3601(options.Station), options),
        HslInstrumentSerialProtocol.YuDianAiBus => new YuDianAIBus { Station = options.Station, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds },
        HslInstrumentSerialProtocol.Dtsu6606 => ConfigureModbus(new DTSU6606Serial(options.Station), options),
        _ => throw new ArgumentOutOfRangeException(nameof(options)),
    };

    private static DeviceSerialPort ConfigureModbus(DeviceSerialPort device, HslInstrumentSerialClientOptions options)
    {
        dynamic modbus = device;
        modbus.AddressStartWithZero = options.AddressStartWithZero;
        modbus.IsStringReverse = options.StringReverse;
        modbus.DataFormat = options.DataFormat switch
        {
            HslDataFormat.ABCD => HslCommunication.Core.DataFormat.ABCD,
            HslDataFormat.BADC => HslCommunication.Core.DataFormat.BADC,
            HslDataFormat.CDAB => HslCommunication.Core.DataFormat.CDAB,
            _ => HslCommunication.Core.DataFormat.DCBA,
        };
        device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
        return device;
    }

    private string DisplayName() => _protocol switch
    {
        HslInstrumentSerialProtocol.YuDianAiBus => "宇电 AIBus",
        HslInstrumentSerialProtocol.Dtsu6606 => "德力西 DTSU6606",
        _ => "DAM3601",
    };

    private static string Format(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"仪表读取失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    private static HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure($"HSL_INSTRUMENT_SERIAL_{operation}_FAILED", Format(result), retryable);
    private static StopBits StopBitsOf(HslSerialStopBits value) => value == HslSerialStopBits.Two ? StopBits.Two : StopBits.One;
    private static Parity ParityOf(HslSerialParity value) => value switch { HslSerialParity.Odd => Parity.Odd, HslSerialParity.Even => Parity.Even, _ => Parity.None };
    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);

    [GeneratedRegex("^(?:s=([0-9]{1,3});)?temperature\\[([0-9]{1,3})\\]$", RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)]
    private static partial Regex DamAddressPattern();
}
