using System.Diagnostics;

namespace InduForge.Collector.DevAgent;

internal sealed class AgentWorker : IAsyncDisposable
{
    private static readonly TimeSpan HeartbeatInterval = TimeSpan.FromSeconds(15);
    private static readonly TimeSpan TaskPollingInterval = TimeSpan.FromSeconds(2);
    private readonly CenterApiClient _apiClient;
    private readonly CollectorTaskExecutor _taskExecutor;
    private readonly IReadOnlyList<AgentProtocolCapability> _capabilities;
    private readonly Func<AgentWorkerUpdate, Task> _onUpdate;
    private readonly AgentFileLogger? _logger;
    private CancellationTokenSource? _runCancellation;
    private Task? _runTask;

    public AgentWorker(CenterApiClient apiClient, CollectorTaskExecutor taskExecutor, IReadOnlyList<AgentProtocolCapability> capabilities, Func<AgentWorkerUpdate, Task> onUpdate, AgentFileLogger? logger = null)
    {
        _apiClient = apiClient;
        _taskExecutor = taskExecutor;
        _capabilities = capabilities;
        _onUpdate = onUpdate;
        _logger = logger;
    }

    public bool IsRunning => _runTask is { IsCompleted: false };

    public void Start(AgentCredentials credentials, CancellationToken applicationCancellation)
    {
        if (IsRunning) return;
        _logger?.Info("worker.start", $"agentId={credentials.AgentId} center={credentials.CenterUrl}");
        _runCancellation = CancellationTokenSource.CreateLinkedTokenSource(applicationCancellation);
        _runTask = RunAsync(credentials, _runCancellation.Token);
    }

    public async Task StopAsync()
    {
        if (_runCancellation is null) return;
        _runCancellation.Cancel();
        if (_runTask is not null)
        {
            try { await _runTask.ConfigureAwait(false); } catch (OperationCanceledException) { }
        }
        _runCancellation.Dispose();
        _runCancellation = null;
        _runTask = null;
        _logger?.Info("worker.stop", "中心连接工作线程已停止");
    }

    private async Task RunAsync(AgentCredentials credentials, CancellationToken cancellationToken)
    {
        var heartbeatAt = DateTimeOffset.MinValue;
        var heartbeatFailureCount = 0;
        var connected = false;
        DateTimeOffset? lastPollingFailureLoggedAt = null;
        while (!cancellationToken.IsCancellationRequested)
        {
            if (DateTimeOffset.UtcNow >= heartbeatAt)
            {
                try
                {
                    await _apiClient.HeartbeatAsync(credentials, new AgentHeartbeatRequest(_capabilities), cancellationToken).ConfigureAwait(false);
                    if (!connected || heartbeatFailureCount > 0)
                    {
                        _logger?.Info("center.heartbeat.connected", $"agentId={credentials.AgentId}");
                    }
                    connected = true;
                    heartbeatFailureCount = 0;
                    heartbeatAt = DateTimeOffset.UtcNow.Add(HeartbeatInterval);
                    await _onUpdate(new AgentWorkerUpdate(AgentConnectionState.Connected, "空闲", "心跳成功", DateTimeOffset.Now, false)).ConfigureAwait(false);
                }
                catch (CenterApiException exception) when (exception.IsCredentialInvalid)
                {
                    await NotifyCredentialsInvalidAsync(exception).ConfigureAwait(false);
                    return;
                }
                catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
                {
                    return;
                }
                catch (Exception exception)
                {
                    connected = false;
                    heartbeatFailureCount++;
                    var retryDelay = HeartbeatRetrySchedule.GetDelay(heartbeatFailureCount);
                    _logger?.Warn("center.heartbeat.failed", $"agentId={credentials.AgentId} retrySeconds={retryDelay.TotalSeconds:0}", exception);
                    heartbeatAt = DateTimeOffset.UtcNow.Add(retryDelay);
                    await _onUpdate(new AgentWorkerUpdate(AgentConnectionState.Disconnected, "空闲", $"心跳失败，{retryDelay.TotalSeconds:0} 秒后重试：{exception.Message}", DateTimeOffset.Now, false)).ConfigureAwait(false);
                    await Task.Delay(TimeSpan.FromSeconds(1), cancellationToken).ConfigureAwait(false);
                    continue;
                }
            }

            try
            {
                var task = await _apiClient.ClaimTaskAsync(credentials, cancellationToken).ConfigureAwait(false);
                if (task is not null)
                {
                    var taskStartedAt = Stopwatch.GetTimestamp();
                    _logger?.Info("task.claimed", $"taskId={task.TaskId} operation={task.Operation}");
                    await _onUpdate(new AgentWorkerUpdate(AgentConnectionState.Connected, task.Operation, $"正在执行任务 {task.TaskId}", DateTimeOffset.Now, false)).ConfigureAwait(false);
                    var completion = await _taskExecutor.ExecuteAsync(task, cancellationToken).ConfigureAwait(false);
                    await _apiClient.CompleteTaskAsync(credentials, task.TaskId, completion, cancellationToken).ConfigureAwait(false);
                    _logger?.Info("task.completed", $"taskId={task.TaskId} operation={task.Operation} status={completion.Status} errorCode={completion.Error?.Code ?? "-"} elapsedMs={Stopwatch.GetElapsedTime(taskStartedAt).TotalMilliseconds:F0}");
                    await _onUpdate(new AgentWorkerUpdate(AgentConnectionState.Connected, "空闲", $"任务 {task.TaskId} 已{(completion.Status == "succeeded" ? "完成" : "失败")}", DateTimeOffset.Now, false)).ConfigureAwait(false);
                }
            }
            catch (CenterApiException exception) when (exception.IsCredentialInvalid)
            {
                await NotifyCredentialsInvalidAsync(exception).ConfigureAwait(false);
                return;
            }
            catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
            {
                return;
            }
            catch (Exception exception)
            {
                // 任务轮询与心跳独立，领取失败不能改变已经确认的心跳节奏。
                var now = DateTimeOffset.UtcNow;
                if (lastPollingFailureLoggedAt is null || now - lastPollingFailureLoggedAt >= TimeSpan.FromSeconds(30))
                {
                    _logger?.Warn("task.polling.failed", "任务轮询失败，30 秒内相同网络异常不重复写入日志", exception);
                    lastPollingFailureLoggedAt = now;
                }
                await _onUpdate(new AgentWorkerUpdate(AgentConnectionState.Connected, "空闲", $"任务轮询失败：{exception.Message}", DateTimeOffset.Now, false)).ConfigureAwait(false);
            }

            await Task.Delay(TaskPollingInterval, cancellationToken).ConfigureAwait(false);
        }
    }

    private Task NotifyCredentialsInvalidAsync(CenterApiException exception)
    {
        _logger?.Warn("center.credentials.invalid", "中心拒绝当前 Agent 凭据", exception);
        return _onUpdate(new AgentWorkerUpdate(AgentConnectionState.NotRegistered, "空闲", exception.Message, DateTimeOffset.Now, true));
    }

    public async ValueTask DisposeAsync() => await StopAsync().ConfigureAwait(false);
}

internal static class HeartbeatRetrySchedule
{
    public static TimeSpan GetDelay(int failureCount) => failureCount switch
    {
        <= 1 => TimeSpan.FromSeconds(5),
        2 => TimeSpan.FromSeconds(15),
        _ => TimeSpan.FromSeconds(30),
    };
}

internal sealed record AgentWorkerUpdate(AgentConnectionState State, string CurrentTask, string Message, DateTimeOffset UpdatedAt, bool CredentialsInvalid);
