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
        var connection = OpcUaConnectionProfile.Parse(profile, _options.ConnectionTimeoutMilliseconds);

        try
        {
            var configuration = await _configuration.Value.ConfigureAwait(false);
            var endpointDescription = await CoreClientUtils.SelectEndpointAsync(
                configuration,
                connection.EndpointUrl,
                useSecurity: false,
                checked((int)connection.Timeout.TotalMilliseconds),
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
                identity: new UserIdentity(),
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

    private static bool IsCertificateError(StatusCode statusCode) =>
        statusCode.Code is StatusCodes.BadCertificateInvalid
            or StatusCodes.BadCertificateUntrusted
            or StatusCodes.BadCertificateTimeInvalid
            or StatusCodes.BadCertificateHostNameInvalid
            or StatusCodes.BadCertificateUriInvalid;
}
