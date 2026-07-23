using System.Reflection;
using System.Text.Json;
using InduForge.Collector.Contracts;
using InduForge.Collector.Drivers.ModbusTcp;
using InduForge.Collector.Drivers.OpcUa;
using InduForge.Collector.Drivers.SiemensS7Tcp;

namespace InduForge.Collector.DevAgent;

internal sealed class DriverRegistry
{
    private const string OpcUaManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.opcua.standard.manifest.json";
    private const string ModbusTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.modbus.tcp.manifest.json";
    private const string SiemensS7TcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.siemens.s7-tcp.manifest.json";
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly Dictionary<string, Func<IIndustrialDriver>> _factories;
    private readonly Dictionary<string, DriverDescriptor> _descriptors;

    public DriverRegistry(IEnumerable<Func<IIndustrialDriver>> factories, IReadOnlyDictionary<string, DriverManifest>? manifests = null)
    {
        var factoryMap = new Dictionary<string, Func<IIndustrialDriver>>(StringComparer.Ordinal);
        var descriptorMap = new Dictionary<string, DriverDescriptor>(StringComparer.Ordinal);
        foreach (var factory in factories)
        {
            var driver = factory();
            var descriptor = BuildDescriptor(driver);
            if (!factoryMap.TryAdd(descriptor.DriverId, factory))
            {
                throw new InvalidOperationException($"驱动 {descriptor.DriverId} 重复注册");
            }
            if (manifests is not null)
            {
                ValidateManifest(descriptor, manifests);
            }
            descriptorMap.Add(descriptor.DriverId, descriptor);
        }
        _factories = factoryMap;
        _descriptors = descriptorMap;
    }

    public IReadOnlyCollection<DriverDescriptor> Descriptors => _descriptors.Values;

    public DriverDescriptor Describe(string driverId) => _descriptors.TryGetValue(driverId, out var descriptor)
        ? descriptor
        : throw new CollectorTaskExecutionException("COLLECTOR_DRIVER_UNSUPPORTED", "当前 Agent 未安装该驱动", false);

    public IIndustrialDriver Create(string driverId)
    {
        if (!_factories.TryGetValue(driverId, out var factory))
        {
            throw new CollectorTaskExecutionException("COLLECTOR_DRIVER_UNSUPPORTED", "当前 Agent 未安装该驱动", false);
        }
        var driver = factory();
        if (!string.Equals(driver.Descriptor.DriverId, driverId, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"驱动工厂返回了不一致的 driverId：{driver.Descriptor.DriverId}");
        }
        return driver;
    }

    public static DriverRegistry CreateDefault()
    {
        var assembly = Assembly.GetExecutingAssembly();
        var opcUaManifest = LoadManifest(assembly, OpcUaManifestResource);
        var modbusTcpManifest = LoadManifest(assembly, ModbusTcpManifestResource);
        var siemensS7TcpManifest = LoadManifest(assembly, SiemensS7TcpManifestResource);
        return new DriverRegistry(
        [
            () => new OpcUaDriver(),
            () => new ModbusTcpDriver(),
            () => new SiemensS7TcpDriver(),
        ],
        new Dictionary<string, DriverManifest>(StringComparer.Ordinal)
        {
            [opcUaManifest.DriverId] = opcUaManifest,
            [modbusTcpManifest.DriverId] = modbusTcpManifest,
            [siemensS7TcpManifest.DriverId] = siemensS7TcpManifest,
        });
    }

    private static DriverDescriptor BuildDescriptor(IIndustrialDriver driver)
    {
        var declared = driver.Descriptor;
        if (string.IsNullOrWhiteSpace(declared.ProtocolFamily) || string.IsNullOrWhiteSpace(declared.DriverId) ||
            string.IsNullOrWhiteSpace(declared.DriverVersion) || declared.SchemaVersions.Count == 0)
        {
            throw new InvalidOperationException("驱动描述缺少必要元数据");
        }

        var operations = new List<string> { DriverOperations.ConnectionTest };
        if (driver is IConnectionSessionDriver)
        {
            operations.Add(DriverOperations.ConnectionOpen);
            operations.Add(DriverOperations.ConnectionClose);
        }
        if (driver is IDeviceBrowser) operations.Add(DriverOperations.DeviceBrowse);
        if (driver is IPointReader) operations.Add(DriverOperations.PointRead);
        if (driver is IPointWriter) operations.Add(DriverOperations.PointWrite);
        if (driver is IPointSubscriptionPreview) operations.Add(DriverOperations.PointSubscriptionPreview);
        if (!declared.Operations.Order().SequenceEqual(operations.Order()))
        {
            throw new InvalidOperationException($"驱动 {declared.DriverId} 声明的操作与实现接口不一致");
        }
        return declared with { Operations = operations };
    }

    private static void ValidateManifest(DriverDescriptor descriptor, IReadOnlyDictionary<string, DriverManifest> manifests)
    {
        if (!manifests.TryGetValue(descriptor.DriverId, out var manifest) ||
            manifest.ProtocolFamily != descriptor.ProtocolFamily ||
            manifest.DriverVersion != descriptor.DriverVersion ||
            !descriptor.SchemaVersions.Contains(manifest.SchemaVersion) ||
            !manifest.Operations.Order().SequenceEqual(descriptor.Operations.Order()))
        {
            throw new InvalidOperationException($"驱动 {descriptor.DriverId} 与内嵌 Manifest 不一致");
        }
    }

    private static DriverManifest LoadManifest(Assembly assembly, string resourceName)
    {
        using var stream = assembly.GetManifestResourceStream(resourceName)
            ?? throw new InvalidOperationException($"缺少内嵌驱动 Manifest：{resourceName}");
        return JsonSerializer.Deserialize<DriverManifest>(stream, JsonOptions)
            ?? throw new InvalidOperationException($"无法解析内嵌驱动 Manifest：{resourceName}");
    }
}

internal sealed record DriverManifest(
    string ProtocolFamily,
    string DriverId,
    string DriverVersion,
    int SchemaVersion,
    IReadOnlyList<string> Operations);
