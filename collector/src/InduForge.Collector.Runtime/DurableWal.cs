using System.Buffers.Binary;
using System.Text;
using System.Text.Json;

namespace InduForge.Collector.Runtime;

public sealed record DurableWalOptions(
    string DirectoryPath,
    long MaxBytes,
    long HighWatermarkBytes,
    Action<WalCompactionStage>? CompactionStageObserver = null,
    Action<WalIoPhase>? IoFaultInjector = null,
    Action<string>? DirectoryFlushObserver = null,
    long DiagnosticReserveBytes = 4 * 1024,
    Action<int>? CompactionFrameObserver = null,
    Action<string, string>? QuarantineMove = null)
{
    private const long MaximumWalBytes = 1L << 30;
    internal void Validate()
    {
        if (string.IsNullOrWhiteSpace(DirectoryPath)) throw new ArgumentException("WAL 目录不能为空", nameof(DirectoryPath));
        if (MaxBytes <= 0) throw new ArgumentOutOfRangeException(nameof(MaxBytes));
        if (MaxBytes > MaximumWalBytes) throw new ArgumentOutOfRangeException(nameof(MaxBytes), "WAL 最大容量不能超过 1GiB");
        if (HighWatermarkBytes <= 0 || HighWatermarkBytes >= MaxBytes)
        {
            throw new ArgumentOutOfRangeException(nameof(HighWatermarkBytes), "高水位必须大于零且严格小于最大容量");
        }
        if (DiagnosticReserveBytes <= 0 || DiagnosticReserveBytes >= MaxBytes)
        {
            throw new ArgumentOutOfRangeException(nameof(DiagnosticReserveBytes), "诊断预留必须大于零且严格小于最大容量");
        }
    }
}

public enum WalCompactionStage
{
    TemporaryDurable,
    PreviousDurable,
    PrimaryDurable,
    PreviousRemoved,
}

public enum WalIoPhase
{
    DataWritten,
    DataFlushed,
    AckWritten,
    AckFlushed,
    TemporaryFlushed,
    PreviousMoved,
    PrimaryMoved,
}

public sealed record WalAppendRequest(string Subject, string EventId, ReadOnlyMemory<byte> Payload, string OwnerId, long Epoch);

public sealed record WalDataRecord(
    string Subject,
    string EventId,
    ReadOnlyMemory<byte> Payload,
    string OwnerId,
    long Epoch,
    long Sequence,
    DateTimeOffset AppendedAt);

public sealed record WalAppendResult(bool Accepted, WalDataRecord? Record, string? RejectionCode)
{
    public static WalAppendResult Backpressure() => new(false, null, "WAL_BACKPRESSURE");
}

/// <summary>读取前取得的原始事件容量 fence；未消耗额度在 Dispose 时归还，禁止半批读取。</summary>
public sealed class WalRawBatchReservation : IDisposable
{
    private DurableWal? _owner;
    private long _remaining;
    private readonly int _perRecord;
    public int Count { get; }

    internal WalRawBatchReservation(DurableWal owner, long bytes, int perRecord, int count)
    {
        _owner = owner; _remaining = bytes; _perRecord = perRecord; Count = count;
    }

    internal DurableWal? Owner => _owner;
    internal void ConsumeUnderWalLock(int actualBytes)
    {
        if (_owner is null || actualBytes > _perRecord || _remaining < _perRecord) throw new WalUnavailableException("WAL 批次预留失效");
        _remaining -= _perRecord;
        _owner.ReleaseRawAdmissionUnderLock(_perRecord);
    }

    public void Dispose()
    {
        var owner = Interlocked.Exchange(ref _owner, null);
        if (owner is null) return;
        lock (owner.AdmissionLock) owner.ReleaseRawAdmissionUnderLock(_remaining);
        _remaining = 0;
    }
}

public sealed record WalDiagnostic(string Code, string Message, long Offset);

public sealed record WalSnapshot(
    int BacklogRecords,
    long BacklogBytes,
    TimeSpan? OldestAge,
    long StoredBytes,
    long ReservedBytes,
    long DiagnosticReserveBytes,
    long AvailableDiagnosticBytes,
    long UsedCapacityBytes,
    long MaxBytes,
    long HighWatermarkBytes,
    bool IsDegraded,
    bool IsRawBackpressured,
    bool IsFull,
    bool IsCorrupted,
    string? LastError);

public sealed class WalCorruptionException : IOException
{
    public WalCorruptionException(string message, IReadOnlyList<WalDiagnostic> diagnostics)
        : base(message) => Diagnostics = diagnostics;

    public IReadOnlyList<WalDiagnostic> Diagnostics { get; }
}

/// <summary>当前实例遇到不确定 I/O 后不能再安全重放，必须关闭并由下一次 Open 恢复。</summary>
public sealed class WalUnavailableException : InvalidOperationException
{
    public WalUnavailableException(string message)
        : base(message)
    {
    }
}

/// <summary>
/// 仅追加的本地可靠队列。序号、事件构造与校验在同一门内完成，Data 先 fsync，
/// 收到真实 PubAck 后才写 ACK；恢复不会跳过任何完整损坏记录。
/// </summary>
public sealed partial class DurableWal : IAsyncDisposable
{
    /// <summary>Collector V1 生产的完整 data.raw.v1 JSON wire payload 上限；Engine 仍可接收更大的 computed 输入。</summary>
    public const int MaximumRawEventPayloadBytes = 64 * 1024;
    // 3×int64、四个长度字段、最长 raw subject(9+UUID)、eventId、owner stableId、frame 与 ACK。
    internal const int MaximumRawReservationBytes = HeaderLength + FooterLength + (3 * sizeof(long)) + (4 * sizeof(int)) + (9 + 36) + 64 + 128 + MaximumRawEventPayloadBytes + AckRecordLength;
    private const uint Magic = 0x4C574649; // "IFWL" 的 little-endian 表示。
    private const byte FormatVersion = 2;
    private const int HeaderLength = 14;
    private const int FooterLength = 4;
    private const int MarkerBodyLength = 16;
    private const int AckBodyLength = MarkerBodyLength + 64;
    private const int AckRecordLength = HeaderLength + AckBodyLength + FooterLength;
    private const int MaximumBodyLength = 64 * 1024 * 1024;
    private static readonly uint[] Crc32CTable = CreateCrc32CTable();
    private static readonly UTF8Encoding StrictUtf8 = new(encoderShouldEmitUTF8Identifier: false, throwOnInvalidBytes: true);
    private readonly DurableWalOptions _options;
    private readonly string _filePath;
    private readonly string _temporaryPath;
    private readonly string _previousPath;
    private readonly FileStream _processLock;
    private readonly SemaphoreSlim _gate = new(1, 1);
    private readonly object _stateLock = new();
    internal object AdmissionLock => _stateLock;
    private readonly CancellationTokenSource _unavailableCancellation = new();
    // 仅保存可重放帧的位置及经验证的固定元数据。payload 不得随 backlog 常驻托管堆；
    // 回放时才在受限大小内读取一帧，避免 1GiB WAL 被一次恢复放大为 1GiB 内存。
    private readonly List<WalPendingIndex> _pending = [];
    private readonly List<WalDiagnostic> _diagnostics = [];
    private long _storedBytes;
    private long _rawAdmissionReservedBytes;

    internal void ReleaseRawAdmissionUnderLock(long bytes)
    {
        _rawAdmissionReservedBytes = checked(_rawAdmissionReservedBytes - bytes);
    }
    private long _maximumSequence;
    private bool _isCorrupted;
    private bool _isBackpressured;
    private bool _stopping;
    private int _replayInFlight;
    private string? _lastError;
    private readonly object _shutdownLock = new();
    private Task? _shutdownTask;
    private TaskCompletionSource _replayDrained = CompletedSignal();
    private TaskCompletionSource _operationsDrained = CompletedSignal();
    private int _activeOperations;

    private DurableWal(DurableWalOptions options, FileStream processLock)
    {
        _options = options;
        _filePath = Path.Combine(options.DirectoryPath, "collector.wal");
        _temporaryPath = Path.Combine(options.DirectoryPath, "collector.wal.compacting");
        _previousPath = Path.Combine(options.DirectoryPath, "collector.wal.previous");
        _processLock = processLock;
    }

    public string StoragePath => _filePath;

    public IReadOnlyList<WalDiagnostic> Diagnostics
    {
        get
        {
            lock (_stateLock) return _diagnostics.ToArray();
        }
    }

    public static async Task<DurableWal> OpenAsync(DurableWalOptions options, CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(options);
        options.Validate();
        DurableStorage.CreateDirectoryDurably(options.DirectoryPath, options.DirectoryFlushObserver);
        FileStream? processLock = null;
        try
        {
            processLock = new FileStream(Path.Combine(options.DirectoryPath, "collector.wal.lock"), FileMode.OpenOrCreate, FileAccess.ReadWrite, FileShare.None);
            var wal = new DurableWal(options, processLock);
            await wal.RecoverAsync(cancellationToken).ConfigureAwait(false);
            return wal;
        }
        catch
        {
            processLock?.Dispose();
            throw;
        }
    }

    /// <summary>
    /// 在 WAL 串行门内生成事件：调用方不能预读序号，也不能让 payload 中的序号与 WAL 不一致。
    /// </summary>
    public async Task<WalAppendResult> AppendDataAsync(
        Func<long, WalAppendRequest> eventFactory,
        CancellationToken cancellationToken = default)
        => await AppendDataCoreAsync(eventFactory, reservation: null, cancellationToken).ConfigureAwait(false);

    public async Task<WalRawBatchReservation?> ReserveRawBatchAsync(int recordCount, CancellationToken cancellationToken = default)
    {
        ArgumentOutOfRangeException.ThrowIfLessThan(recordCount, 1);
        var perRecord = MaximumRawReservationBytes;
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfCorrupted();
            lock (_stateLock)
            {
                var available = _options.MaxBytes - checked(UsedCapacityUnsafe() + _rawAdmissionReservedBytes);
                var count = (int)Math.Min(recordCount, available / perRecord);
                if (count < 1) return null;
                var bytes = checked((long)count * perRecord);
                _rawAdmissionReservedBytes += bytes;
                return new WalRawBatchReservation(this, bytes, perRecord, count);
            }
        }
        finally { _gate.Release(); }
    }

    public Task<WalAppendResult> AppendReservedDataAsync(WalRawBatchReservation reservation, Func<long, WalAppendRequest> eventFactory, CancellationToken cancellationToken = default)
    {
        if (reservation is null || !ReferenceEquals(reservation.Owner, this)) throw new ArgumentException("WAL 批次预留不属于当前实例", nameof(reservation));
        return AppendDataCoreAsync(eventFactory, reservation, cancellationToken);
    }

    private async Task<WalAppendResult> AppendDataCoreAsync(
        Func<long, WalAppendRequest> eventFactory,
        WalRawBatchReservation? reservation,
        CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(eventFactory);
        EnterOperation();
        try
        {
            await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
            try
            {
                ThrowIfCorrupted();
                cancellationToken.ThrowIfCancellationRequested();
                var sequence = checked(GetMaximumSequence() + 1);
                var request = eventFactory(sequence) ?? throw new ArgumentException("事件工厂不能返回 null", nameof(eventFactory));
                CollectorV1EventValidator.Validate(request, sequence);
                if (request.Subject.StartsWith("data.raw.", StringComparison.Ordinal) && request.Payload.Length > MaximumRawEventPayloadBytes) throw new ArgumentOutOfRangeException(nameof(eventFactory), "raw event 超出单条上限");
                var record = new WalDataRecord(
                    request.Subject,
                    request.EventId,
                    request.Payload.ToArray(),
                    request.OwnerId,
                    request.Epoch,
                    sequence,
                    DateTimeOffset.UtcNow);
                var bytes = EncodeData(record);

                var isDiagnostic = string.Equals(request.Subject, "alarm.event", StringComparison.Ordinal);
                lock (_stateLock)
                {
                    if (reservation is not null) reservation.ConsumeUnderWalLock(bytes.Length + AckRecordLength);
                    var ackReservedAfterAppend = checked(ReservedBytesUnsafe() + AckRecordLength);
                    var physicalAfterAppend = checked(_storedBytes + bytes.Length + ackReservedAfterAppend);
                    var diagnosticCost = checked(bytes.Length + AckRecordLength);
                    // 诊断可使用独立 reserve，但绝不能侵占已在 Read 前授予的 raw admission。
                    var usedAfterAppend = isDiagnostic
                        ? checked(physicalAfterAppend + _rawAdmissionReservedBytes)
                        : checked(physicalAfterAppend + RemainingDiagnosticReserveUnsafe() + _rawAdmissionReservedBytes);
                    if ((isDiagnostic && diagnosticCost > _options.DiagnosticReserveBytes) || usedAfterAppend > _options.MaxBytes)
                    {
                        _isBackpressured = !isDiagnostic || usedAfterAppend > _options.MaxBytes;
                        _lastError = isDiagnostic ? "WAL_DIAGNOSTIC_BACKPRESSURE" : "WAL_BACKPRESSURE";
                        return WalAppendResult.Backpressure();
                    }
                }

                long frameOffset;
                lock (_stateLock) frameOffset = _storedBytes;
                await AppendDurablyAsync(bytes, WalIoPhase.DataFlushed, cancellationToken).ConfigureAwait(false);
                var index = DecodeDataIndex(
                    bytes.AsSpan(HeaderLength, bytes.Length - HeaderLength - FooterLength),
                    _filePath,
                    frameOffset,
                    bytes.Length);
                lock (_stateLock)
                {
                    _pending.Add(index);
                    _storedBytes += bytes.Length;
                    _maximumSequence = sequence;
                    _lastError = null;
                    _isBackpressured = false;
                }

                return new WalAppendResult(true, CloneRecord(record), null);
            }
            finally
            {
                _gate.Release();
            }
        }
        finally
        {
            ExitOperation();
        }
    }

    internal Task CompleteAfterPublishAsync(WalReplayPermit permit, CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(permit);
        if (!ReferenceEquals(permit.Owner, this) || !permit.IsActive || !permit.PublishTask.IsCompletedSuccessfully)
        {
            throw new WalUnavailableException("未获得真实 PubAck 的 WAL replay permit");
        }
        return CommitCoreAsync(permit.Index, cancellationToken);
    }

    private async Task CommitCoreAsync(WalPendingIndex record, CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(record);
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfCorrupted();
            cancellationToken.ThrowIfCancellationRequested();
            WalPendingIndex? pending;
            lock (_stateLock)
            {
                // ACK 只能确认严格队首。旧 permit 在已提交后允许幂等返回，但绝不能
                // 借同一 epoch/sequence 搜索并跳过前序记录。
                pending = _pending.Count == 0 ? null : _pending[0];
                if (pending is not null && (pending.Epoch != record.Epoch || pending.Sequence != record.Sequence || !string.Equals(pending.EventId, record.EventId, StringComparison.Ordinal)))
                {
                    if (_pending.Any(item => item.Epoch == record.Epoch && item.Sequence == record.Sequence))
                    {
                        throw new WalUnavailableException("WAL ACK 尝试跳过队首记录");
                    }
                    return;
                }
            }
            if (pending is null) return;

            // PubAck 已确认后再追加 ACK 并 fsync；容量在 Data 时已预留，不会被自身 ACK 拒绝。
            var marker = EncodeAck(pending);
            await AppendDurablyAsync(marker, WalIoPhase.AckFlushed, cancellationToken).ConfigureAwait(false);
            lock (_stateLock)
            {
                _storedBytes += marker.Length;
                _pending.RemoveAll(item => item.Epoch == pending.Epoch && item.Sequence == pending.Sequence);
                _lastError = null;
            }
            await CompactIfBeneficialAsync(cancellationToken, force: string.Equals(pending.Subject, "alarm.event", StringComparison.Ordinal)).ConfigureAwait(false);
            lock (_stateLock)
            {
                _isBackpressured = UsedCapacityUnsafe() >= _options.MaxBytes;
            }
        }
        finally
        {
            _gate.Release();
        }
    }

    public async Task CompactAsync(CancellationToken cancellationToken = default)
    {
        EnterOperation();
        try
        {
            await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
            try
            {
                ThrowIfCorrupted();
                cancellationToken.ThrowIfCancellationRequested();
                lock (_stateLock)
                {
                    // permit 已固定一个旧帧；此时替换主文件会使后续 ACK 的索引失效，延后压缩即可。
                    if (_replayInFlight != 0) return;
                }
                await CompactUnderLockAsync(cancellationToken).ConfigureAwait(false);
            }
            finally
            {
                _gate.Release();
            }
        }
        finally
        {
            ExitOperation();
        }
    }

    /// <summary>仅返回不含 payload 的 backlog 快照；正文只能由单条 replay permit 按需读取。</summary>
    internal IReadOnlyList<WalPendingInfo> GetPendingRecords()
    {
        lock (_stateLock) return _pending.Select(static item => item.ToInfo()).ToArray();
    }

    /// <summary>发布前调用；隔离状态绝不允许把内存残留记录继续送往上游。</summary>
    public void EnsureReplayAvailable()
    {
        lock (_stateLock)
        {
            ThrowIfUnavailableUnsafe();
        }
    }

    /// <summary>
    /// 在状态锁内确认可用并立即发起发布。不会持锁等待网络；故障先取得锁时不会调用
    /// publisher，发布先取得锁时最多留下这一个不可撤回的 at-least-once 在飞请求。
    /// </summary>
    internal async Task<WalReplayPermit?> BeginReplayAsync(Func<WalDataRecord, CancellationToken, Task> publish, CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(publish);
        WalPendingIndex? pending;
        lock (_stateLock)
        {
            ThrowIfUnavailableUnsafe();
            // 已有一个发布在飞时调用方获得 false，保持 WAL 单序且不重复投递。
            if (_replayInFlight != 0) return null;
            pending = _pending.Count == 0 ? null : _pending[0];
            if (pending is null) return null;
            _replayInFlight++;
            if (_replayInFlight == 1) _replayDrained = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        }

        WalDataRecord record;
        try
        {
            // 回放只读取当前一帧；压缩发现 permit 后不会替换索引文件。
            record = await ReadPendingRecordAsync(pending, cancellationToken).ConfigureAwait(false);
        }
        catch
        {
            ReleaseReplayPermit();
            throw;
        }

        var linkedCancellation = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken, _unavailableCancellation.Token);
        lock (_stateLock)
        {
            try
            {
                ThrowIfUnavailableUnsafe();
            }
            catch
            {
                linkedCancellation.Dispose();
                ReleaseReplayPermitUnsafe();
                throw;
            }
            // 在状态锁中发起调用，使 FailCurrentInstance 与发布起点线性化；不持锁 await 网络。
            Task publishTask;
            try { publishTask = publish(record, linkedCancellation.Token); }
            catch (Exception exception) { publishTask = Task.FromException(exception); }
            return new WalReplayPermit(this, pending, record, publishTask, linkedCancellation);
        }
    }

    public WalSnapshot GetSnapshot()
    {
        lock (_stateLock)
        {
            var reserved = ReservedBytesUnsafe();
            var remainingDiagnostic = RemainingDiagnosticReserveUnsafe();
            var used = checked(_storedBytes + reserved + remainingDiagnostic + _rawAdmissionReservedBytes);
            var availableDiagnostic = Math.Min(remainingDiagnostic, Math.Max(0, _options.MaxBytes - checked(_storedBytes + reserved)));
            TimeSpan? oldest = _pending.Count == 0 ? null : DateTimeOffset.UtcNow - _pending[0].AppendedAt;
            return new WalSnapshot(
                _pending.Count,
                _pending.Sum(record => (long)record.PayloadLength),
                oldest,
                _storedBytes,
                checked(reserved + _rawAdmissionReservedBytes),
                _options.DiagnosticReserveBytes,
                availableDiagnostic,
                used,
                _options.MaxBytes,
                _options.HighWatermarkBytes,
                used >= _options.HighWatermarkBytes,
                _isBackpressured || used >= _options.MaxBytes,
                checked(_storedBytes + reserved) >= _options.MaxBytes,
                _isCorrupted,
                _lastError);
        }
    }

    public void RecordPublishFailure(Exception exception)
    {
        ArgumentNullException.ThrowIfNull(exception);
        lock (_stateLock)
        {
            // 隔离原因是恢复决策依据，不能被随后一次发布失败掩盖。
            if (!_isCorrupted) _lastError = exception.GetType().Name;
        }
    }

    public ValueTask DisposeAsync()
    {
        Task shutdown;
        lock (_shutdownLock)
        {
            shutdown = _shutdownTask ??= ShutdownAsync();
        }
        return new ValueTask(shutdown);
    }

    private async Task ShutdownAsync()
    {
        Task replayDrained;
        Task operationsDrained;
        lock (_stateLock)
        {
            _stopping = true;
            replayDrained = _replayDrained.Task;
            operationsDrained = _operationsDrained.Task;
        }
        // Cancel 可能同步执行外部回调，不能放在状态锁内。
        _unavailableCancellation.Cancel();
        await operationsDrained.ConfigureAwait(false);
        await _gate.WaitAsync().ConfigureAwait(false);
        _gate.Release();
        await replayDrained.ConfigureAwait(false);
        _unavailableCancellation.Dispose();
        _processLock.Dispose();
        _gate.Dispose();
    }

    private async Task RecoverAsync(CancellationToken cancellationToken)
    {
        await RestorePreviousIfNecessaryAsync(cancellationToken).ConfigureAwait(false);
        if (!File.Exists(_filePath)) return;

        var scan = await ScanAsync(_filePath, cancellationToken).ConfigureAwait(false);
        if (scan.TornTailOffset is { } tornOffset)
        {
            await TruncateTornTailAsync(_filePath, scan.LastCompleteOffset, tornOffset, cancellationToken).ConfigureAwait(false);
        }

        lock (_stateLock)
        {
            _pending.Clear();
            _pending.AddRange(scan.Pending);
            _storedBytes = new FileInfo(_filePath).Length;
            _maximumSequence = scan.MaximumSequence;
            _isBackpressured = UsedCapacityUnsafe() >= _options.MaxBytes;
        }
    }

    private async Task RestorePreviousIfNecessaryAsync(CancellationToken cancellationToken)
    {
        if (File.Exists(_filePath) || !File.Exists(_previousPath)) return;

        // 主文件缺失时先验证 previous，绝不把它当成空 WAL 或未验证地恢复。
        _ = await ScanAsync(_previousPath, cancellationToken).ConfigureAwait(false);
        File.Move(_previousPath, _filePath);
        DurableStorage.FlushParentDirectory(_options.DirectoryPath);
        AddDiagnostic("WAL_PREVIOUS_RESTORED", "检测到压缩中断，已恢复经过校验的 previous WAL", 0);
    }

    private async Task<RecoveryScan> ScanAsync(string path, CancellationToken cancellationToken)
    {
        await using var stream = new FileStream(path, FileMode.Open, FileAccess.Read, FileShare.Read, 4096, FileOptions.SequentialScan);
        if (stream.Length > _options.MaxBytes) FailClosed(path, 0, "WAL_TOO_LARGE", "WAL 文件超过配置容量");
        var pending = new SortedDictionary<long, WalPendingIndex>();
        long offset = 0;
        long lastCompleteOffset = 0;
        long maximumSequence = 0;
        long lastDataSequence = 0;
        var header = new byte[HeaderLength];
        while (true)
        {
            var headerRead = await ReadAtMostAsync(stream, header, cancellationToken).ConfigureAwait(false);
            if (headerRead == 0) break;
            if (headerRead < HeaderLength)
            {
                return new RecoveryScan(pending.Values.ToArray(), maximumSequence, lastCompleteOffset, offset);
            }
            var headerSpan = header.AsSpan();
            var expectedHeaderCrc = BinaryPrimitives.ReadUInt32LittleEndian(headerSpan[10..14]);
            if (Crc32C(headerSpan[..10]) != expectedHeaderCrc)
            {
                FailClosed(path, offset, "WAL_HEADER_CRC_CORRUPT", "WAL 完整记录头损坏，不能按尾部撕裂处理");
            }

            if (BinaryPrimitives.ReadUInt32LittleEndian(headerSpan[..4]) != Magic || header[4] != FormatVersion)
            {
                FailClosed(path, offset, "WAL_FORMAT_CORRUPT", "WAL 记录格式损坏");
            }

            var type = (RecordType)headerSpan[5];
            var bodyLength = BinaryPrimitives.ReadInt32LittleEndian(headerSpan[6..10]);
            if (!Enum.IsDefined(type) || bodyLength < 0 || bodyLength > MaximumBodyLength)
            {
                FailClosed(path, offset, "WAL_LENGTH_CORRUPT", "WAL 记录长度或类型无效");
            }

            var body = new byte[bodyLength];
            if (await ReadAtMostAsync(stream, body, cancellationToken).ConfigureAwait(false) < bodyLength)
            {
                return new RecoveryScan(pending.Values.ToArray(), maximumSequence, lastCompleteOffset, offset);
            }
            var footer = new byte[FooterLength];
            if (await ReadAtMostAsync(stream, footer, cancellationToken).ConfigureAwait(false) < FooterLength)
            {
                return new RecoveryScan(pending.Values.ToArray(), maximumSequence, lastCompleteOffset, offset);
            }
            var expectedBodyCrc = BinaryPrimitives.ReadUInt32LittleEndian(footer);
            if (Crc32C(body) != expectedBodyCrc)
            {
                FailClosed(path, offset, "WAL_CRC_CORRUPT", "WAL 完整记录正文校验失败");
            }

            switch (type)
            {
                case RecordType.Data:
                    {
                        var data = DecodeDataIndex(body, path, offset, checked(HeaderLength + bodyLength + FooterLength));
                        try
                        {
                            CollectorV1EventValidator.Validate(new WalAppendRequest(data.Subject, data.EventId, body.AsMemory(data.PayloadBodyOffset, data.PayloadLength), data.OwnerId, data.Epoch), data.Sequence);
                        }
                        catch (Exception exception) when (exception is ArgumentException or JsonException)
                        {
                            FailClosed(path, offset, "WAL_EVENT_INVALID", "恢复记录不符合 Collector V1 事件契约: " + exception.Message);
                        }

                        if (data.Sequence <= lastDataSequence || !pending.TryAdd(data.Sequence, data))
                        {
                            FailClosed(path, offset, "WAL_SEQUENCE_INVALID", "WAL Data sequence 必须全局严格递增且唯一");
                        }

                        lastDataSequence = data.Sequence;
                        maximumSequence = Math.Max(maximumSequence, data.Sequence);
                        break;
                    }
                case RecordType.Ack:
                    {
                        var marker = DecodeAck(body, path, offset);
                        var first = pending.Count == 0 ? null : pending.First().Value;
                        if (first is null || first.Epoch != marker.Epoch || first.Sequence != marker.Sequence || !string.Equals(first.EventId, marker.EventId, StringComparison.Ordinal))
                        {
                            FailClosed(path, offset, "WAL_ACK_INVALID", "ACK 必须严格对应队首未确认 Data 的 epoch、sequence 与 eventId");
                        }
                        pending.Remove(first!.Sequence);
                        break;
                    }
                case RecordType.Checkpoint:
                    {
                        var marker = DecodeMarker(body, path, offset);
                        if (marker.Epoch != 0 || marker.Sequence < maximumSequence)
                        {
                            FailClosed(path, offset, "WAL_CHECKPOINT_INVALID", "Checkpoint 不满足内部序号约束");
                        }
                        maximumSequence = marker.Sequence;
                        break;
                    }
            }

            offset += HeaderLength + bodyLength + FooterLength;
            lastCompleteOffset = offset;
        }

        return new RecoveryScan(pending.Values.ToArray(), maximumSequence, lastCompleteOffset, null);
    }

    private static async Task<int> ReadAtMostAsync(Stream stream, Memory<byte> buffer, CancellationToken cancellationToken)
    {
        var read = 0;
        while (read < buffer.Length)
        {
            var current = await stream.ReadAsync(buffer[read..], cancellationToken).ConfigureAwait(false);
            if (current == 0) break;
            read += current;
        }
        return read;
    }

    private async Task TruncateTornTailAsync(string path, long length, long offset, CancellationToken cancellationToken)
    {
        await using var stream = new FileStream(path, FileMode.Open, FileAccess.Write, FileShare.None, 4096, FileOptions.WriteThrough);
        stream.SetLength(length);
        await stream.FlushAsync(cancellationToken).ConfigureAwait(false);
        stream.Flush(flushToDisk: true);
        DurableStorage.FlushParentDirectory(_options.DirectoryPath);
        AddDiagnostic("WAL_TORN_TAIL_TRUNCATED", "WAL 尾部不完整，已截断到最后完整记录", offset);
    }

    private async Task<WalDataRecord> ReadPendingRecordAsync(WalPendingIndex index, CancellationToken cancellationToken)
    {
        var frame = new byte[index.FrameLength];
        try
        {
            await using var stream = new FileStream(_filePath, FileMode.Open, FileAccess.Read, FileShare.ReadWrite, 4096, FileOptions.RandomAccess);
            stream.Position = index.FrameOffset;
            if (await ReadAtMostAsync(stream, frame, cancellationToken).ConfigureAwait(false) != frame.Length)
            {
                FailClosed(_filePath, index.FrameOffset, "WAL_REPLAY_TRUNCATED", "回放目标帧不完整");
            }
            var header = frame.AsSpan(0, HeaderLength);
            if (Crc32C(header[..10]) != BinaryPrimitives.ReadUInt32LittleEndian(header[10..14]))
            {
                FailClosed(_filePath, index.FrameOffset, "WAL_HEADER_CRC_CORRUPT", "回放目标头校验失败");
            }
            var bodyLength = BinaryPrimitives.ReadInt32LittleEndian(header[6..10]);
            if (bodyLength < 0 || bodyLength > MaximumBodyLength || frame.Length != HeaderLength + bodyLength + FooterLength)
            {
                FailClosed(_filePath, index.FrameOffset, "WAL_LENGTH_CORRUPT", "回放目标长度无效");
            }
            var body = frame.AsSpan(HeaderLength, bodyLength);
            if (Crc32C(body) != BinaryPrimitives.ReadUInt32LittleEndian(frame.AsSpan(HeaderLength + bodyLength, FooterLength)))
            {
                FailClosed(_filePath, index.FrameOffset, "WAL_CRC_CORRUPT", "回放目标正文校验失败");
            }
            var record = DecodeData(body, _filePath, checked((int)Math.Min(index.FrameOffset, int.MaxValue)));
            try
            {
                CollectorV1EventValidator.Validate(new WalAppendRequest(record.Subject, record.EventId, record.Payload, record.OwnerId, record.Epoch), record.Sequence);
            }
            catch (Exception exception) when (exception is ArgumentException or JsonException)
            {
                FailClosed(_filePath, index.FrameOffset, "WAL_EVENT_INVALID", "回放目标不符合 Collector V1 事件契约: " + exception.Message);
            }
            if (record.Epoch != index.Epoch || record.Sequence != index.Sequence || !string.Equals(record.EventId, index.EventId, StringComparison.Ordinal))
            {
                FailClosed(_filePath, index.FrameOffset, "WAL_INDEX_MISMATCH", "回放索引与磁盘记录不一致");
            }
            return record;
        }
        catch (WalCorruptionException) { throw; }
        catch (Exception exception) when (exception is IOException or ArgumentException or JsonException)
        {
            FailCurrentInstance("WAL_IO_UNCERTAIN", "读取回放目标失败，必须重启恢复", exception);
            throw;
        }
    }

    private async Task AppendDurablyAsync(byte[] bytes, WalIoPhase phase, CancellationToken cancellationToken)
    {
        cancellationToken.ThrowIfCancellationRequested();
        var createsPrimary = !File.Exists(_filePath);
        try
        {
            await using var stream = new FileStream(_filePath, FileMode.Append, FileAccess.Write, FileShare.Read, 4096, FileOptions.WriteThrough);
            await stream.WriteAsync(bytes, CancellationToken.None).ConfigureAwait(false);
            _options.IoFaultInjector?.Invoke(phase == WalIoPhase.AckFlushed ? WalIoPhase.AckWritten : WalIoPhase.DataWritten);
            await stream.FlushAsync(CancellationToken.None).ConfigureAwait(false);
            stream.Flush(flushToDisk: true);
            if (createsPrimary) DurableStorage.FlushParentDirectory(_options.DirectoryPath);
            _options.IoFaultInjector?.Invoke(phase);
        }
        catch (Exception exception)
        {
            FailCurrentInstance("WAL_IO_UNCERTAIN", "WAL 写入或刷盘结果不确定，必须重启恢复", exception);
            throw;
        }
    }

    private async Task CompactIfBeneficialAsync(CancellationToken cancellationToken, bool force = false)
    {
        lock (_stateLock)
        {
            if (!force && _pending.Count != 0 && UsedCapacityUnsafe() < _options.HighWatermarkBytes) return;
        }
        await CompactUnderLockAsync(cancellationToken).ConfigureAwait(false);
    }

    private async Task CompactUnderLockAsync(CancellationToken cancellationToken)
    {
        cancellationToken.ThrowIfCancellationRequested();
        try
        {
            WalPendingIndex[] pending;
            long maximumSequence;
            lock (_stateLock)
            {
                if (_replayInFlight != 0) return;
                pending = _pending.ToArray();
                maximumSequence = _maximumSequence;
            }

            if (File.Exists(_temporaryPath)) File.Delete(_temporaryPath);
            await using (var temporary = new FileStream(_temporaryPath, FileMode.CreateNew, FileAccess.Write, FileShare.None, 4096, FileOptions.WriteThrough))
            {
                await temporary.WriteAsync(EncodeMarker(RecordType.Checkpoint, 0, maximumSequence), CancellationToken.None).ConfigureAwait(false);
                foreach (var index in pending)
                {
                    // 压缩同样是逐帧流式：Read/Encode/Write 后释放，不把整个 backlog 堆积到内存。
                    var record = await ReadPendingRecordAsync(index, cancellationToken).ConfigureAwait(false);
                    var frame = EncodeData(record);
                    _options.CompactionFrameObserver?.Invoke(1);
                    try
                    {
                        await temporary.WriteAsync(frame, CancellationToken.None).ConfigureAwait(false);
                    }
                    finally
                    {
                        _options.CompactionFrameObserver?.Invoke(0);
                    }
                }
                await temporary.FlushAsync(CancellationToken.None).ConfigureAwait(false);
                temporary.Flush(flushToDisk: true);
            }

            DurableStorage.FlushParentDirectory(_options.DirectoryPath);
            _options.IoFaultInjector?.Invoke(WalIoPhase.TemporaryFlushed);
            _options.CompactionStageObserver?.Invoke(WalCompactionStage.TemporaryDurable);
            if (File.Exists(_previousPath))
            {
                File.Delete(_previousPath);
                DurableStorage.FlushParentDirectory(_options.DirectoryPath);
            }

            if (File.Exists(_filePath))
            {
                File.Move(_filePath, _previousPath);
                DurableStorage.FlushParentDirectory(_options.DirectoryPath);
                _options.IoFaultInjector?.Invoke(WalIoPhase.PreviousMoved);
                _options.CompactionStageObserver?.Invoke(WalCompactionStage.PreviousDurable);
            }

            File.Move(_temporaryPath, _filePath);
            DurableStorage.FlushParentDirectory(_options.DirectoryPath);
            _options.IoFaultInjector?.Invoke(WalIoPhase.PrimaryMoved);
            _options.CompactionStageObserver?.Invoke(WalCompactionStage.PrimaryDurable);
            if (File.Exists(_previousPath))
            {
                File.Delete(_previousPath);
                DurableStorage.FlushParentDirectory(_options.DirectoryPath);
            }
            _options.CompactionStageObserver?.Invoke(WalCompactionStage.PreviousRemoved);
            var scan = await ScanAsync(_filePath, cancellationToken).ConfigureAwait(false);
            lock (_stateLock)
            {
                _pending.Clear();
                _pending.AddRange(scan.Pending);
                _storedBytes = new FileInfo(_filePath).Length;
                _maximumSequence = scan.MaximumSequence;
            }
        }
        catch (Exception exception)
        {
            FailCurrentInstance("WAL_IO_UNCERTAIN", "WAL 压缩结果不确定，必须重启恢复", exception);
            throw;
        }
    }

    private static byte[] EncodeData(WalDataRecord record)
    {
        using var body = new MemoryStream();
        WriteInt64(body, record.Epoch);
        WriteInt64(body, record.Sequence);
        WriteInt64(body, record.AppendedAt.UtcTicks);
        WriteBytes(body, StrictUtf8.GetBytes(record.Subject));
        WriteBytes(body, StrictUtf8.GetBytes(record.EventId));
        WriteBytes(body, StrictUtf8.GetBytes(record.OwnerId));
        WriteBytes(body, record.Payload.Span);
        var encodedBody = body.ToArray();
        if (encodedBody.Length > MaximumBodyLength) throw new ArgumentOutOfRangeException(nameof(record), "Data record 超过 WAL 单帧上限");
        return Frame(RecordType.Data, encodedBody);
    }

    private static byte[] EncodeMarker(RecordType type, long epoch, long sequence)
    {
        Span<byte> body = stackalloc byte[MarkerBodyLength];
        BinaryPrimitives.WriteInt64LittleEndian(body[..8], epoch);
        BinaryPrimitives.WriteInt64LittleEndian(body[8..], sequence);
        return Frame(type, body);
    }

    private static byte[] EncodeAck(WalPendingIndex pending)
    {
        Span<byte> body = stackalloc byte[AckBodyLength];
        BinaryPrimitives.WriteInt64LittleEndian(body[..8], pending.Epoch);
        BinaryPrimitives.WriteInt64LittleEndian(body[8..16], pending.Sequence);
        if (pending.EventId.Length != 64 || !StrictUtf8.GetBytes(pending.EventId, body[16..]).Equals(64)) throw new InvalidDataException("ACK eventId 无效");
        return Frame(RecordType.Ack, body);
    }

    private static byte[] Frame(RecordType type, ReadOnlySpan<byte> body)
    {
        var bytes = new byte[checked(HeaderLength + body.Length + FooterLength)];
        var header = bytes.AsSpan(0, HeaderLength);
        BinaryPrimitives.WriteUInt32LittleEndian(header[..4], Magic);
        header[4] = FormatVersion;
        header[5] = (byte)type;
        BinaryPrimitives.WriteInt32LittleEndian(header[6..10], body.Length);
        BinaryPrimitives.WriteUInt32LittleEndian(header[10..14], Crc32C(header[..10]));
        body.CopyTo(bytes.AsSpan(HeaderLength));
        BinaryPrimitives.WriteUInt32LittleEndian(bytes.AsSpan(HeaderLength + body.Length, FooterLength), Crc32C(body));
        return bytes;
    }

    private WalDataRecord DecodeData(ReadOnlySpan<byte> body, string path, int offset)
    {
        try
        {
            var cursor = 0;
            var epoch = ReadInt64(body, ref cursor);
            var sequence = ReadInt64(body, ref cursor);
            var appendedTicks = ReadInt64(body, ref cursor);
            var subject = StrictUtf8.GetString(ReadBytes(body, ref cursor));
            var eventId = StrictUtf8.GetString(ReadBytes(body, ref cursor));
            var ownerId = StrictUtf8.GetString(ReadBytes(body, ref cursor));
            var payload = ReadBytes(body, ref cursor).ToArray();
            if (cursor != body.Length) throw new InvalidDataException("Data 记录存在尾随字节");
            return new WalDataRecord(subject, eventId, payload, ownerId, epoch, sequence, new DateTimeOffset(appendedTicks, TimeSpan.Zero));
        }
        catch (Exception exception) when (exception is ArgumentException or DecoderFallbackException or InvalidDataException or ArgumentOutOfRangeException)
        {
            FailClosed(path, offset, "WAL_DATA_CORRUPT", "WAL Data 记录解码失败: " + exception.Message);
            throw;
        }
    }

    private WalPendingIndex DecodeDataIndex(ReadOnlySpan<byte> body, string path, long frameOffset, int frameLength)
    {
        try
        {
            var cursor = 0;
            var epoch = ReadInt64(body, ref cursor);
            var sequence = ReadInt64(body, ref cursor);
            var appendedTicks = ReadInt64(body, ref cursor);
            var subject = StrictUtf8.GetString(ReadBytes(body, ref cursor));
            var eventId = StrictUtf8.GetString(ReadBytes(body, ref cursor));
            var ownerId = StrictUtf8.GetString(ReadBytes(body, ref cursor));
            EnsureRemaining(body, cursor, 4);
            var payloadLength = BinaryPrimitives.ReadInt32LittleEndian(body.Slice(cursor, 4));
            cursor += 4;
            if (payloadLength < 0 || payloadLength > MaximumBodyLength) throw new InvalidDataException("payload 长度无效");
            var payloadBodyOffset = cursor;
            EnsureRemaining(body, cursor, payloadLength);
            cursor += payloadLength;
            if (cursor != body.Length) throw new InvalidDataException("Data 记录存在尾随字节");
            return new WalPendingIndex(frameOffset, frameLength, payloadLength, payloadBodyOffset, subject, eventId, ownerId, epoch, sequence, new DateTimeOffset(appendedTicks, TimeSpan.Zero));
        }
        catch (Exception exception) when (exception is ArgumentException or DecoderFallbackException or InvalidDataException or ArgumentOutOfRangeException)
        {
            FailClosed(path, frameOffset, "WAL_DATA_CORRUPT", "WAL Data 记录索引解码失败: " + exception.Message);
            throw;
        }
    }

    private (long Epoch, long Sequence) DecodeMarker(ReadOnlySpan<byte> body, string path, long offset)
    {
        if (body.Length != MarkerBodyLength) FailClosed(path, offset, "WAL_MARKER_CORRUPT", "WAL 标记记录长度无效");
        return (BinaryPrimitives.ReadInt64LittleEndian(body[..8]), BinaryPrimitives.ReadInt64LittleEndian(body[8..]));
    }

    private (long Epoch, long Sequence, string EventId) DecodeAck(ReadOnlySpan<byte> body, string path, long offset)
    {
        if (body.Length != AckBodyLength) FailClosed(path, offset, "WAL_ACK_CORRUPT", "WAL ACK 记录长度无效");
        var eventId = StrictUtf8.GetString(body[16..]);
        if (eventId.Length != 64 || eventId.Any(character => !((character >= 'a' && character <= 'f') || (character >= '0' && character <= '9')))) FailClosed(path, offset, "WAL_ACK_CORRUPT", "WAL ACK eventId 无效");
        return (BinaryPrimitives.ReadInt64LittleEndian(body[..8]), BinaryPrimitives.ReadInt64LittleEndian(body[8..16]), eventId);
    }

    private static void WriteInt64(Stream stream, long value)
    {
        Span<byte> buffer = stackalloc byte[8];
        BinaryPrimitives.WriteInt64LittleEndian(buffer, value);
        stream.Write(buffer);
    }

    private static void WriteBytes(Stream stream, ReadOnlySpan<byte> value)
    {
        Span<byte> length = stackalloc byte[4];
        BinaryPrimitives.WriteInt32LittleEndian(length, value.Length);
        stream.Write(length);
        stream.Write(value);
    }

    private static long ReadInt64(ReadOnlySpan<byte> body, ref int cursor)
    {
        EnsureRemaining(body, cursor, 8);
        var value = BinaryPrimitives.ReadInt64LittleEndian(body.Slice(cursor, 8));
        cursor += 8;
        return value;
    }

    private static ReadOnlySpan<byte> ReadBytes(ReadOnlySpan<byte> body, ref int cursor)
    {
        EnsureRemaining(body, cursor, 4);
        var length = BinaryPrimitives.ReadInt32LittleEndian(body.Slice(cursor, 4));
        cursor += 4;
        if (length < 0 || length > MaximumBodyLength) throw new InvalidDataException("字段长度无效");
        EnsureRemaining(body, cursor, length);
        var value = body.Slice(cursor, length);
        cursor += length;
        return value;
    }

    private static void EnsureRemaining(ReadOnlySpan<byte> body, int cursor, int required)
    {
        if (cursor < 0 || required < 0 || body.Length - cursor < required) throw new InvalidDataException("记录提前结束");
    }

    private static WalDataRecord CloneRecord(WalDataRecord record) => record with { Payload = record.Payload.ToArray() };

    private long GetMaximumSequence()
    {
        lock (_stateLock) return _maximumSequence;
    }

    private long ReservedBytesUnsafe() => checked((long)_pending.Count * AckRecordLength);

    private long DiagnosticUsageUnsafe() => _pending
        .Where(record => string.Equals(record.Subject, "alarm.event", StringComparison.Ordinal))
        .Sum(record => checked((long)record.FrameLength + AckRecordLength));

    private long RemainingDiagnosticReserveUnsafe() => Math.Max(0, _options.DiagnosticReserveBytes - DiagnosticUsageUnsafe());

    private long UsedCapacityUnsafe() => checked(_storedBytes + ReservedBytesUnsafe() + RemainingDiagnosticReserveUnsafe());

    private void AddDiagnostic(string code, string message, long offset)
    {
        lock (_stateLock) _diagnostics.Add(new WalDiagnostic(code, message, offset));
    }

    private void ThrowIfCorrupted()
    {
        lock (_stateLock)
        {
            ThrowIfUnavailableUnsafe();
        }
    }

    private void ThrowIfUnavailableUnsafe()
    {
        if (_isCorrupted || _stopping) throw new WalUnavailableException("WAL 已损坏或正在关闭，禁止继续写入或重放: " + (_lastError ?? "unknown"));
    }

    private static TaskCompletionSource CompletedSignal()
    {
        var signal = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        signal.SetResult();
        return signal;
    }

    private void EnterOperation()
    {
        lock (_stateLock)
        {
            ThrowIfUnavailableUnsafe();
            if (_activeOperations++ == 0) _operationsDrained = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        }
    }

    private void ExitOperation()
    {
        lock (_stateLock)
        {
            if (_activeOperations <= 0) return;
            _activeOperations--;
            if (_activeOperations == 0) _operationsDrained.TrySetResult();
        }
    }

    private void ReleaseReplayPermit()
    {
        lock (_stateLock) ReleaseReplayPermitUnsafe();
    }

    private void ReleaseReplayPermitUnsafe()
    {
        if (_replayInFlight <= 0) return;
        _replayInFlight--;
        if (_replayInFlight == 0) _replayDrained.TrySetResult();
    }

    /// <summary>
    /// 写入、刷盘或改名报错时无法证明介质实际完成到哪一步。当前进程继续写会把
    /// 未知尾部变成中段损坏，因此只能隔离实例并交给下次 Open 的恢复扫描裁决。
    /// </summary>
    private void FailCurrentInstance(string code, string message, Exception exception)
    {
        lock (_stateLock)
        {
            _isCorrupted = true;
            _lastError = code;
            _diagnostics.Add(new WalDiagnostic(code, message + ": " + exception.GetType().Name, -1));
        }
        _unavailableCancellation.Cancel();
    }

    private void FailClosed(string path, long offset, string code, string message)
    {
        IReadOnlyList<WalDiagnostic> diagnostics;
        lock (_stateLock)
        {
            _isCorrupted = true;
            _lastError = code;
            _diagnostics.Add(new WalDiagnostic(code, message, offset));
            diagnostics = _diagnostics.ToArray();
        }
        _unavailableCancellation.Cancel();

        var quarantineDirectory = Path.Combine(_options.DirectoryPath, "quarantine");
        if (File.Exists(path))
        {
            try
            {
                Directory.CreateDirectory(quarantineDirectory);
                // 与 WAL 同目录创建 quarantine，优先同卷 rename，避免 1GiB 损坏文件 CopyTo
                // 带来的同步长停顿和双倍磁盘占用。若部署把子目录挂到另一卷，拒绝回退复制，
                // 保留原文件并记录隔离失败，不能用“清理”掩盖根因。
                var target = Path.Combine(quarantineDirectory, Path.GetFileName(path) + "." + DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() + ".bad");
                if (_options.QuarantineMove is null) File.Move(path, target, overwrite: false);
                else _options.QuarantineMove(path, target);
            }
            catch (Exception quarantineException)
            {
                lock (_stateLock)
                {
                    _diagnostics.Add(new WalDiagnostic("WAL_QUARANTINE_FAILED", "隔离 rename 失败，原 WAL 保留: " + quarantineException.GetType().Name, offset));
                }
            }
        }
        lock (_stateLock) diagnostics = _diagnostics.ToArray();
        throw new WalCorruptionException(message, diagnostics);
    }

    private static uint Crc32C(ReadOnlySpan<byte> value)
    {
        var crc = 0xffffffffu;
        foreach (var item in value) crc = Crc32CTable[(crc ^ item) & 0xff] ^ (crc >> 8);
        return ~crc;
    }

    private static uint[] CreateCrc32CTable()
    {
        var table = new uint[256];
        for (uint index = 0; index < table.Length; index++)
        {
            var crc = index;
            for (var bit = 0; bit < 8; bit++) crc = (crc & 1) == 0 ? crc >> 1 : 0x82f63b78u ^ (crc >> 1);
            table[index] = crc;
        }
        return table;
    }

    private enum RecordType : byte
    {
        Data = 1,
        Ack = 2,
        Checkpoint = 3,
    }

    internal sealed record WalPendingInfo(string Subject, string EventId, string OwnerId, long Epoch, long Sequence, int PayloadLength, DateTimeOffset AppendedAt);

    internal sealed record WalPendingIndex(
        long FrameOffset,
        int FrameLength,
        int PayloadLength,
        int PayloadBodyOffset,
        string Subject,
        string EventId,
        string OwnerId,
        long Epoch,
        long Sequence,
        DateTimeOffset AppendedAt)
    {
        internal WalPendingInfo ToInfo() => new(Subject, EventId, OwnerId, Epoch, Sequence, PayloadLength, AppendedAt);
    }

    private sealed record RecoveryScan(IReadOnlyList<WalPendingIndex> Pending, long MaximumSequence, long LastCompleteOffset, long? TornTailOffset);

    internal sealed class WalReplayPermit : IDisposable
    {
        private readonly DurableWal _owner;
        private readonly CancellationTokenSource _cancellation;
        private int _disposed;

        internal WalReplayPermit(DurableWal owner, WalPendingIndex index, WalDataRecord record, Task publishTask, CancellationTokenSource cancellation)
        {
            _owner = owner;
            Index = index;
            Record = record;
            PublishTask = publishTask;
            _cancellation = cancellation;
        }

        internal WalDataRecord Record { get; }

        internal WalPendingIndex Index { get; }

        internal DurableWal Owner => _owner;

        internal bool IsActive => Volatile.Read(ref _disposed) == 0;

        internal Task PublishTask { get; }

        public void Dispose()
        {
            if (Interlocked.Exchange(ref _disposed, 1) != 0) return;
            _cancellation.Dispose();
            _owner.ReleaseReplayPermit();
        }
    }
}
