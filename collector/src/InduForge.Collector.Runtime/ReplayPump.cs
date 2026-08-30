namespace InduForge.Collector.Runtime;

public sealed record ReplayPumpOptions(TimeSpan IdleDelay, TimeSpan InitialRetryDelay, TimeSpan MaximumRetryDelay)
{
    public static ReplayPumpOptions Default { get; } = new(
        TimeSpan.FromMilliseconds(100),
        TimeSpan.FromMilliseconds(250),
        TimeSpan.FromSeconds(30));

    internal void Validate()
    {
        if (IdleDelay < TimeSpan.Zero || InitialRetryDelay <= TimeSpan.Zero || MaximumRetryDelay < InitialRetryDelay)
        {
            throw new ArgumentOutOfRangeException(nameof(InitialRetryDelay));
        }
    }
}

/// <summary>
/// 单线程顺序回放器：一条消息收到真实 PubAck 并落 ACK 标记后，才发布下一条。
/// </summary>
public sealed class ReplayPump
{
    private readonly DurableWal _wal;
    private readonly IJetStreamPublisher _publisher;
    private readonly ReplayPumpOptions _options;
    private int _isRunning;

    public ReplayPump(DurableWal wal, IJetStreamPublisher publisher, ReplayPumpOptions? options = null)
    {
        _wal = wal ?? throw new ArgumentNullException(nameof(wal));
        _publisher = publisher ?? throw new ArgumentNullException(nameof(publisher));
        _options = options ?? ReplayPumpOptions.Default;
        _options.Validate();
    }

    public async Task RunAsync(CancellationToken cancellationToken)
    {
        if (Interlocked.Exchange(ref _isRunning, 1) != 0)
        {
            throw new InvalidOperationException("ReplayPump 已在运行");
        }

        try
        {
            var retryDelay = _options.InitialRetryDelay;
            while (true)
            {
                cancellationToken.ThrowIfCancellationRequested();
                var published = await PumpOnceAsync(cancellationToken).ConfigureAwait(false);
                if (published)
                {
                    retryDelay = _options.InitialRetryDelay;
                    continue;
                }

                if (_wal.GetPendingRecords().Count == 0)
                {
                    await Task.Delay(_options.IdleDelay, cancellationToken).ConfigureAwait(false);
                    continue;
                }

                await Task.Delay(retryDelay, cancellationToken).ConfigureAwait(false);
                retryDelay = NextRetryDelay(retryDelay);
            }
        }
        finally
        {
            Volatile.Write(ref _isRunning, 0);
        }
    }

    /// <summary>为确定性测试及受控宿主调度提供单次顺序处理。</summary>
    public async Task<bool> PumpOnceAsync(CancellationToken cancellationToken)
    {
        cancellationToken.ThrowIfCancellationRequested();
        using var permit = await _wal.BeginReplayAsync(_publisher.PublishAsync, cancellationToken).ConfigureAwait(false);
        if (permit is null) return false;

        try
        {
            await permit.PublishTask.ConfigureAwait(false);
            await _wal.CompleteAfterPublishAsync(permit, cancellationToken).ConfigureAwait(false);
            return true;
        }
        catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
        {
            throw;
        }
        catch (OperationCanceledException)
        {
            _wal.EnsureReplayAvailable();
            throw;
        }
        catch (WalUnavailableException)
        {
            throw;
        }
        catch (Exception exception)
        {
            _wal.RecordPublishFailure(exception);
            return false;
        }
    }

    private TimeSpan NextRetryDelay(TimeSpan current) =>
        current >= _options.MaximumRetryDelay
            ? _options.MaximumRetryDelay
            : TimeSpan.FromMilliseconds(Math.Min(current.TotalMilliseconds * 2, _options.MaximumRetryDelay.TotalMilliseconds));
}
