using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.SiemensS7Tcp;

public sealed class SiemensVariantDriverException : IndustrialDriverException
{
    public SiemensVariantDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException) { }
}
