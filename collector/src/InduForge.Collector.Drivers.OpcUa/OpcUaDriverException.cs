namespace InduForge.Collector.Drivers.OpcUa;

public sealed class OpcUaDriverException : InduForge.Collector.Contracts.IndustrialDriverException
{
    public OpcUaDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException)
    {
    }
}
