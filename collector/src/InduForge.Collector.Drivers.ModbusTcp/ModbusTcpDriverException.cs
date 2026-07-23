using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusTcp;

public sealed class ModbusTcpDriverException : IndustrialDriverException
{
    public ModbusTcpDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException)
    {
    }
}
