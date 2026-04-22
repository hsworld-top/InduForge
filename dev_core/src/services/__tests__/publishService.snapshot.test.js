const path = require("path");
const fs = require("fs-extra");
const zlib = require("zlib");

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

const mockProjectRuntimeUser = {
  findAll: jest.fn(),
};

const mockProjectRole = {
  findAll: jest.fn(),
};

const mockProjectUserRoleBinding = {
  findAll: jest.fn(),
};

const mockProjectRoleGrant = {
  findAll: jest.fn(),
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
  ProjectRuntimeUser: mockProjectRuntimeUser,
  ProjectRole: mockProjectRole,
  ProjectUserRoleBinding: mockProjectUserRoleBinding,
  ProjectRoleGrant: mockProjectRoleGrant,
  User: mockUser,
}));

const publishService = require("../publishService");

const readIfpEntry = async (ifpPath, entryName) => {
  const buffer = await fs.readFile(ifpPath);
  const eocdSignature = 0x06054b50;
  const centralHeaderSignature = 0x02014b50;
  const localHeaderSignature = 0x04034b50;

  let eocdOffset = -1;
  for (let offset = buffer.length - 22; offset >= 0; offset -= 1) {
    if (buffer.readUInt32LE(offset) === eocdSignature) {
      eocdOffset = offset;
      break;
    }
  }

  if (eocdOffset < 0) {
    throw new Error("无法找到 ZIP 结束目录");
  }

  const centralDirectoryOffset = buffer.readUInt32LE(eocdOffset + 16);
  const totalEntries = buffer.readUInt16LE(eocdOffset + 10);

  let cursor = centralDirectoryOffset;
  for (let index = 0; index < totalEntries; index += 1) {
    if (buffer.readUInt32LE(cursor) !== centralHeaderSignature) {
      throw new Error("ZIP 中央目录损坏");
    }

    const method = buffer.readUInt16LE(cursor + 10);
    const compressedSize = buffer.readUInt32LE(cursor + 20);
    const fileNameLength = buffer.readUInt16LE(cursor + 28);
    const extraLength = buffer.readUInt16LE(cursor + 30);
    const commentLength = buffer.readUInt16LE(cursor + 32);
    const localHeaderOffset = buffer.readUInt32LE(cursor + 42);
    const fileName = buffer.toString("utf8", cursor + 46, cursor + 46 + fileNameLength);

    if (fileName === entryName) {
      if (buffer.readUInt32LE(localHeaderOffset) !== localHeaderSignature) {
        throw new Error(`ZIP 本地文件头损坏: ${entryName}`);
      }

      const localFileNameLength = buffer.readUInt16LE(localHeaderOffset + 26);
      const localExtraLength = buffer.readUInt16LE(localHeaderOffset + 28);
      const dataStart = localHeaderOffset + 30 + localFileNameLength + localExtraLength;
      const compressed = buffer.slice(dataStart, dataStart + compressedSize);

      if (method === 0) {
        return compressed.toString("utf8");
      }
      if (method === 8) {
        return zlib.inflateRawSync(compressed).toString("utf8");
      }
      throw new Error(`暂不支持的 ZIP 压缩方式: ${method}`);
    }

    cursor += 46 + fileNameLength + extraLength + commentLength;
  }

  throw new Error(`未找到 ZIP 条目: ${entryName}`);
};

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

  test("collectProjectData 会附带运行态安全快照", async () => {
    const runtimeUser = {
      id: "runtime-user-1",
      username: "operator",
      displayName: "运维员",
      status: "active",
    };
    const runtimeRole = {
      id: "role-1",
      code: "viewer",
      name: "查看者",
      description: "只读访问",
      isSystem: false,
      status: "active",
    };
    const binding = {
      id: "binding-1",
      runtimeUserId: runtimeUser.id,
      roleId: runtimeRole.id,
      assignedAt: "2026-04-20T12:00:00Z",
    };
    const grant = {
      id: "grant-1",
      roleId: runtimeRole.id,
      resourceType: "page",
      resourceId: "*",
      action: "read",
      effect: "allow",
      scopeConfig: { source: "snapshot" },
    };

    mockDesignPage.findAll.mockResolvedValue([]);
    mockDataDomainClient.getProjectArtifact.mockResolvedValue({
      connections: [],
      queries: [],
      datapoints: [],
      mqtt: {},
      protocols: {},
    });
    mockProjectRuntimeUser.findAll.mockResolvedValue([runtimeUser]);
    mockProjectRole.findAll.mockResolvedValue([runtimeRole]);
    mockProjectUserRoleBinding.findAll.mockResolvedValue([binding]);
    mockProjectRoleGrant.findAll.mockResolvedValue([grant]);

    const result = await publishService.collectProjectData("project-2", "Bearer publish-token");

    expect(mockProjectRuntimeUser.findAll).toHaveBeenCalledWith({
      where: { projectId: "project-2" },
      order: [["createdAt", "ASC"]],
      raw: true,
    });
    expect(mockProjectRole.findAll).toHaveBeenCalledWith({
      where: { projectId: "project-2" },
      order: [["createdAt", "ASC"]],
      raw: true,
    });
    expect(mockProjectUserRoleBinding.findAll).toHaveBeenCalledWith({
      where: { projectId: "project-2" },
      order: [["createdAt", "ASC"]],
      raw: true,
    });
    expect(mockProjectRoleGrant.findAll).toHaveBeenCalledWith({
      where: { projectId: "project-2" },
      order: [["createdAt", "ASC"]],
      raw: true,
    });
    expect(result.runtimeSecuritySnapshot).toEqual(
      expect.objectContaining({
        version: expect.any(String),
        users: [runtimeUser],
        roles: [runtimeRole],
        bindings: [binding],
        grants: [grant],
      }),
    );
  });

  test("bundleIFP 会把运行态安全快照写入最终产物", async () => {
    const runtimeSecuritySnapshot = {
      version: "2026-04-21 10:00:00",
      users: [
        {
          id: "runtime-user-1",
          username: "operator",
        },
      ],
      roles: [
        {
          id: "role-1",
          code: "viewer",
        },
      ],
      bindings: [
        {
          id: "binding-1",
          runtimeUserId: "runtime-user-1",
          roleId: "role-1",
        },
      ],
      grants: [
        {
          id: "grant-1",
          roleId: "role-1",
          resourceType: "page",
          resourceId: "*",
          action: "read",
          effect: "allow",
        },
      ],
    };

    const ifpResult = await publishService.bundleIFP(
      "deployment-1",
      "project-3",
      {
        version: "1.0.0",
        name: "演示工程",
        code: "demo-project",
        security: {
          requireAuth: true,
        },
      },
      {
        pages: [
          {
            id: "page-1",
            name: "首页",
            type: "page",
            sortOrder: 1,
          },
        ],
        artifactVersion: "1.0",
        artifactGeneratedAt: "2026-04-20T12:00:00Z",
        connections: [],
        relationalConfigs: [],
        queries: [],
        mqttConfigs: [],
        mqttSubscriptions: [],
        mqttTagGroups: [],
        mqttTags: [],
        dataPoints: [],
        protocols: {},
        runtimeSecuritySnapshot,
      },
    );

    try {
      const runtimeSecurityJson = await readIfpEntry(ifpResult.ifpPath, "runtime-security.json");
      const manifestJson = await readIfpEntry(ifpResult.ifpPath, "manifest.json");

      expect(JSON.parse(runtimeSecurityJson)).toEqual(runtimeSecuritySnapshot);
      expect(JSON.parse(manifestJson)).toEqual(
        expect.objectContaining({
          security: expect.objectContaining({
            runtimeSecuritySnapshot: expect.objectContaining({
              included: true,
              fileName: "runtime-security.json",
              version: runtimeSecuritySnapshot.version,
            }),
          }),
        }),
      );
    } finally {
      await fs.remove(ifpResult.ifpPath).catch(() => undefined);
    }
  });

  test("bundleIFP 会在输出流错误时拒绝", async () => {
    const { PassThrough } = require("stream");
    const output = new PassThrough();
    const createWriteStreamSpy = jest.spyOn(fs, "createWriteStream").mockImplementation(() => {
      setImmediate(() => output.emit("error", new Error("disk full")));
      return output;
    });

    await expect(
      publishService.bundleIFP(
        "deployment-error",
        "project-error",
        {
          version: "1.0.0",
          name: "演示工程",
          code: "demo-project",
          security: {},
        },
        {
          pages: [],
          artifactVersion: "1.0",
          artifactGeneratedAt: "2026-04-20T12:00:00Z",
          connections: [],
          relationalConfigs: [],
          queries: [],
          mqttConfigs: [],
          mqttSubscriptions: [],
          mqttTagGroups: [],
          mqttTags: [],
          dataPoints: [],
          protocols: {},
          runtimeSecuritySnapshot: {
            version: "2026-04-21 10:00:00",
            users: [],
            roles: [],
            bindings: [],
            grants: [],
          },
        },
      ),
    ).rejects.toThrow("disk full");

    createWriteStreamSpy.mockRestore();
  });

  test("publish 会把 runtimeSecuritySnapshot 同步写入数据库和最终产物", async () => {
    const runtimeSecuritySnapshot = {
      version: "2026-04-21 11:00:00",
      users: [
        {
          id: "runtime-user-1",
          username: "operator",
        },
      ],
      roles: [
        {
          id: "role-1",
          code: "viewer",
        },
      ],
      bindings: [
        {
          id: "binding-1",
          runtimeUserId: "runtime-user-1",
          roleId: "role-1",
        },
      ],
      grants: [
        {
          id: "grant-1",
          roleId: "role-1",
          resourceType: "page",
          resourceId: "*",
          action: "read",
          effect: "allow",
        },
      ],
    };
    const updatedDeployment = {
      update: jest.fn().mockResolvedValue(undefined),
      buildLog: [],
    };

    mockProject.findByPk.mockResolvedValue({
      id: "project-publish",
      tenantId: "tenant-1",
      name: "演示工程",
      code: "demo-project",
      entryConfig: { homePageId: "page-1" },
    });
    mockDeployment.findOne.mockResolvedValue(null);
    mockDeployment.create.mockResolvedValue(updatedDeployment);
    mockDesignPage.findAll.mockImplementation(({ where }) => {
      if (where?.type === "page") {
        return Promise.resolve([
          {
            id: "page-1",
            name: "首页",
            type: "page",
            sortOrder: 1,
            isHome: true,
            schemaContent: { components: [] },
          },
        ]);
      }
      return Promise.resolve([
        {
          id: "page-1",
          name: "首页",
          type: "page",
          sortOrder: 1,
          isHome: true,
          schemaContent: { components: [] },
        },
      ]);
    });
    mockDataDomainClient.getProjectArtifact.mockResolvedValue({
      version: "1.0",
      generatedAt: "2026-04-20T12:00:00Z",
      connections: [],
      queries: [],
      datapoints: [],
      mqtt: {
        connections: [],
        subscriptions: [],
        tagGroups: [],
        tags: [],
      },
      protocols: {},
    });
    mockProjectRuntimeUser.findAll.mockResolvedValue(runtimeSecuritySnapshot.users);
    mockProjectRole.findAll.mockResolvedValue(runtimeSecuritySnapshot.roles);
    mockProjectUserRoleBinding.findAll.mockResolvedValue(runtimeSecuritySnapshot.bindings);
    mockProjectRoleGrant.findAll.mockResolvedValue(runtimeSecuritySnapshot.grants);

    const result = await publishService.publish("project-publish", {
      version: "1.0.0",
      deployedBy: "user-1",
      authorization: "Bearer publish-token",
    });

    const updatePayload = updatedDeployment.update.mock.calls.at(-1)[0];
    const artifactFileName = updatePayload.buildConfig.artifactFileName;
    const ifpPath = path.join(ARTIFACTS_DIR, artifactFileName);

    try {
      const runtimeSecurityJson = await readIfpEntry(ifpPath, "runtime-security.json");

      expect(result).toEqual(
        expect.objectContaining({
          artifactUrl: expect.any(String),
          artifactHash: expect.any(String),
        }),
      );
      expect(updatePayload.manifest).toEqual(
        expect.objectContaining({
          security: expect.objectContaining({
            runtimeSecuritySnapshot: expect.objectContaining({
              included: true,
              fileName: "runtime-security.json",
              version: expect.any(String),
            }),
          }),
        }),
      );
      const packagedRuntimeSecurity = JSON.parse(runtimeSecurityJson);
      expect(packagedRuntimeSecurity).toEqual(
        expect.objectContaining({
          version: updatePayload.manifest.security.runtimeSecuritySnapshot.version,
          users: runtimeSecuritySnapshot.users,
          roles: runtimeSecuritySnapshot.roles,
          bindings: runtimeSecuritySnapshot.bindings,
          grants: runtimeSecuritySnapshot.grants,
        }),
      );
    } finally {
      await fs.remove(ifpPath).catch(() => undefined);
    }
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
