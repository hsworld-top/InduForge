namespace InduForge.Collector.Runtime;

/// <summary>
/// 宿主停止时的 WAL 排空判定。ReplayPump 独立于采集停止令牌，只有确认没有待确认记录后才取消它。
/// </summary>
internal static class CollectorDrainCoordinator
{
    internal const string DrainTimeoutReasonCode = "COLLECTOR_WAL_DRAIN_TIMEOUT";
    internal const int DrainTimeoutExitCode = 4;

    /// <summary>先停止采集令牌，再等待独立 replay 令牌排空；无论结果都由此处统一结束 replay。</summary>
    internal static async Task<bool> StopAcquisitionAndDrainAsync(
        Func<CancellationToken, Task> stopAcquisition,
        DurableWal wal,
        CancellationTokenSource replayCancellation,
        TimeProvider? timeProvider,
        CancellationToken deadline)
    {
        ArgumentNullException.ThrowIfNull(stopAcquisition);
        ArgumentNullException.ThrowIfNull(wal);
        ArgumentNullException.ThrowIfNull(replayCancellation);
        try
        {
            await stopAcquisition(deadline).ConfigureAwait(false);
            return await WaitForDrainAsync(wal, deadline, timeProvider).ConfigureAwait(false);
        }
        catch (OperationCanceledException) when (deadline.IsCancellationRequested)
        {
            return false;
        }
        finally
        {
            replayCancellation.Cancel();
        }
    }

    internal static async Task<bool> WaitForDrainAsync(DurableWal wal, CancellationToken deadline, TimeProvider? timeProvider = null)
    {
        ArgumentNullException.ThrowIfNull(wal);
        var clock = timeProvider ?? TimeProvider.System;
        while (wal.GetSnapshot().BacklogRecords != 0)
        {
            if (deadline.IsCancellationRequested) return false;
            try
            {
                // 使用可注入时钟，使宿主的 deadline 分支无需依赖脆弱的长时间 sleep。
                await Task.Delay(TimeSpan.FromMilliseconds(25), clock, deadline).ConfigureAwait(false);
            }
            catch (OperationCanceledException) when (deadline.IsCancellationRequested)
            {
                return false;
            }
        }
        return true;
    }
}
