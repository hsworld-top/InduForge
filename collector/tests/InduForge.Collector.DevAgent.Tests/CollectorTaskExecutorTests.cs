using System.Text.Json;
using InduForge.Collector.Contracts;
using InduForge.Collector.DevAgent;

namespace InduForge.Collector.DevAgent.Tests;

public sealed class CollectorTaskExecutorTests
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
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
        var readResult = JsonSerializer.SerializeToElement(result.Result, JsonOptions);
        var value = Assert.Single(readResult.GetProperty("values").EnumerateArray());
        Assert.Equal("point-1", value.GetProperty("pointId").GetString());
        Assert.True(value.GetProperty("succeeded").GetBoolean());
        Assert.Equal(12.5, value.GetProperty("value").GetDouble());
        Assert.Equal("Good", value.GetProperty("quality").GetString());
    }

    [Fact]
    public async Task ExecutePointReadPassesProtocolSpecificConfigurationToDriver()
    {
        var state = new TestDriverState();
        var request = JsonSerializer.SerializeToElement(new
        {
            driverId = "test.driver",
            driverVersion = "1.0.0",
            schemaVersion = 1,
            connection = new
            {
                protocolFamily = "modbus",
                config = new { host = "127.0.0.1", port = 502, dataFormat = "CDAB" },
                secrets = new { },
            },
            input = new
            {
                points = new[]
                {
                    new
                    {
                        pointId = "point-1",
                        address = new { station = 1, area = "holdingRegister", address = 20 },
                        dataType = "float32",
                        elementCount = 2,
                        readOptions = new { },
                    },
                },
            },
        });
        var task = new CollectorTaskEnvelope("task-1", "project-1", "agent-1", "point.read", "running", request, "2026-07-23 12:00:00");

        var result = await new CollectorTaskExecutor(
            new DriverRegistry([() => new TestDriver(state)]))
            .ExecuteAsync(task, CancellationToken.None);

        Assert.Equal("succeeded", result.Status);
        Assert.Equal("modbus", state.Profile?.ProtocolFamily);
        Assert.Equal("CDAB", state.Profile?.Config.GetProperty("dataFormat").GetString());
        var point = Assert.Single(state.Request!.Points);
        Assert.Equal("holdingRegister", point.Address.GetProperty("area").GetString());
        Assert.Equal("float32", point.DataType);
        Assert.Equal(2, point.ElementCount);
    }

    [Fact]
    public async Task ExecutePointReadKeepsValidPointsWhenOneAddressIsInvalid()
    {
        var request = JsonSerializer.SerializeToElement(new
        {
            driverId = "test.driver",
            driverVersion = "1.0.0",
            schemaVersion = 1,
            connection = new { protocolFamily = "opcua", config = new { host = "127.0.0.1", port = 18540, endpointPath = "/induforge/sim", securityMode = "None", securityPolicy = "None", authenticationType = "anonymous" }, secrets = new { } },
            input = new
            {
                points = new object[]
                {
                    new { pointId = "point-1", address = new { nodeId = "ns=2;s=Temperature" }, dataType = "float64", elementCount = 1, readOptions = new { } },
                    new { pointId = "point-2", address = new { nodeId = "111" }, dataType = "float64", elementCount = 1, readOptions = new { } },
                    new { pointId = "point-3", address = new { nodeId = "i=32851" }, dataType = "float64", elementCount = 1, readOptions = new { } },
                },
            },
        });
        var task = new CollectorTaskEnvelope("task-1", "project-1", "agent-1", "point.read", "running", request, "2026-07-21 08:50:25");

        var result = await new CollectorTaskExecutor(CreateRegistry()).ExecuteAsync(task, CancellationToken.None);

        Assert.Equal("succeeded", result.Status);
        var values = JsonSerializer.SerializeToElement(result.Result, JsonOptions)
            .GetProperty("values")
            .EnumerateArray()
            .ToArray();
        Assert.Equal(3, values.Length);
        Assert.True(values[0].GetProperty("succeeded").GetBoolean());
        Assert.Equal("point-1", values[0].GetProperty("pointId").GetString());
        Assert.False(values[1].GetProperty("succeeded").GetBoolean());
        Assert.Equal("point-2", values[1].GetProperty("pointId").GetString());
        Assert.Equal("OPCUA_NODE_ID_INVALID", values[1].GetProperty("errorCode").GetString());
        Assert.True(values[2].GetProperty("succeeded").GetBoolean());
        Assert.Equal("point-3", values[2].GetProperty("pointId").GetString());
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
    public async Task LongConnectionReusesSessionForPointRead()
    {
        var state = new SessionDriverState();
        await using var executor = new CollectorTaskExecutor(
            new DriverRegistry([() => new SessionDriver(state)]));
        var connectionId = Guid.NewGuid();
        var workspaceSessionId = Guid.NewGuid();
        await executor.ExecuteAsync(
            CreateSessionTask(DriverOperations.ConnectionOpen, connectionId, new { workspaceSessionId }),
            CancellationToken.None);

        var result = await executor.ExecuteAsync(
            CreateSessionTask(
                DriverOperations.PointRead,
                connectionId,
                new
                {
                    workspaceSessionId,
                    points = new[]
                    {
                        new
                        {
                            pointId = "point-1",
                            address = new { nodeId = "ns=2;s=Temperature" },
                            dataType = "float64",
                            elementCount = 1,
                            readOptions = new { },
                        },
                    },
                }),
            CancellationToken.None);

        Assert.Equal("succeeded", result.Status);
        Assert.Equal(1, state.OpenCount);
        Assert.Equal(1, state.SessionReadCount);
        Assert.Equal(0, state.OneShotReadCount);
        var readResult = JsonSerializer.SerializeToElement(result.Result, JsonOptions);
        Assert.Equal("point-1", Assert.Single(readResult.GetProperty("values").EnumerateArray()).GetProperty("pointId").GetString());
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

    private sealed class SessionDriverState
    {
        public int OpenCount { get; set; }
        public int SessionBrowseCount { get; set; }
        public int OneShotBrowseCount { get; set; }
        public int SessionReadCount { get; set; }
        public int OneShotReadCount { get; set; }
        public int DisposeCount { get; set; }
    }

    private sealed class SessionDriver(SessionDriverState state) :
        IIndustrialDriver,
        IConnectionSessionDriver,
        IDeviceBrowser,
        IPointReader
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
                DriverOperations.PointRead,
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

        public Task<ReadResult> ReadAsync(
            ConnectionProfile profile,
            ReadRequest request,
            CancellationToken cancellationToken)
        {
            state.OneShotReadCount++;
            return Task.FromResult(CreateReadResult(request));
        }
    }

    private sealed class SessionDriverSession(SessionDriverState state) :
        IIndustrialConnectionSession,
        IDeviceBrowserSession,
        IPointReaderSession
    {
        public bool IsConnected { get; private set; } = true;

        public string? ServerName => "session-test";

        public Task<BrowseResult> BrowseAsync(BrowseRequest request, CancellationToken cancellationToken)
        {
            state.SessionBrowseCount++;
            return Task.FromResult(new BrowseResult([], []));
        }

        public Task<ReadResult> ReadAsync(ReadRequest request, CancellationToken cancellationToken)
        {
            state.SessionReadCount++;
            return Task.FromResult(CreateReadResult(request));
        }

        public ValueTask DisposeAsync()
        {
            IsConnected = false;
            state.DisposeCount++;
            return ValueTask.CompletedTask;
        }
    }
    private static ReadResult CreateReadResult(ReadRequest request) =>
        new(
            request.Points
                .Select(CreateReadValue)
                .ToArray(),
            []);

    private static PointReadValue CreateReadValue(PointReadRequest point)
    {
        var nodeId = point.Address.TryGetProperty("nodeId", out var nodeIdElement)
            ? nodeIdElement.GetString()
            : null;
        if (nodeId == "111")
        {
            return new PointReadValue(
                point.Key,
                false,
                null,
                point.DataType,
                "Bad",
                null,
                null,
                "OPCUA_NODE_ID_INVALID",
                "OPC UA NodeId 无效");
        }

        return new PointReadValue(
            point.Key,
            true,
            12.5,
            "float64",
            "Good",
            DateTimeOffset.UtcNow,
            DateTimeOffset.UtcNow,
            null,
            null);
    }

    private sealed class TestDriverState
    {
        public ConnectionProfile? Profile { get; set; }
        public ReadRequest? Request { get; set; }
    }

    private sealed class TestDriver(TestDriverState? state = null) : IIndustrialDriver, IPointReader
    {
        public DriverDescriptor Descriptor { get; } = new(
            "opcua", "test.driver", "1.0.0", [1],
            [DriverOperations.ConnectionTest, DriverOperations.PointRead]);

        public Task<ConnectionTestResult> TestConnectionAsync(ConnectionProfile profile, CancellationToken cancellationToken)
        {
            if (state is not null) state.Profile = profile;
            return Task.FromResult(new ConnectionTestResult(true, TimeSpan.Zero, "test", []));
        }

        public Task<ReadResult> ReadAsync(ConnectionProfile profile, ReadRequest request, CancellationToken cancellationToken)
        {
            if (state is not null)
            {
                state.Profile = profile;
                state.Request = request;
            }
            return Task.FromResult(CreateReadResult(request));
        }
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

