using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;
using InduForge.Collector.Drivers.SiemensS7Tcp;

namespace InduForge.Collector.Drivers.SiemensS7Tcp.Tests;

public sealed class SiemensS7TcpAddressTests
{
    [Theory]
    [InlineData("input", null, 10, 2, "bool", "I10.2", HslValueType.Boolean)]
    [InlineData("output", null, 11, null, "int16", "Q11", HslValueType.Signed16)]
    [InlineData("marker", null, 12, null, "float32", "M12", HslValueType.SinglePrecision)]
    [InlineData("dataBlock", 3, 14, 7, "bool", "DB3.14.7", HslValueType.Boolean)]
    [InlineData("dataBlock", 3, 16, null, "float64", "DB3.16", HslValueType.DoublePrecision)]
    [InlineData("timer", null, 20, null, "uint16", "T20", HslValueType.Unsigned16)]
    [InlineData("counter", null, 21, null, "int16", "C21", HslValueType.Signed16)]
    public void ParsesStructuredAddress(
        string area,
        int? dbNumber,
        int byteOffset,
        int? bitOffset,
        string dataType,
        string expectedAddress,
        HslValueType expectedType)
    {
        var address = SiemensS7TcpAddress.Parse(CreatePoint(area, dbNumber, byteOffset, bitOffset, dataType));

        Assert.Equal(expectedAddress, address.HslAddress);
        Assert.Equal(expectedType, address.ValueType);
    }

    [Theory]
    [InlineData("dataBlock", null, 10, null, "int16", "S7_DB_NUMBER_REQUIRED")]
    [InlineData("marker", 1, 10, null, "int16", "S7_DB_NUMBER_NOT_ALLOWED")]
    [InlineData("marker", null, 10, null, "bool", "S7_BIT_OFFSET_REQUIRED")]
    [InlineData("marker", null, 10, 1, "int16", "S7_BIT_OFFSET_NOT_ALLOWED")]
    [InlineData("timer", null, 10, null, "float32", "S7_TIMER_COUNTER_TYPE_INVALID")]
    public void RejectsInvalidAddressCombinations(
        string area,
        int? dbNumber,
        int byteOffset,
        int? bitOffset,
        string dataType,
        string expectedCode)
    {
        var exception = Assert.Throws<SiemensS7TcpDriverException>(() =>
            SiemensS7TcpAddress.Parse(CreatePoint(area, dbNumber, byteOffset, bitOffset, dataType)));

        Assert.Equal(expectedCode, exception.Code);
    }

    private static PointReadRequest CreatePoint(
        string area,
        int? dbNumber,
        int byteOffset,
        int? bitOffset,
        string dataType)
    {
        var address = new Dictionary<string, object?>
        {
            ["area"] = area,
            ["byteOffset"] = byteOffset,
        };
        if (dbNumber is not null) address["dbNumber"] = dbNumber;
        if (bitOffset is not null) address["bitOffset"] = bitOffset;
        return new PointReadRequest(
            "point",
            JsonSerializer.SerializeToElement(address),
            dataType,
            1,
            JsonSerializer.SerializeToElement(new { }));
    }
}
