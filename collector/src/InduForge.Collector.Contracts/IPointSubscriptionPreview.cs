namespace InduForge.Collector.Contracts;

public interface IPointSubscriptionPreview
{
    Task<SubscriptionPreviewResult> PreviewSubscriptionAsync(ConnectionProfile profile, SubscriptionPreviewRequest request, CancellationToken cancellationToken);
}
