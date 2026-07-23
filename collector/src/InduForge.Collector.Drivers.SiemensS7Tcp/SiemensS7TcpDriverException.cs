using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp;

public sealed class SiemensS7TcpDriverException : IndustrialDriverException
{
    public SiemensS7TcpDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException)
    {
    }
}
