using System.Text;
using HslCommunication;
using HslCommunication.Core;
using HslCommunication.Core.Device;
using HslCommunication.ModBus;

namespace InduForge.Collector.Adapters.Hsl;

public sealed class HslModbusTcpClient : IHslModbusTcpClient
{
    private readonly DeviceTcpNet _client;
    private readonly SemaphoreSlim _operationGate = new(1, 1);
    private bool _connected;
    private bool _disposed;

    public HslModbusTcpClient(HslModbusTcpClientOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _client = CreateClient(options);
    }

    public bool IsConnected => _connected && !_disposed;

    public async Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken)
    {
        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (_connected) return HslOperationResult.Success();

            var result = await _client.ConnectServerAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = result.IsSuccess;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_CONNECT_FAILED", FormatError(result), retryable: true);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_CONNECT_FAILED", exception.Message, retryable: true);
        }
        finally
        {
            _operationGate.Release();
        }
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

            var result = await _client.ConnectCloseAsync().WaitAsync(cancellationToken).ConfigureAwait(false);
            _connected = false;
            return result.IsSuccess
                ? HslOperationResult.Success()
                : HslOperationResult.Failure("HSL_CLOSE_FAILED", FormatError(result), retryable: false);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            _connected = false;
            return HslOperationResult.Failure("HSL_CLOSE_FAILED", exception.Message, retryable: false);
        }
        finally
        {
            _operationGate.Release();
        }
    }

    public async Task<HslReadResult> ReadAsync(HslReadRequest request, CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(request);
        if (request.ElementCount < 1)
        {
            return HslReadResult.Failure("HSL_ELEMENT_COUNT_INVALID", "读取元素数量必须大于 0", retryable: false);
        }

        await _operationGate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (!_connected)
            {
                return HslReadResult.Failure("HSL_NOT_CONNECTED", "HSL 通信连接尚未建立", retryable: true);
            }

            return await ReadCoreAsync(request, cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception)
        {
            return HslReadResult.Failure("HSL_READ_FAILED", exception.Message, retryable: true);
        }
        finally
        {
            _operationGate.Release();
        }
    }

    public async ValueTask DisposeAsync()
    {
        if (_disposed) return;
        await CloseAsync(CancellationToken.None).ConfigureAwait(false);
        _disposed = true;
        (_client as IDisposable)?.Dispose();
        _operationGate.Dispose();
    }

    private async Task<HslReadResult> ReadCoreAsync(HslReadRequest request, CancellationToken cancellationToken)
    {
        var count = checked((ushort)request.ElementCount);
        return request.DataType switch
        {
            HslValueType.Boolean => request.Area switch
            {
                _ => request.ElementCount == 1
                    ? Map(await _client.ReadBoolAsync(WithFunctionCode(request.Address, request.Area)).WaitAsync(cancellationToken).ConfigureAwait(false))
                    : Map(await _client.ReadBoolAsync(WithFunctionCode(request.Address, request.Area), count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            },
            HslValueType.Signed8 => await ReadRawAsync(
                request,
                value => request.ElementCount == 1 ? unchecked((sbyte)value[0]) : value.Select(item => unchecked((sbyte)item)).ToArray(),
                cancellationToken).ConfigureAwait(false),
            HslValueType.Unsigned8 => await ReadRawAsync(
                request,
                value => request.ElementCount == 1 ? value[0] : value,
                cancellationToken).ConfigureAwait(false),
            HslValueType.Signed16 => request.ElementCount == 1
                ? Map(await _client.ReadInt16Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned16 => request.ElementCount == 1
                ? Map(await _client.ReadUInt16Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed32 => request.ElementCount == 1
                ? Map(await _client.ReadInt32Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned32 => request.ElementCount == 1
                ? Map(await _client.ReadUInt32Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed64 => request.ElementCount == 1
                ? Map(await _client.ReadInt64Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned64 => request.ElementCount == 1
                ? Map(await _client.ReadUInt64Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.SinglePrecision => request.ElementCount == 1
                ? Map(await _client.ReadFloatAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.DoublePrecision => request.ElementCount == 1
                ? Map(await _client.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await _client.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Text => Map(await _client.ReadStringAsync(request.Address, count, Encoding.UTF8).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Binary => await ReadRawAsync(request, value => value, cancellationToken).ConfigureAwait(false),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", "HSL 读取数据类型不受支持", retryable: false),
        };
    }

    private async Task<HslReadResult> ReadRawAsync(
        HslReadRequest request,
        Func<byte[], object> convert,
        CancellationToken cancellationToken)
    {
        var registerCount = checked((ushort)Math.Ceiling(request.ElementCount / 2d));
        var result = await _client.ReadAsync(request.Address, registerCount).WaitAsync(cancellationToken).ConfigureAwait(false);
        if (!result.IsSuccess)
        {
            return HslReadResult.Failure("HSL_READ_FAILED", FormatError(result), retryable: true);
        }

        var bytes = result.Content.Take(request.ElementCount).ToArray();
        return HslReadResult.Success(convert(bytes));
    }

    private static HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess
        ? HslReadResult.Success(result.Content)
        : HslReadResult.Failure("HSL_READ_FAILED", FormatError(result), retryable: true);

    private static string FormatError(OperateResult result) =>
        string.IsNullOrWhiteSpace(result.Message)
            ? $"HSL 操作失败，错误码 {result.ErrorCode}"
            : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";

    private static DeviceTcpNet CreateClient(HslModbusTcpClientOptions options)
    {
        DeviceTcpNet client = options.Protocol switch
        {
            HslModbusNetworkProtocol.Tcp => new ModbusTcpNet(options.Host, options.Port, station: 1),
            HslModbusNetworkProtocol.RtuOverTcp => new ModbusRtuOverTcp(options.Host, options.Port, station: 1),
            HslModbusNetworkProtocol.AsciiOverTcp => new ModbusAsciiOverTcp(options.Host, options.Port, station: 1),
            HslModbusNetworkProtocol.Udp => new ModbusUdpNet(options.Host, options.Port, station: 1),
            _ => throw new ArgumentOutOfRangeException(nameof(options), "不支持的 Modbus 网络协议"),
        };
        client.ConnectTimeOut = options.TimeoutMilliseconds;
        client.ReceiveTimeOut = options.ReceiveTimeoutMilliseconds;

        var modbus = (IModbus)client;
        modbus.AddressStartWithZero = true;
        modbus.DataFormat = MapDataFormat(options.DataFormat);
        modbus.IsStringReverse = false;
        switch (client)
        {
            case ModbusRtuOverTcp rtuOverTcp:
                rtuOverTcp.StationCheckMatch = true;
                break;
            case ModbusTcpNet tcp:
                tcp.StationCheckMatch = true;
                break;
        }
        return client;
    }

    // 统一通过功能码前缀读取线圈和离散输入，使所有 Modbus 网络变体复用同一读取实现。
    private static string WithFunctionCode(string address, HslModbusReadArea area)
    {
        var functionCode = area switch
        {
            HslModbusReadArea.Coil => 1,
            HslModbusReadArea.DiscreteInput => 2,
            _ => 0,
        };
        if (functionCode == 0) return address;
        var separator = address.IndexOf(';');
        return separator < 0
            ? $"x={functionCode};{address}"
            : address.Insert(separator + 1, $"x={functionCode};");
    }

    private static DataFormat MapDataFormat(HslDataFormat dataFormat) => dataFormat switch
    {
        HslDataFormat.ABCD => DataFormat.ABCD,
        HslDataFormat.BADC => DataFormat.BADC,
        HslDataFormat.CDAB => DataFormat.CDAB,
        HslDataFormat.DCBA => DataFormat.DCBA,
        _ => throw new ArgumentOutOfRangeException(nameof(dataFormat)),
    };

    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);
}
