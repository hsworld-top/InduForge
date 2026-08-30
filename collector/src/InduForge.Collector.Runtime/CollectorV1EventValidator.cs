using System.Globalization;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.RegularExpressions;

namespace InduForge.Collector.Runtime;

/// <summary>
/// Collector 会落盘后重放的消息必须就是 Runtime V1 正式事件。此处使用本地严格校验，
/// 避免运行时依赖网络 schema 解析器，也避免不完整或伪造的事件进入 WAL。
/// </summary>
internal static partial class CollectorV1EventValidator
{
    private const char UnitSeparator = '\x1f';

    public static void Validate(WalAppendRequest request, long sequence)
    {
        ArgumentNullException.ThrowIfNull(request);
        if (request.Epoch < 1) throw Invalid("epoch 必须大于或等于 1");
        if (!EventIdPattern().IsMatch(request.EventId)) throw Invalid("eventId 必须是 64 位小写十六进制");
        if (!StableIdPattern().IsMatch(request.OwnerId)) throw Invalid("ownerId 必须是 stableId");

        RejectDuplicateProperties(request.Payload.Span);
        using var document = JsonDocument.Parse(request.Payload);
        var root = document.RootElement;
        if (root.ValueKind != JsonValueKind.Object) throw Invalid("事件 payload 必须是 JSON 对象");

        if (request.Subject.StartsWith("data.raw.", StringComparison.Ordinal))
        {
            ValidateRaw(root, request, sequence);
            return;
        }

        if (string.Equals(request.Subject, "alarm.event", StringComparison.Ordinal))
        {
            ValidateDataGap(root, request);
            return;
        }

        throw Invalid("subject 不在 Collector V1 上行白名单中");
    }

    internal static string ComputeDataGapEventId(
        string deploymentId,
        string collectorId,
        string connectionId,
        string ownerId,
        long epoch,
        long fromSequence,
        long toSequence,
        string detectedAt,
        string reason)
    {
        if (!StableIdPattern().IsMatch(deploymentId) || !StableIdPattern().IsMatch(collectorId) || !StableIdPattern().IsMatch(ownerId) || !UuidPattern().IsMatch(connectionId))
        {
            throw Invalid("DATA_GAP 事实身份字段不符合契约");
        }
        if (epoch < 1 || fromSequence < 0 || toSequence < fromSequence || !UtcPattern().IsMatch(detectedAt) || !IsValidUtc(detectedAt))
        {
            throw Invalid("DATA_GAP 事实范围或时间不符合契约");
        }
        _ = RequireAllowedDataGapReason(reason);
        return ComputeDigest(["alarm.event.v1", "data-gap", deploymentId, collectorId, connectionId, ownerId, Decimal(epoch), Decimal(fromSequence), Decimal(toSequence), detectedAt, reason]);
    }

    internal static string ComputeRawEventId(string deploymentId, string pointId, string ownerId, long epoch, long sequence) =>
        ComputeDigest(["data.raw.v1", deploymentId, pointId, ownerId, Decimal(epoch), Decimal(sequence)]);

    private static void ValidateRaw(JsonElement root, WalAppendRequest request, long sequence)
    {
        RequireOnly(root, ["schemaVersion", "subject", "eventId", "deploymentId", "accountId", "pointId", "ownerId", "epoch", "sequence", "value", "quality", "sourceTimestamp", "serverTimestamp", "receivedAt", "source"]);
        RequireString(root, "schemaVersion", "data.raw.v1");
        RequireString(root, "subject", request.Subject);
        RequireString(root, "eventId", request.EventId);
        var deploymentId = RequireStableId(root, "deploymentId");
        _ = RequireStableId(root, "accountId");
        var pointId = RequireUuid(root, "pointId");
        RequireString(root, "ownerId", request.OwnerId);
        RequireInt64(root, "epoch", request.Epoch, minimum: 1);
        RequireInt64(root, "sequence", sequence, minimum: 0);
        if (!request.Subject.Equals("data.raw." + pointId, StringComparison.Ordinal)) throw Invalid("raw subject 尾段必须等于 pointId");
        if (!root.TryGetProperty("value", out var value) || !IsFiniteJsonValue(value)) throw Invalid("value 必须是有限 JSON 值");
        RequireEnum(root, "quality", ["good", "bad", "unknown"]);
        _ = RequireUtc(root, "sourceTimestamp");
        _ = RequireUtc(root, "serverTimestamp");
        _ = RequireUtc(root, "receivedAt");
        var source = RequireObject(root, "source");
        RequireOnly(source, ["collectorId", "connectionId", "variableId"]);
        _ = RequireStableId(source, "collectorId");
        _ = RequireUuid(source, "connectionId");
        _ = RequireUuid(source, "variableId");

        RequireDigest(request.EventId, ["data.raw.v1", deploymentId, pointId, request.OwnerId, Decimal(request.Epoch), Decimal(sequence)]);
    }

    private static void ValidateDataGap(JsonElement root, WalAppendRequest request)
    {
        RequireOnly(root, ["schemaVersion", "subject", "eventId", "deploymentId", "accountId", "kind", "operation", "ownerId", "epoch", "sourceTimestamp", "serverTimestamp", "receivedAt", "dataGap"]);
        RequireString(root, "schemaVersion", "alarm.event.v1");
        RequireString(root, "subject", "alarm.event");
        RequireString(root, "eventId", request.EventId);
        var deploymentId = RequireStableId(root, "deploymentId");
        _ = RequireStableId(root, "accountId");
        RequireString(root, "kind", "data-gap");
        RequireString(root, "operation", "DATA_GAP");
        RequireString(root, "ownerId", request.OwnerId);
        RequireInt64(root, "epoch", request.Epoch, minimum: 1);
        _ = RequireUtc(root, "sourceTimestamp");
        _ = RequireUtc(root, "serverTimestamp");
        _ = RequireUtc(root, "receivedAt");

        var dataGap = RequireObject(root, "dataGap");
        RequireOnly(dataGap, ["collectorId", "connectionId", "fromSequence", "toSequence", "fromSourceTimestamp", "toSourceTimestamp", "detectedAt", "reason"]);
        var collectorId = RequireStableId(dataGap, "collectorId");
        var connectionId = RequireUuid(dataGap, "connectionId");
        var fromSequence = RequireNonNegativeInt64(dataGap, "fromSequence");
        var toSequence = RequireNonNegativeInt64(dataGap, "toSequence");
        if (fromSequence > toSequence) throw Invalid("dataGap.fromSequence 不能大于 toSequence");
        RequireOptionalUtcOrNull(dataGap, "fromSourceTimestamp");
        RequireOptionalUtcOrNull(dataGap, "toSourceTimestamp");
        var detectedAt = RequireUtc(dataGap, "detectedAt");
        var reason = RequireEnum(dataGap, "reason", ["wal-capacity", "wal-corruption", "manual-cleanup", "unrecoverable"]);
        var expected = ComputeDataGapEventId(deploymentId, collectorId, connectionId, request.OwnerId, request.Epoch, fromSequence, toSequence, detectedAt, reason);
        RequireDigest(request.EventId, expected);
    }

    private static void RequireOnly(JsonElement objectElement, params string[] allowed)
    {
        if (objectElement.ValueKind != JsonValueKind.Object) throw Invalid("字段必须是 JSON 对象");
        foreach (var property in objectElement.EnumerateObject())
        {
            if (!allowed.Contains(property.Name, StringComparer.Ordinal)) throw Invalid("payload 含有未允许字段: " + property.Name);
        }
        foreach (var required in allowed)
        {
            if (required is "fromSourceTimestamp" or "toSourceTimestamp") continue;
            if (!objectElement.TryGetProperty(required, out _)) throw Invalid("payload 缺少字段: " + required);
        }
    }

    private static string RequireStableId(JsonElement root, string name)
    {
        var value = RequireString(root, name);
        if (!StableIdPattern().IsMatch(value)) throw Invalid(name + " 必须是 stableId");
        return value;
    }

    private static string RequireUuid(JsonElement root, string name)
    {
        var value = RequireString(root, name);
        if (!UuidPattern().IsMatch(value)) throw Invalid(name + " 必须是 canonical lowercase UUID");
        return value;
    }

    private static string RequireString(JsonElement root, string name, string? expected = null)
    {
        if (!root.TryGetProperty(name, out var value) || value.ValueKind != JsonValueKind.String) throw Invalid("payload." + name + " 必须是字符串");
        var actual = value.GetString() ?? throw Invalid("payload." + name + " 不能为空");
        if (expected is not null && !string.Equals(actual, expected, StringComparison.Ordinal)) throw Invalid("payload." + name + " 与 WAL 记录不一致");
        return actual;
    }

    private static long RequireInt64(JsonElement root, string name, long expected, long minimum)
    {
        if (!root.TryGetProperty(name, out var value) || !value.TryGetInt64(out var actual) || actual != expected || actual < minimum)
        {
            throw Invalid("payload." + name + " 与 WAL 记录不一致");
        }
        return actual;
    }

    private static long RequireNonNegativeInt64(JsonElement root, string name)
    {
        if (!root.TryGetProperty(name, out var value) || !value.TryGetInt64(out var actual) || actual < 0) throw Invalid("payload." + name + " 必须是非负整数");
        return actual;
    }

    private static JsonElement RequireObject(JsonElement root, string name)
    {
        if (!root.TryGetProperty(name, out var value) || value.ValueKind != JsonValueKind.Object) throw Invalid("payload." + name + " 必须是对象");
        return value;
    }

    private static string RequireEnum(JsonElement root, string name, params string[] values)
    {
        var actual = RequireString(root, name);
        if (!values.Contains(actual, StringComparer.Ordinal)) throw Invalid("payload." + name + " 枚举值无效");
        return actual;
    }

    private static string RequireAllowedDataGapReason(string reason)
    {
        if (!new[] { "wal-capacity", "wal-corruption", "manual-cleanup", "unrecoverable" }.Contains(reason, StringComparer.Ordinal)) throw Invalid("dataGap.reason 枚举值无效");
        return reason;
    }

    private static string RequireUtc(JsonElement root, string name)
    {
        var value = RequireString(root, name);
        if (!UtcPattern().IsMatch(value) || !IsValidUtc(value)) throw Invalid("payload." + name + " 必须是严格 UTC RFC3339 Z 时间");
        return value;
    }

    private static void RequireOptionalUtcOrNull(JsonElement root, string name)
    {
        if (!root.TryGetProperty(name, out var value)) return;
        if (value.ValueKind == JsonValueKind.Null) return;
        var text = value.ValueKind == JsonValueKind.String ? value.GetString() : null;
        if (text is null || !UtcPattern().IsMatch(text) || !IsValidUtc(text)) throw Invalid("payload." + name + " 必须是 UTC 时间或 null");
    }

    private static bool IsFiniteJsonValue(JsonElement value) => value.ValueKind switch
    {
        JsonValueKind.Null or JsonValueKind.True or JsonValueKind.False or JsonValueKind.String => true,
        JsonValueKind.Number => value.TryGetDouble(out var number) && double.IsFinite(number),
        JsonValueKind.Array => value.EnumerateArray().All(IsFiniteJsonValue),
        JsonValueKind.Object => value.EnumerateObject().All(property => IsFiniteJsonValue(property.Value)),
        _ => false,
    };

    /// <summary>System.Text.Json 对重复属性采用最后值，正式事件不能接受这种歧义。</summary>
    private static void RejectDuplicateProperties(ReadOnlySpan<byte> utf8)
    {
        var reader = new Utf8JsonReader(utf8, new JsonReaderOptions { CommentHandling = JsonCommentHandling.Disallow });
        if (!reader.Read()) throw Invalid("payload 为空");
        ValidateJsonValue(ref reader);
        if (reader.Read()) throw Invalid("payload 包含尾随 JSON 内容");
    }

    private static void ValidateJsonValue(ref Utf8JsonReader reader)
    {
        if (reader.TokenType == JsonTokenType.StartObject)
        {
            var names = new HashSet<string>(StringComparer.Ordinal);
            while (reader.Read() && reader.TokenType != JsonTokenType.EndObject)
            {
                if (reader.TokenType != JsonTokenType.PropertyName) throw Invalid("payload 对象格式无效");
                var name = reader.GetString() ?? throw Invalid("payload 属性名无效");
                if (!names.Add(name)) throw Invalid("payload 含有重复属性: " + name);
                if (!reader.Read()) throw Invalid("payload 对象提前结束");
                ValidateJsonValue(ref reader);
            }
            if (reader.TokenType != JsonTokenType.EndObject) throw Invalid("payload 对象提前结束");
            return;
        }
        if (reader.TokenType == JsonTokenType.StartArray)
        {
            while (reader.Read() && reader.TokenType != JsonTokenType.EndArray) ValidateJsonValue(ref reader);
            if (reader.TokenType != JsonTokenType.EndArray) throw Invalid("payload 数组提前结束");
        }
    }

    private static bool IsValidUtc(string value)
    {
        var withoutZone = value[..^1];
        var dot = withoutZone.IndexOf('.');
        if (dot >= 0 && withoutZone.Length - dot - 1 > 7) withoutZone = withoutZone[..(dot + 8)];
        return DateTime.TryParseExact(withoutZone, ["yyyy-MM-dd'T'HH:mm:ss", "yyyy-MM-dd'T'HH:mm:ss.FFFFFFF"], CultureInfo.InvariantCulture, DateTimeStyles.None, out _);
    }

    private static void RequireDigest(string eventId, IReadOnlyList<string> fields)
    {
        RequireDigest(eventId, ComputeDigest(fields));
    }

    private static void RequireDigest(string eventId, string expected)
    {
        if (!CryptographicOperations.FixedTimeEquals(Encoding.ASCII.GetBytes(eventId), Encoding.ASCII.GetBytes(expected))) throw Invalid("eventId 不符合 Runtime V1 SHA-256 公式");
    }

    private static string ComputeDigest(IReadOnlyList<string> fields)
    {
        if (fields.Any(field => field.Contains(UnitSeparator, StringComparison.Ordinal))) throw Invalid("eventId 输入不能包含 Unit Separator");
        return Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes(string.Join(UnitSeparator, fields)))).ToLowerInvariant();
    }

    private static string Decimal(long value) => value.ToString(CultureInfo.InvariantCulture);

    private static ArgumentException Invalid(string message) => new(message);

    [GeneratedRegex("^[a-f0-9]{64}$", RegexOptions.CultureInvariant)]
    private static partial Regex EventIdPattern();

    [GeneratedRegex("^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$", RegexOptions.CultureInvariant)]
    private static partial Regex StableIdPattern();

    [GeneratedRegex("^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$", RegexOptions.CultureInvariant)]
    private static partial Regex UuidPattern();

    [GeneratedRegex("^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(?:\\.\\d{1,9})?Z$", RegexOptions.CultureInvariant)]
    private static partial Regex UtcPattern();
}
