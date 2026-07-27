using System.IO.Ports;
using HslCommunication;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.Siemens;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslSiemensVariantClient : IHslSiemensVariantClient
{
    private readonly HslSiemensVariantClientOptions _options;
    private readonly DeviceCommunication _device;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslSiemensVariantClient(HslSiemensVariantClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _options = options;
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

            var result = await OpenAsync(cancellationToken).ConfigureAwait(false);
            if (result.IsSuccess && _options.Protocol == HslSiemensVariantProtocol.MpiSerial && _options.HandshakeCheck)
            {
                // Demo 默认在 MPI 打开串口后执行握手；握手失败时立即关闭串口，避免保留伪连接。
                result = await Task.Run(((SiemensMPI)_device).Handle, cancellationToken).ConfigureAwait(false);
                if (!result.IsSuccess) ((DeviceSerialPort)_device).Close();
            }

            _connected = result.IsSuccess;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure(ErrorCode("CONNECT"), FormatError(result), true);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure(ErrorCode("CONNECT"), exception.Message, true);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed || !_connected) { _connected = false; return HslOperationResult.Success(); }

            var result = await CloseCoreAsync(cancellationToken).ConfigureAwait(false);
            _connected = false;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure(ErrorCode("CLOSE"), FormatError(result), false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure(ErrorCode("CLOSE"), exception.Message, false);
        }
        finally { _operationGate.Release(); }
    }

    public async Task<HslReadResult> ReadAsync(HslS7ReadRequest request, CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(request);
        if (request.ElementCount < 1)
        {
            return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", false);
        }

        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected) return HslReadResult.Failure("HSL_NOT_CONNECTED", $"{DisplayName}连接尚未建立", true);
            return await HslSiemensReader.ReadAsync(
                _device,
                request,
                ErrorCode("READ"),
                DisplayName,
                cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception exception) { return HslReadResult.Failure(ErrorCode("READ"), exception.Message, true); }
        finally { _operationGate.Release(); }
    }

    public async ValueTask DisposeAsync()
    {
        if (_disposed) return;
        await CloseAsync(CancellationToken.None).ConfigureAwait(false);
        _disposed = true;
        (_device as IDisposable)?.Dispose();
        _operationGate.Dispose();
    }

    private async Task<OperateResult> OpenAsync(CancellationToken cancellationToken) => _device switch
    {
        SiemensWebApi webApi => await webApi.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false),
        DeviceTcpNet tcp => await tcp.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false),
        DeviceSerialPort serial => await Task.Run(serial.Open, cancellationToken).ConfigureAwait(false),
        _ => throw new InvalidOperationException("未知的 Siemens 通信设备类型"),
    };

    private async Task<OperateResult> CloseCoreAsync(CancellationToken cancellationToken)
    {
        if (_device is SiemensWebApi webApi)
        {
            return await webApi.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
        }
        if (_device is DeviceTcpNet tcp)
        {
            return await tcp.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
        }
        if (_device is DeviceSerialPort serial)
        {
            await Task.Run(serial.Close, cancellationToken).ConfigureAwait(false);
            return OperateResult.CreateSuccessResult();
        }
        throw new InvalidOperationException("未知的 Siemens 通信设备类型");
    }

    private static DeviceCommunication CreateDevice(HslSiemensVariantClientOptions options) => options.Protocol switch
    {
        HslSiemensVariantProtocol.PpiSerial => ConfigureSerial(
            new SiemensPPI { Station = options.Station, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds }, options),
        HslSiemensVariantProtocol.PpiOverTcp => new SiemensPPIOverTcp(options.Host, options.Port)
        {
            Station = options.Station,
            ConnectTimeOut = options.ConnectTimeoutMilliseconds,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
        },
        HslSiemensVariantProtocol.MpiSerial => ConfigureSerial(
            new SiemensMPI { Station = options.Station, ReceiveTimeOut = options.ReceiveTimeoutMilliseconds }, options),
        HslSiemensVariantProtocol.FetchWrite => new SiemensFetchWriteNet(options.Host, options.Port)
        {
            ConnectTimeOut = options.ConnectTimeoutMilliseconds,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
        },
        HslSiemensVariantProtocol.WebApi => new SiemensWebApi(options.Host, options.Port)
        {
            UserName = options.UserName ?? string.Empty,
            Password = options.Password ?? string.Empty,
            UseHttps = options.UseHttps,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
        },
        HslSiemensVariantProtocol.S7Plus => new SiemensS7Plus(options.Host, options.Port)
        {
            ConnectTimeOut = options.ConnectTimeoutMilliseconds,
            ReceiveTimeOut = options.ReceiveTimeoutMilliseconds,
        },
        _ => throw new ArgumentOutOfRangeException(nameof(options)),
    };

    private static T ConfigureSerial<T>(T device, HslSiemensVariantClientOptions options) where T : DeviceSerialPort
    {
        if (string.IsNullOrWhiteSpace(options.PortName))
        {
            throw new ArgumentException("Siemens 串口名称不能为空", nameof(options));
        }
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

    private string DisplayName => _options.Protocol switch
    {
        HslSiemensVariantProtocol.PpiSerial => "Siemens PPI 串口",
        HslSiemensVariantProtocol.PpiOverTcp => "Siemens PPI TCP 透传",
        HslSiemensVariantProtocol.MpiSerial => "Siemens MPI 串口",
        HslSiemensVariantProtocol.FetchWrite => "Siemens Fetch/Write",
        HslSiemensVariantProtocol.WebApi => "Siemens Web API",
        HslSiemensVariantProtocol.S7Plus => "Siemens S7 Plus",
        _ => "Siemens",
    };

    private string ErrorCode(string operation) => $"HSL_SIEMENS_{_options.Protocol.ToString().ToUpperInvariant()}_{operation}_FAILED";

    private string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message)
        ? $"{DisplayName}操作失败，错误码 {result.ErrorCode}"
        : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";

    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
