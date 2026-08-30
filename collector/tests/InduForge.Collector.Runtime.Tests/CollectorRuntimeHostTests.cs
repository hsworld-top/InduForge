using InduForge.Collector.Contracts;
using InduForge.Collector.DriverHosting;

namespace InduForge.Collector.Runtime.Tests;

public sealed class CollectorRuntimeHostTests
{
    [Fact]
    public void DefaultHostOnlyAllowListsOpcUaAndModbusTcp()
    {
        var host = CollectorRuntimeHost.CreateDefault();

        Assert.Equal(
            ["modbus.tcp", "opcua.standard"],
            host.Drivers.Select(driver => driver.DriverId).Order(StringComparer.Ordinal));
    }

    [Fact]
    public void PreflightRejectsUnknownDriver()
    {
        var host = CollectorRuntimeHost.CreateDefault();

        Assert.Throws<CollectorRuntimeConfigurationException>(() => host.Preflight(new CollectorRuntimeConfiguration(
        [
            new RuntimeDriverConfiguration("siemens.s7-tcp", "siemens", "1.0.0", 1, [DriverOperations.PointRead]),
        ])));
    }

    [Theory]
    [InlineData("linux-x64")]
    [InlineData("linux-arm64")]
    public void PreflightAcceptsManifestAuthorizedRuntimeRid(string runtimeIdentifier)
    {
        var host = CollectorRuntimeHost.CreateForTest(new DriverHostPlatform("runtime", runtimeIdentifier));

        host.Preflight(new CollectorRuntimeConfiguration(
        [
            new RuntimeDriverConfiguration("modbus.tcp", "modbus", "1.0.0", 1, [DriverOperations.PointRead]),
            new RuntimeDriverConfiguration("opcua.standard", "opcua", "1.0.0", 1, [DriverOperations.PointRead]),
        ]));
    }

    [Fact]
    public void PreflightRejectsRuntimeRidOutsideManifestAllowList()
    {
        var host = CollectorRuntimeHost.CreateForTest(new DriverHostPlatform("runtime", "osx-arm64"));

        Assert.Throws<CollectorRuntimeConfigurationException>(() => host.Preflight(new CollectorRuntimeConfiguration(
        [
            new RuntimeDriverConfiguration("modbus.tcp", "modbus", "1.0.0", 1, [DriverOperations.PointRead]),
        ])));
    }
}
