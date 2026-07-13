using System.Diagnostics;
using InduForge.Collector.Contracts;
using Opc.Ua;
using Opc.Ua.Client;
using CollectorBrowseRequest = InduForge.Collector.Contracts.BrowseRequest;
using CollectorBrowseResult = InduForge.Collector.Contracts.BrowseResult;
using CollectorReadRequest = InduForge.Collector.Contracts.ReadRequest;

namespace InduForge.Collector.Drivers.OpcUa;

public sealed class OpcUaDriver : IIndustrialDriver
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly IOpcUaConnectionFactory _connectionFactory;

    public OpcUaDriver()
        : this(new OpcUaConnectionFactory(new OpcUaDriverOptions()))
    {
    }

    internal OpcUaDriver(IOpcUaConnectionFactory connectionFactory)
    {
        _connectionFactory = connectionFactory;
    }

    public ProtocolCapability Capability { get; } = new(
        "opcua",
        "1.0",
        [DriverOperations.ConnectionTest, DriverOperations.OpcUaBrowse, DriverOperations.OpcUaRead]);

    public async Task<ConnectionTestResult> TestConnectionAsync(
        ConnectionProfile profile,
        CancellationToken cancellationToken)
    {
        var stopwatch = Stopwatch.StartNew();
        using var session = await _connectionFactory.ConnectAsync(profile, cancellationToken).ConfigureAwait(false);
        stopwatch.Stop();

        return new ConnectionTestResult(
            Connected: session.Connected,
            Elapsed: stopwatch.Elapsed,
            ServerName: session.Endpoint?.Server?.ApplicationName?.Text,
            Diagnostics: NoDiagnostics);
    }

    public async Task<CollectorBrowseResult> BrowseAsync(
        ConnectionProfile profile,
        CollectorBrowseRequest request,
        CancellationToken cancellationToken)
    {
        if (request.MaxDepth != 1)
        {
            throw new OpcUaDriverException(
                "OPCUA_BROWSE_DEPTH_UNSUPPORTED",
                "当前版本只支持浏览指定父节点的直接子节点",
                retryable: false);
        }

        var parentNodeId = ParseNodeId(request.ParentNodeId);
        using var session = await _connectionFactory.ConnectAsync(profile, cancellationToken).ConfigureAwait(false);
        var browser = new Browser(session, new BrowserOptions
        {
            BrowseDirection = BrowseDirection.Forward,
            ReferenceTypeId = ReferenceTypeIds.HierarchicalReferences,
            IncludeSubtypes = true,
            NodeClassMask = (int)(NodeClass.Object | NodeClass.Variable | NodeClass.Method | NodeClass.View),
            ResultMask = (uint)BrowseResultMask.All,
        })
        {
            ContinueUntilDone = true,
        };
        var references = await browser.BrowseAsync(parentNodeId, cancellationToken).ConfigureAwait(false);
        var nodes = references
            .Select(reference => OpcUaNodeMapper.Map(reference, session.NamespaceUris))
            .OrderBy(node => node.DisplayName, StringComparer.OrdinalIgnoreCase)
            .ToArray();

        return new CollectorBrowseResult(nodes, NoDiagnostics);
    }

    public async Task<ReadResult> ReadAsync(
        ConnectionProfile profile,
        CollectorReadRequest request,
        CancellationToken cancellationToken)
    {
        if (request.NodeIds.Count == 0)
        {
            return new ReadResult([], NoDiagnostics);
        }

        var readValues = new ReadValueIdCollection(
            request.NodeIds.Select(nodeId => new ReadValueId
            {
                NodeId = ParseNodeId(nodeId),
                AttributeId = Attributes.Value,
            }));
        using var session = await _connectionFactory.ConnectAsync(profile, cancellationToken).ConfigureAwait(false);
        var response = await session.ReadAsync(
            requestHeader: null,
            maxAge: 0,
            timestampsToReturn: TimestampsToReturn.Both,
            nodesToRead: readValues,
            cancellationToken).ConfigureAwait(false);

        ClientBase.ValidateResponse(response.Results, readValues);
        ClientBase.ValidateDiagnosticInfos(response.DiagnosticInfos, readValues);

        var values = response.Results
            .Select((value, index) => OpcUaNodeMapper.MapDataValue(request.NodeIds[index], value))
            .ToArray();
        return new ReadResult(values, NoDiagnostics);
    }

    private static NodeId ParseNodeId(string nodeId)
    {
        if (string.IsNullOrWhiteSpace(nodeId))
        {
            throw new OpcUaDriverException("OPCUA_NODE_ID_REQUIRED", "OPC UA NodeId 不能为空", retryable: false);
        }

        try
        {
            return NodeId.Parse(nodeId);
        }
        catch (Exception exception) when (exception is FormatException or ServiceResultException)
        {
            throw new OpcUaDriverException("OPCUA_NODE_ID_INVALID", "OPC UA NodeId 无效", retryable: false, exception);
        }
    }
}
