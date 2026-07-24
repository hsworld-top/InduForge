using InduForge.Collector.Adapters.Hsl;

namespace InduForge.Collector.DevAgent;

internal static class Program
{
    [STAThread]
    private static void Main()
    {
        ApplicationConfiguration.Initialize();
        AgentFileLogger logger;
        try
        {
            logger = new AgentFileLogger();
        }
        catch (Exception exception)
        {
            MessageBox.Show(
                $"无法在程序目录创建日志文件，请确认目录可写。\n\n{AppContext.BaseDirectory}\n\n{exception.Message}",
                "采集调试代理启动失败",
                MessageBoxButtons.OK,
                MessageBoxIcon.Error);
            return;
        }

        RegisterUnhandledExceptionLogging(logger);
        logger.WriteFailed += exception => MessageBox.Show(
            $"日志写入失败，后续异常可能无法保存。\n\n{logger.LogDirectory}\n\n{exception.Message}",
            "采集调试代理日志异常",
            MessageBoxButtons.OK,
            MessageBoxIcon.Warning);
        logger.Info("agent.start", $"version={Application.ProductVersion} processId={Environment.ProcessId} baseDirectory={AppContext.BaseDirectory}");
        InitializeHslAuthorization(logger);
        try
        {
            RunApplication(logger);
        }
        catch (Exception exception)
        {
            logger.Error("agent.fatal", "采集调试代理发生未处理异常", exception);
            MessageBox.Show(exception.Message, "采集调试代理异常", MessageBoxButtons.OK, MessageBoxIcon.Error);
        }
        finally
        {
            logger.Info("agent.exit", "采集调试代理已退出");
        }
    }

    private static void InitializeHslAuthorization(AgentFileLogger logger)
    {
        try
        {
            var result = HslAuthorizationInitializer.Initialize(HslAuthorizationSettings.AuthorizationCode);
            switch (result.Status)
            {
                case HslAuthorizationStatus.Activated:
                    logger.Info("hsl.authorization.activated", "HSL 通信组件授权成功");
                    break;
                case HslAuthorizationStatus.Rejected:
                    logger.Warn("hsl.authorization.rejected", "HSL 授权码无效，HSL 驱动将继续受试用时长限制");
                    break;
                default:
                    logger.Warn("hsl.authorization.not_configured", "未配置 HSL 授权码，HSL 驱动将使用试用模式");
                    break;
            }
        }
        catch (Exception exception)
        {
            // 授权初始化异常不阻止代理启动，OPC UA 等非 HSL 驱动仍应保持可用。
            logger.Error("hsl.authorization.failed", "HSL 授权初始化异常，HSL 驱动将使用试用模式", exception);
        }
    }

    private static void RunApplication(AgentFileLogger logger)
    {
        var singleInstance = new SingleInstanceCoordinator();
        if (!singleInstance.IsPrimaryInstance)
        {
            try
            {
                if (singleInstance.ExistingInstanceInCurrentSession)
                {
                    if (!singleInstance.NotifyExistingInstanceAsync().GetAwaiter().GetResult())
                    {
                        MessageBox.Show("采集调试代理已在运行，但无法唤起现有窗口。", "InduForge 采集调试代理", MessageBoxButtons.OK, MessageBoxIcon.Information);
                    }
                }
                else
                {
                    MessageBox.Show("采集调试代理已在其他 Windows 会话运行，当前会话不能再次启动。", "InduForge 采集调试代理", MessageBoxButtons.OK, MessageBoxIcon.Information);
                }
            }
            finally
            {
                singleInstance.DisposeAsync().AsTask().GetAwaiter().GetResult();
            }
            return;
        }

        try
        {
            Application.Run(new TrayApplicationContext(singleInstance, logger));
        }
        catch
        {
            singleInstance.DisposeAsync().AsTask().GetAwaiter().GetResult();
            throw;
        }
    }

    private static void RegisterUnhandledExceptionLogging(AgentFileLogger logger)
    {
        Application.SetUnhandledExceptionMode(UnhandledExceptionMode.CatchException);
        Application.ThreadException += (_, args) => logger.Error("agent.ui.unhandled", "WinForms UI 线程异常", args.Exception);
        AppDomain.CurrentDomain.UnhandledException += (_, args) =>
        {
            var exception = args.ExceptionObject as Exception ?? new InvalidOperationException(args.ExceptionObject?.ToString() ?? "未知异常");
            logger.Error("agent.domain.unhandled", $"AppDomain 未处理异常 terminating={args.IsTerminating}", exception);
        };
        TaskScheduler.UnobservedTaskException += (_, args) =>
        {
            logger.Error("agent.task.unobserved", "未观察的 Task 异常", args.Exception);
            args.SetObserved();
        };
    }
}
