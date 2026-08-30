namespace InduForge.Collector.Runtime;

/// <summary>
/// 可靠上行的发布边界。实现只有收到 JetStream PubAck 后才可正常返回。
/// </summary>
public interface IJetStreamPublisher
{
    Task PublishAsync(WalDataRecord record, CancellationToken cancellationToken);
}
