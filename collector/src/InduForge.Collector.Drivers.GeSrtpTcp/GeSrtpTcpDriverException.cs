using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.GeSrtpTcp;

public sealed class GeSrtpTcpDriverException : IndustrialDriverException
{
    public GeSrtpTcpDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException) { }
}
