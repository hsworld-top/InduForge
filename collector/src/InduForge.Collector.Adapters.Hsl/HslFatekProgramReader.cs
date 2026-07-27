using System.Text;
using HslCommunication;
using HslCommunication.Core.Device;

namespace InduForge.Collector.Adapters.Hsl;

internal static class HslFatekProgramReader
{
    // TCP 与串口只负责连接生命周期，读取统一走这里，保证两种传输的数据类型与错误语义一致。
    public static async Task<HslReadResult> ReadAsync(
        DeviceCommunication client,
        HslFatekProgramReadRequest request,
        string errorCode,
        string displayName,
        CancellationToken cancellationToken)
    {
        var count = checked((ushort)request.ElementCount);
        return request.DataType switch
        {
            HslValueType.Boolean => request.ElementCount == 1
                ? Map(await client.ReadBoolAsync(request.Address).WaitAsync(cancellationToken))
                : Map(await client.ReadBoolAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed8 => await ReadRawAsync(bytes => request.ElementCount == 1
                ? unchecked((sbyte)bytes[0])
                : bytes.Select(value => unchecked((sbyte)value)).ToArray()),
            HslValueType.Unsigned8 => await ReadRawAsync(bytes => request.ElementCount == 1 ? bytes[0] : bytes),
            HslValueType.Signed16 => request.ElementCount == 1
                ? Map(await client.ReadInt16Async(request.Address).WaitAsync(cancellationToken))
                : Map(await client.ReadInt16Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned16 => request.ElementCount == 1
                ? Map(await client.ReadUInt16Async(request.Address).WaitAsync(cancellationToken))
                : Map(await client.ReadUInt16Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed32 => request.ElementCount == 1
                ? Map(await client.ReadInt32Async(request.Address).WaitAsync(cancellationToken))
                : Map(await client.ReadInt32Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned32 => request.ElementCount == 1
                ? Map(await client.ReadUInt32Async(request.Address).WaitAsync(cancellationToken))
                : Map(await client.ReadUInt32Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed64 => request.ElementCount == 1
                ? Map(await client.ReadInt64Async(request.Address).WaitAsync(cancellationToken))
                : Map(await client.ReadInt64Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned64 => request.ElementCount == 1
                ? Map(await client.ReadUInt64Async(request.Address).WaitAsync(cancellationToken))
                : Map(await client.ReadUInt64Async(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.SinglePrecision => request.ElementCount == 1
                ? Map(await client.ReadFloatAsync(request.Address).WaitAsync(cancellationToken))
                : Map(await client.ReadFloatAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.DoublePrecision => request.ElementCount == 1
                ? Map(await client.ReadDoubleAsync(request.Address).WaitAsync(cancellationToken))
                : Map(await client.ReadDoubleAsync(request.Address, count).WaitAsync(cancellationToken)),
            HslValueType.Text => Map(await client.ReadStringAsync(request.Address, count, Encoding.UTF8).WaitAsync(cancellationToken)),
            HslValueType.Binary => await ReadRawAsync(bytes => bytes),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", $"{displayName}数据类型不受支持", false),
        };

        async Task<HslReadResult> ReadRawAsync(Func<byte[], object> convert)
        {
            var wordCount = checked((ushort)Math.Ceiling(request.ElementCount / 2d));
            var result = await client.ReadAsync(request.Address, wordCount).WaitAsync(cancellationToken).ConfigureAwait(false);
            return result.IsSuccess
                ? HslReadResult.Success(convert(result.Content.Take(request.ElementCount).ToArray()))
                : HslReadResult.Failure(errorCode, FormatError(result), true);
        }

        HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess
            ? HslReadResult.Success(result.Content)
            : HslReadResult.Failure(errorCode, FormatError(result), true);

        string FormatError(OperateResult result) => string.IsNullOrWhiteSpace(result.Message)
            ? $"{displayName}读取失败，错误码 {result.ErrorCode}"
            : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    }
}
