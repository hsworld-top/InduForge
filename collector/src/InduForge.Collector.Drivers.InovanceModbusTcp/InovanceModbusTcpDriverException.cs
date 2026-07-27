using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.InovanceModbusTcp;

public sealed class InovanceModbusTcpDriverException : IndustrialDriverException
{
    public InovanceModbusTcpDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException) { }
}
