using System.Text;
using System.Text.RegularExpressions;

namespace InduForge.Collector.DevAgent;

internal sealed partial class AgentFileLogger
{
    private const int RetainedFileCount = 14;
    private readonly object _syncRoot = new();
    private readonly string _logDirectory;
    private DateOnly? _lastCleanupDate;
    private bool _writeFailureReported;

    public AgentFileLogger(string? baseDirectory = null)
    {
        _logDirectory = Path.Combine(baseDirectory ?? AppContext.BaseDirectory, "logs");
        Directory.CreateDirectory(_logDirectory);
        var probePath = Path.Combine(_logDirectory, $".write-probe-{Environment.ProcessId}");
        File.WriteAllText(probePath, string.Empty, Encoding.UTF8);
        File.Delete(probePath);
    }

    internal string LogDirectory => _logDirectory;
    public event Action<Exception>? WriteFailed;

    public void Info(string eventName, string message) => Write("INFO", eventName, message, null);

    public void Warn(string eventName, string message, Exception? exception = null) => Write("WARN", eventName, message, exception);

    public void Error(string eventName, string message, Exception exception) => Write("ERROR", eventName, message, exception);

    private void Write(string level, string eventName, string message, Exception? exception)
    {
        lock (_syncRoot)
        {
            try
            {
                var now = DateTimeOffset.Now;
                var logPath = Path.Combine(_logDirectory, $"collector-dev-agent-{now:yyyyMMdd}.log");
                var detail = exception is null ? string.Empty : $"{Environment.NewLine}{exception}";
                var line = $"{now:yyyy-MM-dd HH:mm:ss.fff} {level,-5} {eventName} {Sanitize(message + detail)}{Environment.NewLine}";
                File.AppendAllText(logPath, line, new UTF8Encoding(false));
                CleanupIfNeeded(DateOnly.FromDateTime(now.DateTime));
            }
            catch (Exception writeException)
            {
                if (_writeFailureReported) return;
                _writeFailureReported = true;
                try { WriteFailed?.Invoke(writeException); } catch { }
            }
        }
    }

    private void CleanupIfNeeded(DateOnly currentDate)
    {
        if (_lastCleanupDate == currentDate) return;
        _lastCleanupDate = currentDate;
        foreach (var file in new DirectoryInfo(_logDirectory)
                     .GetFiles("collector-dev-agent-*.log")
                     .OrderByDescending(file => file.Name, StringComparer.Ordinal)
                     .Skip(RetainedFileCount))
        {
            file.Delete();
        }
    }

    internal static string Sanitize(string value) => SensitiveValuePattern().Replace(value, match => $"{match.Groups[1].Value}[REDACTED]");

    [GeneratedRegex("(?i)(Bearer\\s+|[\"']?(?:registrationCode|agentToken)[\"']?\\s*[=:]\\s*[\"']?)[^\\s,;\"']+")]
    private static partial Regex SensitiveValuePattern();
}
