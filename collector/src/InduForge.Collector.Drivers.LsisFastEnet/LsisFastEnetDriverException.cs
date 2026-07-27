using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.LsisFastEnet;
public sealed class LsisFastEnetDriverException(string code, string message, bool retryable, Exception? innerException = null) : IndustrialDriverException(code, message, retryable, innerException);
