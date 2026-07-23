using InduForge.Collector.Adapters.Hsl;

namespace InduForge.Collector.Adapters.Hsl.Tests;

public sealed class HslPublicContractTests
{
    [Fact]
    public void PublicApiDoesNotExposeHslCommunicationTypes()
    {
        var exposedNamespaces = typeof(HslReadResult).Assembly
            .GetExportedTypes()
            .SelectMany(type =>
                type.GetProperties().Select(property => property.PropertyType.Namespace)
                    .Concat(type.GetMethods().Select(method => method.ReturnType.Namespace))
                    .Concat(type.GetMethods().SelectMany(method => method.GetParameters()).Select(parameter => parameter.ParameterType.Namespace)))
            .Where(value => value is not null)
            .ToArray();

        Assert.DoesNotContain(
            exposedNamespaces,
            value => value!.StartsWith("HslCommunication", StringComparison.Ordinal));
    }

    [Fact]
    public void FailedReadKeepsStableErrorInformation()
    {
        var result = HslReadResult.Failure("HSL_READ_FAILED", "读取失败", retryable: true);

        Assert.False(result.Succeeded);
        Assert.Null(result.Value);
        Assert.Equal("HSL_READ_FAILED", result.ErrorCode);
        Assert.Equal("读取失败", result.ErrorMessage);
        Assert.True(result.Retryable);
    }
}
