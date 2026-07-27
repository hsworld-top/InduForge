using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Freedom;
public sealed class FreedomDriverException(string code, string message, bool retryable, Exception? innerException = null) : IndustrialDriverException(code, message, retryable, innerException);
