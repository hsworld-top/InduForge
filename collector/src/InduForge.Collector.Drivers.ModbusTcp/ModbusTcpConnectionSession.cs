using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusTcp;

internal sealed class ModbusTcpConnectionSession : IIndustrialConnectionSession, IPointReaderSession
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IHslModbusTcpClient _client;

    public ModbusTcpConnectionSession(IHslModbusTcpClient client, string serverName)
    {
        _client = client;
        ServerName = serverName;
    }

    public bool IsConnected => _client.IsConnected;

    public string? ServerName { get; }

    public async Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
    {
        if (request.Points.Count == 0)
        {
            return new ReadResult([], NoDiagnostics);
        }

        var readAt = DateTimeOffset.UtcNow;
        var values = new PointReadValue?[request.Points.Count];
        var validPoints = new List<ModbusReadPoint>();
        for (var index = 0; index < request.Points.Count; index++)
        {
            var point = request.Points[index];
            try
            {
                validPoints.Add(new ModbusReadPoint(index, point, ModbusTcpAddress.Parse(point)));
            }
            catch (ModbusTcpDriverException exception)
            {
                values[index] = Failure(point, exception.Code, exception.Message);
            }
        }

        foreach (var segment in BuildSegments(validPoints))
        {
            await ReadSegmentAsync(segment, values, readAt, cancellationToken).ConfigureAwait(false);
        }

        return new ReadResult(values.Select(value => value!).ToArray(), NoDiagnostics);
    }

    public ValueTask DisposeAsync() => _client.DisposeAsync();

    private async Task ReadSegmentAsync(
        ModbusReadSegment segment,
        PointReadValue?[] values,
        DateTimeOffset readAt,
        CancellationToken cancellationToken)
    {
        var mergedRequest = segment.First.Address.ToHslRequest() with
        {
            ElementCount = segment.Points.Sum(point => point.Request.ElementCount),
        };
        var result = await _client.ReadAsync(mergedRequest, cancellationToken).ConfigureAwait(false);
        if (result.Succeeded && TrySplitSegmentResult(segment, result.Value, values, readAt))
        {
            return;
        }

        if (segment.Points.Count == 1)
        {
            var point = segment.First;
            values[point.Index] = Failure(
                point.Request,
                result.ErrorCode ?? "MODBUS_READ_FAILED",
                result.ErrorMessage ?? "Modbus TCP 变量读取失败");
            return;
        }

        // 合并读取失败后逐点降级，避免一个坏地址拖垮同批其他变量。
        foreach (var point in segment.Points)
        {
            var singleResult = await _client.ReadAsync(point.Address.ToHslRequest(), cancellationToken).ConfigureAwait(false);
            values[point.Index] = singleResult.Succeeded
                ? Success(point.Request, singleResult.Value, readAt)
                : Failure(
                    point.Request,
                    singleResult.ErrorCode ?? "MODBUS_READ_FAILED",
                    singleResult.ErrorMessage ?? "Modbus TCP 变量读取失败");
        }
    }

    private static bool TrySplitSegmentResult(
        ModbusReadSegment segment,
        object? mergedValue,
        PointReadValue?[] values,
        DateTimeOffset readAt)
    {
        if (segment.Points.Count == 1)
        {
            var point = segment.First;
            values[point.Index] = Success(point.Request, mergedValue, readAt);
            return true;
        }
        if (mergedValue is not Array array || array.Length < segment.Points.Sum(point => point.Request.ElementCount))
        {
            return false;
        }

        var offset = 0;
        foreach (var point in segment.Points)
        {
            object? value;
            if (point.Request.ElementCount == 1)
            {
                value = array.GetValue(offset);
            }
            else
            {
                var slice = Array.CreateInstance(array.GetType().GetElementType() ?? typeof(object), point.Request.ElementCount);
                Array.Copy(array, offset, slice, 0, point.Request.ElementCount);
                value = slice;
            }

            values[point.Index] = Success(point.Request, value, readAt);
            offset += point.Request.ElementCount;
        }
        return true;
    }

    private static ModbusReadSegment[] BuildSegments(IReadOnlyList<ModbusReadPoint> points)
    {
        var segments = new List<ModbusReadSegment>();
        foreach (var group in points.GroupBy(point => new
                 {
                     point.Address.Station,
                     point.Address.Area,
                     point.Address.ValueType,
                 }))
        {
            ModbusReadSegment? current = null;
            foreach (var point in group.OrderBy(GetStartUnit))
            {
                if (current is null || !CanAppend(current, point))
                {
                    current = new ModbusReadSegment([point]);
                    segments.Add(current);
                    continue;
                }
                current.Points.Add(point);
            }
        }
        return segments.OrderBy(segment => segment.First.Index).ToArray();
    }

    private static bool CanAppend(ModbusReadSegment segment, ModbusReadPoint next)
    {
        if (!CanMergeType(next.Address.ValueType)) return false;
        var previous = segment.Points[^1];
        return GetStartUnit(next) == GetStartUnit(previous) + GetSpanUnits(previous) &&
               FitsProtocolReadLimit(segment.First, next);
    }

    // 连续变量只在同一 Modbus 报文容量内合并，超过上限时自动开启下一读取段。
    private static bool FitsProtocolReadLimit(ModbusReadPoint first, ModbusReadPoint last)
    {
        var start = GetStartUnit(first);
        var end = GetStartUnit(last) + GetSpanUnits(last);
        if (first.Address.ValueType == HslValueType.Boolean &&
            first.Address.Area is ModbusTcpArea.InputRegister or ModbusTcpArea.HoldingRegister)
        {
            var firstRegister = start / 16;
            var endRegister = (end + 15) / 16;
            return endRegister - firstRegister <= 125;
        }
        return end - start <= (first.Address.Area is ModbusTcpArea.Coil or ModbusTcpArea.DiscreteInput ? 2000 : 125);
    }

    private static bool CanMergeType(HslValueType valueType) => valueType is not
        (HslValueType.Signed8 or HslValueType.Unsigned8 or HslValueType.Text or HslValueType.Binary);

    private static long GetStartUnit(ModbusReadPoint point)
    {
        if (point.Address.ValueType == HslValueType.Boolean &&
            point.Address.Area is ModbusTcpArea.InputRegister or ModbusTcpArea.HoldingRegister)
        {
            return (long)point.Address.StartAddress * 16 + point.Address.BitIndex!.Value;
        }
        return point.Address.StartAddress;
    }

    private static long GetSpanUnits(ModbusReadPoint point)
    {
        if (point.Address.ValueType == HslValueType.Boolean) return point.Request.ElementCount;
        return RegistersPerElement(point.Address.ValueType) * point.Request.ElementCount;
    }

    private static int RegistersPerElement(HslValueType valueType) => valueType switch
    {
        HslValueType.Signed16 or HslValueType.Unsigned16 => 1,
        HslValueType.Signed32 or HslValueType.Unsigned32 or HslValueType.SinglePrecision => 2,
        HslValueType.Signed64 or HslValueType.Unsigned64 or HslValueType.DoublePrecision => 4,
        _ => 1,
    };

    private static PointReadValue Success(PointReadRequest point, object? value, DateTimeOffset readAt) => new(
        point.Key,
        true,
        value,
        point.DataType,
        "Good",
        readAt,
        readAt,
        null,
        null);

    private static PointReadValue Failure(PointReadRequest point, string errorCode, string errorMessage) => new(
        point.Key,
        false,
        null,
        point.DataType,
        "Bad",
        null,
        null,
        errorCode,
        errorMessage);

    private sealed record ModbusReadPoint(int Index, PointReadRequest Request, ModbusTcpAddress Address);

    private sealed class ModbusReadSegment(List<ModbusReadPoint> points)
    {
        public List<ModbusReadPoint> Points { get; } = points;

        public ModbusReadPoint First => Points[0];
    }
}
