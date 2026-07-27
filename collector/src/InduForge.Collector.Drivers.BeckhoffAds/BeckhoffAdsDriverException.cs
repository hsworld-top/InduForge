using InduForge.Collector.Contracts;

namespace InduForge.Collector.Drivers.BeckhoffAds;

public sealed class BeckhoffAdsDriverException : IndustrialDriverException
{
    public BeckhoffAdsDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(code, message, retryable, innerException)
    {
    }
}
