using InduForge.Collector.Contracts;
using Opc.Ua;
using Opc.Ua.Client;
using CollectorBrowseRequest = InduForge.Collector.Contracts.BrowseRequest;
using CollectorBrowseResult = InduForge.Collector.Contracts.BrowseResult;
using CollectorReadRequest = InduForge.Collector.Contracts.ReadRequest;

namespace InduForge.Collector.Drivers.OpcUa;

internal sealed class OpcUaConnectionSession :
    IIndustrialConnectionSession,
    IDeviceBrowserSession,
    IPointReaderSession
{
    private static readonly IReadOnlyList<DriverDiagnostic> NoDiagnostics = [];
    private readonly ISession _session;

    public OpcUaConnectionSession(ISession session)
    {
        _session = session;
    }

    public bool IsConnected => _session.Connected;

    public string? ServerName => _session.Endpoint?.Server?.ApplicationName?.Text;

    public async Task<CollectorBrowseResult> BrowseAsync(CollectorBrowseRequest request, CancellationToken cancellationToken)
    {
        if (request.MaxDepth != 1)
        {
            throw new OpcUaDriverException(
                "OPCUA_BROWSE_DEPTH_UNSUPPORTED",
                "当前版本只支持浏览指定父节点的直接子节点",
                retryable: false);
        }

        var parentNodeId = ParseNodeId(request.ParentNodeId);
        var browser = new Browser(_session, new BrowserOptions
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
            .Select(reference => OpcUaNodeMapper.Map(reference, _session.NamespaceUris))
            .ToArray();
        var dataTypes = await ReadBrowseDataTypesAsync(nodes, cancellationToken).ConfigureAwait(false);
        var enrichedNodes = nodes
            .Select(node => dataTypes.TryGetValue(node.NodeId, out var dataType)
                ? node with { DataType = dataType }
                : node)
            .OrderBy(node => node.DisplayName, StringComparer.OrdinalIgnoreCase)
            .ToArray();

        return new CollectorBrowseResult(enrichedNodes, NoDiagnostics);
    }

    public async Task<ReadResult> ReadAsync(CollectorReadRequest request, CancellationToken cancellationToken)
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
        var response = await _session.ReadAsync(
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

    public ValueTask DisposeAsync()
    {
        _session.Dispose();
        return ValueTask.CompletedTask;
    }

    private async Task<IReadOnlyDictionary<string, string>> ReadBrowseDataTypesAsync(
        IReadOnlyList<IndustrialNode> nodes,
        CancellationToken cancellationToken)
    {
        var variables = nodes.Where(node => node.NodeClass == "variable").ToArray();
        if (variables.Length == 0)
        {
            return new Dictionary<string, string>(StringComparer.Ordinal);
        }

        var valuesToRead = new ReadValueIdCollection(
            variables.Select(node => new ReadValueId
            {
                NodeId = ParseNodeId(node.NodeId),
                AttributeId = Attributes.DataType,
            }));
        var response = await _session.ReadAsync(
            requestHeader: null,
            maxAge: 0,
            timestampsToReturn: TimestampsToReturn.Neither,
            nodesToRead: valuesToRead,
            cancellationToken).ConfigureAwait(false);

        ClientBase.ValidateResponse(response.Results, valuesToRead);
        ClientBase.ValidateDiagnosticInfos(response.DiagnosticInfos, valuesToRead);

        var result = new Dictionary<string, string>(StringComparer.Ordinal);
        for (var index = 0; index < variables.Length; index++)
        {
            var value = response.Results[index];
            if (!StatusCode.IsGood(value.StatusCode) || value.Value is not NodeId dataTypeId)
            {
                continue;
            }

            var builtInType = DataTypes.GetBuiltInType(dataTypeId, _session.TypeTree);
            var dataType = OpcUaAddressMapper.MapDataType(builtInType);
            if (!string.IsNullOrWhiteSpace(dataType))
            {
                result[variables[index].NodeId] = dataType;
            }
        }

        return result;
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
