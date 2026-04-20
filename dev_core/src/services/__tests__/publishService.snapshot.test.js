const mockDataDomainClient = {
  getProjectSnapshot: jest.fn(),
};

const mockProject = {
  findByPk: jest.fn(),
};

const mockDesignPage = {
  findAll: jest.fn(),
};

const mockDeployment = {
  findByPk: jest.fn(),
  findOne: jest.fn(),
  create: jest.fn(),
};

const mockNodeDeployment = {
  count: jest.fn(),
  findAndCountAll: jest.fn(),
};

const mockUser = {};

jest.mock("../dataDomainClient", () => ({
  dataDomainClient: mockDataDomainClient,
}));

jest.mock("../../models", () => ({
  Project: mockProject,
  DesignPage: mockDesignPage,
  Deployment: mockDeployment,
  NodeDeployment: mockNodeDeployment,
  User: mockUser,
}));

const publishService = require("../publishService");

describe("publishService snapshot integration", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("collectProjectData 会通过 data_service 快照读取数据域内容", async () => {
    const pageRecord = {
      id: "page-1",
      name: "首页",
      type: "page",
      sortOrder: 1,
    };
    mockDesignPage.findAll.mockResolvedValue([pageRecord]);
    mockDataDomainClient.getProjectSnapshot.mockResolvedValue({
      connections: [{ id: "conn-1", name: "主库", type: "relational", status: "connected", config: {} }],
      relationalConfigs: [{ connectionId: "conn-1", dbType: "postgresql", host: "172.21.242.174", port: 5432 }],
      queries: [{ id: "query-1", connectionId: "conn-1", name: "查询1", queryType: "sql", config: { sql: "select 1" } }],
      mqttConfigs: [],
      mqttSubscriptions: [],
      mqttTagGroups: [],
      mqttTags: [],
      datapoints: [{ id: "dp-1", path: "device.temp", sourceType: "query", dataType: "number" }],
    });

    const result = await publishService.collectProjectData("project-1", "Bearer publish-token");

    expect(mockDesignPage.findAll).toHaveBeenCalledWith({
      where: { projectId: "project-1" },
      order: [
        ["type", "ASC"],
        ["sortOrder", "ASC"],
      ],
    });
    expect(mockDataDomainClient.getProjectSnapshot).toHaveBeenCalledWith(
      "project-1",
      "Bearer publish-token",
    );
    expect(result).toEqual(
      expect.objectContaining({
        pages: [pageRecord],
        connections: expect.arrayContaining([
          expect.objectContaining({ id: "conn-1" }),
        ]),
        dataPoints: expect.arrayContaining([
          expect.objectContaining({ id: "dp-1" }),
        ]),
      }),
    );
  });

  test("compileManifest 使用快照数据点的 camelCase 字段生成清单", async () => {
    const manifest = await publishService.compileManifest(
      {
        id: "project-1",
        tenantId: "tenant-1",
        name: "演示工程",
        code: "demo-project",
        entryConfig: { homePageId: "page-1" },
      },
      {
        connections: [
          { id: "conn-1", name: "主库", type: "relational" },
          { id: "conn-2", name: "MQTT", type: "mqtt" },
        ],
        dataPoints: [
          { path: "device.temp", sourceType: "query", dataType: "number" },
        ],
        queries: [{ id: "query-1" }],
      },
      "1.0.0",
    );

    expect(manifest.dataRequirements).toEqual({
      connections: [
        { id: "conn-1", name: "主库", type: "relational" },
        { id: "conn-2", name: "MQTT", type: "mqtt" },
      ],
      dataPoints: [
        { path: "device.temp", sourceType: "query", dataType: "number" },
      ],
    });
    expect(manifest.capabilities).toEqual(["mqtt", "database", "query"]);
  });
});
