using System.Collections.Concurrent;
using System.Globalization;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;
using NATS.Client.Core;
using NATS.Client.JetStream;
using NATS.Client.JetStream.Models;
using NATS.Net;

namespace InduForge.Collector.Runtime.Tests;

public sealed class DurableWalTests
{
    private const string OwnerId = "collector-line1-a";
    private const string PointId = "22222222-2222-4222-8222-222222222222";
    private const string ConnectionId = "aaaaaaaa-1111-4111-8111-111111111111";
    private const string VariableId = "bbbbbbbb-1111-4111-8111-111111111111";
    private static readonly JsonSerializerOptions JsonOptions = new() { PropertyNamingPolicy = JsonNamingPolicy.CamelCase };

    [Fact]
    public async Task RecordWithoutPubAckIsRecoveredForReplay()
    {
        await using var directory = new TemporaryDirectory();
        WalDataRecord written;
        await using (var wal = await OpenAsync(directory.Path))
        {
            written = await AppendRawAsync(wal);
        }

        await using var recovered = await OpenAsync(directory.Path);
        var pending = Assert.Single(recovered.GetPendingRecords());
        Assert.Equal(written.Subject, pending.Subject);
        Assert.Equal(written.EventId, pending.EventId);
        // backlog 索引不常驻 payload；正文由 replay permit 单帧按需读取。
        Assert.Equal(written.Sequence, pending.Sequence);
    }

    [Fact]
    public async Task PubAckCommitIsRecoveredWithoutReplay()
    {
        await using var directory = new TemporaryDirectory();
        await using (var wal = await OpenAsync(directory.Path))
        {
            _ = await AppendRawAsync(wal);
            await AcknowledgeNextAsync(wal);
        }

        await using var recovered = await OpenAsync(directory.Path);
        Assert.Empty(recovered.GetPendingRecords());
    }

    [Fact]
    public async Task FailedPublishRetriesExactMsgIdAndPayload()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);
        var record = await AppendRawAsync(wal, paddingLength: 20);
        var publisher = new FailOncePublisher();
        var pump = new ReplayPump(wal, publisher);

        Assert.False(await pump.PumpOnceAsync(CancellationToken.None));
        Assert.Single(wal.GetPendingRecords()); // publish 未得到 PubAck，绝不能写 ACK。
        Assert.True(await pump.PumpOnceAsync(CancellationToken.None));

        Assert.Equal(2, publisher.Records.Count);
        Assert.Equal(record.EventId, publisher.Records[0].EventId);
        Assert.True(record.Payload.Span.SequenceEqual(publisher.Records[0].Payload.Span));
        Assert.True(publisher.Records[0].Payload.Span.SequenceEqual(publisher.Records[1].Payload.Span));
        Assert.Empty(wal.GetPendingRecords());
    }

    [Fact]
    public async Task RejectsInvalidCollectorV1ContractBeforeWriting()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);

        await Assert.ThrowsAnyAsync<ArgumentException>(() => wal.AppendDataAsync(sequence =>
            CreateRawRequest(sequence, eventId: "invalid", epoch: 0)));
        await Assert.ThrowsAsync<ArgumentException>(() => wal.AppendDataAsync(sequence =>
            CreateRawRequest(sequence, subject: "telemetry.line1")));
        await Assert.ThrowsAsync<ArgumentException>(() => wal.AppendDataAsync(sequence =>
            CreateRawRequest(sequence, sequenceInPayload: sequence + 1)));
        Assert.False(File.Exists(wal.StoragePath));
    }

    [Fact]
    public async Task RejectsNonCanonicalUuidFieldsRequiredByRuntimeSchemas()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);
        var raw = CreateRawRequest(1);
        var upperPoint = Mutate(raw, root => root["pointId"] = "AAAAAAAA-1111-4111-8111-111111111111");
        var stableConnection = Mutate(raw, root => root["source"]!["connectionId"] = "line1-plc");
        var wrongVariableType = Mutate(raw, root => root["source"]!["variableId"] = 1);
        var gap = CreateDataGapRequest(1);
        var upperGapConnection = Mutate(gap, root => root["dataGap"]!["connectionId"] = ConnectionId.ToUpperInvariant());

        await Assert.ThrowsAsync<ArgumentException>(() => wal.AppendDataAsync(_ => upperPoint));
        await Assert.ThrowsAsync<ArgumentException>(() => wal.AppendDataAsync(_ => stableConnection));
        await Assert.ThrowsAsync<ArgumentException>(() => wal.AppendDataAsync(_ => wrongVariableType));
        await Assert.ThrowsAsync<ArgumentException>(() => wal.AppendDataAsync(_ => upperGapConnection));
    }

    [Fact]
    public async Task ConfirmedLossBuildsAndDurablyReplaysFormalDataGap()
    {
        await using var directory = new TemporaryDirectory();
        var fact = CreateConfirmedLossFact();
        await using (var wal = await OpenAsync(directory.Path))
        {
            var appended = await wal.AppendConfirmedLossAsync(fact);
            Assert.True(appended.Accepted);
            Assert.Equal("alarm.event", appended.Record!.Subject);
        }

        await using var recovered = await OpenAsync(directory.Path);
        var publisher = new RecordingPublisher();
        Assert.True(await new ReplayPump(recovered, publisher).PumpOnceAsync(CancellationToken.None));
        Assert.Single(publisher.Records);
        Assert.Equal("alarm.event", publisher.Records[0].Subject);
    }

    [Fact]
    public async Task ConfirmedLossRejectsUnknownRangeAndFailsClosedOnWriteUncertainty()
    {
        await using var invalidDirectory = new TemporaryDirectory();
        await using var invalid = await OpenAsync(invalidDirectory.Path);
        var invalidFact = CreateConfirmedLossFact() with { ToSequence = 9, FromSequence = 10 };
        await Assert.ThrowsAsync<ArgumentOutOfRangeException>(() => invalid.AppendConfirmedLossAsync(invalidFact));

        await using var failureDirectory = new TemporaryDirectory();
        await using var failing = await DurableWal.OpenAsync(new DurableWalOptions(failureDirectory.Path, 1_000_000, 500_000, IoFaultInjector: phase =>
        {
            if (phase == WalIoPhase.DataFlushed) throw new IOException("模拟 DATA_GAP 刷盘不确定");
        }));
        await Assert.ThrowsAsync<IOException>(() => failing.AppendConfirmedLossAsync(CreateConfirmedLossFact()));
        Assert.True(failing.GetSnapshot().IsCorrupted);
    }

    [Fact]
    public async Task ConfirmedLossUsesDiagnosticReserveAndFullWalDoesNotClaimIt()
    {
        await using var reserveDirectory = new TemporaryDirectory();
        await using var reserveWal = await DurableWal.OpenAsync(new DurableWalOptions(reserveDirectory.Path, 12_000, 7_000, DiagnosticReserveBytes: 2_000));
        _ = await AppendRawAsync(reserveWal, paddingLength: 6_500);
        Assert.False((await reserveWal.AppendDataAsync(sequence => CreateRawRequest(sequence, paddingLength: 3_000))).Accepted);
        var gap = await reserveWal.AppendConfirmedLossAsync(CreateConfirmedLossFact());
        Assert.True(gap.Accepted);
        Assert.Equal("alarm.event", gap.Record!.Subject);

        await using var fullDirectory = new TemporaryDirectory();
        await using var fullWal = await DurableWal.OpenAsync(new DurableWalOptions(fullDirectory.Path, 1_000, 500, DiagnosticReserveBytes: 100));
        var before = fullWal.GetSnapshot().BacklogRecords;
        var rejected = await fullWal.AppendConfirmedLossAsync(CreateConfirmedLossFact());
        Assert.False(rejected.Accepted);
        Assert.Equal("WAL_BACKPRESSURE", rejected.RejectionCode);
        Assert.Equal(before, fullWal.GetSnapshot().BacklogRecords);
    }

    [Fact]
    public async Task RecoveryRejectsCrcValidAckThatSkipsQueueHead()
    {
        await using var directory = new TemporaryDirectory();
        string path;
        WalDataRecord second;
        await using (var wal = await OpenAsync(directory.Path))
        {
            _ = await AppendRawAsync(wal);
            second = await AppendRawAsync(wal);
            path = wal.StoragePath;
        }
        await AppendBytesAsync(path, BuildAck(second));
        var exception = await Assert.ThrowsAsync<WalCorruptionException>(() => OpenAsync(directory.Path));
        Assert.Contains(exception.Diagnostics, item => item.Code == "WAL_ACK_INVALID");
    }

    [Fact]
    public async Task RecoveryRejectsAckEventIdMismatchAndLegacyV1Ack()
    {
        await using var mismatchDirectory = new TemporaryDirectory();
        string mismatchPath;
        WalDataRecord first;
        await using (var wal = await OpenAsync(mismatchDirectory.Path))
        {
            first = await AppendRawAsync(wal);
            mismatchPath = wal.StoragePath;
        }
        await AppendBytesAsync(mismatchPath, BuildAck(first, new string('0', 64)));
        var mismatch = await Assert.ThrowsAsync<WalCorruptionException>(() => OpenAsync(mismatchDirectory.Path));
        Assert.Contains(mismatch.Diagnostics, item => item.Code == "WAL_ACK_INVALID");

        await using var legacyDirectory = new TemporaryDirectory();
        string legacyPath;
        WalDataRecord legacy;
        await using (var wal = await OpenAsync(legacyDirectory.Path))
        {
            legacy = await AppendRawAsync(wal);
            legacyPath = wal.StoragePath;
        }
        await AppendBytesAsync(legacyPath, BuildLegacyAck(legacy));
        var oldAck = await Assert.ThrowsAsync<WalCorruptionException>(() => OpenAsync(legacyDirectory.Path));
        Assert.Contains(oldAck.Diagnostics, item => item.Code == "WAL_ACK_CORRUPT");
    }

    [Fact]
    public async Task RecoveryRejectsAckAfterCheckpointInsteadOfTreatingItAsIdempotent()
    {
        await using var directory = new TemporaryDirectory();
        string path;
        WalDataRecord record;
        await using (var wal = await OpenAsync(directory.Path))
        {
            record = await AppendRawAsync(wal);
            Assert.True(await AcknowledgeNextAsync(wal)); // 空 backlog 会压缩为 checkpoint。
            path = wal.StoragePath;
        }
        await AppendBytesAsync(path, BuildAck(record));
        var exception = await Assert.ThrowsAsync<WalCorruptionException>(() => OpenAsync(directory.Path));
        Assert.Contains(exception.Diagnostics, item => item.Code == "WAL_ACK_INVALID");
    }

    [Fact]
    public async Task JetStreamPublisherIntegrationUsesMsgIdAndRejectsDuplicateWhenExplicitlyEnabled()
    {
        var url = Environment.GetEnvironmentVariable("INDUFORGE_NATS_INTEGRATION_URL");
        if (string.IsNullOrWhiteSpace(url)) return;

        var stream = "IFWAL" + Guid.NewGuid().ToString("N")[..16].ToUpperInvariant();
        await using var client = new NatsClient(new NatsOpts { Url = url });
        await client.ConnectAsync();
        var jetStream = client.CreateJetStreamContext();
        await jetStream.CreateStreamAsync(new StreamConfig { Name = stream, Subjects = ["data.raw." + PointId] });
        try
        {
            var request = CreateRawRequest(1);
            var record = new WalDataRecord(request.Subject, request.EventId, request.Payload, request.OwnerId, request.Epoch, 1, DateTimeOffset.UtcNow);
            var publisher = new NatsJetStreamPublisher(jetStream);
            await publisher.PublishAsync(record, CancellationToken.None);
            await publisher.PublishAsync(record, CancellationToken.None);
            var streamInfo = await jetStream.GetStreamAsync(stream);
            Assert.Equal(1L, streamInfo.Info.State.Messages);

            // 模拟“服务端已持久化，但本地尚未来得及写 ACK”：重放 duplicate PubAck
            // 必须被归一为成功，随后 durable ACK 使重开不再重投。
            await using var directory = new TemporaryDirectory();
            await using (var wal = await OpenAsync(directory.Path))
            {
                var appended = await wal.AppendDataAsync(_ => request);
                Assert.True(appended.Accepted);
                Assert.True(await new ReplayPump(wal, publisher).PumpOnceAsync(CancellationToken.None));
            }
            await using var recovered = await OpenAsync(directory.Path);
            Assert.Empty(recovered.GetPendingRecords());
            streamInfo = await jetStream.GetStreamAsync(stream);
            Assert.Equal(1L, streamInfo.Info.State.Messages);
        }
        finally
        {
            await jetStream.DeleteStreamAsync(stream);
        }
    }

    [Fact]
    public async Task AcceptsOnlyDataGapAlarmBranch()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);
        var accepted = await wal.AppendDataAsync(CreateDataGapRequest);
        Assert.True(accepted.Accepted);

        await Assert.ThrowsAsync<ArgumentException>(() => wal.AppendDataAsync(sequence => CreateDataGapRequest(sequence) with
        {
            Payload = JsonSerializer.SerializeToUtf8Bytes(new
            {
                schemaVersion = "alarm.event.v1",
                subject = "alarm.event",
                eventId = EventIdForDataGap(sequence),
                ownerId = OwnerId,
                epoch = 7,
                kind = "transition",
                operation = "OPENED",
            }, JsonOptions),
            EventId = EventIdForDataGap(sequence),
        }));
    }

    [Fact]
    public async Task StrictValidatorRejectsMissingExtraBadTimeQualityReasonAndDigest()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);
        var raw = CreateRawRequest(1);
        var missing = Mutate(raw, root => root.Remove("source"));
        var extra = Mutate(raw, root => root["unexpected"] = true);
        var badTime = Mutate(raw, root => root["receivedAt"] = "2026-08-30T10:20:30+08:00");
        var badQuality = Mutate(raw, root => root["quality"] = "stale");
        var badDigest = raw with { EventId = new string('0', 64), Payload = ReplaceEventId(raw.Payload, new string('0', 64)) };
        foreach (var request in new[] { missing, extra, badTime, badQuality, badDigest })
        {
            await Assert.ThrowsAsync<ArgumentException>(() => wal.AppendDataAsync(_ => request));
        }

        var gap = CreateDataGapRequest(1);
        var badReason = Mutate(gap, root => root["dataGap"]!["reason"] = "source-unavailable");
        var reverseRange = Mutate(gap, root => root["dataGap"]!["fromSequence"] = 1025);
        foreach (var request in new[] { badReason, reverseRange })
        {
            await Assert.ThrowsAsync<ArgumentException>(() => wal.AppendDataAsync(_ => request));
        }
    }

    [Theory]
    [InlineData("{\"schemaVersion\":\"data.raw.v1\",\"schemaVersion\":\"x\"}")]
    [InlineData("{\"source\":{\"collectorId\":\"a\",\"collectorId\":\"b\"}}")]
    [InlineData("{\"value\":[{\"x\":1,\"x\":2}]}")]
    public async Task DuplicatePropertiesAreRejectedAtEveryJsonObjectLevel(string payload)
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);
        var request = new WalAppendRequest("data.raw." + PointId, new string('a', 64), Encoding.UTF8.GetBytes(payload), OwnerId, 7);
        await Assert.ThrowsAsync<ArgumentException>(() => wal.AppendDataAsync(_ => request));
    }

    [Fact]
    public async Task UncertainDataOrAckWriteFailsCurrentInstanceAndRecoveryDecides()
    {
        await using var dataDirectory = new TemporaryDirectory();
        await using (var wal = await DurableWal.OpenAsync(new DurableWalOptions(dataDirectory.Path, 1_000_000, 500_000, IoFaultInjector: phase =>
        {
            if (phase == WalIoPhase.DataFlushed) throw new IOException("完整写后故障");
        })))
        {
            await Assert.ThrowsAsync<IOException>(() => wal.AppendDataAsync(sequence => CreateRawRequest(sequence)));
            Assert.True(wal.GetSnapshot().IsCorrupted);
            await Assert.ThrowsAnyAsync<InvalidOperationException>(() => wal.AppendDataAsync(sequence => CreateRawRequest(sequence)));
        }
        await using (var recovered = await OpenAsync(dataDirectory.Path))
        {
            Assert.Single(recovered.GetPendingRecords());
        }

        await using var ackDirectory = new TemporaryDirectory();
        await using (var wal = await DurableWal.OpenAsync(new DurableWalOptions(ackDirectory.Path, 1_000_000, 500_000, IoFaultInjector: phase =>
        {
            if (phase == WalIoPhase.AckFlushed) throw new IOException("确认写后故障");
        })))
        {
            _ = await AppendRawAsync(wal);
            Assert.False(await AcknowledgeNextAsync(wal));
            Assert.True(wal.GetSnapshot().IsCorrupted);
        }
        await using (var recovered = await OpenAsync(ackDirectory.Path))
        {
            Assert.Empty(recovered.GetPendingRecords());
        }
    }

    [Fact]
    public async Task OnlyOneProcessCanOpenWalAndCancellationStopsBeforeWriting()
    {
        await using var directory = new TemporaryDirectory();
        await using var first = await OpenAsync(directory.Path);
        await Assert.ThrowsAsync<IOException>(() => OpenAsync(directory.Path));
        using var cancellation = new CancellationTokenSource();
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => first.AppendDataAsync(sequence => CreateRawRequest(sequence), cancellation.Token));
        Assert.False(File.Exists(first.StoragePath));
    }

    [Fact]
    public async Task WriteBoundaryAndCompactionFaultsQuarantineCurrentInstance()
    {
        await using var writeDirectory = new TemporaryDirectory();
        await using (var wal = await DurableWal.OpenAsync(new DurableWalOptions(writeDirectory.Path, 1_000_000, 500_000, IoFaultInjector: phase =>
        {
            if (phase == WalIoPhase.DataWritten) throw new IOException("模拟短写边界故障");
        })))
        {
            await Assert.ThrowsAsync<IOException>(() => wal.AppendDataAsync(sequence => CreateRawRequest(sequence)));
            Assert.True(wal.GetSnapshot().IsCorrupted);
        }

        await using var compactDirectory = new TemporaryDirectory();
        await using (var wal = await DurableWal.OpenAsync(new DurableWalOptions(compactDirectory.Path, 1_000_000, 500_000, IoFaultInjector: phase =>
        {
            if (phase == WalIoPhase.TemporaryFlushed) throw new IOException("模拟压缩临界故障");
        })))
        {
            _ = await AppendRawAsync(wal);
            await Assert.ThrowsAsync<IOException>(() => wal.CompactAsync());
            Assert.True(wal.GetSnapshot().IsCorrupted);
        }
        await using var recovered = await OpenAsync(compactDirectory.Path);
        Assert.Single(recovered.GetPendingRecords());
    }

    [Fact]
    public async Task FaultedWalNeverPublishesAndReplayRunExitsWithOriginalError()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(directory.Path, 1_000_000, 500_000, IoFaultInjector: phase =>
        {
            if (phase == WalIoPhase.AckFlushed) throw new IOException("确认后故障");
        }));
        _ = await AppendRawAsync(wal);
        Assert.False(await AcknowledgeNextAsync(wal));
        wal.RecordPublishFailure(new IOException("不得覆盖隔离原因"));
        var publisher = new RecordingPublisher();
        var pump = new ReplayPump(wal, publisher);

        await Assert.ThrowsAsync<WalUnavailableException>(() => pump.PumpOnceAsync(CancellationToken.None));
        await Assert.ThrowsAsync<WalUnavailableException>(() => pump.RunAsync(CancellationToken.None));
        Assert.Empty(publisher.Records);
        Assert.Equal("WAL_IO_UNCERTAIN", wal.GetSnapshot().LastError);
    }

    [Fact]
    public async Task PublishThatStartsBeforeFaultIsCancelledAndNoSecondPublishStarts()
    {
        await using var directory = new TemporaryDirectory();
        var injectFault = false;
        await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(directory.Path, 1_000_000, 500_000, IoFaultInjector: phase =>
        {
            if (injectFault && phase == WalIoPhase.DataFlushed) throw new IOException("发布在飞时写入故障");
        }));
        _ = await AppendRawAsync(wal);
        var publisher = new CancellationAwarePublisher();
        var pump = new ReplayPump(wal, publisher);
        var running = pump.PumpOnceAsync(CancellationToken.None);
        await publisher.Started.Task.WaitAsync(TimeSpan.FromSeconds(2));

        injectFault = true;
        await Assert.ThrowsAsync<IOException>(() => wal.AppendDataAsync(sequence => CreateRawRequest(sequence)));
        await Assert.ThrowsAsync<WalUnavailableException>(() => running);
        Assert.True(publisher.CancellationObserved);
        Assert.Equal(1, publisher.PublishCount);
        await Assert.ThrowsAsync<WalUnavailableException>(() => pump.PumpOnceAsync(CancellationToken.None));
        Assert.Equal(1, publisher.PublishCount);
    }

    [Fact]
    public async Task NewNestedWalDirectoryFlushesParentsInsideOut()
    {
        await using var directory = new TemporaryDirectory();
        var nested = Path.Combine(directory.Path, "one", "two");
        var flushed = new List<string>();
        await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(nested, 1_000_000, 500_000, DirectoryFlushObserver: flushed.Add));

        Assert.Equal([Path.Combine(directory.Path, "one"), directory.Path], flushed);
        Assert.False(File.Exists(wal.StoragePath));
    }

    [Fact]
    public void UnsupportedDirectoryDurabilityPlatformsFailClosed()
    {
        DurableStorage.EnsurePlatformSupported(isWindows: false, isUnix: true);
        Assert.Throws<PlatformNotSupportedException>(() => DurableStorage.EnsurePlatformSupported(isWindows: true, isUnix: false));
        Assert.Throws<PlatformNotSupportedException>(() => DurableStorage.EnsurePlatformSupported(isWindows: false, isUnix: false));
    }

    [Fact]
    public async Task ExistingDirectoryCannotBypassWindowsDurabilityFailClosed()
    {
        await using var directory = new TemporaryDirectory();
        Assert.Throws<PlatformNotSupportedException>(() => DurableStorage.CreateDirectoryDurably(directory.Path, isWindows: true, isUnix: false));
    }

    [Fact]
    public async Task TornTailIsTruncatedButCompleteBadHeaderFailsClosed()
    {
        await using var directory = new TemporaryDirectory();
        string walPath;
        await using (var wal = await OpenAsync(directory.Path))
        {
            _ = await AppendRawAsync(wal);
            walPath = wal.StoragePath;
        }
        await AppendBytesAsync(walPath, [0x49, 0x46, 0x57]);
        await using (var recovered = await OpenAsync(directory.Path))
        {
            Assert.Single(recovered.GetPendingRecords());
            Assert.Contains(recovered.Diagnostics, item => item.Code == "WAL_TORN_TAIL_TRUNCATED");
        }

        var bytes = await File.ReadAllBytesAsync(walPath);
        bytes[6] ^= 0x01; // 篡改末条长度且不更新独立 header CRC。
        await File.WriteAllBytesAsync(walPath, bytes);
        var exception = await Assert.ThrowsAsync<WalCorruptionException>(() => OpenAsync(directory.Path));
        Assert.Contains(exception.Diagnostics, item => item.Code == "WAL_HEADER_CRC_CORRUPT");
    }

    [Fact]
    public async Task MidFileCrcCorruptionFailsClosedAndQuarantines()
    {
        await using var directory = new TemporaryDirectory();
        string walPath;
        await using (var wal = await OpenAsync(directory.Path))
        {
            _ = await AppendRawAsync(wal);
            walPath = wal.StoragePath;
        }
        var bytes = await File.ReadAllBytesAsync(walPath);
        bytes[20] ^= 0x7f;
        await File.WriteAllBytesAsync(walPath, bytes);

        var exception = await Assert.ThrowsAsync<WalCorruptionException>(() => OpenAsync(directory.Path));
        Assert.Contains(exception.Diagnostics, item => item.Code == "WAL_CRC_CORRUPT");
        var quarantined = Assert.Single(Directory.EnumerateFiles(Path.Combine(directory.Path, "quarantine")));
        Assert.False(File.Exists(walPath)); // 隔离为同卷 rename，不复制损坏 WAL 占用双倍空间。
        Assert.Equal(bytes.Length, new FileInfo(quarantined).Length);
    }

    [Fact]
    public async Task QuarantineRenameFailureKeepsRootCauseAndOriginalWal()
    {
        await using var directory = new TemporaryDirectory();
        string path;
        await using (var wal = await OpenAsync(directory.Path))
        {
            _ = await AppendRawAsync(wal);
            path = wal.StoragePath;
        }
        var bytes = await File.ReadAllBytesAsync(path);
        bytes[20] ^= 0x7f;
        await File.WriteAllBytesAsync(path, bytes);

        var options = new DurableWalOptions(directory.Path, 1_000_000, 500_000, QuarantineMove: (_, _) => throw new IOException("模拟跨卷 rename 失败"));
        var exception = await Assert.ThrowsAsync<WalCorruptionException>(() => DurableWal.OpenAsync(options));
        Assert.Contains(exception.Diagnostics, item => item.Code == "WAL_CRC_CORRUPT");
        Assert.Contains(exception.Diagnostics, item => item.Code == "WAL_QUARANTINE_FAILED");
        Assert.True(File.Exists(path));
    }

    [Fact]
    public async Task CapacityReservesAckSpaceAndCommittedAckIsNeverRejected()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(directory.Path, 12_000, 2_000));
        _ = await AppendRawAsync(wal, paddingLength: 5_000);
        var snapshot = wal.GetSnapshot();
        Assert.True(snapshot.ReservedBytes > 0);
        Assert.Equal(snapshot.StoredBytes + snapshot.ReservedBytes + snapshot.AvailableDiagnosticBytes, snapshot.UsedCapacityBytes);
        Assert.True(snapshot.IsDegraded);

        var rejected = await wal.AppendDataAsync(sequence => CreateRawRequest(sequence, paddingLength: 9_000));
        Assert.False(rejected.Accepted);
        Assert.Equal("WAL_BACKPRESSURE", rejected.RejectionCode);
        Assert.True(await AcknowledgeNextAsync(wal));
        Assert.Empty(wal.GetPendingRecords());
    }

    [Fact]
    public async Task DiagnosticReserveLetsDataGapPassRawBackpressureAndIsReclaimed()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(directory.Path, 12_000, 7_000, DiagnosticReserveBytes: 2_000));
        _ = await AppendRawAsync(wal, paddingLength: 6_500);
        var rawRejected = await wal.AppendDataAsync(sequence => CreateRawRequest(sequence, paddingLength: 3_000));
        Assert.False(rawRejected.Accepted);
        Assert.True(wal.GetSnapshot().IsRawBackpressured);
        Assert.True(wal.GetSnapshot().AvailableDiagnosticBytes > 0);

        var diagnostic = await wal.AppendDataAsync(CreateDataGapRequest);
        Assert.True(diagnostic.Accepted);
        Assert.True(await AcknowledgeNextAsync(wal));
        var publisher = new RecordingPublisher();
        var pump = new ReplayPump(wal, publisher);
        Assert.True(await pump.PumpOnceAsync(CancellationToken.None));
        Assert.Single(publisher.Records);
        Assert.Equal("alarm.event", publisher.Records[0].Subject);
        Assert.Equal(2_000, wal.GetSnapshot().AvailableDiagnosticBytes);

        var nextDiagnostic = await wal.AppendDataAsync(CreateDataGapRequest);
        Assert.True(nextDiagnostic.Accepted);
        Assert.True(await AcknowledgeNextAsync(wal));

        await using var tinyDirectory = new TemporaryDirectory();
        await using var tiny = await DurableWal.OpenAsync(new DurableWalOptions(tinyDirectory.Path, 1_000_000, 500_000, DiagnosticReserveBytes: 100));
        var oversized = await tiny.AppendDataAsync(CreateDataGapRequest);
        Assert.False(oversized.Accepted);
        Assert.Equal("WAL_BACKPRESSURE", oversized.RejectionCode);
        Assert.False(tiny.GetSnapshot().IsRawBackpressured);
    }

    [Fact]
    public async Task DiagnosticReserveIsRebuiltAfterRecovery()
    {
        await using var directory = new TemporaryDirectory();
        var options = new DurableWalOptions(directory.Path, 12_000, 7_000, DiagnosticReserveBytes: 2_000);
        await using (var wal = await DurableWal.OpenAsync(options))
        {
            _ = await wal.AppendDataAsync(CreateDataGapRequest);
            Assert.True(await AcknowledgeNextAsync(wal));
        }

        await using var recovered = await DurableWal.OpenAsync(options);
        Assert.Equal(2_000, recovered.GetSnapshot().AvailableDiagnosticBytes);
        Assert.True((await recovered.AppendDataAsync(CreateDataGapRequest)).Accepted);
    }

    [Fact]
    public async Task CompactionPreservesMaximumSequenceAcrossRecovery()
    {
        await using var directory = new TemporaryDirectory();
        await using (var wal = await OpenAsync(directory.Path))
        {
            _ = await AppendRawAsync(wal, epoch: 7);
            _ = await AppendRawAsync(wal, epoch: 8);
            Assert.True(await AcknowledgeNextAsync(wal));
            Assert.True(await AcknowledgeNextAsync(wal));
        }

        await using var recovered = await OpenAsync(directory.Path);
        var next = await AppendRawAsync(recovered, epoch: 9);
        Assert.Equal(3, next.Sequence);
    }

    [Fact]
    public async Task FirstWalCreationAndCheckpointRecoveryRemainDurable()
    {
        await using var directory = new TemporaryDirectory();
        await using (var wal = await OpenAsync(directory.Path))
        {
            Assert.False(File.Exists(wal.StoragePath));
            _ = await AppendRawAsync(wal);
            Assert.True(File.Exists(wal.StoragePath));
            Assert.True(await AcknowledgeNextAsync(wal)); // 空 pending 会写内部 checkpoint 并压缩 ACK。
        }

        await using var recovered = await OpenAsync(directory.Path);
        var next = await AppendRawAsync(recovered);
        Assert.Equal(2, next.Sequence);
    }

    [Fact]
    public async Task PreviousFileIsValidatedAndRestoredAfterInterruptedCompaction()
    {
        await using var directory = new TemporaryDirectory();
        var options = new DurableWalOptions(directory.Path, 1_000_000, 500_000, stage =>
        {
            if (stage == WalCompactionStage.PreviousDurable) throw new IOException("模拟断电");
        });
        await using (var wal = await DurableWal.OpenAsync(options))
        {
            _ = await AppendRawAsync(wal);
            await Assert.ThrowsAsync<IOException>(() => wal.CompactAsync());
            Assert.False(File.Exists(wal.StoragePath));
        }

        await using var recovered = await OpenAsync(directory.Path);
        Assert.Single(recovered.GetPendingRecords());
        Assert.Contains(recovered.Diagnostics, item => item.Code == "WAL_PREVIOUS_RESTORED");
    }

    [Fact]
    public async Task CorruptPrimaryNeverFallsBackToPrevious()
    {
        await using var directory = new TemporaryDirectory();
        var options = new DurableWalOptions(directory.Path, 1_000_000, 500_000, stage =>
        {
            if (stage == WalCompactionStage.PreviousDurable) throw new IOException("模拟断电");
        });
        string primaryPath;
        await using (var wal = await DurableWal.OpenAsync(options))
        {
            _ = await AppendRawAsync(wal);
            primaryPath = wal.StoragePath;
            await Assert.ThrowsAsync<IOException>(() => wal.CompactAsync());
        }

        var previousPath = primaryPath + ".previous";
        File.Copy(previousPath, primaryPath);
        var corrupt = await File.ReadAllBytesAsync(primaryPath);
        corrupt[20] ^= 0x20;
        await File.WriteAllBytesAsync(primaryPath, corrupt);

        var exception = await Assert.ThrowsAsync<WalCorruptionException>(() => OpenAsync(directory.Path));
        Assert.Contains(exception.Diagnostics, item => item.Code == "WAL_CRC_CORRUPT");
        Assert.True(File.Exists(previousPath));
    }

    [Fact]
    public async Task DuplicateOrOrphanAckFailsClosed()
    {
        await using var directory = new TemporaryDirectory();
        string walPath;
        await using (var wal = await OpenAsync(directory.Path))
        {
            _ = await AppendRawAsync(wal);
            _ = await AppendRawAsync(wal);
            Assert.True(await AcknowledgeNextAsync(wal));
            walPath = wal.StoragePath;
        }
        var bytes = await File.ReadAllBytesAsync(walPath);
        await AppendBytesAsync(walPath, bytes[^98..]); // v2 ACK: header + epoch/sequence/eventId + footer。

        var exception = await Assert.ThrowsAsync<WalCorruptionException>(() => OpenAsync(directory.Path));
        Assert.Contains(exception.Diagnostics, item => item.Code == "WAL_ACK_INVALID");
    }

    [Fact]
    public async Task ReplayPumpHonorsCancellation()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);
        var pump = new ReplayPump(wal, new FailOncePublisher());
        using var cancellation = new CancellationTokenSource();
        var running = pump.RunAsync(cancellation.Token);
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => running);
    }

    [Fact]
    public async Task ConcurrentAppendSnapshotAndReplayRemainOrdered()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);
        var publisher = new RecordingPublisher();
        var pump = new ReplayPump(wal, publisher, new ReplayPumpOptions(TimeSpan.FromMilliseconds(1), TimeSpan.FromMilliseconds(1), TimeSpan.FromMilliseconds(10)));
        using var cancellation = new CancellationTokenSource();
        var replay = pump.RunAsync(cancellation.Token);
        var snapshots = Task.Run(() =>
        {
            for (var index = 0; index < 500; index++)
            {
                _ = wal.GetSnapshot();
                _ = wal.GetPendingRecords();
                _ = wal.Diagnostics;
            }
        });
        var appenders = Enumerable.Range(0, 4).Select(_ => Task.Run(async () =>
        {
            for (var index = 0; index < 15; index++)
            {
                var result = await wal.AppendDataAsync(sequence => CreateRawRequest(sequence));
                Assert.True(result.Accepted);
            }
        }));

        await Task.WhenAll(appenders.Append(snapshots));
        var deadline = DateTimeOffset.UtcNow + TimeSpan.FromSeconds(5);
        while (wal.GetSnapshot().BacklogRecords != 0 && DateTimeOffset.UtcNow < deadline) await Task.Delay(10);
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => replay);
        Assert.Equal(60, publisher.Records.Length);
        Assert.Equal(Enumerable.Range(1, 60).Select(value => (long)value), publisher.Records.Select(record => record.Sequence));
    }

    [Fact]
    public async Task OversizedFrameIsRejectedBeforeIoAndDoesNotConsumeSequence()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);
        var before = File.Exists(wal.StoragePath) ? new FileInfo(wal.StoragePath).Length : 0;

        await Assert.ThrowsAsync<ArgumentOutOfRangeException>(() =>
            wal.AppendDataAsync(sequence => CreateRawRequest(sequence, (64 * 1024 * 1024) + 1)));

        var after = File.Exists(wal.StoragePath) ? new FileInfo(wal.StoragePath).Length : 0;
        Assert.Equal(before, after);
        var valid = await AppendRawAsync(wal);
        Assert.Equal(1, valid.Sequence);
    }

    [Fact]
    public async Task ConcurrentReplayPumpsStartExactlyOnePublish()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);
        _ = await AppendRawAsync(wal);
        var publisher = new BlockingPublisher();
        var first = new ReplayPump(wal, publisher);
        var second = new ReplayPump(wal, publisher);

        var firstPump = first.PumpOnceAsync(CancellationToken.None);
        await publisher.Started.Task.WaitAsync(TimeSpan.FromSeconds(2));
        Assert.False(await second.PumpOnceAsync(CancellationToken.None));
        Assert.Equal(1, publisher.PublishCount);
        publisher.Release();
        Assert.True(await firstPump);
        Assert.Empty(wal.GetPendingRecords());
    }

    [Fact]
    public async Task DisposeWaitsForInFlightReplayBeforeReleasingProcessLock()
    {
        await using var directory = new TemporaryDirectory();
        var wal = await OpenAsync(directory.Path);
        _ = await AppendRawAsync(wal);
        var publisher = new BlockingPublisher();
        var pump = new ReplayPump(wal, publisher);
        var replay = pump.PumpOnceAsync(CancellationToken.None);
        await publisher.Started.Task.WaitAsync(TimeSpan.FromSeconds(2));

        var disposeOne = wal.DisposeAsync().AsTask();
        var disposeTwo = wal.DisposeAsync().AsTask();
        await Assert.ThrowsAsync<IOException>(() => OpenAsync(directory.Path));
        Assert.False(disposeOne.IsCompleted);
        publisher.Release();
        await Assert.ThrowsAsync<WalUnavailableException>(() => replay);
        await Task.WhenAll(disposeOne, disposeTwo);
        await using var reopened = await OpenAsync(directory.Path);
    }

    [Fact]
    public async Task BacklogIndexDoesNotExposeOrRetainPayloadAndCompactionRebuildsOffsets()
    {
        await using var directory = new TemporaryDirectory();
        await using var wal = await OpenAsync(directory.Path);
        _ = await AppendRawAsync(wal, 300 * 1024);
        _ = await AppendRawAsync(wal, 300 * 1024);
        var indexType = typeof(DurableWal).GetNestedType("WalPendingIndex", System.Reflection.BindingFlags.NonPublic);
        Assert.NotNull(indexType);
        Assert.DoesNotContain(indexType!.GetProperties(), property => property.Name.Contains("Payload", StringComparison.Ordinal) && property.PropertyType == typeof(ReadOnlyMemory<byte>));

        Assert.True(await AcknowledgeNextAsync(wal));
        await wal.CompactAsync();
        var publisher = new RecordingPublisher();
        var pump = new ReplayPump(wal, publisher);
        Assert.True(await pump.PumpOnceAsync(CancellationToken.None));
        Assert.Single(publisher.Records);
        Assert.Equal(2, publisher.Records[0].Sequence);
    }

    [Fact]
    public async Task CompactionWritesOneFrameAtATime()
    {
        await using var directory = new TemporaryDirectory();
        var inFlight = 0;
        var maximum = 0;
        await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(
            directory.Path,
            1_000_000,
            500_000,
            CompactionFrameObserver: value =>
            {
                _ = Interlocked.Exchange(ref inFlight, value);
                maximum = Math.Max(maximum, value);
            }));
        _ = await AppendRawAsync(wal, 200 * 1024);
        _ = await AppendRawAsync(wal, 200 * 1024);
        await wal.CompactAsync();
        Assert.Equal(1, maximum);
        Assert.Equal(0, Volatile.Read(ref inFlight));
    }

    private static Task<DurableWal> OpenAsync(string path) =>
        DurableWal.OpenAsync(new DurableWalOptions(path, 1_000_000, 500_000));

    private static Task<bool> AcknowledgeNextAsync(DurableWal wal) =>
        new ReplayPump(wal, new RecordingPublisher()).PumpOnceAsync(CancellationToken.None);

    private static async Task<WalDataRecord> AppendRawAsync(DurableWal wal, int paddingLength = 0, long epoch = 7)
    {
        var result = await wal.AppendDataAsync(sequence => CreateRawRequest(sequence, paddingLength, epoch));
        Assert.True(result.Accepted);
        return Assert.IsType<WalDataRecord>(result.Record);
    }

    private static WalAppendRequest CreateRawRequest(long sequence, int paddingLength = 0, long epoch = 7, long? sequenceInPayload = null, string? subject = null, string? eventId = null)
    {
        subject ??= "data.raw." + PointId;
        eventId ??= EventIdForRaw(sequence, epoch);
        var payload = JsonSerializer.SerializeToUtf8Bytes(new
        {
            schemaVersion = "data.raw.v1",
            subject,
            eventId,
            deploymentId = "deployment-line1-prod",
            accountId = "account-deployment-line1-prod",
            ownerId = OwnerId,
            epoch,
            pointId = PointId,
            sequence = sequenceInPayload ?? sequence,
            value = paddingLength == 0 ? (object)36.5 : new string('x', paddingLength),
            quality = "good",
            sourceTimestamp = "2026-08-30T10:20:30.123Z",
            serverTimestamp = "2026-08-30T10:20:30.150Z",
            receivedAt = "2026-08-30T10:20:30.160Z",
            source = new { collectorId = OwnerId, connectionId = ConnectionId, variableId = VariableId },
        }, JsonOptions);
        return new WalAppendRequest(subject, eventId, payload, OwnerId, epoch);
    }

    private static WalAppendRequest CreateDataGapRequest(long sequence)
    {
        const string subject = "alarm.event";
        var eventId = EventIdForDataGap(sequence);
        var payload = JsonSerializer.SerializeToUtf8Bytes(new
        {
            schemaVersion = "alarm.event.v1",
            subject,
            eventId,
            deploymentId = "deployment-line1-prod",
            accountId = "account-deployment-line1-prod",
            ownerId = OwnerId,
            epoch = 7,
            kind = "data-gap",
            operation = "DATA_GAP",
            sourceTimestamp = "2026-08-30T10:20:30Z",
            serverTimestamp = "2026-08-30T10:20:31Z",
            receivedAt = "2026-08-30T10:20:31Z",
            dataGap = new { collectorId = OwnerId, connectionId = ConnectionId, fromSequence = 1000, toSequence = 1024, fromSourceTimestamp = "2026-08-30T10:00:00Z", toSourceTimestamp = "2026-08-30T10:20:30Z", detectedAt = "2026-08-30T10:20:31Z", reason = "wal-capacity" },
        }, JsonOptions);
        return new WalAppendRequest(subject, eventId, payload, OwnerId, 7);
    }

    private static ConfirmedLossFact CreateConfirmedLossFact() => new(
        "deployment-line1-prod",
        "account-deployment-line1-prod",
        OwnerId,
        OwnerId,
        7,
        ConnectionId,
        1000,
        1024,
        DateTimeOffset.Parse("2026-08-30T10:00:00Z", CultureInfo.InvariantCulture),
        DateTimeOffset.Parse("2026-08-30T10:20:30Z", CultureInfo.InvariantCulture),
        DateTimeOffset.Parse("2026-08-30T10:20:31Z", CultureInfo.InvariantCulture),
        ConfirmedLossReason.WalCapacity);

    private static byte[] BuildAck(WalDataRecord record, string? eventId = null)
    {
        var body = new byte[80];
        System.Buffers.Binary.BinaryPrimitives.WriteInt64LittleEndian(body.AsSpan(0, 8), record.Epoch);
        System.Buffers.Binary.BinaryPrimitives.WriteInt64LittleEndian(body.AsSpan(8, 8), record.Sequence);
        Encoding.ASCII.GetBytes(eventId ?? record.EventId, body.AsSpan(16));
        var frame = new byte[14 + body.Length + 4];
        System.Buffers.Binary.BinaryPrimitives.WriteUInt32LittleEndian(frame.AsSpan(0, 4), 0x4C574649);
        frame[4] = 2;
        frame[5] = 2;
        System.Buffers.Binary.BinaryPrimitives.WriteInt32LittleEndian(frame.AsSpan(6, 4), body.Length);
        System.Buffers.Binary.BinaryPrimitives.WriteUInt32LittleEndian(frame.AsSpan(10, 4), Crc32C(frame.AsSpan(0, 10)));
        body.CopyTo(frame, 14);
        System.Buffers.Binary.BinaryPrimitives.WriteUInt32LittleEndian(frame.AsSpan(94, 4), Crc32C(body));
        return frame;
    }

    private static byte[] BuildLegacyAck(WalDataRecord record)
    {
        var body = new byte[16];
        System.Buffers.Binary.BinaryPrimitives.WriteInt64LittleEndian(body.AsSpan(0, 8), record.Epoch);
        System.Buffers.Binary.BinaryPrimitives.WriteInt64LittleEndian(body.AsSpan(8, 8), record.Sequence);
        var frame = new byte[14 + body.Length + 4];
        System.Buffers.Binary.BinaryPrimitives.WriteUInt32LittleEndian(frame.AsSpan(0, 4), 0x4C574649);
        frame[4] = 2;
        frame[5] = 2;
        System.Buffers.Binary.BinaryPrimitives.WriteInt32LittleEndian(frame.AsSpan(6, 4), body.Length);
        System.Buffers.Binary.BinaryPrimitives.WriteUInt32LittleEndian(frame.AsSpan(10, 4), Crc32C(frame.AsSpan(0, 10)));
        body.CopyTo(frame, 14);
        System.Buffers.Binary.BinaryPrimitives.WriteUInt32LittleEndian(frame.AsSpan(30, 4), Crc32C(body));
        return frame;
    }

    private static uint Crc32C(ReadOnlySpan<byte> value)
    {
        var crc = 0xffffffffu;
        foreach (var item in value)
        {
            crc ^= item;
            for (var bit = 0; bit < 8; bit++) crc = (crc & 1) == 0 ? crc >> 1 : 0x82f63b78u ^ (crc >> 1);
        }
        return ~crc;
    }

    private static string EventIdForRaw(long sequence, long epoch) => Hash("data.raw.v1", "deployment-line1-prod", PointId, OwnerId, epoch.ToString(CultureInfo.InvariantCulture), sequence.ToString(CultureInfo.InvariantCulture));

    private static string EventIdForDataGap(long sequence) => Hash("alarm.event.v1", "data-gap", "deployment-line1-prod", OwnerId, ConnectionId, OwnerId, "7", "1000", "1024", "2026-08-30T10:20:31Z", "wal-capacity");

    private static string Hash(params string[] fields) =>
        Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes(string.Join('\x1f', fields)))).ToLowerInvariant();

    private static WalAppendRequest Mutate(WalAppendRequest request, Action<JsonObject> mutate)
    {
        var root = JsonNode.Parse(Encoding.UTF8.GetString(request.Payload.Span))?.AsObject() ?? throw new InvalidOperationException();
        mutate(root);
        return request with { Payload = JsonSerializer.SerializeToUtf8Bytes(root, JsonOptions) };
    }

    private static ReadOnlyMemory<byte> ReplaceEventId(ReadOnlyMemory<byte> payload, string eventId)
    {
        var root = JsonNode.Parse(Encoding.UTF8.GetString(payload.Span))?.AsObject() ?? throw new InvalidOperationException();
        root["eventId"] = eventId;
        return JsonSerializer.SerializeToUtf8Bytes(root, JsonOptions);
    }

    private static async Task AppendBytesAsync(string path, byte[] bytes)
    {
        await using var stream = new FileStream(path, FileMode.Append, FileAccess.Write, FileShare.None, 4096, FileOptions.WriteThrough);
        await stream.WriteAsync(bytes);
        await stream.FlushAsync();
        stream.Flush(flushToDisk: true);
    }

    private sealed class FailOncePublisher : IJetStreamPublisher
    {
        public List<WalDataRecord> Records { get; } = [];

        public Task PublishAsync(WalDataRecord record, CancellationToken cancellationToken)
        {
            Records.Add(record with { Payload = record.Payload.ToArray() });
            if (Records.Count == 1) throw new IOException("temporary publish failure");
            return Task.CompletedTask;
        }
    }

    private sealed class RecordingPublisher : IJetStreamPublisher
    {
        public ConcurrentQueue<WalDataRecord> Queue { get; } = new();
        public WalDataRecord[] Records => Queue.ToArray();

        public Task PublishAsync(WalDataRecord record, CancellationToken cancellationToken)
        {
            Queue.Enqueue(record with { Payload = record.Payload.ToArray() });
            return Task.CompletedTask;
        }
    }

    private sealed class CancellationAwarePublisher : IJetStreamPublisher
    {
        private int _publishCount;

        public TaskCompletionSource<bool> Started { get; } = new(TaskCreationOptions.RunContinuationsAsynchronously);

        public bool CancellationObserved { get; private set; }

        public int PublishCount => Volatile.Read(ref _publishCount);

        public async Task PublishAsync(WalDataRecord record, CancellationToken cancellationToken)
        {
            Interlocked.Increment(ref _publishCount);
            Started.TrySetResult(true);
            try
            {
                await Task.Delay(Timeout.InfiniteTimeSpan, cancellationToken);
            }
            catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
            {
                CancellationObserved = true;
                throw;
            }
        }
    }

    private sealed class BlockingPublisher : IJetStreamPublisher
    {
        private int _publishCount;
        private readonly TaskCompletionSource _release = new(TaskCreationOptions.RunContinuationsAsynchronously);
        public TaskCompletionSource<bool> Started { get; } = new(TaskCreationOptions.RunContinuationsAsynchronously);
        public int PublishCount => Volatile.Read(ref _publishCount);

        public async Task PublishAsync(WalDataRecord record, CancellationToken cancellationToken)
        {
            Interlocked.Increment(ref _publishCount);
            Started.TrySetResult(true);
            await _release.Task.ConfigureAwait(false);
        }

        public void Release() => _release.TrySetResult();
    }

    private sealed class TemporaryDirectory : IAsyncDisposable
    {
        public TemporaryDirectory()
        {
            Path = System.IO.Path.Combine(System.IO.Path.GetTempPath(), "induforge-runtime-tests", Guid.NewGuid().ToString("N"));
            Directory.CreateDirectory(Path);
        }

        public string Path { get; }

        public ValueTask DisposeAsync()
        {
            if (Directory.Exists(Path)) Directory.Delete(Path, recursive: true);
            return ValueTask.CompletedTask;
        }
    }
}
