using InduForge.Collector.Adapters.Hsl;

namespace InduForge.Collector.Adapters.Hsl.Tests;

public sealed class HslPublicContractTests
{
    [Fact]
    public void PublicApiDoesNotExposeHslCommunicationTypes()
    {
        var exposedNamespaces = typeof(HslReadResult).Assembly
            .GetExportedTypes()
            .SelectMany(type =>
                type.GetProperties().Select(property => property.PropertyType.Namespace)
                    .Concat(type.GetMethods().Select(method => method.ReturnType.Namespace))
                    .Concat(type.GetMethods().SelectMany(method => method.GetParameters()).Select(parameter => parameter.ParameterType.Namespace)))
            .Where(value => value is not null)
            .ToArray();

        Assert.DoesNotContain(
            exposedNamespaces,
            value => value!.StartsWith("HslCommunication", StringComparison.Ordinal));
    }

    [Fact]
    public void FailedReadKeepsStableErrorInformation()
    {
        var result = HslReadResult.Failure("HSL_READ_FAILED", "读取失败", retryable: true);

        Assert.False(result.Succeeded);
        Assert.Null(result.Value);
        Assert.Equal("HSL_READ_FAILED", result.ErrorCode);
        Assert.Equal("读取失败", result.ErrorMessage);
        Assert.True(result.Retryable);
    }

    [Fact]
    public async Task S7FactoryCreatesClientWithoutExposingHslTypes()
    {
        var factory = new HslS7TcpClientFactory();

        var client = factory.Create(new HslS7TcpClientOptions(
            "127.0.0.1",
            102,
            5000,
            HslSiemensPlc.S1200,
            0,
            1,
            null,
            null));

        Assert.IsAssignableFrom<IHslS7TcpClient>(client);
        await client.DisposeAsync();
    }

    [Fact]
    public async Task ModbusRtuFactoryCreatesClientWithoutOpeningSerialPort()
    {
        var factory = new HslModbusRtuClientFactory();

        var client = factory.Create(new HslModbusRtuClientOptions(
            "COM1",
            9600,
            8,
            HslSerialParity.None,
            HslSerialStopBits.One,
            3000,
            HslDataFormat.ABCD));

        Assert.IsAssignableFrom<IHslModbusRtuClient>(client);
        Assert.False(client.IsConnected);
        await client.DisposeAsync();
    }

    [Fact]
    public async Task MelsecMcFactoryCreatesClientWithoutOpeningNetworkConnection()
    {
        var client = new HslMelsecMcClientFactory().Create(new HslMelsecMcClientOptions(
            "127.0.0.1",
            6000,
            5000,
            0,
            0,
            1023));

        Assert.IsAssignableFrom<IHslMelsecMcClient>(client);
        Assert.False(client.IsConnected);
        await client.DisposeAsync();
    }

    [Fact]
    public async Task OmronFinsFactoryCreatesTcpAndUdpClientsWithoutOpeningConnection()
    {
        var factory = new HslOmronFinsClientFactory();
        var common = new HslOmronFinsClientOptions(
            HslOmronFinsTransport.Tcp,
            "127.0.0.1",
            9600,
            5000,
            5000,
            HslOmronPlcType.CSCJ,
            500,
            2,
            0,
            HslDataFormat.CDAB,
            true,
            false);

        await using var tcp = factory.Create(common);
        await using var udp = factory.Create(common with { Transport = HslOmronFinsTransport.Udp });

        Assert.False(tcp.IsConnected);
        Assert.False(udp.IsConnected);
    }

    [Fact]
    public async Task AllenBradleyFactoryCreatesClientWithoutOpeningConnection()
    {
        await using var client = new HslAllenBradleyClientFactory().Create(new HslAllenBradleyClientOptions(
            "127.0.0.1",
            44818,
            5000,
            5000,
            0,
            null,
            false,
            true,
            HslDataFormat.DCBA));

        Assert.False(client.IsConnected);
    }

    [Fact]
    public async Task BeckhoffAdsFactoryCreatesClientWithoutOpeningConnection()
    {
        await using var client = new HslBeckhoffAdsClientFactory().Create(new HslBeckhoffAdsClientOptions(
            "127.0.0.1", 48898, 5000, 5000, true, 851, null, null, true, HslDataFormat.DCBA));

        Assert.False(client.IsConnected);
    }

    [Fact]
    public async Task Iec104FactoryCreatesClientWithoutOpeningConnection()
    {
        await using var client = new HslIec104ClientFactory().Create(new HslIec104ClientOptions(
            "127.0.0.1", 2404, 5000, 10000, 1, 20, 6, 3000));

        Assert.False(client.IsConnected);
    }

    [Theory]
    [InlineData(1, HslIec104InformationType.SinglePoint)]
    [InlineData(30, HslIec104InformationType.SinglePoint)]
    [InlineData(13, HslIec104InformationType.ShortFloatMeasured)]
    [InlineData(36, HslIec104InformationType.ShortFloatMeasured)]
    [InlineData(15, HslIec104InformationType.IntegratedTotal)]
    public void Iec104MapsTimedAndUntimedTypeIds(byte typeId, HslIec104InformationType expected)
    {
        Assert.Equal(expected, HslIec104Client.ResolveInformationType(typeId));
    }

    [Fact]
    public async Task PanasonicMewtocolFactoryCreatesClientWithoutOpeningConnection()
    {
        await using var client = new HslPanasonicMewtocolClientFactory().Create(new HslPanasonicMewtocolClientOptions(
            "127.0.0.1", 5000, 5000, 5000, 0xEE, HslDataFormat.DCBA));

        Assert.False(client.IsConnected);
    }

    [Fact]
    public async Task LsisFastEnetFactoryCreatesClientWithoutOpeningConnection()
    {
        await using var client = new HslLsisFastEnetClientFactory().Create(new HslLsisFastEnetClientOptions(
            "127.0.0.1", 2004, 5000, 5000, HslLsisCpuType.XGK, "LSIS-XGT", 0, 3, HslDataFormat.DCBA));
        Assert.False(client.IsConnected);
    }

    [Fact]
    public async Task GeSrtpFactoryCreatesClientWithoutOpeningConnection()
    {
        await using var client = new HslGeSrtpClientFactory().Create(new HslGeSrtpClientOptions(
            "127.0.0.1", 18245, 5000, 10000, HslDataFormat.DCBA));
        Assert.False(client.IsConnected);
    }

    [Fact]
    public async Task InovanceModbusTcpFactoryCreatesClientWithoutOpeningConnection()
    {
        await using var client = new HslInovanceModbusTcpClientFactory().Create(new HslInovanceModbusTcpClientOptions(
            "127.0.0.1", 502, 5000, 10000, 1, HslInovanceSeries.AM, true, HslDataFormat.CDAB, false));
        Assert.False(client.IsConnected);
    }

    [Fact]
    public async Task FatekProgramTcpFactoryCreatesClientWithoutOpeningConnection()
    {
        await using var client = new HslFatekProgramTcpClientFactory().Create(new HslFatekProgramTcpClientOptions(
            "127.0.0.1", 2000, 5000, 10000, 1));
        Assert.False(client.IsConnected);
    }

    [Fact]
    public void BlankAuthorizationCodeKeepsTrialModeWithoutActivationRequest()
    {
        var result = HslAuthorizationInitializer.Initialize("  ");

        Assert.Equal(HslAuthorizationStatus.NotConfigured, result.Status);
        Assert.False(result.IsActivated);
    }
}
