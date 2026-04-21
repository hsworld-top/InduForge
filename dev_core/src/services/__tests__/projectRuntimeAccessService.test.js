const fs = require("fs");
const path = require("path");
const bcrypt = require("bcryptjs");

const mockProjectRole = {
  findOne: jest.fn(),
  findOrCreate: jest.fn(),
  create: jest.fn(),
};

const mockProjectRuntimeUser = {
  findOne: jest.fn(),
  findOrCreate: jest.fn(),
  create: jest.fn(),
};

const mockProjectUserRoleBinding = {
  findOne: jest.fn(),
  findOrCreate: jest.fn(),
  create: jest.fn(),
};

const mockProjectRoleGrant = {
  findAll: jest.fn(),
};

jest.mock("../../models", () => ({
  ProjectRole: mockProjectRole,
  ProjectRuntimeUser: mockProjectRuntimeUser,
  ProjectUserRoleBinding: mockProjectUserRoleBinding,
  ProjectRoleGrant: mockProjectRoleGrant,
}));

const {
  DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
  ensureRuntimeAdminBootstrap,
  buildEffectiveRoleGrantMap,
} = require("../projectRuntimeAccessService");

describe("projectRuntimeAccessService", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("ensureRuntimeAdminBootstrap 缺少 initialPassword 时应失败", async () => {
    await expect(
      ensureRuntimeAdminBootstrap({
        project: { id: "project-1", name: "演示工程" },
        creator: { id: "user-creator-1", username: "alice" },
      }),
    ).rejects.toThrow("initialPassword 不能为空");
  });

  test("ensureRuntimeAdminBootstrap 会把创建者用户名作为默认账号并创建角色、用户与绑定", async () => {
    const transaction = { id: "tx-1" };
    const project = { id: "project-1", name: "演示工程" };
    const creator = {
      id: "user-creator-1",
      username: "alice",
      fullName: "张三",
    };

    mockProjectRole.findOne.mockResolvedValue(null);
    mockProjectRole.findOrCreate.mockImplementation(async ({ defaults }) => [
      {
        id: "role-1",
        ...defaults,
      },
      true,
    ]);

    mockProjectRuntimeUser.findOne.mockResolvedValue(null);
    mockProjectRuntimeUser.findOrCreate.mockImplementation(async ({ defaults }) => [
      {
        id: "user-1",
        ...defaults,
      },
      true,
    ]);

    mockProjectUserRoleBinding.findOne.mockResolvedValue(null);
    mockProjectUserRoleBinding.findOrCreate.mockImplementation(async ({ defaults }) => [
      {
        id: "binding-1",
        ...defaults,
      },
      true,
    ]);

    const result = await ensureRuntimeAdminBootstrap({
      project,
      creator,
      initialPassword: "Initial#123",
      transaction,
    });

    const createdUserCall = mockProjectRuntimeUser.findOrCreate.mock.calls[0][0];

    expect(mockProjectRole.findOrCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        defaults: expect.objectContaining({
          projectId: "project-1",
          code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
          createdBy: creator.id,
          updatedBy: creator.id,
        }),
        transaction,
      }),
    );
    expect(mockProjectRuntimeUser.findOrCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        defaults: expect.objectContaining({
          projectId: "project-1",
          username: creator.username,
          createdBy: creator.id,
          updatedBy: creator.id,
        }),
        transaction,
      }),
    );
    expect(mockProjectUserRoleBinding.findOrCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        defaults: expect.objectContaining({
          projectId: "project-1",
          runtimeUserId: "user-1",
          roleId: "role-1",
          createdBy: creator.id,
        }),
        transaction,
      }),
    );
    expect(
      await bcrypt.compare("Initial#123", createdUserCall.defaults.passwordHash),
    ).toBe(true);
    expect(result.role.code).toBe(DEFAULT_RUNTIME_ADMIN_ROLE_CODE);
    expect(result.runtimeUser.username).toBe(creator.username);
    expect(result.runtimeUser.passwordHash).toBeUndefined();
  });

  test("ensureRuntimeAdminBootstrap 已有记录时会复用而不重复创建", async () => {
    const transaction = { id: "tx-2" };
    const project = { id: "project-1", name: "演示工程" };
    const creator = {
      id: "user-creator-1",
      username: "alice",
      fullName: "张三",
    };
    const existingRole = {
      id: "role-1",
      projectId: "project-1",
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
      createdBy: creator.id,
      updatedBy: creator.id,
    };
    const existingUser = {
      id: "user-1",
      projectId: "project-1",
      username: creator.username,
      createdBy: creator.id,
      updatedBy: creator.id,
      passwordHash: await bcrypt.hash("Initial#123", 12),
    };
    const existingBinding = {
      id: "binding-1",
      projectId: "project-1",
      runtimeUserId: "user-1",
      roleId: "role-1",
    };

    mockProjectRole.findOne.mockResolvedValueOnce(null).mockResolvedValue(existingRole);
    mockProjectRole.findOrCreate.mockImplementation(async ({ defaults }) => [
      {
        ...defaults,
        id: "role-1",
      },
      false,
    ]);
    mockProjectRuntimeUser.findOne
      .mockResolvedValueOnce(null)
      .mockResolvedValue(existingUser);
    mockProjectRuntimeUser.findOrCreate.mockImplementation(async ({ defaults }) => [
      {
        ...defaults,
        id: "user-1",
      },
      false,
    ]);
    mockProjectUserRoleBinding.findOne
      .mockResolvedValueOnce(null)
      .mockResolvedValue(existingBinding);
    mockProjectUserRoleBinding.findOrCreate.mockImplementation(async ({ defaults }) => [
      {
        ...defaults,
        id: "binding-1",
      },
      false,
    ]);

    const firstResult = await ensureRuntimeAdminBootstrap({
      project,
      creator,
      initialPassword: "Initial#123",
      transaction,
    });
    const secondResult = await ensureRuntimeAdminBootstrap({
      project,
      creator,
      initialPassword: "Initial#123",
      transaction,
    });

    expect(firstResult.binding.id).toBe("binding-1");
    expect(secondResult.binding.id).toBe("binding-1");
    expect(mockProjectRole.findOrCreate).toHaveBeenCalledTimes(1);
    expect(mockProjectRuntimeUser.findOrCreate).toHaveBeenCalledTimes(1);
    expect(mockProjectUserRoleBinding.findOrCreate).toHaveBeenCalledTimes(1);
    expect(secondResult.runtimeUser.passwordHash).toBeUndefined();
  });

  test("ensureRuntimeAdminBootstrap 遇到同名但非创建者来源的账号时应拒绝自动提权", async () => {
    const transaction = { id: "tx-3" };
    const project = { id: "project-1", name: "演示工程" };
    const creator = {
      id: "user-creator-1",
      username: "alice",
      fullName: "张三",
    };

    mockProjectRole.findOne.mockResolvedValue({
      id: "role-1",
      projectId: "project-1",
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
    });
    mockProjectRuntimeUser.findOne.mockResolvedValue({
      id: "user-999",
      projectId: "project-1",
      username: creator.username,
      createdBy: "someone-else",
      updatedBy: "someone-else",
    });

    await expect(
      ensureRuntimeAdminBootstrap({
        project,
        creator,
        initialPassword: "Initial#123",
        transaction,
      }),
    ).rejects.toThrow("拒绝自动提权");
    expect(mockProjectRuntimeUser.findOrCreate).not.toHaveBeenCalled();
  });

  test("ensureRuntimeAdminBootstrap 在 findOrCreate 命中已有同名账号时也要校验归属", async () => {
    const transaction = { id: "tx-4" };
    const project = { id: "project-1", name: "演示工程" };
    const creator = {
      id: "user-creator-1",
      username: "alice",
      fullName: "张三",
    };

    mockProjectRole.findOne.mockResolvedValue({
      id: "role-1",
      projectId: "project-1",
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
    });
    mockProjectRuntimeUser.findOne.mockResolvedValue(null);
    mockProjectRuntimeUser.findOrCreate.mockImplementation(async ({ defaults }) => [
      {
        id: "user-999",
        ...defaults,
        createdBy: "someone-else",
        updatedBy: "someone-else",
      },
      false,
    ]);

    await expect(
      ensureRuntimeAdminBootstrap({
        project,
        creator,
        initialPassword: "Initial#123",
        transaction,
      }),
    ).rejects.toThrow("拒绝自动提权");
  });

  test("buildEffectiveRoleGrantMap 会按 resourceId 粒度聚合并让 deny 覆盖 allow", () => {
    const grantMap = buildEffectiveRoleGrantMap([
      {
        roleCode: "PROJECT_RUNTIME_ADMIN",
        resourceType: "project",
        resourceId: "*",
        action: "deploy",
        effect: "allow",
      },
      {
        roleCode: "PROJECT_RUNTIME_OPERATOR",
        resourceType: "project",
        resourceId: "project-1",
        action: "deploy",
        effect: "allow",
      },
      {
        roleCode: "PROJECT_RUNTIME_OPERATOR",
        resourceType: "project",
        resourceId: "project-1",
        action: "deploy",
        effect: "deny",
      },
      {
        roleCode: "PROJECT_RUNTIME_VIEWER",
        resourceType: "project",
        resourceId: "project-2",
        action: "deploy",
        effect: "allow",
      },
      {
        roleCode: "PROJECT_RUNTIME_DENY",
        resourceType: "project",
        resourceId: "project-2",
        action: "deploy",
        effect: "deny",
      },
    ]);

    expect(grantMap.project["*"].deploy).toEqual(
      expect.objectContaining({
        allowRoles: ["PROJECT_RUNTIME_ADMIN"],
        denyRoles: [],
        effectiveAllowRoles: ["PROJECT_RUNTIME_ADMIN"],
        effectiveDenyRoles: [],
      }),
    );
    expect(grantMap.project["project-1"].deploy).toEqual(
      expect.objectContaining({
        allowRoles: [],
        denyRoles: ["PROJECT_RUNTIME_OPERATOR"],
        effectiveAllowRoles: ["PROJECT_RUNTIME_ADMIN"],
        effectiveDenyRoles: ["PROJECT_RUNTIME_OPERATOR"],
      }),
    );
    expect(grantMap.project["project-2"].deploy).toEqual(
      expect.objectContaining({
        allowRoles: ["PROJECT_RUNTIME_VIEWER"],
        denyRoles: ["PROJECT_RUNTIME_DENY"],
        effectiveAllowRoles: ["PROJECT_RUNTIME_ADMIN", "PROJECT_RUNTIME_VIEWER"],
        effectiveDenyRoles: ["PROJECT_RUNTIME_DENY"],
      }),
    );
  });

  test("buildEffectiveRoleGrantMap 会让 wildcard deny 对具体实例仍然生效，且具体 deny 也能覆盖 wildcard allow", () => {
    const denyGrantMap = buildEffectiveRoleGrantMap([
      {
        roleCode: "ROLE_WILDCARD_DENY",
        resourceType: "project",
        resourceId: "*",
        action: "start",
        effect: "deny",
      },
      {
        roleCode: "ROLE_SPECIFIC_ALLOW",
        resourceType: "project",
        resourceId: "project-1",
        action: "start",
        effect: "allow",
      },
    ]);

    expect(denyGrantMap.project["project-1"].start).toEqual(
      expect.objectContaining({
        effectiveAllowRoles: ["ROLE_SPECIFIC_ALLOW"],
        effectiveDenyRoles: ["ROLE_WILDCARD_DENY"],
      }),
    );

    const allowGrantMap = buildEffectiveRoleGrantMap([
      {
        roleCode: "ROLE_WILDCARD_ALLOW",
        resourceType: "project",
        resourceId: "*",
        action: "stop",
        effect: "allow",
      },
      {
        roleCode: "ROLE_SPECIFIC_DENY",
        resourceType: "project",
        resourceId: "project-2",
        action: "stop",
        effect: "deny",
      },
    ]);

    expect(allowGrantMap.project["project-2"].stop).toEqual(
      expect.objectContaining({
        effectiveAllowRoles: ["ROLE_WILDCARD_ALLOW"],
        effectiveDenyRoles: ["ROLE_SPECIFIC_DENY"],
      }),
    );
  });

  test("init.sql 包含同工程一致性复合约束", () => {
    const initSql = fs.readFileSync(
      path.join(__dirname, "../../../database/init.sql"),
      "utf8",
    );

    expect(initSql).toContain('CONSTRAINT project_runtime_users_id_project_uq UNIQUE ("id", "projectId")');
    expect(initSql).toContain('CONSTRAINT project_roles_id_project_uq UNIQUE ("id", "projectId")');
    expect(initSql).toContain('FOREIGN KEY ("runtimeUserId", "projectId")');
    expect(initSql).toContain('FOREIGN KEY ("roleId", "projectId")');
    expect(initSql).toContain('CONSTRAINT project_role_grants_role_project_fk');
  });
});
