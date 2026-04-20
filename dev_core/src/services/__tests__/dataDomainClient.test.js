const { DataDomainClient } = require("../dataDomainClient");

const buildJsonResponse = (body, options = {}) => ({
  ok: options.ok ?? true,
  status: options.status ?? 200,
  statusText: options.statusText ?? "OK",
  headers: {
    get: (name) => (name && name.toLowerCase() === "content-type" ? "application/json" : null),
  },
  json: jest.fn().mockResolvedValue(body),
  text: jest.fn().mockResolvedValue(JSON.stringify(body)),
});

describe("dataDomainClient", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    global.fetch = jest.fn();
  });

  afterEach(() => {
    delete global.fetch;
  });

  test("getProjectSnapshot 会规范化 data_service 返回的 PascalCase 快照字段", async () => {
    global.fetch.mockResolvedValue(
      buildJsonResponse({
        success: true,
        data: {
          connections: [
            {
              ID: "conn-1",
              ProjectID: "project-1",
              Name: "主库",
              Type: "relational",
              Status: "connected",
              Config: {
                dbType: "postgresql",
                host: "172.21.242.174",
              },
            },
          ],
          relationalConfigs: [
            {
              ConnectionID: "conn-1",
              DBType: "postgresql",
              Host: "172.21.242.174",
              Port: 5432,
              Database: "data_service",
              Username: "postgres",
              Password: "postgres",
              SSL: false,
              SSLConfig: {},
            },
          ],
          queries: [
            {
              ID: "query-1",
              ProjectID: "project-1",
              ConnectionID: "conn-1",
              Name: "查询1",
              QueryType: "sql",
              Config: { sql: "select 1" },
              IsEnabled: true,
              TimeoutMS: 30000,
              CacheEnabled: false,
              CacheTtlSeconds: 0,
            },
          ],
          datapoints: [
            {
              ID: "dp-1",
              ProjectID: "project-1",
              Path: "device.temp",
              Name: "温度",
              SourceType: "query",
              SourceID: "query-1",
              SourceConfig: { column: "temp" },
              DataType: "number",
              Tags: [],
              RefreshMode: "auto",
              Status: "active",
            },
          ],
        },
      }),
    );

    const client = new DataDomainClient({
      baseUrl: "http://data-service.test",
      timeoutMs: 2000,
    });

    const snapshot = await client.getProjectSnapshot("project-1", "Bearer token-1");

    expect(global.fetch).toHaveBeenCalledWith(
      "http://data-service.test/api/v1/data/projects/project-1/snapshot",
      expect.objectContaining({
        method: "GET",
        headers: expect.objectContaining({
          Authorization: "Bearer token-1",
          Accept: "application/json",
        }),
      }),
    );
    expect(snapshot.connections[0]).toEqual({
      id: "conn-1",
      projectId: "project-1",
      name: "主库",
      type: "relational",
      status: "connected",
      config: {
        dbType: "postgresql",
        host: "172.21.242.174",
      },
      createdAt: undefined,
      updatedAt: undefined,
    });
    expect(snapshot.queries[0].timeoutMs).toBe(30000);
    expect(snapshot.datapoints[0]).toEqual(
      expect.objectContaining({
        id: "dp-1",
        sourceType: "query",
        sourceId: "query-1",
        dataType: "number",
      }),
    );
  });

  test("replaceProjectSnapshot 会使用 PUT 调用 data_service 快照覆盖接口", async () => {
    global.fetch.mockResolvedValue(
      buildJsonResponse({
        success: true,
        data: { updated: true },
      }),
    );

    const client = new DataDomainClient({
      baseUrl: "http://data-service.test",
      timeoutMs: 2000,
    });
    const snapshot = {
      connections: [{ id: "conn-1", name: "主库", type: "relational", status: "connected", config: {} }],
      relationalConfigs: [],
      queries: [],
      mqttConfigs: [],
      mqttSubscriptions: [],
      mqttTagGroups: [],
      mqttTags: [],
      datapoints: [],
    };

    const result = await client.replaceProjectSnapshot("project-1", snapshot, "Bearer token-2");

    expect(result).toEqual({ updated: true });
    expect(global.fetch).toHaveBeenCalledWith(
      "http://data-service.test/api/v1/data/projects/project-1/snapshot",
      expect.objectContaining({
        method: "PUT",
        headers: expect.objectContaining({
          Authorization: "Bearer token-2",
          "Content-Type": "application/json",
        }),
        body: JSON.stringify(snapshot),
      }),
    );
  });
});
