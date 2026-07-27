using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusRtu.Tests;

public sealed class ModbusRtuAddressTests
{
    [Theory]
    [InlineData("coil", "s=1;100", HslModbusReadArea.Coil)]
    [InlineData("discreteInput", "s=1;100", HslModbusReadArea.DiscreteInput)]
    [InlineData("holdingRegister", "s=1;100.3", HslModbusReadArea.Register)]
    [InlineData("inputRegister", "s=1;x=4;100.3", HslModbusReadArea.Register)]
    public void ParseBuildsHslAddressForBooleanAreas(string area, string expectedAddress, HslModbusReadArea expectedArea)
    {
        var includeBit = area.EndsWith("Register", StringComparison.Ordinal);
        var address = includeBit
            ? JsonSerializer.SerializeToElement(new { station = 1, area, address = 100, bitIndex = 3 })
            : JsonSerializer.SerializeToElement(new { station = 1, area, address = 100 });
        var point = CreatePoint(address, "bool");

        var result = ModbusRtuAddress.Parse(point);

        Assert.Equal(expectedAddress, result.HslAddress);
        Assert.Equal(expectedArea, result.ReadArea);
        Assert.Equal(HslValueType.Boolean, result.ValueType);
    }

    [Fact]
    public void ParseBuildsInputRegisterNumericAddress()
    {
        var result = ModbusRtuAddress.Parse(CreatePoint(
            JsonSerializer.SerializeToElement(new { station = 2, area = "inputRegister", address = 20 }),
            "float32"));

        Assert.Equal("s=2;x=4;20", result.HslAddress);
        Assert.Equal(HslValueType.SinglePrecision, result.ValueType);
    }

    [Theory]
    [InlineData("coil", "int16", false)]
    [InlineData("coil", "bool", true)]
    [InlineData("holdingRegister", "bool", false)]
    [InlineData("holdingRegister", "int16", true)]
    public void ParseRejectsInvalidAreaAndBitCombinations(string area, string dataType, bool includeBit)
    {
        var address = includeBit
            ? JsonSerializer.SerializeToElement(new { station = 1, area, address = 100, bitIndex = 2 })
            : JsonSerializer.SerializeToElement(new { station = 1, area, address = 100 });

        Assert.Throws<ModbusRtuDriverException>(() => ModbusRtuAddress.Parse(CreatePoint(address, dataType)));
    }

    [Theory]
    [InlineData(0, 100)]
    [InlineData(248, 100)]
    [InlineData(1, -1)]
    [InlineData(1, 65536)]
    public void ParseRejectsStationAndAddressOutsideProtocolRange(int station, int address)
    {
        var point = CreatePoint(
            JsonSerializer.SerializeToElement(new { station, area = "holdingRegister", address }),
            "int16");

        Assert.Throws<ModbusRtuDriverException>(() => ModbusRtuAddress.Parse(point));
    }

    [Theory]
    [InlineData("coil", "bool", 2001, null)]
    [InlineData("holdingRegister", "int16", 126, null)]
    [InlineData("holdingRegister", "int32", 63, null)]
    [InlineData("holdingRegister", "bool", 1986, 15)]
    public void ParseRejectsSinglePointReadsAboveProtocolLimit(
        string area,
        string dataType,
        int elementCount,
        int? bitIndex)
    {
        var address = new Dictionary<string, object?>
        {
            ["station"] = 1,
            ["area"] = area,
            ["address"] = 100,
        };
        if (bitIndex is not null) address["bitIndex"] = bitIndex;

        var exception = Assert.Throws<ModbusRtuDriverException>(() =>
            ModbusRtuAddress.Parse(CreatePoint(JsonSerializer.SerializeToElement(address), dataType, elementCount)));

        Assert.Equal("MODBUS_ELEMENT_COUNT_TOO_LARGE", exception.Code);
    }

    private static PointReadRequest CreatePoint(JsonElement address, string dataType, int elementCount = 1) => new(
        "point-1",
        address,
        dataType,
        elementCount,
        JsonSerializer.SerializeToElement(new { }));
}
