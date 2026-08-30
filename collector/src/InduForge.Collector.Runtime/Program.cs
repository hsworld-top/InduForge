using System.Text.Json;

namespace InduForge.Collector.Runtime;

internal static class Program
{
    private static async Task<int> Main(string[] args)
    {
        var host = CollectorRuntimeHost.CreateDefault();
        if (args.Length == 2 && string.Equals(args[0], "--preflight", StringComparison.Ordinal))
        {
            try
            {
                await using var stream = File.OpenRead(args[1]);
                var configuration = await JsonSerializer.DeserializeAsync<CollectorRuntimeConfiguration>(stream)
                    .ConfigureAwait(false)
                    ?? throw new CollectorRuntimeConfigurationException("运行态配置为空");
                host.Preflight(configuration);
                Console.WriteLine("配置预检通过。");
                return 0;
            }
            catch (Exception exception) when (exception is IOException or JsonException or CollectorRuntimeConfigurationException)
            {
                Console.Error.WriteLine($"配置预检失败：{exception.Message}");
                return 2;
            }
        }

        if (args.Length == 0)
        {
            // 采集循环尚未进入本阶段；后续仅在确认具体丢失范围时调用 DurableWal.AppendConfirmedLossAsync，
            // 不能在宿主骨架中凭空生成 DATA_GAP。
            await host.StartAsync().ConfigureAwait(false);
            await host.StopAsync().ConfigureAwait(false);
            Console.WriteLine("industrial_collector 驱动宿主基础已就绪；本阶段未启动采集或业务处理循环。");
            return 0;
        }

        Console.Error.WriteLine("用法：industrial_collector [--preflight <config.json>]");
        return 64;
    }
}
