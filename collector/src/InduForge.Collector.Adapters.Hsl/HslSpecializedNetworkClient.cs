using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Core.Device;
using HslCommunication.DCS;
using HslCommunication.Profinet.OrientalMotor;
using HslCommunication.Profinet.Toyota;
using HslCommunication.Profinet.Turck;
using HslCommunication.Robot.Estun;
using HslCommunication.Robot.FANUC;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslSpecializedNetworkClient : IHslSpecializedNetworkClient
{
    private readonly DeviceTcpNet _device;
    private readonly string _errorPrefix;
    private readonly string _displayName;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslSpecializedNetworkClient(HslSpecializedNetworkClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        (_device, _errorPrefix, _displayName) = options.Protocol switch
        {
            HslSpecializedNetworkProtocol.OrientalMotorEip => ((DeviceTcpNet)CreateOrientalMotor(options), "ORIENTAL_MOTOR", "东方马达 EtherNet/IP"),
            HslSpecializedNetworkProtocol.ToyoPuc => ((DeviceTcpNet)new ToyoPuc(options.Host, options.Port), "TOYO_PUC", "东洋 PUC"),
            HslSpecializedNetworkProtocol.TurckReader => ((DeviceTcpNet)new ReaderNet(options.Host, options.Port), "TURCK_READER", "Turck Reader"),
            HslSpecializedNetworkProtocol.DcsNanJingAuto => ((DeviceTcpNet)CreateDcsNanJingAuto(options), "DCS_NANJING_AUTO", "南京自动化 DCS"),
            HslSpecializedNetworkProtocol.EstunRobot => ((DeviceTcpNet)CreateEstunRobot(options), "ESTUN_ROBOT", "埃斯顿机器人"),
            HslSpecializedNetworkProtocol.FanucRobot => ((DeviceTcpNet)CreateFanucRobot(options), "FANUC_ROBOT", "FANUC 机器人"),
            _ => throw new ArgumentOutOfRangeException(nameof(options)),
        };
        _device.ConnectTimeOut = options.ConnectTimeoutMilliseconds;
        _device.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();
            var result = await _device.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = result.IsSuccess;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CONNECT", result, true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure($"HSL_{_errorPrefix}_CONNECT_FAILED", exception.Message, true); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); }
            var result = await _device.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = false;
            return result.IsSuccess ? HslOperationResult.Success() : Failure("CLOSE", result, false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { _connected = false; return HslOperationResult.Failure($"HSL_{_errorPrefix}_CLOSE_FAILED", exception.Message, false); }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslSpecializedNetworkReadRequest request, CancellationToken cancellationToken)
    {
        if (request.ElementCount < 1) return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", false);
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", $"{_displayName}连接尚未建立", true);
            return await HslTypedDeviceReader.ReadAsync(_device, request.Address, request.DataType, request.ElementCount, $"HSL_{_errorPrefix}_READ_FAILED", _displayName, cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure($"HSL_{_errorPrefix}_READ_FAILED", exception.Message, true); }
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

    private static OrientalMotorEipNet CreateOrientalMotor(HslSpecializedNetworkClientOptions options) => new(options.Host, options.Port)
    {
        RunIdleHeader = options.RunIdleHeader,
        RPITime = options.RpiTimeMilliseconds,
        ActualTimeout = options.ActualTimeoutSeconds,
    };

    private static DcsNanJingAuto CreateDcsNanJingAuto(HslSpecializedNetworkClientOptions options) => new(options.Host, options.Port, options.Station)
    {
        AddressStartWithZero = options.AddressStartWithZero,
        DataFormat = options.DataFormat switch
        {
            HslDataFormat.ABCD => DataFormat.ABCD,
            HslDataFormat.BADC => DataFormat.BADC,
            HslDataFormat.CDAB => DataFormat.CDAB,
            HslDataFormat.DCBA => DataFormat.DCBA,
            _ => throw new ArgumentOutOfRangeException(nameof(options)),
        },
        IsStringReverse = options.IsStringReverse,
    };

    private static EstunTcpNet CreateEstunRobot(HslSpecializedNetworkClientOptions options) => new(options.Host, options.Port, options.Station)
    {
        Station = options.Station,
        DataFormat = MapDataFormat(options.DataFormat),
    };

    private static FanucInterfaceNet CreateFanucRobot(HslSpecializedNetworkClientOptions options) => new(options.Host, options.Port)
    {
        FanucDataRetainTime = options.DataRetainTimeMilliseconds,
    };

    private static DataFormat MapDataFormat(HslDataFormat value) => value switch
    {
        HslDataFormat.ABCD => DataFormat.ABCD,
        HslDataFormat.BADC => DataFormat.BADC,
        HslDataFormat.CDAB => DataFormat.CDAB,
        HslDataFormat.DCBA => DataFormat.DCBA,
        _ => throw new ArgumentOutOfRangeException(nameof(value)),
    };

    private HslOperationResult Failure(string operation, OperateResult result, bool retryable) => HslOperationResult.Failure(
        $"HSL_{_errorPrefix}_{operation}_FAILED",
        string.IsNullOrWhiteSpace(result.Message) ? $"{_displayName}操作失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）",
        retryable);

    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
