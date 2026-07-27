using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Cimon;
public sealed class CimonDriverException(string code, string message, bool retryable, Exception? innerException = null) : IndustrialDriverException(code, message, retryable, innerException);
