using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.PanasonicMewtocolTcp;

public sealed class PanasonicMewtocolTcpDriverException : IndustrialDriverException
{
    public PanasonicMewtocolTcpDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException) { }
}
