namespace InduForge.Collector.Runtime.Tests;

using System.Net;
using System.Net.Sockets;
using System.Text.Json;

public sealed class CollectorRuntimeV1Tests
{
    [Fact]
    public void ValueNormalizerRejectsLossyIntegerAndUnspecifiedDateTime()
    {
        Assert.Throws<InvalidCastException>(() => CollectorValueNormalizer.Normalize(1.5d, "int32"));
        Assert.Throws<InvalidCastException>(() => CollectorValueNormalizer.Normalize(DateTime.SpecifyKind(DateTime.UnixEpoch, DateTimeKind.Unspecified), "datetime"));
    }

    [Fact]
    public void ProgramOptionsAcceptOnlyReferencesAndPaths()
    {
        var options = Program.ParseOptions(["--artifact", "/trusted/artifact.json", "--binding", "/trusted/binding.json", "--index", "/trusted/index.json", "--wal", "/trusted/wal", "--listen", "http://127.0.0.1:18081/", "--site-id", "site-a", "--production", "true"]);
        Assert.True(options.Production);
        Assert.Equal("site-a", options.SiteId);
        Assert.Throws<CollectorRuntimeConfigurationException>(() => Program.ParseOptions(["--unknown", "x", "--binding", "x", "--index", "x", "--wal", "x", "--listen", "x", "--production", "false"]));
    }

    [Fact]
    public void NormalizerRequiresExactModbusArrayShape()
    {
        var normalized = CollectorValueNormalizer.Normalize(new ushort[] { 1, 2 }, "uint16", 2);
        Assert.Equal("[1,2]", normalized.GetRawText());
        Assert.Throws<InvalidCastException>(() => CollectorValueNormalizer.Normalize(new ushort[] { 1 }, "uint16", 2));
    }

    [Fact]
    public void NatsCredentialIsStrictAndProductionRejectsNoAuth()
    {
        using var token = JsonDocument.Parse("{\"schemaVersion\":\"nats-credential.v1\",\"authType\":\"token\",\"token\":\"secret\"}");
        Assert.NotNull(NatsCredential.Parse(token.RootElement, production: true).ToOptions());
        using var none = JsonDocument.Parse("{\"schemaVersion\":\"nats-credential.v1\",\"authType\":\"none\"}");
        Assert.Throws<CollectorRuntimeConfigurationException>(() => NatsCredential.Parse(none.RootElement, production: true));
        using var resource = JsonDocument.Parse("{\"url\":\"nats://127.0.0.1:4222\",\"accountId\":\"account-a\"}");
        var parsed = RetryingNatsPublisher.ParseServerResource(resource.RootElement);
        Assert.Equal("account-a", parsed.AccountId);
        Assert.DoesNotContain("127.0.0.1", parsed.ToString(), StringComparison.Ordinal);
        Assert.DoesNotContain("account-a", parsed.ToString(), StringComparison.Ordinal);
    }

    [Theory]
    [InlineData("{\"value\":")]
    [InlineData("{\"value\":1,\"value\":2}")]
    public void StrictJsonMapsMalformedAndDuplicateInputToStableConfigurationError(string json)
    {
        var exception = Assert.Throws<CollectorRuntimeConfigurationException>(() => StrictJson.Parse(System.Text.Encoding.UTF8.GetBytes(json)));
        Assert.Equal("COLLECTOR_CONFIGURATION_INVALID", exception.Message);
    }

    [Fact]
    public async Task WindowsProductionFileSecurityRejectsWritableFileAndReparseTarget()
    {
        if (!OperatingSystem.IsWindows()) return;
        var directory = Path.Combine(Path.GetTempPath(), "induforge-windows-security-" + Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(directory);
        try
        {
            var writable = Path.Combine(directory, "index.json");
            await File.WriteAllTextAsync(writable, "{}");
            await Assert.ThrowsAsync<CollectorRuntimeConfigurationException>(() => SecureFile.ReadAsync(writable, 1024, requireReadOnly: true, CancellationToken.None));

            var reparse = Path.Combine(directory, "index-link.json");
            try
            {
                File.CreateSymbolicLink(reparse, writable);
            }
            catch (UnauthorizedAccessException)
            {
                return; // 未启用开发者模式时 Windows 不允许普通进程创建符号链接；生产部署仍由 handle 校验拒绝。
            }
            await Assert.ThrowsAsync<CollectorRuntimeConfigurationException>(() => SecureFile.ReadAsync(reparse, 1024, requireReadOnly: false, CancellationToken.None));
        }
        finally
        {
            if (Directory.Exists(directory)) Directory.Delete(directory, recursive: true);
        }
    }

    [Fact]
    public async Task DrainCoordinatorStopsAcquisitionBeforeIndependentReplayAndCommitsAck()
    {
        var directory = Path.Combine(Path.GetTempPath(), "induforge-drain-" + Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(directory);
        try
        {
            await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(directory, 1_000_000, 500_000));
            await AppendRawAsync(wal);
            using var acquisitionCancellation = new CancellationTokenSource();
            using var replayCancellation = new CancellationTokenSource();
            var publisher = new RecordingPublisher();
            var replay = new ReplayPump(wal, publisher, new ReplayPumpOptions(TimeSpan.Zero, TimeSpan.FromMilliseconds(1), TimeSpan.FromMilliseconds(2)));
            var runningReplay = replay.RunAsync(replayCancellation.Token);
            var replayWasActiveWhenAcquisitionStopped = false;
            using var deadline = new CancellationTokenSource(TimeSpan.FromSeconds(1));

            var drained = await CollectorDrainCoordinator.StopAcquisitionAndDrainAsync(_ =>
            {
                acquisitionCancellation.Cancel();
                replayWasActiveWhenAcquisitionStopped = !replayCancellation.IsCancellationRequested;
                return Task.CompletedTask;
            }, wal, replayCancellation, timeProvider: null, deadline: deadline.Token);
            await Assert.ThrowsAnyAsync<OperationCanceledException>(() => runningReplay);

            Assert.True(acquisitionCancellation.IsCancellationRequested);
            Assert.True(replayWasActiveWhenAcquisitionStopped);
            Assert.True(drained);
            Assert.Single(publisher.Records);
            Assert.Equal(0, wal.GetSnapshot().BacklogRecords);
        }
        finally
        {
            if (Directory.Exists(directory)) Directory.Delete(directory, recursive: true);
        }
    }

    [Fact]
    public async Task DrainCoordinatorTimeoutKeepsWalAndUsesStableExitMapping()
    {
        var directory = Path.Combine(Path.GetTempPath(), "induforge-drain-timeout-" + Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(directory);
        try
        {
            await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(directory, 1_000_000, 500_000));
            await AppendRawAsync(wal);
            using var acquisitionCancellation = new CancellationTokenSource();
            using var replayCancellation = new CancellationTokenSource();
            using var expiredDeadline = new CancellationTokenSource();
            expiredDeadline.Cancel(); // 注入已到期 deadline，避免测试等待真实 20 秒。

            var drained = await CollectorDrainCoordinator.StopAcquisitionAndDrainAsync(_ =>
            {
                acquisitionCancellation.Cancel();
                return Task.CompletedTask;
            }, wal, replayCancellation, new FakeTimeProvider(), expiredDeadline.Token);

            Assert.False(drained);
            Assert.True(acquisitionCancellation.IsCancellationRequested);
            Assert.True(replayCancellation.IsCancellationRequested);
            Assert.Equal(1, wal.GetSnapshot().BacklogRecords);
            Assert.Equal("COLLECTOR_WAL_DRAIN_TIMEOUT", CollectorDrainCoordinator.DrainTimeoutReasonCode);
            Assert.Equal(4, CollectorDrainCoordinator.DrainTimeoutExitCode);
        }
        finally
        {
            if (Directory.Exists(directory)) Directory.Delete(directory, recursive: true);
        }
    }

    [Theory]
    [InlineData("{\"url\":\"nats://127.0.0.1:4222\"}")]
    [InlineData("{\"url\":\"nats://127.0.0.1:4222\",\"accountId\":\"account-a\",\"extra\":true}")]
    public void NatsResourceRequiresExactAccountScopedShape(string json)
    {
        using var resource = JsonDocument.Parse(json);
        Assert.Throws<CollectorRuntimeConfigurationException>(() => RetryingNatsPublisher.ParseServerResource(resource.RootElement));
    }

    [Fact]
    public async Task NatsAccountMismatchFailsBeforeCredentialResolution()
    {
        var resources = new StaticResourceResolver("{\"url\":\"nats://127.0.0.1:4222\",\"accountId\":\"account-b\"}");
        var secrets = new CountingSecretResolver();
        await using var publisher = new RetryingNatsPublisher(resources, secrets, "site-resource://nats", "secret://nats", "account-a", production: true);
        var record = new WalDataRecord("data.raw.x", "event", ReadOnlyMemory<byte>.Empty, "owner", 1, 1, DateTimeOffset.UtcNow);

        var exception = await Assert.ThrowsAsync<CollectorRuntimeConfigurationException>(() => publisher.PublishAsync(record, CancellationToken.None));
        Assert.Equal("COLLECTOR_CONFIGURATION_INVALID", exception.Message);
        Assert.Equal(0, secrets.ResolveCount);
    }

    [Fact]
    public async Task StatusEndpointsUseExactHealthDataAndJsonErrorEnvelopes()
    {
        var directory = Path.Combine(Path.GetTempPath(), "induforge-status-" + Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(directory);
        try
        {
            await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(Path.Combine(directory, "wal"), 1_000_000, 500_000));
            var configuration = new CollectorLoadedConfiguration(
                new CollectorArtifact("artifact-a", 1, "project-a", "1.0.0", [], []),
                new CollectorBinding("binding-a", 1, "deployment-a", "account-a", "artifact-a", 1, "digest-a", "collector-a", "owner-a", 1, new Dictionary<string, BindingConnection>(), "site-resource://nats", "secret://nats", 1_000_000, 500_000, 4_096));
            var host = CollectorRuntimeHost.CreateDefault();
            await using var orchestrator = new CollectorOrchestrator(configuration, host, wal, new StaticResourceResolver("{}"), new CountingSecretResolver());
            var prefix = "http://127.0.0.1:" + GetAvailablePort() + "/";
            await using var status = new CollectorStatusServer(prefix, "site-a", configuration, wal, orchestrator);
            status.MarkRunning();
            status.Start();
            using var client = new HttpClient { BaseAddress = new Uri(prefix) };

            using var healthy = await client.GetAsync("health");
            Assert.Equal(HttpStatusCode.OK, healthy.StatusCode);
            using var healthyJson = JsonDocument.Parse(await healthy.Content.ReadAsStringAsync());
            Assert.Equal(0, healthyJson.RootElement.GetProperty("code").GetInt32());
            var healthData = healthyJson.RootElement.GetProperty("data");
            Assert.Equal(["observedAt", "status"], healthData.EnumerateObject().Select(item => item.Name).Order(StringComparer.Ordinal));
            Assert.Equal("UP", healthData.GetProperty("status").GetString());

            using var full = await client.GetAsync("api/v1/status");
            var fullText = await full.Content.ReadAsStringAsync();
            using var fullJson = JsonDocument.Parse(fullText);
            Assert.Equal("runtime-health-status.v1", fullJson.RootElement.GetProperty("data").GetProperty("schemaVersion").GetString());
            Assert.Equal("site-a", fullJson.RootElement.GetProperty("data").GetProperty("siteId").GetString());
            Assert.Equal("account-a", fullJson.RootElement.GetProperty("data").GetProperty("accountId").GetString());
            Assert.DoesNotContain("nats://", fullText, StringComparison.Ordinal);
            Assert.DoesNotContain("token", fullText, StringComparison.OrdinalIgnoreCase);
            Assert.DoesNotContain("username", fullText, StringComparison.OrdinalIgnoreCase);
            Assert.DoesNotContain("password", fullText, StringComparison.OrdinalIgnoreCase);

            using var unknown = await client.PostAsync("unknown", new StringContent("ignored"));
            await AssertErrorEnvelopeAsync(unknown, HttpStatusCode.NotFound, allow: null);
            using var method = new HttpRequestMessage(HttpMethod.Get, "health") { Content = new StringContent("body") };
            using var notAllowed = await client.SendAsync(method);
            await AssertErrorEnvelopeAsync(notAllowed, HttpStatusCode.MethodNotAllowed, "GET");

            status.MarkStopping();
            using var down = await client.GetAsync("health");
            Assert.Equal(HttpStatusCode.ServiceUnavailable, down.StatusCode);
            using var downJson = JsonDocument.Parse(await down.Content.ReadAsStringAsync());
            Assert.NotEqual(0, downJson.RootElement.GetProperty("code").GetInt32());
            Assert.Equal("DOWN", downJson.RootElement.GetProperty("data").GetProperty("status").GetString());
        }
        finally
        {
            if (Directory.Exists(directory)) Directory.Delete(directory, recursive: true);
        }
    }

    private static int GetAvailablePort()
    {
        using var listener = new TcpListener(IPAddress.Loopback, 0);
        listener.Start();
        return ((IPEndPoint)listener.LocalEndpoint).Port;
    }

    private static async Task AssertErrorEnvelopeAsync(HttpResponseMessage response, HttpStatusCode expectedStatus, string? allow)
    {
        Assert.Equal(expectedStatus, response.StatusCode);
        var actualAllow = response.Headers.TryGetValues("Allow", out var values)
            ? values.SingleOrDefault()
            : response.Content.Headers.TryGetValues("Allow", out var contentValues) ? contentValues.SingleOrDefault() : null;
        Assert.Equal(allow, actualAllow);
        using var body = JsonDocument.Parse(await response.Content.ReadAsStringAsync());
        Assert.NotEqual(0, body.RootElement.GetProperty("code").GetInt32());
        Assert.Equal(JsonValueKind.Null, body.RootElement.GetProperty("data").ValueKind);
        Assert.False(string.IsNullOrEmpty(body.RootElement.GetProperty("reqId").GetString()));
    }

    private static async Task AppendRawAsync(DurableWal wal)
    {
        const string deploymentId = "deployment-a";
        const string accountId = "account-a";
        const string ownerId = "owner-a";
        const string pointId = "22222222-2222-4222-8222-222222222222";
        var result = await wal.AppendDataAsync(sequence =>
        {
            var subject = "data.raw." + pointId;
            var eventId = CollectorV1EventValidator.ComputeRawEventId(deploymentId, pointId, ownerId, 1, sequence);
            var payload = JsonSerializer.SerializeToUtf8Bytes(new
            {
                schemaVersion = "data.raw.v1",
                subject,
                eventId,
                deploymentId,
                accountId,
                pointId,
                ownerId,
                epoch = 1,
                sequence,
                value = 1,
                quality = "good",
                sourceTimestamp = "2026-08-30T10:20:30.123Z",
                serverTimestamp = "2026-08-30T10:20:30.150Z",
                receivedAt = "2026-08-30T10:20:30.160Z",
                source = new { collectorId = "collector-a", connectionId = "11111111-1111-4111-8111-111111111111", variableId = "33333333-3333-4333-8333-333333333333" },
            });
            return new WalAppendRequest(subject, eventId, payload, ownerId, 1);
        });
        Assert.True(result.Accepted);
    }

    private sealed class RecordingPublisher : IJetStreamPublisher
    {
        public List<WalDataRecord> Records { get; } = [];
        public Task PublishAsync(WalDataRecord record, CancellationToken cancellationToken)
        {
            Records.Add(record);
            return Task.CompletedTask;
        }
    }

    private sealed class FakeTimeProvider : TimeProvider
    {
        public override DateTimeOffset GetUtcNow() => DateTimeOffset.UnixEpoch;
    }

    private sealed class StaticResourceResolver(string json) : ICollectorResourceResolver
    {
        public ValueTask<CollectorResource> ResolveResourceAsync(string reference, CancellationToken cancellationToken) => ValueTask.FromResult(new CollectorResource(JsonDocument.Parse(json)));
    }

    private sealed class CountingSecretResolver : ICollectorSecretResolver
    {
        public int ResolveCount { get; private set; }
        public ValueTask<CollectorSecret> ResolveSecretAsync(string reference, CancellationToken cancellationToken)
        {
            ResolveCount++;
            return ValueTask.FromResult(new CollectorSecret(JsonDocument.Parse("{\"schemaVersion\":\"nats-credential.v1\",\"authType\":\"token\",\"token\":\"secret\"}")));
        }
    }

    [Fact]
    public async Task RawReservationUsesConservativeWireAndAckBudget()
    {
        var directory = Path.Combine(Path.GetTempPath(), "induforge-reservation-" + Guid.NewGuid().ToString("N"));
        try
        {
            await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(directory, 1_000_000, 500_000));
            using var reservation = await wal.ReserveRawBatchAsync(2);
            Assert.NotNull(reservation);
            Assert.Equal(2, reservation.Count);
            Assert.Equal(65_929, DurableWal.MaximumRawReservationBytes);
        }
        finally
        {
            if (Directory.Exists(directory)) Directory.Delete(directory, recursive: true);
        }
    }
}
