using System.Text;
using HslCommunication;
using HslCommunication.Core.Device;
using HslCommunication.Profinet.Siemens;

namespace InduForge.Collector.Adapters.Hsl;

internal static class HslSiemensReader
{
    // Siemens 各协议共享相同的类型化读取入口，传输差异只留在连接适配器中处理。
    public static async Task<HslReadResult> ReadAsync(
        DeviceCommunication client,
        HslS7ReadRequest request,
        string errorCode,
        string displayName,
        CancellationToken cancellationToken)
    {
        var count = checked((ushort)request.ElementCount);
        return request.DataType switch
        {
            HslValueType.Boolean => request.ElementCount == 1
                ? Map(await client.ReadBoolAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await client.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed8 => await ReadRawAsync(value => request.ElementCount == 1
                ? unchecked((sbyte)value[0])
                : value.Select(item => unchecked((sbyte)item)).ToArray()).ConfigureAwait(false),
            HslValueType.Unsigned8 => await ReadRawAsync(value => request.ElementCount == 1 ? value[0] : value).ConfigureAwait(false),
            HslValueType.Signed16 => request.ElementCount == 1
                ? Map(await client.ReadInt16Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await client.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned16 => request.ElementCount == 1
                ? Map(await client.ReadUInt16Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await client.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed32 => request.ElementCount == 1
                ? Map(await client.ReadInt32Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await client.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned32 => request.ElementCount == 1
                ? Map(await client.ReadUInt32Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await client.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Signed64 => request.ElementCount == 1
                ? Map(await client.ReadInt64Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await client.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Unsigned64 => request.ElementCount == 1
                ? Map(await client.ReadUInt64Async(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await client.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.SinglePrecision => request.ElementCount == 1
                ? Map(await client.ReadFloatAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await client.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.DoublePrecision => request.ElementCount == 1
                ? Map(await client.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false))
                : Map(await client.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Text => Map(await client.ReadStringAsync(request.Address, count, Encoding.UTF8).WaitAsync(cancellationToken).ConfigureAwait(false)),
            HslValueType.Binary => await ReadRawAsync(value => value).ConfigureAwait(false),
            HslValueType.DateTime => await ReadDateTimeAsync().ConfigureAwait(false),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", $"{displayName}读取数据类型不受支持", false),
        };

        async Task<HslReadResult> ReadRawAsync(Func<byte[], object> convert)
        {
            var result = await client.ReadAsync(request.Address, count).WaitAsync(cancellationToken).ConfigureAwait(false);
            return result.IsSuccess
                ? HslReadResult.Success(convert(result.Content))
                : HslReadResult.Failure(errorCode, FormatError(result), true);
        }

        HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess
            ? HslReadResult.Success(result.Content)
            : HslReadResult.Failure(errorCode, FormatError(result), true);

        async Task<HslReadResult> ReadDateTimeAsync() => client switch
        {
            SiemensS7Net s7 => Map(await s7.ReadDateTimeAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)),
            SiemensWebApi webApi => Map(await webApi.ReadDateTimeAsync(request.Address).WaitAsync(cancellationToken).ConfigureAwait(false)),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", $"{displayName}不支持 datetime 数据类型", false),
        };

        string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message)
            ? $"{displayName}读取失败，错误码 {result.ErrorCode}"
            : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    }
}
