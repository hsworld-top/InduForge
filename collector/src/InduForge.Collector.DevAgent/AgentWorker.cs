namespace InduForge.Collector.DevAgent;

internal sealed class AgentWorker : IAsyncDisposable
{
    private static readonly TimeSpan HeartbeatInterval = TimeSpan.FromSeconds(15);
    private static readonly TimeSpan TaskPollingInterval = TimeSpan.FromSeconds(2);
    private readonly CenterApiClient _apiClient;
    private readonly CollectorTaskExecutor _taskExecutor;
    private readonly IReadOnlyList<AgentProtocolCapability> _capabilities;
    private readonly Func<AgentWorkerUpdate, Task> _onUpdate;
    private CancellationTokenSource? _runCancellation;
    private Task? _runTask;

    public AgentWorker(CenterApiClient apiClient, CollectorTaskExecutor taskExecutor, IReadOnlyList<AgentProtocolCapability> capabilities, Func<AgentWorkerUpdate, Task> onUpdate)
    {
        _apiClient = apiClient;
        _taskExecutor = taskExecutor;
        _capabilities = capabilities;
        _onUpdate = onUpdate;
    }

    public bool IsRunning => _runTask is { IsCompleted: false };

    public void Start(AgentCredentials credentials, CancellationToken applicationCancellation)
    {
        if (IsRunning) return;
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
    }

    private async Task RunAsync(AgentCredentials credentials, CancellationToken cancellationToken)
    {
        var heartbeatAt = DateTimeOffset.MinValue;
        var heartbeatFailureCount = 0;
        while (!cancellationToken.IsCancellationRequested)
        {
            if (DateTimeOffset.UtcNow >= heartbeatAt)
            {
                try
                {
                    await _apiClient.HeartbeatAsync(credentials, new AgentHeartbeatRequest(_capabilities), cancellationToken).ConfigureAwait(false);
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
                    heartbeatFailureCount++;
                    var retryDelay = HeartbeatRetrySchedule.GetDelay(heartbeatFailureCount);
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
                    await _onUpdate(new AgentWorkerUpdate(AgentConnectionState.Connected, task.Operation, $"正在执行任务 {task.TaskId}", DateTimeOffset.Now, false)).ConfigureAwait(false);
                    var completion = await _taskExecutor.ExecuteAsync(task, cancellationToken).ConfigureAwait(false);
                    await _apiClient.CompleteTaskAsync(credentials, task.TaskId, completion, cancellationToken).ConfigureAwait(false);
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
                await _onUpdate(new AgentWorkerUpdate(AgentConnectionState.Connected, "空闲", $"任务轮询失败：{exception.Message}", DateTimeOffset.Now, false)).ConfigureAwait(false);
            }

            await Task.Delay(TaskPollingInterval, cancellationToken).ConfigureAwait(false);
        }
    }

    private Task NotifyCredentialsInvalidAsync(CenterApiException exception) =>
        _onUpdate(new AgentWorkerUpdate(AgentConnectionState.NotRegistered, "空闲", exception.Message, DateTimeOffset.Now, true));

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
