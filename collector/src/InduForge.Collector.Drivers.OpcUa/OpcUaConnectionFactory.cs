using System.Text;
using InduForge.Collector.Contracts;
using Opc.Ua;
using Opc.Ua.Client;

namespace InduForge.Collector.Drivers.OpcUa;

internal interface IOpcUaConnectionFactory
{
    Task<ISession> ConnectAsync(ConnectionProfile profile, CancellationToken cancellationToken);
}

internal sealed class OpcUaConnectionFactory : IOpcUaConnectionFactory
{
    private readonly OpcUaDriverOptions _options;
    private readonly Lazy<Task<ApplicationConfiguration>> _configuration;
    private readonly DefaultSessionFactory _sessionFactory = new(DefaultTelemetry.Create(_ => { }));

    public OpcUaConnectionFactory(OpcUaDriverOptions options)
    {
        _options = options;
        _configuration = new Lazy<Task<ApplicationConfiguration>>(CreateConfigurationAsync);
    }

    public async Task<ISession> ConnectAsync(ConnectionProfile profile, CancellationToken cancellationToken)
    {
        ValidateProfile(profile);

        try
        {
            var configuration = await _configuration.Value.ConfigureAwait(false);
            var timeout = profile.Timeout ?? TimeSpan.FromMilliseconds(_options.ConnectionTimeoutMilliseconds);
            var endpointDescription = await CoreClientUtils.SelectEndpointAsync(
                configuration,
                profile.EndpointUrl,
                useSecurity: false,
                checked((int)timeout.TotalMilliseconds),
                _sessionFactory.Telemetry,
                cancellationToken).ConfigureAwait(false);
            var endpoint = new ConfiguredEndpoint(
                collection: null,
                endpointDescription,
                EndpointConfiguration.Create(configuration));

            return await _sessionFactory.CreateAsync(
                configuration,
                endpoint,
                updateBeforeConnect: false,
                sessionName: _options.ApplicationName,
                sessionTimeout: _options.SessionTimeoutMilliseconds,
                identity: CreateIdentity(profile.Authentication),
                preferredLocales: null,
                cancellationToken).ConfigureAwait(false);
        }
        catch (ServiceResultException exception) when (IsCertificateError(exception.StatusCode))
        {
            throw new OpcUaDriverException(
                "OPCUA_BAD_CERTIFICATE",
                "OPC UA 证书校验失败",
                retryable: false,
                exception);
        }
        catch (OperationCanceledException)
        {
            throw;
        }
        catch (Exception exception) when (exception is not OpcUaDriverException)
        {
            throw new OpcUaDriverException(
                "OPCUA_CONNECT_FAILED",
                "无法连接 OPC UA 服务",
                retryable: true,
                exception);
        }
    }

    private async Task<ApplicationConfiguration> CreateConfigurationAsync()
    {
        var pkiRoot = Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
            "InduForge",
            "CollectorDevAgent",
            "pki");
        var configuration = new ApplicationConfiguration
        {
            ApplicationName = _options.ApplicationName,
            ApplicationUri = $"urn:{Environment.MachineName}:InduForge:CollectorDevAgent",
            ProductUri = "urn:InduForge:CollectorDevAgent",
            ApplicationType = ApplicationType.Client,
            SecurityConfiguration = new SecurityConfiguration
            {
                ApplicationCertificate = new CertificateIdentifier(),
                TrustedIssuerCertificates = new CertificateTrustList
                {
                    StoreType = CertificateStoreType.Directory,
                    StorePath = Path.Combine(pkiRoot, "issuer"),
                },
                TrustedPeerCertificates = new CertificateTrustList
                {
                    StoreType = CertificateStoreType.Directory,
                    StorePath = Path.Combine(pkiRoot, "trusted"),
                },
                RejectedCertificateStore = new CertificateTrustList
                {
                    StoreType = CertificateStoreType.Directory,
                    StorePath = Path.Combine(pkiRoot, "rejected"),
                },
                AutoAcceptUntrustedCertificates = false,
                AddAppCertToTrustedStore = true,
            },
            TransportConfigurations = [],
            TransportQuotas = new TransportQuotas
            {
                OperationTimeout = _options.ConnectionTimeoutMilliseconds,
            },
            ClientConfiguration = new ClientConfiguration
            {
                DefaultSessionTimeout = checked((int)_options.SessionTimeoutMilliseconds),
            },
        };

        await configuration.ValidateAsync(ApplicationType.Client, CancellationToken.None).ConfigureAwait(false);
        return configuration;
    }

    private static UserIdentity CreateIdentity(ConnectionAuthentication authentication) => authentication.Type switch
    {
        AuthenticationType.Anonymous => new UserIdentity(),
        AuthenticationType.Username when !string.IsNullOrWhiteSpace(authentication.Username) =>
            new UserIdentity(authentication.Username, Encoding.UTF8.GetBytes(authentication.Password ?? string.Empty)),
        AuthenticationType.Username => throw new OpcUaDriverException(
            "OPCUA_USERNAME_REQUIRED",
            "OPC UA 用户名认证缺少用户名",
            retryable: false),
        AuthenticationType.Certificate => throw new OpcUaDriverException(
            "OPCUA_CERTIFICATE_AUTH_UNSUPPORTED",
            "当前版本尚未启用 OPC UA 用户证书认证",
            retryable: false),
        _ => throw new OpcUaDriverException(
            "OPCUA_AUTH_UNSUPPORTED",
            "不支持的 OPC UA 认证方式",
            retryable: false),
    };

    private static void ValidateProfile(ConnectionProfile profile)
    {
        if (!string.Equals(profile.ProtocolType, "opcua", StringComparison.OrdinalIgnoreCase))
        {
            throw new OpcUaDriverException("OPCUA_PROTOCOL_MISMATCH", "连接配置不是 OPC UA 协议", retryable: false);
        }

        if (!Uri.TryCreate(profile.EndpointUrl, UriKind.Absolute, out var endpointUri) ||
            !string.Equals(endpointUri.Scheme, "opc.tcp", StringComparison.OrdinalIgnoreCase))
        {
            throw new OpcUaDriverException("OPCUA_ENDPOINT_INVALID", "OPC UA Endpoint 地址无效", retryable: false);
        }

        if (!string.Equals(profile.SecurityMode, "None", StringComparison.OrdinalIgnoreCase) ||
            !string.Equals(profile.SecurityPolicy, "None", StringComparison.OrdinalIgnoreCase))
        {
            throw new OpcUaDriverException(
                "OPCUA_SECURITY_UNSUPPORTED",
                "当前版本只支持 SecurityMode=None 和 SecurityPolicy=None",
                retryable: false);
        }
    }

    private static bool IsCertificateError(StatusCode statusCode) =>
        statusCode.Code is StatusCodes.BadCertificateInvalid
            or StatusCodes.BadCertificateUntrusted
            or StatusCodes.BadCertificateTimeInvalid
            or StatusCodes.BadCertificateHostNameInvalid
            or StatusCodes.BadCertificateUriInvalid;
}
