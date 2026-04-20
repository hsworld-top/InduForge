const path = require("path");
const fs = require("fs-extra");

const mockDataDomainClient = {
  getProjectSnapshot: jest.fn(),
  getProjectArtifact: jest.fn(),
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
const ARTIFACTS_DIR = path.join(process.cwd(), "artifacts");

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

  afterEach(async () => {
    const cleanupTargets = [
      "project-download-with-config_v1.0.0_100.ifp",
      "project-download-legacy_v1.0.0_100.ifp",
      "project-download-legacy_v1.0.0_200.ifp",
    ];

    await Promise.all(
      cleanupTargets.map((fileName) =>
        fs.remove(path.join(ARTIFACTS_DIR, fileName)).catch(() => undefined)
      ),
    );
  });

  test("collectProjectData 会通过 data_service artifact 读取数据域内容", async () => {
    const pageRecord = {
      id: "page-1",
      name: "首页",
      type: "page",
      sortOrder: 1,
    };
    mockDesignPage.findAll.mockResolvedValue([pageRecord]);
    mockDataDomainClient.getProjectArtifact.mockResolvedValue({
      version: "1.0",
      generatedAt: "2026-04-20T12:00:00Z",
      connections: [{ id: "conn-1", name: "主库", type: "relational", status: "connected", config: {} }],
      queries: [{ id: "query-1", connectionId: "conn-1", name: "查询1", queryType: "sql", config: { sql: "select 1" } }],
      datapoints: [{ id: "dp-1", path: "device.temp", sourceType: "query", dataType: "number" }],
      mqtt: {
        connections: [],
        subscriptions: [],
        tagGroups: [],
        tags: [],
      },
      protocols: {
        kafka: [],
        http: [],
        websocket: [],
        redis: [],
      },
    });

    const result = await publishService.collectProjectData("project-1", "Bearer publish-token");

    expect(mockDesignPage.findAll).toHaveBeenCalledWith({
      where: { projectId: "project-1" },
      order: [
        ["type", "ASC"],
        ["sortOrder", "ASC"],
      ],
    });
    expect(mockDataDomainClient.getProjectArtifact).toHaveBeenCalledWith(
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

  test("getArtifactPath 会优先读取 buildConfig 中固化的 artifactFileName", async () => {
    const fileName = "project-download-with-config_v1.0.0_100.ifp";
    const artifactPath = path.join(ARTIFACTS_DIR, fileName);
    await fs.ensureDir(ARTIFACTS_DIR);
    await fs.writeFile(artifactPath, "artifact-content");

    mockDeployment.findByPk.mockResolvedValue({
      artifactUrl: "/api/v1/publish/deployment/deployment-1/download",
      projectId: "project-download-with-config",
      version: "1.0.0",
      buildConfig: {
        artifactFileName: fileName,
      },
    });

    await expect(publishService.getArtifactPath("deployment-1")).resolves.toBe(artifactPath);
  });

  test("getArtifactPath 会为旧记录按工程与版本回溯最新制品文件", async () => {
    const olderFileName = "project-download-legacy_v1.0.0_100.ifp";
    const newerFileName = "project-download-legacy_v1.0.0_200.ifp";
    const olderPath = path.join(ARTIFACTS_DIR, olderFileName);
    const newerPath = path.join(ARTIFACTS_DIR, newerFileName);
    await fs.ensureDir(ARTIFACTS_DIR);
    await fs.writeFile(olderPath, "artifact-old");
    await fs.writeFile(newerPath, "artifact-new");

    mockDeployment.findByPk.mockResolvedValue({
      artifactUrl: "/api/v1/publish/deployment/deployment-legacy/download",
      projectId: "project-download-legacy",
      version: "1.0.0",
      buildConfig: {},
    });

    await expect(publishService.getArtifactPath("deployment-legacy")).resolves.toBe(newerPath);
  });
});
