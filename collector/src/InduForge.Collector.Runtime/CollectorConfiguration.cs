using System.Security.Cryptography;
using System.Runtime.InteropServices;
using System.Text;
using System.Text.Json;
using System.Text.RegularExpressions;
using Microsoft.Win32.SafeHandles;

namespace InduForge.Collector.Runtime;

/// <summary>资源和凭据解析边界。SecretValue 不公开内容，也不会被默认日志格式化。</summary>
public interface ICollectorResourceResolver
{
    ValueTask<CollectorResource> ResolveResourceAsync(string reference, CancellationToken cancellationToken);
}

public interface ICollectorSecretResolver
{
    ValueTask<CollectorSecret> ResolveSecretAsync(string reference, CancellationToken cancellationToken);
}

public sealed class CollectorResource : IDisposable
{
    private readonly JsonDocument _document;
    internal CollectorResource(JsonDocument document) => _document = document;
    public JsonElement Value => _document.RootElement.Clone();
    public override string ToString() => "CollectorResource { redacted }";
    public void Dispose() => _document.Dispose();
}

public sealed class CollectorSecret : IDisposable
{
    private readonly JsonDocument _document;
    internal CollectorSecret(JsonDocument document) => _document = document;
    internal JsonElement Value => _document.RootElement.Clone();
    public override string ToString() => "CollectorSecret { redacted }";
    public void Dispose() => _document.Dispose();
}

/// <summary>
/// 受限站点索引：资源值可内嵌，secret 仅引用同目录的普通文件。索引本身从不承载 secret 值。
/// </summary>
public sealed class StrictCollectorIndexResolver : ICollectorResourceResolver, ICollectorSecretResolver, IDisposable
{
    private const int MaximumResourceBytes = 1024 * 1024;
    private readonly Dictionary<string, CollectorResource> _resources;
    private readonly Dictionary<string, string> _secrets;
    private readonly string _directory;

    private StrictCollectorIndexResolver(string directory, Dictionary<string, CollectorResource> resources, Dictionary<string, string> secrets)
    {
        _directory = directory;
        _resources = resources;
        _secrets = secrets;
    }

    public static async Task<StrictCollectorIndexResolver> OpenAsync(string indexPath, bool production = false, CancellationToken cancellationToken = default)
    {
        var bytes = await SecureFile.ReadAsync(indexPath, MaximumResourceBytes, requireReadOnly: production, cancellationToken).ConfigureAwait(false);
        using var document = StrictJson.Parse(bytes);
        var root = document.RootElement;
        StrictJson.RequireObject(root, "index");
        StrictJson.RequireOnly(root, "schemaVersion", "resources", "secrets");
        StrictJson.RequireString(root, "schemaVersion", "collector-runtime-index.v1");
        var resources = new Dictionary<string, CollectorResource>(StringComparer.Ordinal);
        if (!root.TryGetProperty("resources", out var resourceValues) || resourceValues.ValueKind != JsonValueKind.Object) throw ConfigurationError.Invalid();
        foreach (var entry in resourceValues.EnumerateObject())
        {
            StrictJson.RequireResourceReference(entry.Name);
            var serialized = JsonSerializer.SerializeToUtf8Bytes(entry.Value);
            if (serialized.Length > MaximumResourceBytes) throw ConfigurationError.Invalid();
            resources.Add(entry.Name, new CollectorResource(StrictJson.Parse(serialized)));
        }
        var secrets = new Dictionary<string, string>(StringComparer.Ordinal);
        if (!root.TryGetProperty("secrets", out var secretValues) || secretValues.ValueKind != JsonValueKind.Object) throw ConfigurationError.Invalid();
        foreach (var entry in secretValues.EnumerateObject())
        {
            StrictJson.RequireSecretReference(entry.Name);
            if (entry.Value.ValueKind != JsonValueKind.String) throw ConfigurationError.Invalid();
            var relative = entry.Value.GetString();
            if (string.IsNullOrWhiteSpace(relative) || Path.IsPathRooted(relative) || relative.Contains('\0') || relative.Split(['/', '\\']).Any(segment => segment is "" or "." or "..")) throw ConfigurationError.Invalid();
            secrets.Add(entry.Name, relative);
        }
        return new StrictCollectorIndexResolver(Path.GetDirectoryName(Path.GetFullPath(indexPath))!, resources, secrets);
    }

    public ValueTask<CollectorResource> ResolveResourceAsync(string reference, CancellationToken cancellationToken)
    {
        cancellationToken.ThrowIfCancellationRequested();
        if (!_resources.TryGetValue(reference, out var value)) throw ConfigurationError.Invalid();
        return ValueTask.FromResult(value);
    }

    public async ValueTask<CollectorSecret> ResolveSecretAsync(string reference, CancellationToken cancellationToken)
    {
        if (!_secrets.TryGetValue(reference, out var relative)) throw ConfigurationError.Invalid();
        var path = Path.GetFullPath(Path.Combine(_directory, relative));
        if (!path.StartsWith(_directory + Path.DirectorySeparatorChar, StringComparison.Ordinal)) throw ConfigurationError.Invalid();
        var bytes = await SecureFile.ReadAsync(path, MaximumResourceBytes, requireReadOnly: true, cancellationToken).ConfigureAwait(false);
        return new CollectorSecret(StrictJson.Parse(bytes));
    }

    public void Dispose()
    {
        foreach (var resource in _resources.Values) resource.Dispose();
    }
}

public sealed record CollectorArtifact(
    string ArtifactId, int ArtifactRevision, string ProjectId, string CollectorVersion,
    IReadOnlyList<CollectorConnection> Connections, IReadOnlyList<CollectorPoint> Points);
public sealed record CollectorConnection(string ConnectionId, string ProtocolFamily, string DriverId, string DriverVersion, int SchemaVersion, bool Enabled, IReadOnlyList<string> RequiredOperations, Acquisition Acquisition);
public sealed record CollectorPoint(string DatapointId, string ConnectionId, string VariableId, string DataType, int ElementCount, JsonElement Address, JsonElement ReadOptions, bool Enabled, Acquisition Acquisition);
public sealed record Acquisition(int IntervalMs, decimal Deadband, bool ChangeOnly);
public sealed record CollectorBinding(string BindingId, int BindingRevision, string DeploymentId, string AccountId, string ArtifactId, int ArtifactRevision, string ArtifactDigest, string CollectorId, string OwnerId, long Epoch, IReadOnlyDictionary<string, BindingConnection> Connections, string NatsResourceRef, string NatsSecretRef, long MaxBytes, long HighWatermarkBytes, long DiagnosticReserveBytes);
public sealed record BindingConnection(string ResourceRef, IReadOnlyList<string> SecretRefs);
public sealed record CollectorLoadedConfiguration(CollectorArtifact Artifact, CollectorBinding Binding);

public static class CollectorConfigurationLoader
{
    private const int ArtifactLimit = 16 * 1024 * 1024;
    private const int BindingLimit = 1024 * 1024;

    public static async Task<CollectorLoadedConfiguration> LoadAsync(string artifactPath, string bindingPath, bool production, CancellationToken cancellationToken)
    {
        var artifactBytes = await SecureFile.ReadAsync(artifactPath, ArtifactLimit, production, cancellationToken).ConfigureAwait(false);
        var bindingBytes = await SecureFile.ReadAsync(bindingPath, BindingLimit, production, cancellationToken).ConfigureAwait(false);
        using var artifactDocument = StrictJson.Parse(artifactBytes);
        using var bindingDocument = StrictJson.Parse(bindingBytes);
        var artifact = ParseArtifact(artifactDocument.RootElement);
        var binding = ParseBinding(bindingDocument.RootElement);
        var digest = "sha256:" + Convert.ToHexString(SHA256.HashData(artifactBytes)).ToLowerInvariant();
        if (!CryptographicOperations.FixedTimeEquals(Encoding.ASCII.GetBytes(digest), Encoding.ASCII.GetBytes(binding.ArtifactDigest)) ||
            !string.Equals(artifact.ArtifactId, binding.ArtifactId, StringComparison.Ordinal) || artifact.ArtifactRevision != binding.ArtifactRevision)
            throw ConfigurationError.Invalid();
        var enabled = artifact.Connections.Where(connection => connection.Enabled).ToDictionary(connection => connection.ConnectionId, StringComparer.Ordinal);
        if (enabled.Count != binding.Connections.Count || enabled.Keys.Except(binding.Connections.Keys, StringComparer.Ordinal).Any()) throw ConfigurationError.Invalid();
        if (artifact.Points.Where(point => point.Enabled).Any(point => !enabled.ContainsKey(point.ConnectionId))) throw ConfigurationError.Invalid();
        return new CollectorLoadedConfiguration(artifact, binding);
    }

    private static CollectorArtifact ParseArtifact(JsonElement root)
    {
        StrictJson.RequireObject(root, "artifact");
        StrictJson.RequireOnly(root, "schemaVersion", "artifactId", "artifactRevision", "projectId", "collectorVersion", "connections", "pointMappings", "wal");
        StrictJson.RequireString(root, "schemaVersion", "collector-runtime-artifact.v1");
        var artifactId = StrictJson.RequireStableId(root, "artifactId");
        var revision = StrictJson.RequirePositiveInt(root, "artifactRevision");
        var projectId = StrictJson.RequireUuid(root, "projectId");
        var collectorVersion = StrictJson.RequireNonEmptyString(root, "collectorVersion");
        var wal = StrictJson.RequireProperty(root, "wal");
        StrictJson.RequireObject(wal, "wal"); StrictJson.RequireOnly(wal, "fsync", "checksum", "commitMarker");
        StrictJson.RequireString(wal, "fsync", "before-publish"); StrictJson.RequireString(wal, "checksum", "crc32c"); StrictJson.RequireString(wal, "commitMarker", "after-jetstream-puback");
        var connectionsElement = StrictJson.RequireArray(root, "connections", minimum: 1);
        var connections = new List<CollectorConnection>(); var ids = new HashSet<string>(StringComparer.Ordinal);
        foreach (var item in connectionsElement.EnumerateArray())
        {
            StrictJson.RequireObject(item, "connection"); StrictJson.RequireOnly(item, "connectionId", "protocolFamily", "driverId", "driverVersion", "schemaVersion", "enabled", "requiredOperations", "defaultAcquisition");
            var id = StrictJson.RequireUuid(item, "connectionId"); if (!ids.Add(id)) throw ConfigurationError.Invalid();
            var operations = StrictJson.RequireStringArray(item, "requiredOperations", minimum: 1, unique: true);
            if (!operations.Contains("point.read", StringComparer.Ordinal)) throw ConfigurationError.Invalid();
            connections.Add(new CollectorConnection(id, StrictJson.RequireStableId(item, "protocolFamily"), StrictJson.RequireStableId(item, "driverId"), StrictJson.RequireNonEmptyString(item, "driverVersion"), StrictJson.RequirePositiveInt(item, "schemaVersion"), StrictJson.RequireBoolean(item, "enabled"), operations, ParseAcquisition(StrictJson.RequireProperty(item, "defaultAcquisition"))));
        }
        var pointsElement = StrictJson.RequireArray(root, "pointMappings", minimum: 1);
        var points = new List<CollectorPoint>(); var pointIds = new HashSet<string>(StringComparer.Ordinal); var pairs = new HashSet<string>(StringComparer.Ordinal);
        foreach (var item in pointsElement.EnumerateArray())
        {
            StrictJson.RequireObject(item, "point"); StrictJson.RequireOnly(item, "datapointId", "connectionId", "variableId", "dataType", "elementCount", "addressSchemaVersion", "address", "readOptions", "acquisitionMode", "acquisitionOverrides", "effectiveAcquisition", "enabled");
            var pointId = StrictJson.RequireUuid(item, "datapointId"); var connectionId = StrictJson.RequireUuid(item, "connectionId"); var variableId = StrictJson.RequireUuid(item, "variableId");
            if (!pointIds.Add(pointId) || !pairs.Add(connectionId + "\u001f" + variableId)) throw ConfigurationError.Invalid();
            var dataType = StrictJson.RequireEnum(item, "dataType", ["bool", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64", "decimal", "string", "bytes", "datetime"]);
            var elementCount = StrictJson.RequireInt32(item, "elementCount", 1, 4096);
            if (dataType == "bytes" && elementCount != 1) throw ConfigurationError.Invalid();
            var address = StrictJson.RequireProperty(item, "address"); StrictJson.RequireObject(address, "address"); if (!address.EnumerateObject().Any()) throw ConfigurationError.Invalid();
            var options = StrictJson.RequireProperty(item, "readOptions"); StrictJson.RequireObject(options, "readOptions");
            var mode = StrictJson.RequireEnum(item, "acquisitionMode", ["inherit", "override"]);
            var effective = ParseAcquisition(StrictJson.RequireProperty(item, "effectiveAcquisition"));
            Acquisition expected;
            var owner = connections.SingleOrDefault(connection => string.Equals(connection.ConnectionId, connectionId, StringComparison.Ordinal));
            if (owner is null) throw ConfigurationError.Invalid();
            if (StrictJson.RequirePositiveInt(item, "addressSchemaVersion") != owner.SchemaVersion) throw ConfigurationError.Invalid();
            if (mode == "inherit") { if (item.TryGetProperty("acquisitionOverrides", out _)) throw ConfigurationError.Invalid(); expected = owner.Acquisition; }
            else { if (!item.TryGetProperty("acquisitionOverrides", out var overrides)) throw ConfigurationError.Invalid(); expected = MergeAcquisition(owner.Acquisition, overrides); }
            if (effective != expected) throw ConfigurationError.Invalid();
            points.Add(new CollectorPoint(pointId, connectionId, variableId, dataType, elementCount, address.Clone(), options.Clone(), StrictJson.RequireBoolean(item, "enabled"), effective));
        }
        return new CollectorArtifact(artifactId, revision, projectId, collectorVersion, connections, points);
    }

    private static CollectorBinding ParseBinding(JsonElement root)
    {
        StrictJson.RequireObject(root, "binding"); StrictJson.RequireOnly(root, "schemaVersion", "bindingId", "bindingRevision", "deploymentId", "accountId", "artifact", "collectorId", "ownership", "connections", "nats", "walCapacity");
        StrictJson.RequireString(root, "schemaVersion", "collector-runtime-binding.v1");
        var artifact = StrictJson.RequireProperty(root, "artifact"); StrictJson.RequireObject(artifact, "artifact"); StrictJson.RequireOnly(artifact, "artifactId", "artifactRevision", "artifactDigest");
        var ownership = StrictJson.RequireProperty(root, "ownership"); StrictJson.RequireObject(ownership, "ownership"); StrictJson.RequireOnly(ownership, "ownerId", "epoch");
        var connections = new Dictionary<string, BindingConnection>(StringComparer.Ordinal);
        foreach (var item in StrictJson.RequireArray(root, "connections", minimum: 1).EnumerateArray())
        {
            StrictJson.RequireObject(item, "binding connection"); StrictJson.RequireOnly(item, "connectionId", "resourceRef", "secretRefs"); var id = StrictJson.RequireUuid(item, "connectionId");
            var resource = StrictJson.RequireResourceReference(StrictJson.RequireNonEmptyString(item, "resourceRef"));
            var secrets = item.TryGetProperty("secretRefs", out var values) ? StrictJson.RequireReferenceArray(values, secret: true) : [];
            if (!connections.TryAdd(id, new BindingConnection(resource, secrets))) throw ConfigurationError.Invalid();
        }
        var nats = StrictJson.RequireProperty(root, "nats"); StrictJson.RequireObject(nats, "nats"); StrictJson.RequireOnly(nats, "serverResourceRef", "credentialSecretRef", "rawSubjectPrefix");
        StrictJson.RequireString(nats, "rawSubjectPrefix", "data.raw");
        var capacity = StrictJson.RequireProperty(root, "walCapacity"); StrictJson.RequireObject(capacity, "wal capacity"); StrictJson.RequireOnly(capacity, "maxBytes", "highWatermarkBytes", "diagnosticReserveBytes", "dataGapPolicy");
        var max = StrictJson.RequireInt64(capacity, "maxBytes", 1_048_576, 1_073_741_824); var high = StrictJson.RequireInt64(capacity, "highWatermarkBytes", 1, long.MaxValue); var reserve = StrictJson.RequireInt64(capacity, "diagnosticReserveBytes", 1, long.MaxValue);
        if (high >= max || reserve >= max) throw ConfigurationError.Invalid(); StrictJson.RequireString(capacity, "dataGapPolicy", "emit-alarm-event");
        var digest = StrictJson.RequireString(artifact, "artifactDigest"); if (!Regex.IsMatch(digest, "^sha256:[a-f0-9]{64}$", RegexOptions.CultureInvariant)) throw ConfigurationError.Invalid();
        return new CollectorBinding(StrictJson.RequireStableId(root, "bindingId"), StrictJson.RequirePositiveInt(root, "bindingRevision"), StrictJson.RequireStableId(root, "deploymentId"), StrictJson.RequireStableId(root, "accountId"), StrictJson.RequireStableId(artifact, "artifactId"), StrictJson.RequirePositiveInt(artifact, "artifactRevision"), digest, StrictJson.RequireStableId(root, "collectorId"), StrictJson.RequireStableId(ownership, "ownerId"), StrictJson.RequirePositiveInt(ownership, "epoch"), connections, StrictJson.RequireResourceReference(StrictJson.RequireNonEmptyString(nats, "serverResourceRef")), StrictJson.RequireSecretReference(StrictJson.RequireNonEmptyString(nats, "credentialSecretRef")), max, high, reserve);
    }

    private static Acquisition ParseAcquisition(JsonElement element)
    {
        StrictJson.RequireObject(element, "acquisition"); StrictJson.RequireOnly(element, "intervalMs", "deadband", "changeOnly");
        var interval = StrictJson.RequireInt32(element, "intervalMs", 1, 86_400_000); var deadband = StrictJson.RequireDecimal(element, "deadband", 0); var changeOnly = StrictJson.RequireBoolean(element, "changeOnly");
        return new Acquisition(interval, deadband, changeOnly);
    }
    private static Acquisition MergeAcquisition(Acquisition defaults, JsonElement overrides)
    {
        StrictJson.RequireObject(overrides, "acquisition overrides"); StrictJson.RequireOnly(overrides, "intervalMs", "deadband", "changeOnly"); if (!overrides.EnumerateObject().Any()) throw ConfigurationError.Invalid();
        return new Acquisition(overrides.TryGetProperty("intervalMs", out var i) ? StrictJson.RequireInt32Value(i, 1, 86_400_000) : defaults.IntervalMs, overrides.TryGetProperty("deadband", out var d) ? StrictJson.RequireDecimalValue(d, 0) : defaults.Deadband, overrides.TryGetProperty("changeOnly", out var c) ? StrictJson.RequireBooleanValue(c) : defaults.ChangeOnly);
    }
}

internal static class ConfigurationError
{
    // 对外只公开稳定错误码，避免泄露挂载路径、摘要或凭据内容。
    internal static CollectorRuntimeConfigurationException Invalid() => new("COLLECTOR_CONFIGURATION_INVALID");
}

internal static class SecureFile
{
    internal static async Task<byte[]> ReadAsync(string path, int maximum, bool requireReadOnly, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(path) || !Path.IsPathFullyQualified(path) || path.Contains('\0')) throw ConfigurationError.Invalid();
        if (OperatingSystem.IsWindows())
        {
            await using var verified = WindowsSecureFile.OpenVerified(path, requireReadOnly);
            if (verified.Length > maximum) throw ConfigurationError.Invalid();
            return await ReadExactlyAsync(verified, maximum, cancellationToken).ConfigureAwait(false);
        }
        RejectLinkedParents(path);
        var info = new FileInfo(path);
        if (!info.Exists || info.LinkTarget is not null || (info.Attributes & FileAttributes.ReparsePoint) != 0) throw ConfigurationError.Invalid();
        if (requireReadOnly && !OperatingSystem.IsWindows())
        {
            var mode = File.GetUnixFileMode(path);
            if ((mode & (UnixFileMode.GroupRead | UnixFileMode.OtherRead)) != 0) throw ConfigurationError.Invalid();
        }
        if (requireReadOnly && !OperatingSystem.IsWindows() && !IsMostSpecificMountReadOnly(path)) throw ConfigurationError.Invalid();
        if (info.Length > maximum) throw ConfigurationError.Invalid();
        await using var stream = OpenNoFollow(path);
        if (stream.Length != info.Length || stream.Length > maximum) throw ConfigurationError.Invalid();
        var bytes = await ReadExactlyAsync(stream, maximum, cancellationToken).ConfigureAwait(false);
        var post = new FileInfo(path);
        if (!post.Exists || post.LinkTarget is not null || post.Length != bytes.Length || post.LastWriteTimeUtc != info.LastWriteTimeUtc) throw ConfigurationError.Invalid();
        return bytes;
    }

    private static async Task<byte[]> ReadExactlyAsync(FileStream stream, int maximum, CancellationToken cancellationToken)
    {
        if (stream.Length > maximum) throw ConfigurationError.Invalid();
        var bytes = new byte[stream.Length]; var read = 0;
        while (read < bytes.Length) { var amount = await stream.ReadAsync(bytes.AsMemory(read), cancellationToken).ConfigureAwait(false); if (amount == 0) throw ConfigurationError.Invalid(); read += amount; }
        return bytes;
    }

    private static FileStream OpenNoFollow(string path)
    {
        if (OperatingSystem.IsWindows()) return new FileStream(path, FileMode.Open, FileAccess.Read, FileShare.Read, 64 * 1024, FileOptions.SequentialScan);
        var descriptor = open(path, 0 | (OperatingSystem.IsMacOS() ? 0x100 : 0x20000)); // O_RDONLY | O_NOFOLLOW
        if (descriptor < 0) throw ConfigurationError.Invalid();
        return new FileStream(new SafeFileHandle((IntPtr)descriptor, ownsHandle: true), FileAccess.Read, 64 * 1024, isAsync: false);
    }

    private static void RejectLinkedParents(string path)
    {
        var current = Path.GetDirectoryName(path);
        while (!string.IsNullOrEmpty(current))
        {
            var info = new DirectoryInfo(current);
            if (info.LinkTarget is not null || (info.Attributes & FileAttributes.ReparsePoint) != 0) throw ConfigurationError.Invalid();
            current = info.Parent?.FullName;
        }
    }

    private static bool IsMostSpecificMountReadOnly(string path)
    {
        // Linux mountinfo 的 mount point 位于可选字段之前的第 5 列；最长前缀就是最具体 mount。
        if (!OperatingSystem.IsLinux()) return true; // Windows ACL 无法由 .NET 完整判断，运行部署需额外审计。
        try
        {
            var full = Path.GetFullPath(path); string? bestMount = null; string? bestOptions = null;
            foreach (var line in File.ReadLines("/proc/self/mountinfo"))
            {
                var fields = line.Split(' '); if (fields.Length < 6) continue;
                var mount = fields[4].Replace("\\040", " ", StringComparison.Ordinal);
                if ((full == mount || full.StartsWith(mount.EndsWith('/') ? mount : mount + "/", StringComparison.Ordinal)) && (bestMount is null || mount.Length > bestMount.Length)) { bestMount = mount; bestOptions = fields[5]; }
            }
            return bestOptions?.Split(',').Contains("ro", StringComparer.Ordinal) == true;
        }
        catch (IOException) { return false; }
    }

#pragma warning disable CA2101 // LPUTF8Str 显式指定 Unix 路径封送，分析器无法识别该组合。
    [DllImport("libc", CharSet = CharSet.Ansi, SetLastError = true)]
    private static extern int open([MarshalAs(UnmanagedType.LPUTF8Str)] string path, int flags);
#pragma warning restore CA2101
}

/// <summary>
/// Windows 生产挂载的文件必须以 handle 为准验证。路径属性在打开前后都可能被 junction 替换，
/// 因此只接受最终对象与请求路径一致、且当前进程无法取得写入或删除能力的普通文件。
/// </summary>
internal static class WindowsSecureFile
{
    private const uint FileAttributeReparsePoint = 0x400;
    private const uint GenericWrite = 0x40000000;
    private const uint Delete = 0x00010000;
    private const uint WriteDac = 0x00040000;
    private const uint WriteOwner = 0x00080000;
    private const uint FileAddFile = 0x00000002;
    private const uint FileDeleteChild = 0x00000040;
    private const uint ShareAll = 0x00000007;
    private const uint OpenExisting = 3;
    private const uint FileFlagBackupSemantics = 0x02000000;
    private const int FileBasicInfo = 0;
    private const int ErrorAccessDenied = 5;

    internal static FileStream OpenVerified(string path, bool requireReadOnly)
    {
        try
        {
            // 必须在受保护读取句柄建立前探测，否则 FileShare.Read 会把自身的拒绝共享误判成 ACL 拒绝。
            if (requireReadOnly) VerifyNoMutationCapability(path);
            var handle = File.OpenHandle(path, FileMode.Open, FileAccess.Read, FileShare.Read, FileOptions.SequentialScan);
            try
            {
                VerifyFinalObject(handle, path);
                return new FileStream(handle, FileAccess.Read, 64 * 1024, isAsync: false);
            }
            catch
            {
                handle.Dispose();
                throw;
            }
        }
        catch (CollectorRuntimeConfigurationException) { throw; }
        catch (Exception exception) when (exception is IOException or UnauthorizedAccessException or ArgumentException or NotSupportedException)
        {
            throw ConfigurationError.Invalid();
        }
    }

    private static void VerifyFinalObject(SafeFileHandle handle, string requestedPath)
    {
        if (!GetFileInformationByHandleEx(handle, FileBasicInfo, out var basic, (uint)Marshal.SizeOf<FileBasicInformation>()) || (basic.FileAttributes & FileAttributeReparsePoint) != 0)
            throw ConfigurationError.Invalid();
        var final = new StringBuilder(32_768);
        var length = GetFinalPathNameByHandleW(handle, final, (uint)final.Capacity, 0);
        if (length == 0 || length >= final.Capacity || !string.Equals(NormalizeFinalPath(final.ToString()), NormalizeFinalPath(requestedPath), StringComparison.OrdinalIgnoreCase))
            throw ConfigurationError.Invalid();
    }

    private static string NormalizeFinalPath(string path)
    {
        var full = Path.GetFullPath(path).Replace('/', '\\');
        if (full.StartsWith("\\\\?\\UNC\\", StringComparison.OrdinalIgnoreCase)) return full;
        return full.StartsWith("\\\\?\\", StringComparison.Ordinal) ? full : "\\\\?\\" + full;
    }

    private static void VerifyNoMutationCapability(string path)
    {
        // 这是 capability 探测，不修改文件。可打开即说明 ACL 允许当前进程篡改或替换，生产环境应 fail-closed。
        if (ProbeMutation(path, GenericWrite, 0) != MutationCapability.Denied ||
            ProbeMutation(path, Delete, 0) != MutationCapability.Denied ||
            ProbeMutation(path, WriteDac, 0) != MutationCapability.Denied ||
            ProbeMutation(path, WriteOwner, 0) != MutationCapability.Denied) throw ConfigurationError.Invalid();
        var parent = Path.GetDirectoryName(path);
        if (string.IsNullOrEmpty(parent) ||
            ProbeMutation(parent, GenericWrite, FileFlagBackupSemantics) != MutationCapability.Denied ||
            ProbeMutation(parent, Delete, FileFlagBackupSemantics) != MutationCapability.Denied ||
            ProbeMutation(parent, WriteDac, FileFlagBackupSemantics) != MutationCapability.Denied ||
            ProbeMutation(parent, WriteOwner, FileFlagBackupSemantics) != MutationCapability.Denied ||
            ProbeMutation(parent, FileAddFile, FileFlagBackupSemantics) != MutationCapability.Denied ||
            ProbeMutation(parent, FileDeleteChild, FileFlagBackupSemantics) != MutationCapability.Denied) throw ConfigurationError.Invalid();
    }

    private static MutationCapability ProbeMutation(string path, uint desiredAccess, uint flags)
    {
        using var handle = CreateFileW(path, desiredAccess, ShareAll, IntPtr.Zero, OpenExisting, flags, IntPtr.Zero);
        if (!handle.IsInvalid) return MutationCapability.Allowed;
        // 仅 ERROR_ACCESS_DENIED 能证明当前令牌缺少该具体能力；共享冲突、I/O 或其他错误都 fail-closed。
        return Marshal.GetLastWin32Error() == ErrorAccessDenied ? MutationCapability.Denied : MutationCapability.Unknown;
    }

    private enum MutationCapability { Allowed, Denied, Unknown }

    [StructLayout(LayoutKind.Sequential)]
    private struct FileBasicInformation
    {
        internal long CreationTime;
        internal long LastAccessTime;
        internal long LastWriteTime;
        internal long ChangeTime;
        internal uint FileAttributes;
    }

    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    [return: MarshalAs(UnmanagedType.Bool)]
    private static extern bool GetFileInformationByHandleEx(SafeFileHandle fileInformation, int fileInformationClass, out FileBasicInformation fileInformationData, uint bufferSize);

 #pragma warning disable CA1838 // Win32 API 返回可变长 UTF-16 路径，缓冲区避免额外托管复制。
    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern uint GetFinalPathNameByHandleW(SafeFileHandle file, StringBuilder path, uint pathLength, uint flags);
 #pragma warning restore CA1838

    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern SafeFileHandle CreateFileW(string fileName, uint desiredAccess, uint shareMode, IntPtr securityAttributes, uint creationDisposition, uint flagsAndAttributes, IntPtr templateFile);
}

internal static class StrictJson
{
    private static readonly Regex Stable = new("^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$", RegexOptions.CultureInvariant);
    private static readonly Regex Uuid = new("^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$", RegexOptions.CultureInvariant);
    internal static JsonDocument Parse(ReadOnlyMemory<byte> bytes)
    {
        try { RejectDuplicates(bytes.Span); return JsonDocument.Parse(bytes); }
        catch (JsonException) { throw ConfigurationError.Invalid(); }
        catch (InvalidOperationException) { throw ConfigurationError.Invalid(); }
    }
    internal static void RequireObject(JsonElement value, string _) { if (value.ValueKind != JsonValueKind.Object) throw ConfigurationError.Invalid(); }
    internal static JsonElement RequireProperty(JsonElement value, string name) => value.TryGetProperty(name, out var property) ? property : throw ConfigurationError.Invalid();
    /// <summary>仅拒绝未声明字段。必填字段由各读取器在实际读取时强制，避免把 Schema optional 误当 required。</summary>
    internal static void RequireOnly(JsonElement value, params string[] names)
    {
        var allowed = new HashSet<string>(names, StringComparer.Ordinal);
        foreach (var p in value.EnumerateObject()) if (!allowed.Contains(p.Name)) throw ConfigurationError.Invalid();
    }
    internal static string RequireString(JsonElement value, string name, string? expected = null) { var text = RequireNonEmptyString(value, name); if (expected is not null && !string.Equals(text, expected, StringComparison.Ordinal)) throw ConfigurationError.Invalid(); return text; }
    internal static string RequireNonEmptyString(JsonElement value, string name) { if (!value.TryGetProperty(name, out var property) || property.ValueKind != JsonValueKind.String || string.IsNullOrEmpty(property.GetString())) throw ConfigurationError.Invalid(); return property.GetString()!; }
    internal static string RequireStableId(JsonElement value, string name) => RequireStableId(RequireNonEmptyString(value, name));
    internal static string RequireStableId(string value) => !string.IsNullOrEmpty(value) && Stable.IsMatch(value) ? value : throw ConfigurationError.Invalid();
    internal static string RequireUuid(JsonElement value, string name) { var text = RequireNonEmptyString(value, name); return Uuid.IsMatch(text) ? text : throw ConfigurationError.Invalid(); }
    internal static int RequirePositiveInt(JsonElement value, string name) => RequireInt32(value, name, 1, int.MaxValue);
    internal static int RequireInt32(JsonElement value, string name, int minimum, int maximum) => value.TryGetProperty(name, out var property) ? RequireInt32Value(property, minimum, maximum) : throw ConfigurationError.Invalid();
    internal static int RequireInt32Value(JsonElement value, int minimum, int maximum) => value.TryGetInt32(out var number) && number >= minimum && number <= maximum ? number : throw ConfigurationError.Invalid();
    internal static long RequireInt64(JsonElement value, string name, long minimum, long maximum) => value.TryGetProperty(name, out var property) && property.TryGetInt64(out var number) && number >= minimum && number <= maximum ? number : throw ConfigurationError.Invalid();
    internal static decimal RequireDecimal(JsonElement value, string name, decimal minimum) => value.TryGetProperty(name, out var property) ? RequireDecimalValue(property, minimum) : throw ConfigurationError.Invalid();
    internal static decimal RequireDecimalValue(JsonElement value, decimal minimum) => value.TryGetDecimal(out var number) && number >= minimum ? number : throw ConfigurationError.Invalid();
    internal static bool RequireBoolean(JsonElement value, string name) => value.TryGetProperty(name, out var p) ? RequireBooleanValue(p) : throw ConfigurationError.Invalid();
    internal static bool RequireBooleanValue(JsonElement value) => value.ValueKind is JsonValueKind.True ? true : value.ValueKind is JsonValueKind.False ? false : throw ConfigurationError.Invalid();
    internal static string RequireEnum(JsonElement value, string name, string[] allowed) { var text = RequireString(value, name); return allowed.Contains(text, StringComparer.Ordinal) ? text : throw ConfigurationError.Invalid(); }
    internal static JsonElement RequireArray(JsonElement value, string name, int minimum) { var array = RequireProperty(value, name); if (array.ValueKind != JsonValueKind.Array || array.GetArrayLength() < minimum) throw ConfigurationError.Invalid(); return array; }
    internal static IReadOnlyList<string> RequireStringArray(JsonElement value, string name, int minimum, bool unique) { var result = RequireArray(value, name, minimum).EnumerateArray().Select(item => item.ValueKind == JsonValueKind.String && !string.IsNullOrEmpty(item.GetString()) ? item.GetString()! : throw ConfigurationError.Invalid()).ToArray(); if (unique && result.Distinct(StringComparer.Ordinal).Count() != result.Length) throw ConfigurationError.Invalid(); return result; }
    internal static string RequireResourceReference(string value) { if (!Regex.IsMatch(value, "^site-resource://[A-Za-z0-9][A-Za-z0-9._:/@-]{0,255}$", RegexOptions.CultureInvariant)) throw ConfigurationError.Invalid(); return value; }
    internal static string RequireSecretReference(string value) { if (!Regex.IsMatch(value, "^secret://[A-Za-z0-9][A-Za-z0-9._:/@-]{0,255}$", RegexOptions.CultureInvariant)) throw ConfigurationError.Invalid(); return value; }
    internal static IReadOnlyList<string> RequireReferenceArray(JsonElement array, bool secret) { if (array.ValueKind != JsonValueKind.Array || array.GetArrayLength() == 0) throw ConfigurationError.Invalid(); var values = array.EnumerateArray().Select(value => value.ValueKind == JsonValueKind.String ? (secret ? RequireSecretReference(value.GetString()!) : RequireResourceReference(value.GetString()!)) : throw ConfigurationError.Invalid()).ToArray(); if (values.Distinct(StringComparer.Ordinal).Count() != values.Length) throw ConfigurationError.Invalid(); return values; }
    private static void RejectDuplicates(ReadOnlySpan<byte> bytes) { var reader = new Utf8JsonReader(bytes, new JsonReaderOptions { CommentHandling = JsonCommentHandling.Disallow }); if (!reader.Read()) throw ConfigurationError.Invalid(); Walk(ref reader); if (reader.Read()) throw ConfigurationError.Invalid(); }
    private static void Walk(ref Utf8JsonReader reader) { if (reader.TokenType == JsonTokenType.StartObject) { var names = new HashSet<string>(StringComparer.Ordinal); while (reader.Read() && reader.TokenType != JsonTokenType.EndObject) { if (reader.TokenType != JsonTokenType.PropertyName || !names.Add(reader.GetString() ?? string.Empty) || !reader.Read()) throw ConfigurationError.Invalid(); Walk(ref reader); } if (reader.TokenType != JsonTokenType.EndObject) throw ConfigurationError.Invalid(); } else if (reader.TokenType == JsonTokenType.StartArray) { while (reader.Read() && reader.TokenType != JsonTokenType.EndArray) Walk(ref reader); if (reader.TokenType != JsonTokenType.EndArray) throw ConfigurationError.Invalid(); } }
}
