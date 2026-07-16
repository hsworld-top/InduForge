using InduForge.Collector.DevAgent;

namespace InduForge.Collector.DevAgent.Tests;

public sealed class AgentFileLoggerTests
{
    [Fact]
    public void WritesExceptionChainAndRedactsSensitiveValues()
    {
        var baseDirectory = CreateTempDirectory();
        try
        {
            var logger = new AgentFileLogger(baseDirectory);
            var exception = new InvalidOperationException("outer failure", new HttpRequestException("inner failure"));

            logger.Error("test.failure", "Bearer secret-token registrationCode=register-me agentToken=agent-secret", exception);

            var content = File.ReadAllText(Assert.Single(Directory.GetFiles(logger.LogDirectory, "collector-dev-agent-*.log")));
            Assert.Contains("test.failure", content);
            Assert.Contains("outer failure", content);
            Assert.Contains("inner failure", content);
            Assert.Contains("[REDACTED]", content);
            Assert.DoesNotContain("secret-token", content);
            Assert.DoesNotContain("register-me", content);
            Assert.DoesNotContain("agent-secret", content);
        }
        finally
        {
            Directory.Delete(baseDirectory, true);
        }
    }

    [Fact]
    public void KeepsOnlyFourteenDailyLogFiles()
    {
        var baseDirectory = CreateTempDirectory();
        try
        {
            var logDirectory = Path.Combine(baseDirectory, "logs");
            Directory.CreateDirectory(logDirectory);
            for (var index = 0; index < 16; index++)
            {
                File.WriteAllText(Path.Combine(logDirectory, $"collector-dev-agent-202606{index + 1:00}.log"), "old");
            }

            var logger = new AgentFileLogger(baseDirectory);
            logger.Info("test.start", "retention");

            Assert.Equal(14, Directory.GetFiles(logDirectory, "collector-dev-agent-*.log").Length);
        }
        finally
        {
            Directory.Delete(baseDirectory, true);
        }
    }

    private static string CreateTempDirectory()
    {
        var path = Path.Combine(Path.GetTempPath(), $"induforge-agent-log-{Guid.NewGuid():N}");
        Directory.CreateDirectory(path);
        return path;
    }
}
