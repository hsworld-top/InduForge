using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusTcp.Tests;

public sealed class ModbusNetworkVariantDriverTests
{
    [Theory]
    [InlineData("modbus.rtu-over-tcp", HslModbusNetworkProtocol.RtuOverTcp)]
    [InlineData("modbus.ascii-over-tcp", HslModbusNetworkProtocol.AsciiOverTcp)]
    [InlineData("modbus.udp", HslModbusNetworkProtocol.Udp)]
    public async Task VariantPassesProtocolAndDemoDefaults(string driverId, HslModbusNetworkProtocol protocol)
    {
        var factory = new CapturingFactory();
        IConnectionSessionDriver driver = protocol switch
        {
            HslModbusNetworkProtocol.RtuOverTcp => new ModbusRtuOverTcpDriver(factory),
            HslModbusNetworkProtocol.AsciiOverTcp => new ModbusAsciiOverTcpDriver(factory),
            HslModbusNetworkProtocol.Udp => new ModbusUdpDriver(factory),
            _ => throw new ArgumentOutOfRangeException(nameof(protocol)),
        };

        await using var session = await driver.OpenSessionAsync(
            new ConnectionProfile("modbus", JsonSerializer.SerializeToElement(new { host = "127.0.0.1" }), JsonSerializer.SerializeToElement(new { })),
            CancellationToken.None);

        Assert.Equal(driverId, ((IIndustrialDriver)driver).Descriptor.DriverId);
        Assert.NotNull(factory.Options);
        Assert.Equal(protocol, factory.Options.Protocol);
        Assert.Equal(502, factory.Options.Port);
        Assert.Equal(5000, factory.Options.TimeoutMilliseconds);
        Assert.Equal(10000, factory.Options.ReceiveTimeoutMilliseconds);
        Assert.Equal(HslDataFormat.CDAB, factory.Options.DataFormat);
    }

    private sealed class CapturingFactory : IHslModbusTcpClientFactory
    {
        public HslModbusTcpClientOptions? Options { get; private set; }

        public IHslModbusTcpClient Create(HslModbusTcpClientOptions options)
        {
            Options = options;
            return new FakeClient();
        }
    }

    private sealed class FakeClient : IHslModbusTcpClient
    {
        public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslReadRequest request, CancellationToken cancellationToken) => Task.FromResult(HslReadResult.Success((short)1));
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
