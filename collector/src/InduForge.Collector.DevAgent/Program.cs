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
                singleInstance.NotifyExistingInstanceAsync().GetAwaiter().GetResult();
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
