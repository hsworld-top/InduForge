using System.Text.Json;
using InduForge.Collector.Contracts;
using InduForge.Collector.DevAgent;

namespace InduForge.Collector.DevAgent.Tests;

public sealed class CollectorTaskExecutorTests
{
    [Fact]
    public async Task ExecuteAsyncRejectsUnsupportedOperation()
    {
        var request = JsonSerializer.SerializeToElement(new
        {
            driverId = "test.driver",
            driverVersion = "1.0.0",
            schemaVersion = 1,
            connection = new { protocolFamily = "opcua", config = new { host = "127.0.0.1", port = 18540, endpointPath = "/induforge/sim", securityMode = "None", securityPolicy = "None", authenticationType = "anonymous" }, secrets = new { } },
            input = new { },
        });
        var task = new CollectorTaskEnvelope("task-1", "project-1", "agent-1", "point.write", "running", request, "2026-07-13 12:00:00");
        var result = await new CollectorTaskExecutor(CreateRegistry()).ExecuteAsync(task, CancellationToken.None);
        Assert.Equal("failed", result.Status);
        Assert.Equal("COLLECTOR_OPERATION_UNSUPPORTED", result.Error?.Code);
    }

    [Fact]
    public async Task ExecutePointReadUsesSavedStructuredAddresses()
    {
        var request = JsonSerializer.SerializeToElement(new
        {
            driverId = "test.driver",
            driverVersion = "1.0.0",
            schemaVersion = 1,
            connection = new { protocolFamily = "opcua", config = new { host = "127.0.0.1", port = 18540, endpointPath = "/induforge/sim", securityMode = "None", securityPolicy = "None", authenticationType = "anonymous" }, secrets = new { } },
            input = new { points = new[] { new { pointId = "point-1", address = new { nodeId = "ns=2;s=Temperature" }, dataType = "float32", elementCount = 1, readOptions = new { } } } },
        });
        var task = new CollectorTaskEnvelope("task-1", "project-1", "agent-1", "point.read", "running", request, "2026-07-13 12:00:00");

        var result = await new CollectorTaskExecutor(CreateRegistry()).ExecuteAsync(task, CancellationToken.None);

        Assert.Equal("succeeded", result.Status);
        var readResult = Assert.IsType<ReadResult>(result.Result);
        Assert.Equal("ns=2;s=Temperature", Assert.Single(readResult.Values).NodeId);
    }

    [Fact]
    public async Task LongConnectionReusesSessionForBrowseAndClosesIt()
    {
        var state = new SessionDriverState();
        var registry = new DriverRegistry([() => new SessionDriver(state)]);
        await using var executor = new CollectorTaskExecutor(registry);
        var connectionId = Guid.NewGuid();
        var workspaceSessionId = Guid.NewGuid();

        var opened = await executor.ExecuteAsync(
            CreateSessionTask(DriverOperations.ConnectionOpen, connectionId, new { workspaceSessionId }),
            CancellationToken.None);
        var browsed = await executor.ExecuteAsync(
            CreateSessionTask(
                DriverOperations.DeviceBrowse,
                connectionId,
                new { workspaceSessionId, parentNodeId = "ns=0;i=85", maxDepth = 1 }),
            CancellationToken.None);
        var closed = await executor.ExecuteAsync(
            CreateSessionTask(DriverOperations.ConnectionClose, connectionId, new { workspaceSessionId }),
            CancellationToken.None);

        Assert.Equal("succeeded", opened.Status);
        Assert.Equal("succeeded", browsed.Status);
        Assert.Equal("succeeded", closed.Status);
        Assert.Equal(1, state.OpenCount);
        Assert.Equal(1, state.SessionBrowseCount);
        Assert.Equal(0, state.OneShotBrowseCount);
        Assert.Equal(1, state.DisposeCount);
    }

    [Fact]
    public async Task LongConnectionCachesBatchBrowseUntilRefresh()
    {
        var state = new SessionDriverState();
        await using var executor = new CollectorTaskExecutor(
            new DriverRegistry([() => new SessionDriver(state)]));
        var connectionId = Guid.NewGuid();
        var workspaceSessionId = Guid.NewGuid();
        var parentNodeIds = new[] { "node-1", "node-2" };
        await executor.ExecuteAsync(
            CreateSessionTask(DriverOperations.ConnectionOpen, connectionId, new { workspaceSessionId }),
            CancellationToken.None);

        var first = await executor.ExecuteAsync(
            CreateSessionTask(
                DriverOperations.DeviceBrowse,
                connectionId,
                new { workspaceSessionId, parentNodeIds, maxDepth = 1 }),
            CancellationToken.None);
        var second = await executor.ExecuteAsync(
            CreateSessionTask(
                DriverOperations.DeviceBrowse,
                connectionId,
                new { workspaceSessionId, parentNodeIds, maxDepth = 1 }),
            CancellationToken.None);
        var refreshed = await executor.ExecuteAsync(
            CreateSessionTask(
                DriverOperations.DeviceBrowse,
                connectionId,
                new { workspaceSessionId, parentNodeIds, maxDepth = 1, refreshCache = true }),
            CancellationToken.None);

        Assert.Equal(2, Assert.IsType<BrowseBatchResult>(first.Result).Branches.Count);
        Assert.Equal(2, Assert.IsType<BrowseBatchResult>(second.Result).Branches.Count);
        Assert.Equal(2, Assert.IsType<BrowseBatchResult>(refreshed.Result).Branches.Count);
        Assert.Equal(4, state.SessionBrowseCount);
    }

    [Fact]
    public async Task LongConnectionRequiresWorkspaceSessionId()
    {
        var state = new SessionDriverState();
        await using var executor = new CollectorTaskExecutor(
            new DriverRegistry([() => new SessionDriver(state)]));

        var result = await executor.ExecuteAsync(
            CreateSessionTask(DriverOperations.ConnectionOpen, Guid.NewGuid(), new { }),
            CancellationToken.None);

        Assert.Equal("failed", result.Status);
        Assert.Equal("COLLECTOR_SESSION_ID_REQUIRED", result.Error?.Code);
        Assert.Equal(0, state.OpenCount);
    }

    [Fact]
    public async Task ClosingSessionDoesNotParseProtocolProfile()
    {
        var state = new SessionDriverState();
        await using var executor = new CollectorTaskExecutor(
            new DriverRegistry([() => new SessionDriver(state)]));

        var result = await executor.ExecuteAsync(
            CreateSessionTask(
                DriverOperations.ConnectionClose,
                Guid.NewGuid(),
                new { workspaceSessionId = Guid.NewGuid() },
                "future.protocol"),
            CancellationToken.None);

        Assert.Equal("succeeded", result.Status);
        var closeResult = Assert.IsType<ConnectionSessionCloseResult>(result.Result);
        Assert.False(closeResult.Closed);
    }

    private static CollectorTaskEnvelope CreateSessionTask(
        string operation,
        Guid connectionId,
        object input,
        string protocolFamily = "opcua")
    {
        var request = JsonSerializer.SerializeToElement(new
        {
            connectionId,
            driverId = "session.driver",
            driverVersion = "1.0.0",
            schemaVersion = 1,
            connection = new
            {
                protocolFamily,
                config = new
                {
                    host = "127.0.0.1",
                    port = 18540,
                    endpointPath = "/induforge/sim",
                    securityMode = "None",
                    securityPolicy = "None",
                    authenticationType = "anonymous",
                },
                secrets = new { },
            },
            input,
        });
        return new CollectorTaskEnvelope(
            Guid.NewGuid().ToString(),
            "project-1",
            "agent-1",
            operation,
            "running",
            request,
            "2026-07-17 12:00:00");
    }
    private static DriverRegistry CreateRegistry() => new([() => new TestDriver()]);

    [Theory]
    [InlineData("127.0.0.1", 4840, "/factory/server", "opc.tcp://127.0.0.1:4840/factory/server")]
    [InlineData("plc.local", 4840, "factory/server", "opc.tcp://plc.local:4840/factory/server")]
    [InlineData("2001:db8::1", 4840, "/", "opc.tcp://[2001:db8::1]:4840/")]
    public void OpcUaEndpointBuilderCombinesStructuredConfiguration(string host, int port, string path, string expected)
    {
        Assert.Equal(expected, OpcUaEndpointBuilder.Build(host, port, path));
    }

    [Theory]
    [InlineData("opc.tcp://127.0.0.1", 4840)]
    [InlineData("", 4840)]
    [InlineData("127.0.0.1", 0)]
    public void OpcUaEndpointBuilderRejectsInvalidConfiguration(string host, int port)
    {
        Assert.Throws<CollectorTaskExecutionException>(() => OpcUaEndpointBuilder.Build(host, port, "/"));
    }

    private sealed class SessionDriverState
    {
        public int OpenCount { get; set; }
        public int SessionBrowseCount { get; set; }
        public int OneShotBrowseCount { get; set; }
        public int DisposeCount { get; set; }
    }

    private sealed class SessionDriver(SessionDriverState state) :
        IIndustrialDriver,
        IConnectionSessionDriver,
        IDeviceBrowser
    {
        public DriverDescriptor Descriptor { get; } = new(
            "opcua",
            "session.driver",
            "1.0.0",
            [1],
            [
                DriverOperations.ConnectionTest,
                DriverOperations.ConnectionOpen,
                DriverOperations.ConnectionClose,
                DriverOperations.DeviceBrowse,
            ]);

        public Task<ConnectionTestResult> TestConnectionAsync(
            ConnectionProfile profile,
            CancellationToken cancellationToken) =>
            Task.FromResult(new ConnectionTestResult(true, TimeSpan.Zero, "test", []));

        public Task<IIndustrialConnectionSession> OpenSessionAsync(
            ConnectionProfile profile,
            CancellationToken cancellationToken)
        {
            state.OpenCount++;
            return Task.FromResult<IIndustrialConnectionSession>(new SessionDriverSession(state));
        }

        public Task<BrowseResult> BrowseAsync(
            ConnectionProfile profile,
            BrowseRequest request,
            CancellationToken cancellationToken)
        {
            state.OneShotBrowseCount++;
            return Task.FromResult(new BrowseResult([], []));
        }
    }

    private sealed class SessionDriverSession(SessionDriverState state) :
        IIndustrialConnectionSession,
        IDeviceBrowserSession
    {
        public bool IsConnected { get; private set; } = true;

        public string? ServerName => "session-test";

        public Task<BrowseResult> BrowseAsync(BrowseRequest request, CancellationToken cancellationToken)
        {
            state.SessionBrowseCount++;
            return Task.FromResult(new BrowseResult([], []));
        }

        public ValueTask DisposeAsync()
        {
            IsConnected = false;
            state.DisposeCount++;
            return ValueTask.CompletedTask;
        }
    }
    private sealed class TestDriver : IIndustrialDriver, IPointReader
    {
        public DriverDescriptor Descriptor { get; } = new(
            "opcua", "test.driver", "1.0.0", [1],
            [DriverOperations.ConnectionTest, DriverOperations.PointRead]);

        public Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken) =>
            Task.FromResult(new ConnectionTestResult(true, TimeSpan.Zero, "test", []));

        public Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken) =>
            Task.FromResult(new ReadResult(request.NodeIds.Select(nodeId => new IndustrialDataValue(nodeId, 12.5, "float64", "Good", DateTimeOffset.UtcNow, DateTimeOffset.UtcNow)).ToArray(), []));
    }

    [Fact]
    public void CredentialStoreRoundTripsForCurrentUser()
    {
        var path = Path.Combine(Path.GetTempPath(), $"induforge-collector-{Guid.NewGuid():N}.dat");
        try
        {
            var store = new AgentCredentialStore(path);
            var expected = new AgentCredentials("http://127.0.0.1:18102", "agent-1", "secret-token", "tenant-1", "dev-machine");
            store.Save(expected);
            Assert.Equal(expected, store.Load());
            Assert.DoesNotContain("secret-token", File.ReadAllText(path));
            store.Clear();
            Assert.Null(store.Load());
        }
        finally
        {
            if (File.Exists(path)) File.Delete(path);
        }
    }

    [Fact]
    public void CredentialStoreDefaultPathUsesExecutableDataDirectory()
    {
        var baseDirectory = Path.Combine(Path.GetTempPath(), $"induforge-collector-{Guid.NewGuid():N}");
        Assert.Equal(Path.Combine(baseDirectory, "data", "credentials.dat"), AgentCredentialStore.GetDefaultPath(baseDirectory));
    }

    [Fact]
    public void CredentialStoreCreatesDataDirectoryOnStartup()
    {
        var baseDirectory = Path.Combine(Path.GetTempPath(), $"induforge-collector-{Guid.NewGuid():N}");
        try
        {
            _ = new AgentCredentialStore(Path.Combine(baseDirectory, "data", "credentials.dat"));
            Assert.True(Directory.Exists(Path.Combine(baseDirectory, "data")));
        }
        finally
        {
            if (Directory.Exists(baseDirectory)) Directory.Delete(baseDirectory, true);
        }
    }
}

