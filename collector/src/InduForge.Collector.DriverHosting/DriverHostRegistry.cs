using InduForge.Collector.Contracts;

namespace InduForge.Collector.DriverHosting;

/// <summary>
/// 描述宿主加载驱动时必须满足的平台身份；不依赖任何 UI、任务或凭据实现。
/// </summary>
public sealed record DriverHostPlatform(string HostKind, string RuntimeIdentifier);

/// <summary>
/// 协议 Manifest 中与驱动加载安全相关的最小字段。
/// </summary>
public sealed record DriverManifest(
    string ProtocolFamily,
    string DriverId,
    string DriverVersion,
    int SchemaVersion,
    IReadOnlyList<string> Operations,
    IReadOnlyList<string> Transports,
    IReadOnlyDictionary<string, IReadOnlyList<string>>? Platforms = null);

/// <summary>
/// 驱动注册或 Manifest 校验失败时抛出，调用方可据此阻止宿主继续启动。
/// </summary>
public sealed class DriverRegistrationException : InvalidOperationException
{
    public DriverRegistrationException(string message)
        : base(message)
    {
    }
}

/// <summary>
/// 跨宿主的显式驱动注册表。它只接受调用方给出的工厂，不进行程序集扫描。
/// </summary>
public sealed class DriverHostRegistry
{
    private readonly Dictionary<string, Func<IIndustrialDriver>> _factories;
    private readonly Dictionary<string, DriverDescriptor> _descriptors;
    private readonly Dictionary<string, IReadOnlyList<string>> _transports;
    private readonly Dictionary<string, DriverManifest>? _manifests;

    public DriverHostRegistry(
        IEnumerable<Func<IIndustrialDriver>> factories,
        IReadOnlyDictionary<string, DriverManifest>? manifests = null,
        DriverHostPlatform? hostPlatform = null)
    {
        ArgumentNullException.ThrowIfNull(factories);

        _factories = new Dictionary<string, Func<IIndustrialDriver>>(StringComparer.Ordinal);
        _descriptors = new Dictionary<string, DriverDescriptor>(StringComparer.Ordinal);
        _transports = new Dictionary<string, IReadOnlyList<string>>(StringComparer.Ordinal);
        _manifests = manifests is null
            ? null
            : new Dictionary<string, DriverManifest>(manifests, StringComparer.Ordinal);

        foreach (var factory in factories)
        {
            ArgumentNullException.ThrowIfNull(factory);
            var driver = factory() ?? throw new DriverRegistrationException("驱动工厂返回了空实例");
            var descriptor = ValidateDescriptor(driver);

            if (!_factories.TryAdd(descriptor.DriverId, factory))
            {
                throw new DriverRegistrationException($"驱动 {descriptor.DriverId} 重复注册");
            }

            if (_manifests is not null)
            {
                ValidateManifest(descriptor, _manifests, hostPlatform);
                _transports.Add(descriptor.DriverId, _manifests[descriptor.DriverId].Transports ?? []);
            }

            _descriptors.Add(descriptor.DriverId, descriptor);
        }
    }

    public IReadOnlyCollection<DriverDescriptor> Descriptors => _descriptors.Values;

    public bool IsRegistered(string driverId) => _factories.ContainsKey(driverId);

    public bool UsesTransport(string driverId, string transport) =>
        _transports.TryGetValue(driverId, out var transports) &&
        transports.Contains(transport, StringComparer.OrdinalIgnoreCase);

    public DriverDescriptor Describe(string driverId) => _descriptors.TryGetValue(driverId, out var descriptor)
        ? descriptor
        : throw new DriverRegistrationException($"宿主未注册驱动：{driverId}");

    public IIndustrialDriver Create(string driverId)
    {
        if (!_factories.TryGetValue(driverId, out var factory))
        {
            throw new DriverRegistrationException($"宿主未注册驱动：{driverId}");
        }

        var driver = factory() ?? throw new DriverRegistrationException($"驱动 {driverId} 的工厂返回了空实例");
        var descriptor = ValidateDescriptor(driver);
        if (!string.Equals(descriptor.DriverId, driverId, StringComparison.Ordinal) ||
            !_descriptors.TryGetValue(driverId, out var cached) ||
            !EquivalentDescriptor(descriptor, cached))
        {
            throw new DriverRegistrationException($"驱动 {driverId} 的工厂实例与认证描述不一致");
        }

        if (_manifests is not null) ValidateManifest(descriptor, _manifests, hostPlatform: null);

        return driver;
    }

    private static bool EquivalentDescriptor(DriverDescriptor left, DriverDescriptor right) =>
        string.Equals(left.DriverId, right.DriverId, StringComparison.Ordinal) &&
        string.Equals(left.ProtocolFamily, right.ProtocolFamily, StringComparison.Ordinal) &&
        string.Equals(left.DriverVersion, right.DriverVersion, StringComparison.Ordinal) &&
        left.SchemaVersions.Order().SequenceEqual(right.SchemaVersions.Order()) &&
        HaveSameValues(left.Operations, right.Operations);

    /// <summary>
    /// 在配置预检时检查目标平台；延迟到此处可让空配置宿主先完成安全启动。
    /// </summary>
    public void ValidatePlatform(string driverId, DriverHostPlatform hostPlatform)
    {
        ArgumentNullException.ThrowIfNull(hostPlatform);
        if (!_descriptors.TryGetValue(driverId, out var descriptor) ||
            _manifests is null ||
            !_manifests.TryGetValue(driverId, out var manifest))
        {
            throw new DriverRegistrationException($"驱动 {driverId} 缺少认证 Manifest");
        }

        ValidatePlatform(descriptor, manifest, hostPlatform);
    }

    private static DriverDescriptor ValidateDescriptor(IIndustrialDriver driver)
    {
        var declared = driver.Descriptor ?? throw new DriverRegistrationException("驱动描述不能为空");
        if (string.IsNullOrWhiteSpace(declared.ProtocolFamily) ||
            string.IsNullOrWhiteSpace(declared.DriverId) ||
            string.IsNullOrWhiteSpace(declared.DriverVersion) ||
            declared.SchemaVersions is null || declared.SchemaVersions.Count == 0 ||
            declared.SchemaVersions.Any(version => version <= 0))
        {
            throw new DriverRegistrationException("驱动描述缺少必要元数据");
        }

        var implementedOperations = GetImplementedOperations(driver);
        if (!HaveSameValues(declared.Operations, implementedOperations))
        {
            throw new DriverRegistrationException($"驱动 {declared.DriverId} 声明的操作与实现接口不一致");
        }

        // 规范化操作集合，避免调用方因声明列表顺序而产生不同的宿主视图。
        return declared with { Operations = implementedOperations };
    }

    private static void ValidateManifest(
        DriverDescriptor descriptor,
        Dictionary<string, DriverManifest> manifests,
        DriverHostPlatform? hostPlatform)
    {
        if (!manifests.TryGetValue(descriptor.DriverId, out var manifest))
        {
            throw new DriverRegistrationException($"驱动 {descriptor.DriverId} 缺少认证 Manifest");
        }

        if (string.IsNullOrWhiteSpace(manifest.ProtocolFamily) ||
            string.IsNullOrWhiteSpace(manifest.DriverId) ||
            string.IsNullOrWhiteSpace(manifest.DriverVersion) ||
            manifest.SchemaVersion <= 0 ||
            manifest.Operations is null || manifest.Transports is null)
        {
            throw new DriverRegistrationException($"驱动 {descriptor.DriverId} 的 Manifest 缺少必要元数据");
        }

        if (!string.Equals(manifest.DriverId, descriptor.DriverId, StringComparison.Ordinal) ||
            !string.Equals(manifest.ProtocolFamily, descriptor.ProtocolFamily, StringComparison.Ordinal) ||
            !string.Equals(manifest.DriverVersion, descriptor.DriverVersion, StringComparison.Ordinal) ||
            !descriptor.SchemaVersions.Contains(manifest.SchemaVersion) ||
            !HaveSameValues(manifest.Operations, descriptor.Operations))
        {
            throw new DriverRegistrationException($"驱动 {descriptor.DriverId} 与认证 Manifest 不一致");
        }

        if (hostPlatform is not null) ValidatePlatform(descriptor, manifest, hostPlatform);
    }

    private static void ValidatePlatform(
        DriverDescriptor descriptor,
        DriverManifest manifest,
        DriverHostPlatform hostPlatform)
    {
        if (manifest.Platforms is null ||
            !manifest.Platforms.TryGetValue(hostPlatform.HostKind, out var supportedPlatforms) ||
            supportedPlatforms is null ||
            !supportedPlatforms.Contains(hostPlatform.RuntimeIdentifier, StringComparer.OrdinalIgnoreCase))
        {
            throw new DriverRegistrationException(
                $"驱动 {descriptor.DriverId} 未获准在 {hostPlatform.HostKind}/{hostPlatform.RuntimeIdentifier} 运行");
        }
    }

    private static List<string> GetImplementedOperations(IIndustrialDriver driver)
    {
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
        return operations;
    }

    private static bool HaveSameValues(IReadOnlyList<string>? actual, IReadOnlyList<string> expected) =>
        actual is not null && actual.Count == expected.Count &&
        actual.Order(StringComparer.Ordinal).SequenceEqual(expected.Order(StringComparer.Ordinal));
}
