namespace InduForge.Collector.Drivers.MegMeet;

internal sealed class MegMeetDriverException(string code, string message, bool retryable) : Exception(message)
{
    public string Code { get; } = code;
    public bool Retryable { get; } = retryable;
}
