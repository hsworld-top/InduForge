namespace InduForge.Collector.DevAgent;

internal static class Program
{
    [STAThread]
    private static void Main()
    {
        ApplicationConfiguration.Initialize();
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
            Application.Run(new TrayApplicationContext(singleInstance));
        }
        catch
        {
            singleInstance.DisposeAsync().AsTask().GetAwaiter().GetResult();
            throw;
        }
    }
}
