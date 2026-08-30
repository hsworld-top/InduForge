using NATS.Client.JetStream;
using NATS.Client.JetStream.Models;

namespace InduForge.Collector.Runtime;

/// <summary>
/// 官方 NATS.Net JetStream 适配器。禁止降级为 Core NATS 发布，因为需要服务端持久化确认。
/// </summary>
public sealed class NatsJetStreamPublisher(INatsJSContext jetStream) : IJetStreamPublisher
{
    private readonly INatsJSContext _jetStream = jetStream ?? throw new ArgumentNullException(nameof(jetStream));

    public async Task PublishAsync(WalDataRecord record, CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(record);
        try
        {
            var acknowledgment = await _jetStream.PublishAsync(
                subject: record.Subject,
                data: record.Payload.ToArray(),
                opts: new NatsJSPubOpts { MsgId = record.EventId },
                cancellationToken: cancellationToken).ConfigureAwait(false);
            // 相同 MsgId 的 duplicate PubAck 说明服务端已持久化首条消息；这是客户端在
            // 首次 PubAck 丢失后重放时所需的确认，而不是新的发布失败。
            if (acknowledgment.Duplicate) return;
            acknowledgment.EnsureSuccess();
        }
        catch (NatsJSDuplicateMessageException)
        {
            // 官方客户端的 EnsureSuccess 也会以该精确异常表示 duplicate PubAck。
        }
    }
}
