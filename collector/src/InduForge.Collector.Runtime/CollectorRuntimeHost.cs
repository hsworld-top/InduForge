using System.Reflection;
using System.Runtime.InteropServices;
using System.Text.Json;
using InduForge.Collector.Contracts;
using InduForge.Collector.DriverHosting;
using InduForge.Collector.Drivers.ModbusTcp;
using InduForge.Collector.Drivers.OpcUa;

namespace InduForge.Collector.Runtime;

public sealed record RuntimeDriverConfiguration(
    string DriverId,
    string ProtocolFamily,
    string DriverVersion,
    int SchemaVersion,
    IReadOnlyList<string> RequiredOperations);

public sealed record CollectorRuntimeConfiguration(IReadOnlyList<RuntimeDriverConfiguration> Drivers);

public enum CollectorRuntimeState
{
    Created,
    Running,
    Stopped,
}

/// <summary>
/// 运行态的最小宿主：只负责驱动白名单和配置预检，不承担采集循环、缓冲或 WAL。
/// </summary>
public sealed class CollectorRuntimeHost
{
    private static readonly JsonSerializerOptions ManifestJsonOptions = new(JsonSerializerDefaults.Web);
    private readonly DriverHostRegistry _drivers;
    private readonly DriverHostPlatform _platform;

    private CollectorRuntimeHost(DriverHostRegistry drivers, DriverHostPlatform platform)
    {
        _drivers = drivers;
        _platform = platform;
    }

    public CollectorRuntimeState State { get; private set; } = CollectorRuntimeState.Created;

    public IReadOnlyCollection<DriverDescriptor> Drivers => _drivers.Descriptors;

    public static DriverHostPlatform CurrentPlatform { get; } = new("runtime", RuntimeInformation.RuntimeIdentifier);

    /// <summary>创建仅含认证驱动的运行态宿主，平台身份固定为当前真实 Runtime RID。</summary>
    public static CollectorRuntimeHost CreateDefault() => Create(CurrentPlatform);

    /// <summary>仅供同程序集测试显式验证 Manifest RID，生产调用不得伪造平台身份。</summary>
    internal static CollectorRuntimeHost CreateForTest(DriverHostPlatform targetPlatform)
    {
        ArgumentNullException.ThrowIfNull(targetPlatform);
        return Create(targetPlatform);
    }

    private static CollectorRuntimeHost Create(DriverHostPlatform targetPlatform)
    {
        var assembly = typeof(CollectorRuntimeHost).Assembly;
        var manifests = new Dictionary<string, DriverManifest>(StringComparer.Ordinal)
        {
            ["opcua.standard"] = LoadManifest(assembly, "InduForge.Collector.Runtime.Manifests.opcua.standard.json"),
            ["modbus.tcp"] = LoadManifest(assembly, "InduForge.Collector.Runtime.Manifests.modbus.tcp.json"),
        };

        // 仅在编译期明确列出的受认证工厂可进入运行态；禁止从目录或程序集反射扫描驱动。
        return new CollectorRuntimeHost(new DriverHostRegistry(
        [
            () => new OpcUaDriver(),
            () => new ModbusTcpDriver(),
        ],
        manifests), targetPlatform);
    }

    public void Preflight(CollectorRuntimeConfiguration configuration)
    {
        ArgumentNullException.ThrowIfNull(configuration);
        if (configuration.Drivers is null)
        {
            throw new CollectorRuntimeConfigurationException("运行态配置缺少 drivers");
        }

        foreach (var configuredDriver in configuration.Drivers)
        {
            if (string.IsNullOrWhiteSpace(configuredDriver.DriverId) || !_drivers.IsRegistered(configuredDriver.DriverId))
            {
                throw new CollectorRuntimeConfigurationException($"运行态未认证驱动：{configuredDriver.DriverId}");
            }

            var descriptor = _drivers.Describe(configuredDriver.DriverId);
            if (!string.Equals(descriptor.ProtocolFamily, configuredDriver.ProtocolFamily, StringComparison.Ordinal) ||
                !string.Equals(descriptor.DriverVersion, configuredDriver.DriverVersion, StringComparison.Ordinal) ||
                !descriptor.SchemaVersions.Contains(configuredDriver.SchemaVersion))
            {
                throw new CollectorRuntimeConfigurationException($"驱动 {configuredDriver.DriverId} 的协议或 Schema 版本不匹配");
            }

            if (configuredDriver.RequiredOperations is null ||
                configuredDriver.RequiredOperations.Except(descriptor.Operations, StringComparer.Ordinal).Any())
            {
                throw new CollectorRuntimeConfigurationException($"驱动 {configuredDriver.DriverId} 不支持配置要求的操作");
            }

            // 官方 Manifest 尚未授权的平台必须在真正进入运行配置前被阻断。
            try
            {
                _drivers.ValidatePlatform(configuredDriver.DriverId, _platform);
            }
            catch (DriverRegistrationException exception)
            {
                throw new CollectorRuntimeConfigurationException(exception.Message);
            }
        }
    }

    public Task StartAsync(CancellationToken cancellationToken = default)
    {
        cancellationToken.ThrowIfCancellationRequested();
        if (State == CollectorRuntimeState.Running)
        {
            return Task.CompletedTask;
        }

        State = CollectorRuntimeState.Running;
        return Task.CompletedTask;
    }

    public Task StopAsync(CancellationToken cancellationToken = default)
    {
        cancellationToken.ThrowIfCancellationRequested();
        State = CollectorRuntimeState.Stopped;
        return Task.CompletedTask;
    }

    private static DriverManifest LoadManifest(Assembly assembly, string resourceName)
    {
        using var stream = assembly.GetManifestResourceStream(resourceName)
            ?? throw new InvalidOperationException($"缺少内嵌驱动 Manifest：{resourceName}");
        return JsonSerializer.Deserialize<DriverManifest>(stream, ManifestJsonOptions)
            ?? throw new InvalidOperationException($"无法解析内嵌驱动 Manifest：{resourceName}");
    }
}

public sealed class CollectorRuntimeConfigurationException : InvalidOperationException
{
    public CollectorRuntimeConfigurationException(string message)
        : base(message)
    {
    }
}
