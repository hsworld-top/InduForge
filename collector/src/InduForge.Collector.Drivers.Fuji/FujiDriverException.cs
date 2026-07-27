namespace InduForge.Collector.Drivers.Fuji;
internal sealed class FujiDriverException(string code, string message, bool retryable) : Exception(message) { public string Code { get; } = code; public bool Retryable { get; } = retryable; }
