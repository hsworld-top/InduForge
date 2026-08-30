using System.Globalization;
using System.Text.Json;

namespace InduForge.Collector.Runtime;

/// <summary>已确认、范围已知的上行缺口原因；范围未知时调用方必须 fail-closed，不能伪造 DATA_GAP。</summary>
public enum ConfirmedLossReason
{
    WalCapacity,
    WalCorruption,
    ManualCleanup,
    Unrecoverable,
}

/// <summary>
/// 可安全公告的丢失事实。连接、生产者 fence、序列和时间范围均由实际采集状态提供；
/// 此类型刻意不接受预组装 JSON，避免业务方绕开正式 alarm.event.v1 契约。
/// </summary>
public sealed record ConfirmedLossFact(
    string DeploymentId,
    string AccountId,
    string CollectorId,
    string OwnerId,
    long Epoch,
    string ConnectionId,
    long FromSequence,
    long ToSequence,
    DateTimeOffset FromSourceTimestamp,
    DateTimeOffset ToSourceTimestamp,
    DateTimeOffset DetectedAt,
    ConfirmedLossReason Reason);

public sealed partial class DurableWal
{
    /// <summary>
    /// 将已确认且范围完整的丢失事实作为 DATA_GAP 落入同一 WAL。若容量拒绝，返回
    /// WAL_BACKPRESSURE 且不会声称已公告；未知范围或不可信介质状态必须由调用方停机隔离。
    /// </summary>
    public Task<WalAppendResult> AppendConfirmedLossAsync(ConfirmedLossFact fact, CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(fact);
        ValidateConfirmedLossFact(fact);
        return AppendDataAsync(sequence => CreateDataGapRequest(fact), cancellationToken);
    }

    private static WalAppendRequest CreateDataGapRequest(ConfirmedLossFact fact)
    {
        var detectedAt = FormatUtc(fact.DetectedAt);
        var reason = ToWireReason(fact.Reason);
        var eventId = CollectorV1EventValidator.ComputeDataGapEventId(
            fact.DeploymentId,
            fact.CollectorId,
            fact.ConnectionId,
            fact.OwnerId,
            fact.Epoch,
            fact.FromSequence,
            fact.ToSequence,
            detectedAt,
            reason);
        var payload = JsonSerializer.SerializeToUtf8Bytes(new
        {
            schemaVersion = "alarm.event.v1",
            subject = "alarm.event",
            eventId,
            deploymentId = fact.DeploymentId,
            accountId = fact.AccountId,
            kind = "data-gap",
            operation = "DATA_GAP",
            ownerId = fact.OwnerId,
            epoch = fact.Epoch,
            sourceTimestamp = FormatUtc(fact.FromSourceTimestamp),
            serverTimestamp = detectedAt,
            receivedAt = detectedAt,
            dataGap = new
            {
                collectorId = fact.CollectorId,
                connectionId = fact.ConnectionId,
                fromSequence = fact.FromSequence,
                toSequence = fact.ToSequence,
                fromSourceTimestamp = FormatUtc(fact.FromSourceTimestamp),
                toSourceTimestamp = FormatUtc(fact.ToSourceTimestamp),
                detectedAt,
                reason,
            },
        });
        return new WalAppendRequest("alarm.event", eventId, payload, fact.OwnerId, fact.Epoch);
    }

    private static void ValidateConfirmedLossFact(ConfirmedLossFact fact)
    {
        if (fact.FromSequence < 0 || fact.ToSequence < fact.FromSequence) throw new ArgumentOutOfRangeException(nameof(fact), "DATA_GAP 序列范围无效");
        if (fact.FromSourceTimestamp > fact.ToSourceTimestamp || fact.ToSourceTimestamp > fact.DetectedAt) throw new ArgumentOutOfRangeException(nameof(fact), "DATA_GAP 时间范围无效");
        // 最终字段、UUID、stableId、epoch 及摘要仍由与普通上行相同的正式校验器确认。
        _ = ToWireReason(fact.Reason);
    }

    private static string FormatUtc(DateTimeOffset value) => value.ToUniversalTime().ToString("yyyy-MM-dd'T'HH:mm:ss.FFFFFFF'Z'", CultureInfo.InvariantCulture);

    private static string ToWireReason(ConfirmedLossReason reason) => reason switch
    {
        ConfirmedLossReason.WalCapacity => "wal-capacity",
        ConfirmedLossReason.WalCorruption => "wal-corruption",
        ConfirmedLossReason.ManualCleanup => "manual-cleanup",
        ConfirmedLossReason.Unrecoverable => "unrecoverable",
        _ => throw new ArgumentOutOfRangeException(nameof(reason)),
    };
}
