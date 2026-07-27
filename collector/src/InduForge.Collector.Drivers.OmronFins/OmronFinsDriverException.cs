using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.OmronFins;

public sealed class OmronFinsDriverException : IndustrialDriverException
{
    public OmronFinsDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException)
    {
    }
}
