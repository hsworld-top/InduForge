namespace InduForge.Collector.Drivers.Delta;
internal sealed class DeltaDriverException(string code, string message, bool retryable) : Exception(message)
{
    public string Code { get; } = code;
    public bool Retryable { get; } = retryable;
}
