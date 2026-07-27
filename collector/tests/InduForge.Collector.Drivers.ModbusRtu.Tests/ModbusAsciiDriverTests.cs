using System.Text.Json;
using InduForge.Collector.Adapters.Hsl;
using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusRtu.Tests;

public sealed class ModbusAsciiDriverTests
{
    [Fact]
    public async Task PassesAsciiProtocolAndDemoDefaults()
    {
        var factory = new CapturingFactory();
        var driver = new ModbusAsciiDriver(factory);

        await using var session = await driver.OpenSessionAsync(
            new ConnectionProfile(
                "modbus",
                JsonSerializer.SerializeToElement(new { portName = "COM9" }),
                JsonSerializer.SerializeToElement(new { })),
            CancellationToken.None);

        Assert.Equal("modbus.ascii", driver.Descriptor.DriverId);
        Assert.NotNull(factory.Options);
        Assert.Equal(HslModbusSerialProtocol.Ascii, factory.Options.Protocol);
        Assert.Equal(9600, factory.Options.BaudRate);
        Assert.Equal(8, factory.Options.DataBits);
        Assert.Equal(10000, factory.Options.TimeoutMilliseconds);
        Assert.Equal(HslDataFormat.CDAB, factory.Options.DataFormat);
    }

    private sealed class CapturingFactory : IHslModbusRtuClientFactory
    {
        public HslModbusRtuClientOptions? Options { get; private set; }

        public IHslModbusRtuClient Create(HslModbusRtuClientOptions options)
        {
            Options = options;
            return new FakeClient();
        }
    }

    private sealed class FakeClient : IHslModbusRtuClient
    {
        public bool IsConnected { get; private set; }
        public Task<HslOperationResult> ConnectAsync(CancellationToken cancellationToken) { IsConnected = true; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslOperationResult> CloseAsync(CancellationToken cancellationToken) { IsConnected = false; return Task.FromResult(HslOperationResult.Success()); }
        public Task<HslReadResult> ReadAsync(HslReadRequest request, CancellationToken cancellationToken) => Task.FromResult(HslReadResult.Success((short)1));
        public async ValueTask DisposeAsync() { if (IsConnected) await CloseAsync(CancellationToken.None); }
    }
}
