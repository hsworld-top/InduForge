using InduForge.Collector.DevAgent;

namespace InduForge.Collector.DevAgent.Tests;

public sealed class SingleInstanceCoordinatorTests
{
    [Fact]
    public async Task SecondaryInstanceNotifiesPrimaryInstance()
    {
        var instanceName = $"test-{Guid.NewGuid():N}";
        await using var primary = new SingleInstanceCoordinator(instanceName);
        await using var secondary = new SingleInstanceCoordinator(instanceName);
        var activated = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);

        Assert.True(primary.IsPrimaryInstance);
        Assert.False(secondary.IsPrimaryInstance);

        primary.StartListening(() =>
        {
            activated.TrySetResult();
            return Task.CompletedTask;
        });

        Assert.True(await secondary.NotifyExistingInstanceAsync());
        await activated.Task.WaitAsync(TimeSpan.FromSeconds(3));
    }
}
