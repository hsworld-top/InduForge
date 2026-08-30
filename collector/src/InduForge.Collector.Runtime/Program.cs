using System.Text.Json;

namespace InduForge.Collector.Runtime;

internal sealed record CollectorProgramOptions(string ArtifactPath, string BindingPath, string IndexPath, string WalPath, string ListenPrefix, bool Production);

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
            using var stopping = new CancellationTokenSource();
            Console.CancelKeyPress += (_, eventArgs) => { eventArgs.Cancel = true; stopping.Cancel(); };
            var configuration = await CollectorConfigurationLoader.LoadAsync(options.ArtifactPath, options.BindingPath, options.Production, stopping.Token).ConfigureAwait(false);
            using var resolver = await StrictCollectorIndexResolver.OpenAsync(options.IndexPath, options.Production, stopping.Token).ConfigureAwait(false);
            await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(options.WalPath, configuration.Binding.MaxBytes, configuration.Binding.HighWatermarkBytes, DiagnosticReserveBytes: configuration.Binding.DiagnosticReserveBytes), stopping.Token).ConfigureAwait(false);
            var host = CollectorRuntimeHost.CreateDefault();
            await host.StartAsync(stopping.Token).ConfigureAwait(false);
            await using var orchestrator = new CollectorOrchestrator(configuration, host, wal, resolver, resolver);
            await using var publisher = new RetryingNatsPublisher(resolver, resolver, configuration.Binding.NatsResourceRef, configuration.Binding.NatsSecretRef, options.Production);
            var replay = new ReplayPump(wal, publisher);
            await using var status = new CollectorStatusServer(options.ListenPrefix, configuration, wal, orchestrator, () => publisher.IsConnected);
            await orchestrator.StartAsync(stopping.Token).ConfigureAwait(false);
            var acquisition = orchestrator.Completion;
            var replayTask = replay.RunAsync(stopping.Token);
            status.Start();
            var stopped = Task.Delay(Timeout.InfiniteTimeSpan, stopping.Token);
            var completed = await Task.WhenAny(stopped, acquisition, replayTask).ConfigureAwait(false);
            var exitCode = 0;
            if (completed != stopped)
            {
                status.MarkFatal("COLLECTOR_RUNTIME_TASK_FAILED");
                stopping.Cancel();
                exitCode = 3;
            }
            using var shutdown = new CancellationTokenSource(TimeSpan.FromSeconds(20));
            status.MarkStopping();
            try { await orchestrator.StopAsync(shutdown.Token).ConfigureAwait(false); } catch when (exitCode != 0) { }
            try { await replayTask.WaitAsync(shutdown.Token).ConfigureAwait(false); }
            catch (OperationCanceledException) when (stopping.IsCancellationRequested || shutdown.IsCancellationRequested) { }
            await host.StopAsync(shutdown.Token).ConfigureAwait(false);
            status.MarkStopped();
            return exitCode;
        }
        catch (OperationCanceledException) { return 0; }
        catch (Exception) { Console.Error.WriteLine("COLLECTOR_STARTUP_FAILED"); return 2; }
    }

    internal static CollectorProgramOptions ParseOptions(string[] args)
    {
        ArgumentNullException.ThrowIfNull(args);
        if (args.Length != 0 && args.Length != 10 && args.Length != 12) throw ConfigurationError.Invalid();
        var values = new Dictionary<string, string>(StringComparer.Ordinal);
        for (var index = 0; index < args.Length; index += 2)
        {
            if (!args[index].StartsWith("--", StringComparison.Ordinal) || string.IsNullOrWhiteSpace(args[index + 1]) || !values.TryAdd(args[index], args[index + 1])) throw ConfigurationError.Invalid();
        }
        string Required(string argument, string environment) => values.TryGetValue(argument, out var supplied) ? supplied : Environment.GetEnvironmentVariable(environment) ?? throw ConfigurationError.Invalid();
        var permitted = new HashSet<string>(["--artifact", "--binding", "--index", "--wal", "--listen", "--production"], StringComparer.Ordinal);
        if (values.Keys.Any(key => !permitted.Contains(key))) throw ConfigurationError.Invalid();
        var production = values.TryGetValue("--production", out var productionText) ? ParseBoolean(productionText) : ParseBoolean(Environment.GetEnvironmentVariable("INDUFORGE_COLLECTOR_PRODUCTION") ?? "false");
        return new CollectorProgramOptions(Required("--artifact", "INDUFORGE_COLLECTOR_ARTIFACT"), Required("--binding", "INDUFORGE_COLLECTOR_BINDING"), Required("--index", "INDUFORGE_COLLECTOR_INDEX"), Required("--wal", "INDUFORGE_COLLECTOR_WAL"), values.TryGetValue("--listen", out var listen) ? listen : Environment.GetEnvironmentVariable("INDUFORGE_COLLECTOR_LISTEN") ?? "http://127.0.0.1:18081/", production);
    }

    private static bool ParseBoolean(string value) => value switch { "true" => true, "false" => false, _ => throw ConfigurationError.Invalid() };
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
