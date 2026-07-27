namespace InduForge.Collector.Drivers.Xinje;
internal sealed class XinjeDriverException(string code, string message, bool retryable) : Exception(message) { public string Code { get; } = code; public bool Retryable { get; } = retryable; }
