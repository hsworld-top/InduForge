using System.Text.Json;
using NATS.Client.Core;
using NATS.Net;

namespace InduForge.Collector.Runtime;

/// <summary>上游不可用时保持 WAL 与采集宿主运行；ReplayPump 的顺序不因重连而改变。</summary>
public sealed class RetryingNatsPublisher : IJetStreamPublisher, IAsyncDisposable
{
    private readonly ICollectorResourceResolver _resources;
    private readonly ICollectorSecretResolver _secrets;
    private readonly string _resourceReference;
    private readonly string _secretReference;
    private readonly string _accountId;
    private readonly bool _production;
    private readonly SemaphoreSlim _gate = new(1, 1);
    private NatsClient? _client;
    private NatsJetStreamPublisher? _publisher;
    private DateTimeOffset _nextConnect;

    public RetryingNatsPublisher(ICollectorResourceResolver resources, ICollectorSecretResolver secrets, string resourceReference, string secretReference, string accountId, bool production = false)
    {
        _resources = resources ?? throw new ArgumentNullException(nameof(resources));
        _secrets = secrets ?? throw new ArgumentNullException(nameof(secrets));
        _resourceReference = resourceReference;
        _secretReference = secretReference;
        _accountId = StrictJson.RequireStableId(accountId);
        _production = production;
    }

    public bool IsConnected => Volatile.Read(ref _publisher) is not null;

    public async Task PublishAsync(WalDataRecord record, CancellationToken cancellationToken)
    {
        var publisher = await GetPublisherAsync(cancellationToken).ConfigureAwait(false);
        try { await publisher.PublishAsync(record, cancellationToken).ConfigureAwait(false); }
        catch { await DropAsync().ConfigureAwait(false); throw; }
    }

    private async Task<NatsJetStreamPublisher> GetPublisherAsync(CancellationToken cancellationToken)
    {
        if (_publisher is not null) return _publisher;
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_publisher is not null) return _publisher;
            if (DateTimeOffset.UtcNow < _nextConnect) throw new IOException("NATS_UNAVAILABLE");
            try
            {
                var resource = await _resources.ResolveResourceAsync(_resourceReference, cancellationToken).ConfigureAwait(false);
                var server = ParseServerResource(resource.Value);
                // 在读取凭据或建立连接前固定校验租户边界，避免错误 Binding 跨 account 使用资源。
                if (!string.Equals(server.AccountId, _accountId, StringComparison.Ordinal)) throw ConfigurationError.Invalid();
                using var secret = await _secrets.ResolveSecretAsync(_secretReference, cancellationToken).ConfigureAwait(false);
                var credential = NatsCredential.Parse(secret.Value, _production);
                var options = new NatsOpts { Url = server.Url.AbsoluteUri };
                if (credential.ToOptions() is { } authentication) options = options with { AuthOpts = authentication };
                NatsClient? client = new NatsClient(options);
                try
                {
                    using var connectTimeout = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
                    connectTimeout.CancelAfter(TimeSpan.FromSeconds(10));
                    await client.ConnectAsync().AsTask().WaitAsync(connectTimeout.Token).ConfigureAwait(false);
                    _client = client; _publisher = new NatsJetStreamPublisher(client.CreateJetStreamContext()); return _publisher;
                }
                catch
                {
                    await client.DisposeAsync().ConfigureAwait(false);
                    throw;
                }
            }
            catch
            {
                _nextConnect = DateTimeOffset.UtcNow.AddSeconds(2); throw;
            }
        }
        finally { _gate.Release(); }
    }

    private async ValueTask DropAsync()
    {
        await _gate.WaitAsync().ConfigureAwait(false);
        try { _publisher = null; if (_client is not null) { await _client.DisposeAsync().ConfigureAwait(false); _client = null; } _nextConnect = DateTimeOffset.UtcNow.AddSeconds(2); }
        finally { _gate.Release(); }
    }
    public async ValueTask DisposeAsync() { await DropAsync().ConfigureAwait(false); _gate.Dispose(); }

    internal static NatsServerResource ParseServerResource(JsonElement resource)
    {
        StrictJson.RequireObject(resource, "nats resource"); StrictJson.RequireOnly(resource, "url", "accountId");
        var url = StrictJson.RequireNonEmptyString(resource, "url");
        var accountId = StrictJson.RequireStableId(resource, "accountId");
        if (!Uri.TryCreate(url, UriKind.Absolute, out var uri) || uri.Scheme is not ("nats" or "tls") || !string.IsNullOrEmpty(uri.UserInfo) || string.IsNullOrEmpty(uri.Host) || !string.IsNullOrEmpty(uri.Query) || !string.IsNullOrEmpty(uri.Fragment)) throw ConfigurationError.Invalid();
        return new NatsServerResource(uri, accountId);
    }
}

internal sealed record NatsServerResource(Uri Url, string AccountId)
{
    public override string ToString() => "NatsServerResource { redacted }";
}

internal sealed record NatsCredential(string Kind, string? Token, string? Username, string? Password)
{
    internal static NatsCredential Parse(JsonElement secret, bool production)
    {
        StrictJson.RequireObject(secret, "nats credential"); StrictJson.RequireOnly(secret, "schemaVersion", "authType", "token", "username", "password");
        StrictJson.RequireString(secret, "schemaVersion", "nats-credential.v1");
        var type = StrictJson.RequireEnum(secret, "authType", ["none", "token", "username-password"]);
        return type switch
        {
            "none" when !production && !secret.TryGetProperty("token", out _) && !secret.TryGetProperty("username", out _) && !secret.TryGetProperty("password", out _) => new NatsCredential(type, null, null, null),
            "token" when !secret.TryGetProperty("username", out _) && !secret.TryGetProperty("password", out _) => new NatsCredential(type, StrictJson.RequireNonEmptyString(secret, "token"), null, null),
            "username-password" when !secret.TryGetProperty("token", out _) => new NatsCredential(type, null, StrictJson.RequireNonEmptyString(secret, "username"), StrictJson.RequireNonEmptyString(secret, "password")),
            _ => throw ConfigurationError.Invalid(),
        };
    }
    internal NatsAuthOpts? ToOptions() => Kind switch
    {
        "none" => null,
        "token" => new NatsAuthOpts { Token = Token },
        "username-password" => new NatsAuthOpts { Username = Username, Password = Password },
        _ => throw ConfigurationError.Invalid(),
    };
    public override string ToString() => "NatsCredential { redacted }";
}
