using InduForge.Collector.DevAgent;

namespace InduForge.Collector.DevAgent.Tests;

public sealed class AgentLifecycleTests
{
    [Fact]
    public void MachineIdentityHashIsStableAndNormalized()
    {
        var upper = MachineIdentityProvider.HashMachineGuid(" 01234567-89AB-CDEF-0123-456789ABCDEF ");
        var lower = MachineIdentityProvider.HashMachineGuid("01234567-89ab-cdef-0123-456789abcdef");

        Assert.Equal(lower, upper);
        Assert.Equal(64, lower.Length);
    }

    [Theory]
    [InlineData(1, 5)]
    [InlineData(2, 15)]
    [InlineData(3, 30)]
    [InlineData(8, 30)]
    public void HeartbeatFailuresUseConfiguredBackoff(int failures, int expectedSeconds)
    {
        Assert.Equal(TimeSpan.FromSeconds(expectedSeconds), HeartbeatRetrySchedule.GetDelay(failures));
    }
}
