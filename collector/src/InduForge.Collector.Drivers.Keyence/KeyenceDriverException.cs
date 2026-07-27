namespace InduForge.Collector.Drivers.Keyence;

internal sealed class KeyenceDriverException(string code, string message, bool retryable)
    : Exception(message)
{
    public string Code { get; } = code;
    public bool Retryable { get; } = retryable;
}
