using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp;

internal sealed record SiemensVariantAddress(string Address, HslValueType DataType, int ElementCount)
{
    public static SiemensVariantAddress Parse(PointReadRequest point, HslSiemensVariantProtocol protocol)
    {
        if (protocol == HslSiemensVariantProtocol.WebApi)
        {
            return ParseWebApi(point);
        }
        if (protocol == HslSiemensVariantProtocol.S7Plus)
        {
            return ParseS7Plus(point);
        }

        SiemensS7TcpAddress parsed;
        try
        {
            parsed = SiemensS7TcpAddress.Parse(point);
        }
        catch (SiemensS7TcpDriverException exception)
        {
            throw new SiemensVariantDriverException(exception.Code, exception.Message, exception.Retryable, exception);
        }

        if (parsed.ValueType == HslValueType.DateTime)
        {
            throw Invalid("SIEMENS_DATA_TYPE_UNSUPPORTED", "当前 Siemens 协议不支持 datetime 数据类型");
        }
        if (protocol == HslSiemensVariantProtocol.FetchWrite && parsed.ValueType == HslValueType.Boolean)
        {
            throw Invalid("SIEMENS_FETCH_WRITE_BOOL_UNSUPPORTED", "Fetch/Write 不支持单个位地址读取，请按字节或数值类型读取后解析位值");
        }

        return new(parsed.HslAddress, parsed.ValueType, parsed.ElementCount);
    }

    public HslS7ReadRequest ToHslRequest() => new(Address, DataType, ElementCount);

    private static SiemensVariantAddress ParseS7Plus(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object || !point.Address.TryGetProperty("address", out var value) || value.ValueKind != JsonValueKind.String)
        {
            throw Invalid("SIEMENS_S7_PLUS_ADDRESS_REQUIRED", "Siemens S7 Plus 地址必须包含 address 字段");
        }
        var address = value.GetString()?.Trim() ?? string.Empty;
        if (address.Length is < 1 or > 256 || address.Any(char.IsWhiteSpace) || point.ElementCount is < 1 or > ushort.MaxValue)
        {
            throw Invalid("SIEMENS_S7_PLUS_ADDRESS_INVALID", "Siemens S7 Plus 变量地址无效");
        }
        var dataType = SiemensS7TcpAddress.ParseValueType(point.DataType);
        if (dataType == HslValueType.DateTime) throw Invalid("SIEMENS_DATA_TYPE_UNSUPPORTED", "Siemens S7 Plus 暂不支持 datetime 数据类型");
        return new(address, dataType, point.ElementCount);
    }

    private static SiemensVariantAddress ParseWebApi(PointReadRequest point)
    {
        if (point.Address.ValueKind != JsonValueKind.Object ||
            !point.Address.TryGetProperty("tag", out var value) ||
            value.ValueKind != JsonValueKind.String)
        {
            throw Invalid("SIEMENS_WEB_API_TAG_REQUIRED", "Siemens Web API 地址必须包含 tag 字段");
        }

        var tag = value.GetString()?.Trim() ?? string.Empty;
        if (tag.Length is < 1 or > 512)
        {
            throw Invalid("SIEMENS_WEB_API_TAG_INVALID", "Siemens Web API 标签名称无效");
        }
        if (point.ElementCount is < 1 or > ushort.MaxValue)
        {
            throw Invalid("SIEMENS_ELEMENT_COUNT_INVALID", "Siemens 元素数量必须在 1 到 65535 之间");
        }

        var dataType = SiemensS7TcpAddress.ParseValueType(point.DataType);
        if (dataType == HslValueType.DateTime && point.ElementCount != 1)
        {
            throw Invalid("SIEMENS_DATETIME_COUNT_INVALID", "Siemens datetime 变量只支持单元素读取");
        }
        return new(tag, dataType, point.ElementCount);
    }

    private static SiemensVariantDriverException Invalid(string code, string message) => new(code, message, false);
}
