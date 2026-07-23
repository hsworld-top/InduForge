using System.Text.Json;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusTcp.Tests;

public sealed class ModbusTcpDriverIntegrationTests
{
    [Fact]
    public async Task ConnectsAndReadsAllStandardReadAreasThroughHsl()
    {
        await using var server = new ModbusTcpTestServer();
        server.Coils[10] = true;
        server.DiscreteInputs[20] = false;
        server.HoldingRegisters[100] = 1234;
        server.InputRegisters[200] = 5678;
        var driver = new ModbusTcpDriver();
        var profile = CreateProfile(server.Port);

        var connection = await driver.TestConnectionAsync(profile, CancellationToken.None);
        Assert.True(connection.Connected);

        await using var session = await driver.OpenSessionAsync(profile, CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);
        var result = await reader.ReadAsync(
            new ReadRequest([
                CreatePoint("coil", "coil", 10, "bool"),
                CreatePoint("discrete", "discreteInput", 20, "bool"),
                CreatePoint("holding", "holdingRegister", 100, "int16"),
                CreatePoint("input", "inputRegister", 200, "uint16"),
            ]),
            CancellationToken.None);

        Assert.Equal(4, result.Values.Count);
        Assert.True(Assert.IsType<bool>(result.Values[0].Value));
        Assert.False(Assert.IsType<bool>(result.Values[1].Value));
        Assert.Equal(1234, Assert.IsType<short>(result.Values[2].Value));
        Assert.Equal(5678, Assert.IsType<ushort>(result.Values[3].Value));
        Assert.All(result.Values, value => Assert.True(value.Succeeded));
    }

    private static ConnectionProfile CreateProfile(int port) => new(
        "modbus",
        JsonSerializer.SerializeToElement(new
        {
            host = "127.0.0.1",
            port,
            connectTimeoutMs = 3000,
            dataFormat = "ABCD",
        }),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest CreatePoint(
        string key,
        string area,
        int address,
        string dataType) => new(
        key,
        JsonSerializer.SerializeToElement(new { station = 1, area, address }),
        dataType,
        1,
        JsonSerializer.SerializeToElement(new { }));
}
