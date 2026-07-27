using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.AllenBradleyEtherNetIp;

public sealed class AllenBradleyEtherNetIpDriverException : IndustrialDriverException
{
    public AllenBradleyEtherNetIpDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException)
    {
    }
}
