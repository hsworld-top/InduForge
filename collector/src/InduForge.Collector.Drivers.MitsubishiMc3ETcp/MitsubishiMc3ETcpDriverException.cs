using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.MitsubishiMc3ETcp;

public sealed class MitsubishiMc3ETcpDriverException : IndustrialDriverException
{
    public MitsubishiMc3ETcpDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException)
    {
    }
}
