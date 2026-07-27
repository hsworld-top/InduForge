using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.ModbusRtu;

public sealed class ModbusRtuDriverException : IndustrialDriverException
{
    public ModbusRtuDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException)
    {
    }
}
