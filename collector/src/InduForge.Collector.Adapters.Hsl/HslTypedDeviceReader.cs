using System.Text;
using HslCommunication;
using HslCommunication.Core.Device;

namespace InduForge.Collector.Adapters.Hsl;

internal static class HslTypedDeviceReader
{
    public static async Task<HslReadResult> ReadAsync(
        DeviceCommunication client,
        string address,
        HslValueType dataType,
        int elementCount,
        string errorCode,
        string displayName,
        CancellationToken cancellationToken)
    {
        var count = checked((ushort)elementCount);
        return dataType switch
        {
            HslValueType.Boolean => elementCount == 1 ? Map(await client.ReadBoolAsync(address).WaitAsync(cancellationToken)) : Map(await client.ReadBoolAsync(address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed8 => await ReadRawAsync(bytes => elementCount == 1 ? unchecked((sbyte)bytes[0]) : bytes.Select(value => unchecked((sbyte)value)).ToArray()),
            HslValueType.Unsigned8 => await ReadRawAsync(bytes => elementCount == 1 ? bytes[0] : bytes),
            HslValueType.Signed16 => elementCount == 1 ? Map(await client.ReadInt16Async(address).WaitAsync(cancellationToken)) : Map(await client.ReadInt16Async(address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned16 => elementCount == 1 ? Map(await client.ReadUInt16Async(address).WaitAsync(cancellationToken)) : Map(await client.ReadUInt16Async(address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed32 => elementCount == 1 ? Map(await client.ReadInt32Async(address).WaitAsync(cancellationToken)) : Map(await client.ReadInt32Async(address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned32 => elementCount == 1 ? Map(await client.ReadUInt32Async(address).WaitAsync(cancellationToken)) : Map(await client.ReadUInt32Async(address, count).WaitAsync(cancellationToken)),
            HslValueType.Signed64 => elementCount == 1 ? Map(await client.ReadInt64Async(address).WaitAsync(cancellationToken)) : Map(await client.ReadInt64Async(address, count).WaitAsync(cancellationToken)),
            HslValueType.Unsigned64 => elementCount == 1 ? Map(await client.ReadUInt64Async(address).WaitAsync(cancellationToken)) : Map(await client.ReadUInt64Async(address, count).WaitAsync(cancellationToken)),
            HslValueType.SinglePrecision => elementCount == 1 ? Map(await client.ReadFloatAsync(address).WaitAsync(cancellationToken)) : Map(await client.ReadFloatAsync(address, count).WaitAsync(cancellationToken)),
            HslValueType.DoublePrecision => elementCount == 1 ? Map(await client.ReadDoubleAsync(address).WaitAsync(cancellationToken)) : Map(await client.ReadDoubleAsync(address, count).WaitAsync(cancellationToken)),
            HslValueType.Text => Map(await client.ReadStringAsync(address, count, Encoding.UTF8).WaitAsync(cancellationToken)),
            HslValueType.Binary => await ReadRawAsync(bytes => bytes),
            _ => HslReadResult.Failure("HSL_DATA_TYPE_UNSUPPORTED", $"{displayName}数据类型不受支持", false),
        };

        async Task<HslReadResult> ReadRawAsync(Func<byte[], object> convert)
        {
            var result = await client.ReadAsync(address, count).WaitAsync(cancellationToken).ConfigureAwait(false);
            return result.IsSuccess ? HslReadResult.Success(convert(result.Content)) : HslReadResult.Failure(errorCode, Format(result), true);
        }

        HslReadResult Map<T>(OperateResult<T> result) => result.IsSuccess ? HslReadResult.Success(result.Content) : HslReadResult.Failure(errorCode, Format(result), true);
        string Format(OperateResult result) => string.IsNullOrWhiteSpace(result.Message) ? $"{displayName}读取失败，错误码 {result.ErrorCode}" : $"{result.Message}（HSL 错误码 {result.ErrorCode}）";
    }
}
