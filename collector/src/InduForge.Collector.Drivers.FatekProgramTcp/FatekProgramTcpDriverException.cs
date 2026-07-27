using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.FatekProgramTcp;

public sealed class FatekProgramDriverException : IndustrialDriverException
{
    public FatekProgramDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException) { }
}
