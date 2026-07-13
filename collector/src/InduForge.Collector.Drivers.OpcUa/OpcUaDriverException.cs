namespace InduForge.Collector.Drivers.OpcUa;

public sealed class OpcUaDriverException : Exception
{
    public OpcUaDriverException(string code, string message, bool retryable, Exception? innerException = null)
        : base(message, innerException)
    {
        Code = code;
        Retryable = retryable;
    }

    public string Code { get; }

    public bool Retryable { get; }
}
