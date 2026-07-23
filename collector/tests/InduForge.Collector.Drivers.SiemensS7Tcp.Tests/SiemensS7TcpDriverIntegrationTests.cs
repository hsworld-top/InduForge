using System.Net;
using System.Net.Sockets;
using System.Text.Json;
using HslCommunication.Profinet.Siemens;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp.Tests;

public sealed class SiemensS7TcpDriverIntegrationTests
{
    [Fact]
    public async Task ConnectsAndReadsStandardS7AreasThroughHsl()
    {
        using var server = new SiemensS7Server();
        var port = GetFreeTcpPort();
        server.ServerStart(port);
        Assert.True(server.Write("M0.0", true).IsSuccess);
        Assert.True(server.Write("DB1.2", (short)1234).IsSuccess);
        Assert.True(server.Write("DB1.4", 56.5f).IsSuccess);

        var driver = new SiemensS7TcpDriver();
        var profile = CreateProfile(port);
        var connection = await driver.TestConnectionAsync(profile, CancellationToken.None);
        Assert.True(connection.Connected);

        await using var session = await driver.OpenSessionAsync(profile, CancellationToken.None);
        var reader = Assert.IsAssignableFrom<IPointReaderSession>(session);
        var result = await reader.ReadAsync(
            new ReadRequest([
                CreatePoint("marker", new { area = "marker", byteOffset = 0, bitOffset = 0 }, "bool"),
                CreatePoint("temperature", new { area = "dataBlock", dbNumber = 1, byteOffset = 2 }, "int16"),
                CreatePoint("pressure", new { area = "dataBlock", dbNumber = 1, byteOffset = 4 }, "float32"),
            ]),
            CancellationToken.None);

        Assert.True(Assert.IsType<bool>(result.Values[0].Value));
        Assert.Equal((short)1234, Assert.IsType<short>(result.Values[1].Value));
        Assert.Equal(56.5f, Assert.IsType<float>(result.Values[2].Value));
        Assert.All(result.Values, value => Assert.True(value.Succeeded));
    }

    private static int GetFreeTcpPort()
    {
        var listener = new TcpListener(IPAddress.Loopback, 0);
        listener.Start();
        var port = ((IPEndPoint)listener.LocalEndpoint).Port;
        listener.Stop();
        return port;
    }

    private static ConnectionProfile CreateProfile(int port) => new(
        "siemens",
        JsonSerializer.SerializeToElement(new
        {
            host = "127.0.0.1",
            port,
            plcType = "S1200",
            rack = 0,
            slot = 1,
            connectTimeoutMs = 3000,
        }),
        JsonSerializer.SerializeToElement(new { }));

    private static PointReadRequest CreatePoint(string key, object address, string dataType) => new(
        key,
        JsonSerializer.SerializeToElement(address),
        dataType,
        1,
        JsonSerializer.SerializeToElement(new { }));
}
