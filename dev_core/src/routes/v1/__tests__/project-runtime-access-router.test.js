const express = require("express");
const request = require("supertest");

const mockTransaction = { id: "tx-project-create" };
const mockProject = {
  sequelize: {
    transaction: jest.fn(async (callback) => callback(mockTransaction)),
  },
  create: jest.fn(),
  findByPk: jest.fn(),
};

const mockService = {
  ensureRuntimeAdminBootstrap: jest.fn(),
  listRuntimeUsers: jest.fn(),
  createRuntimeUser: jest.fn(),
  updateRuntimeUserStatus: jest.fn(),
  resetRuntimeUserPassword: jest.fn(),
  bindRuntimeUserRoles: jest.fn(),
  listRuntimeRoles: jest.fn(),
  createRuntimeRole: jest.fn(),
  updateRuntimeRole: jest.fn(),
  deleteRuntimeRole: jest.fn(),
};

const mockUpsertProjectSettings = jest.fn();
const mockReplaceProjectSnapshot = jest.fn();
const mockLogger = { error: jest.fn(), warn: jest.fn() };

jest.mock("crypto", () => {
  const actualCrypto = jest.requireActual("crypto");
  return {
    ...actualCrypto,
    randomBytes: jest.fn(() => Buffer.from("0123456789abcdef0123456789abcdef", "hex")),
    randomUUID: jest.fn(() => "uuid-1"),
  };
});

jest.mock("../../../models", () => ({
  Project: mockProject,
  Tenant: {},
  User: {},
  DesignPage: {},
  NodeDeployment: { findAll: jest.fn() },
}));

jest.mock("../../../middlewares/auth", () => ({
  authenticateToken: (req, res, next) => {
    req.user = {
      id: "user-1",
      username: "owner",
      fullName: "工程创建者",
      tenantId: "tenant-1",
      role: "SYSTEM_ADMIN",
    };
    next();
  },
  requireResourceOwnership: () => (req, res, next) => next(),
  hasCapability: () => true,
}));

jest.mock("../../../middlewares/validate", () => ({
  validate: () => (req, res, next) => next(),
}));

jest.mock("../../../utils/logger", () => ({
  logger: mockLogger,
}));

jest.mock("../../../services/projectRuntimeAccessService", () => mockService);

jest.mock("../../../services/projectSettingsStore", () => ({
  upsertProjectSettings: mockUpsertProjectSettings,
  getProjectSettingsRow: jest.fn(),
}));

jest.mock("../../../services/dataDomainClient", () => ({
  dataDomainClient: {
    replaceProjectSnapshot: mockReplaceProjectSnapshot,
    getProjectArtifact: jest.fn(),
  },
}));

jest.mock("../../../services/deploymentService", () => ({
  undeploy: jest.fn(),
}));

jest.mock("../../../services/designAssetService", () => ({
  deleteAssetsByProject: jest.fn(),
  deleteFoldersByProject: jest.fn(),
}));

const router = require("../project");

const createApp = () => {
  const app = express();
  app.use(express.json());
  app.use("/projects", router);
  return app;
};

describe("project runtime access router", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockProject.sequelize.transaction.mockImplementation(async (callback) => callback(mockTransaction));
    mockProject.findByPk.mockResolvedValue({
      id: "project-1",
      tenantId: "tenant-1",
      creator: { id: "user-1", username: "owner", fullName: "工程创建者" },
    });
  });

  test("创建工程后会在事务内初始化默认工程管理员账号并传入随机 initialPassword", async () => {
    mockProject.create.mockResolvedValue({ id: "project-1", tenantId: "tenant-1" });
    mockService.ensureRuntimeAdminBootstrap.mockResolvedValue({});

    const response = await request(createApp()).post("/projects").send({ name: "新工程" });

    expect(response.status).toBe(201);
    expect(mockProject.sequelize.transaction).toHaveBeenCalledTimes(1);
    expect(mockProject.create).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "新工程",
        tenantId: "tenant-1",
        createdBy: "user-1",
      }),
      expect.objectContaining({
        transaction: mockTransaction,
      }),
    );
    expect(mockService.ensureRuntimeAdminBootstrap).toHaveBeenCalledWith(
      expect.objectContaining({
        project: expect.objectContaining({ id: "project-1" }),
        creator: expect.objectContaining({
          id: "user-1",
          username: "owner",
          fullName: "工程创建者",
        }),
        transaction: mockTransaction,
        initialPassword: "0123456789abcdef0123456789abcdef",
      }),
    );
    expect(mockUpsertProjectSettings).toHaveBeenCalledTimes(1);
    expect(JSON.stringify(response.body)).not.toContain("0123456789abcdef0123456789abcdef");
  });

  test("创建工程时如果 bootstrap 失败则返回 500，且不会继续查询工程详情", async () => {
    mockProject.create.mockResolvedValue({ id: "project-1", tenantId: "tenant-1" });
    mockService.ensureRuntimeAdminBootstrap.mockRejectedValue(new Error("bootstrap failed"));

    const response = await request(createApp()).post("/projects").send({ name: "新工程" });

    expect(response.status).toBe(500);
    expect(mockProject.sequelize.transaction).toHaveBeenCalledTimes(1);
    expect(mockProject.findByPk).not.toHaveBeenCalled();
    expect(mockUpsertProjectSettings).not.toHaveBeenCalled();
  });

  test("GET /projects/:id/runtime-users 返回 runtimeUsers", async () => {
    mockService.listRuntimeUsers.mockResolvedValue([
      { id: "runtime-user-1", username: "operator", roleIds: ["role-1"] },
    ]);

    const response = await request(createApp()).get("/projects/project-1/runtime-users");

    expect(response.status).toBe(200);
    expect(response.body.data.runtimeUsers).toEqual([
      expect.objectContaining({ username: "operator", roleIds: ["role-1"] }),
    ]);
  });

  test("POST /projects/:id/runtime-users 可创建用户并绑定角色", async () => {
    mockService.createRuntimeUser.mockResolvedValue({
      id: "runtime-user-1",
      projectId: "project-1",
      username: "operator",
      displayName: "值班员",
      status: "active",
      roleIds: ["role-1", "role-2"],
      roles: [
        { id: "role-1", code: "OPERATOR", name: "值班员" },
        { id: "role-2", code: "VIEWER", name: "观察员" },
      ],
    });

    const response = await request(createApp())
      .post("/projects/project-1/runtime-users")
      .send({
        username: "operator",
        displayName: "值班员",
        initialPassword: "Initial#123",
        roleIds: ["role-1", "role-2"],
      });

    expect(response.status).toBe(201);
    expect(mockService.createRuntimeUser).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: "project-1",
        actorId: "user-1",
        username: "operator",
        displayName: "值班员",
        initialPassword: "Initial#123",
        roleIds: ["role-1", "role-2"],
      }),
    );
    expect(response.body.data.runtimeUser).toEqual(
      expect.objectContaining({
        username: "operator",
        roleIds: ["role-1", "role-2"],
      }),
    );
  });

  test("GET /projects/:id/runtime-roles 返回 runtimeRoles", async () => {
    mockService.listRuntimeRoles.mockResolvedValue([
      { id: "role-1", code: "OPERATOR", name: "值班员", bindingCount: 2, grantCount: 3 },
    ]);

    const response = await request(createApp()).get("/projects/project-1/runtime-roles");

    expect(response.status).toBe(200);
    expect(response.body.data.runtimeRoles).toEqual([
      expect.objectContaining({ code: "OPERATOR", bindingCount: 2, grantCount: 3 }),
    ]);
  });

  test("DELETE /projects/:id/runtime-roles/:roleId 会把业务错误透传为统一响应", async () => {
    mockService.deleteRuntimeRole.mockRejectedValue(
      Object.assign(new Error("系统内置角色不能删除"), {
        errorCode: "B0001",
        statusCode: 400,
        options: { message: "系统内置角色不能删除" },
      }),
    );

    const response = await request(createApp()).delete("/projects/project-1/runtime-roles/role-admin");

    expect(response.status).toBe(400);
    expect(response.body).toEqual(
      expect.objectContaining({
        success: false,
        errorCode: "B0001",
        message: "系统内置角色不能删除",
      }),
    );
  });
});
