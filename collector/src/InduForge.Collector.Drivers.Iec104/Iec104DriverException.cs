using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.Iec104;

public sealed class Iec104DriverException : IndustrialDriverException
{
    public Iec104DriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException)
    {
    }
}
