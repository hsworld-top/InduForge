using System.Text.Json;
using System.Runtime.InteropServices;

namespace InduForge.Collector.Runtime;

internal sealed record CollectorProgramOptions(string ArtifactPath, string BindingPath, string IndexPath, string WalPath, string ListenPrefix, string SiteId, bool Production);

internal static class Program
{
    private static async Task<int> Main(string[] args)
    {
        if (args.Length == 2 && string.Equals(args[0], "--preflight", StringComparison.Ordinal)) return await RunLegacyPreflightAsync(args[1]).ConfigureAwait(false);
        CollectorProgramOptions options;
        try { options = ParseOptions(args); }
        catch (CollectorRuntimeConfigurationException) { Console.Error.WriteLine("COLLECTOR_CONFIGURATION_INVALID"); return 64; }
        try
        {
            using var stopRequested = new CancellationTokenSource();
            using var replayCancellation = new CancellationTokenSource();
            ConsoleCancelEventHandler cancelHandler = (_, eventArgs) => { eventArgs.Cancel = true; stopRequested.Cancel(); };
            Console.CancelKeyPress += cancelHandler;
            using var sigterm = RegisterSigterm(stopRequested);
            var configuration = await CollectorConfigurationLoader.LoadAsync(options.ArtifactPath, options.BindingPath, options.Production, stopRequested.Token).ConfigureAwait(false);
            using var resolver = await StrictCollectorIndexResolver.OpenAsync(options.IndexPath, options.Production, stopRequested.Token).ConfigureAwait(false);
            await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(options.WalPath, configuration.Binding.MaxBytes, configuration.Binding.HighWatermarkBytes, DiagnosticReserveBytes: configuration.Binding.DiagnosticReserveBytes), stopRequested.Token).ConfigureAwait(false);
            var host = CollectorRuntimeHost.CreateDefault();
            await host.StartAsync(stopRequested.Token).ConfigureAwait(false);
            await using var orchestrator = new CollectorOrchestrator(configuration, host, wal, resolver, resolver);
            await using var publisher = new RetryingNatsPublisher(resolver, resolver, configuration.Binding.NatsResourceRef, configuration.Binding.NatsSecretRef, configuration.Binding.AccountId, options.Production);
            var replay = new ReplayPump(wal, publisher);
            await using var status = new CollectorStatusServer(options.ListenPrefix, options.SiteId, configuration, wal, orchestrator, () => publisher.IsConnected);
            await orchestrator.StartAsync(stopRequested.Token).ConfigureAwait(false);
            status.MarkRunning();
            var acquisition = orchestrator.Completion;
            var replayTask = replay.RunAsync(replayCancellation.Token);
            status.Start();
            var stopped = Task.Delay(Timeout.InfiniteTimeSpan, stopRequested.Token);
            var completed = await Task.WhenAny(stopped, acquisition, replayTask).ConfigureAwait(false);
            var exitCode = 0;
            if (completed != stopped)
            {
                var configurationFailure = completed.IsFaulted && completed.Exception?.GetBaseException() is CollectorRuntimeConfigurationException;
                status.MarkFatal(configurationFailure ? "COLLECTOR_CONFIGURATION_INVALID" : "COLLECTOR_RUNTIME_TASK_FAILED");
                stopRequested.Cancel();
                exitCode = configurationFailure ? 64 : 3;
            }
            using var shutdown = new CancellationTokenSource(TimeSpan.FromSeconds(20));
            status.MarkStopping();
            if (exitCode == 0)
            {
                // 收到停止信号后先停止采集，已有 WAL 仍持续等待 PubAck 和 ACK marker，不复用采集取消令牌。
                var drained = await CollectorDrainCoordinator.StopAcquisitionAndDrainAsync(orchestrator.StopAsync, wal, replayCancellation, timeProvider: null, deadline: shutdown.Token).ConfigureAwait(false);
                if (!drained)
                {
                    status.MarkFatal(CollectorDrainCoordinator.DrainTimeoutReasonCode);
                    exitCode = CollectorDrainCoordinator.DrainTimeoutExitCode;
                }
            }
            else
            {
                try { await orchestrator.StopAsync(shutdown.Token).ConfigureAwait(false); } catch { }
                replayCancellation.Cancel();
            }
            try { await replayTask.ConfigureAwait(false); }
            catch (OperationCanceledException) when (replayCancellation.IsCancellationRequested) { }
            catch when (exitCode == 0) { status.MarkFatal("COLLECTOR_REPLAY_FAILED"); exitCode = 3; }
            try { await host.StopAsync(shutdown.Token).ConfigureAwait(false); } catch when (exitCode != 0) { }
            if (exitCode == 0) status.MarkStopped();
            Console.CancelKeyPress -= cancelHandler;
            return exitCode;
        }
        catch (OperationCanceledException) { return 0; }
        catch (CollectorRuntimeConfigurationException) { Console.Error.WriteLine("COLLECTOR_CONFIGURATION_INVALID"); return 64; }
        catch (Exception) { Console.Error.WriteLine("COLLECTOR_STARTUP_FAILED"); return 2; }
    }

    internal static CollectorProgramOptions ParseOptions(string[] args)
    {
        ArgumentNullException.ThrowIfNull(args);
        if (args.Length % 2 != 0 || args.Length > 14) throw ConfigurationError.Invalid();
        var values = new Dictionary<string, string>(StringComparer.Ordinal);
        for (var index = 0; index < args.Length; index += 2)
        {
            if (!args[index].StartsWith("--", StringComparison.Ordinal) || string.IsNullOrWhiteSpace(args[index + 1]) || !values.TryAdd(args[index], args[index + 1])) throw ConfigurationError.Invalid();
        }
        string Required(string argument, string environment) => values.TryGetValue(argument, out var supplied) ? supplied : Environment.GetEnvironmentVariable(environment) ?? throw ConfigurationError.Invalid();
        var permitted = new HashSet<string>(["--artifact", "--binding", "--index", "--wal", "--listen", "--site-id", "--production"], StringComparer.Ordinal);
        if (values.Keys.Any(key => !permitted.Contains(key))) throw ConfigurationError.Invalid();
        var production = values.TryGetValue("--production", out var productionText) ? ParseBoolean(productionText) : ParseBoolean(Environment.GetEnvironmentVariable("INDUFORGE_COLLECTOR_PRODUCTION") ?? "false");
        var siteId = values.TryGetValue("--site-id", out var suppliedSiteId) ? suppliedSiteId : Environment.GetEnvironmentVariable("INDUFORGE_COLLECTOR_SITE_ID");
        if (string.IsNullOrWhiteSpace(siteId))
        {
            if (production) throw ConfigurationError.Invalid();
            siteId = "local";
        }
        return new CollectorProgramOptions(Required("--artifact", "INDUFORGE_COLLECTOR_ARTIFACT"), Required("--binding", "INDUFORGE_COLLECTOR_BINDING"), Required("--index", "INDUFORGE_COLLECTOR_INDEX"), Required("--wal", "INDUFORGE_COLLECTOR_WAL"), values.TryGetValue("--listen", out var listen) ? listen : Environment.GetEnvironmentVariable("INDUFORGE_COLLECTOR_LISTEN") ?? "http://127.0.0.1:18081/", StrictJson.RequireStableId(siteId), production);
    }

    private static bool ParseBoolean(string value) => value switch { "true" => true, "false" => false, _ => throw ConfigurationError.Invalid() };
    private static PosixSignalRegistration? RegisterSigterm(CancellationTokenSource stopRequested)
    {
        if (!OperatingSystem.IsWindows())
        {
            return PosixSignalRegistration.Create(PosixSignal.SIGTERM, context =>
            {
                context.Cancel = true;
                stopRequested.Cancel();
            });
        }
        return null;
    }
    private static async Task<int> RunLegacyPreflightAsync(string path)
    {
        try
        {
            await using var stream = File.OpenRead(path);
            var configuration = await JsonSerializer.DeserializeAsync<CollectorRuntimeConfiguration>(stream).ConfigureAwait(false) ?? throw ConfigurationError.Invalid();
            CollectorRuntimeHost.CreateDefault().Preflight(configuration); Console.WriteLine("配置预检通过。"); return 0;
        }
        catch (Exception exception) when (exception is IOException or JsonException or CollectorRuntimeConfigurationException) { Console.Error.WriteLine("COLLECTOR_CONFIGURATION_INVALID"); return 2; }
    }
}
