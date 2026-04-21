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

const createUniqueConstraintError = () =>
  Object.assign(new Error("unique constraint violation"), {
    name: "SequelizeUniqueConstraintError",
  });

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

  test("ensureRuntimeAdminBootstrap 只提供 project.creator 时也能继承默认账号和显示名", async () => {
    const transaction = { id: "tx-1b" };
    const projectCreator = {
      id: "user-project-creator-1",
      username: "project.owner",
      fullName: "工程创建者",
    };
    const project = {
      id: "project-1",
      name: "演示工程",
      creator: projectCreator,
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
      initialPassword: "Initial#123",
      transaction,
    });

    const createdUserCall = mockProjectRuntimeUser.findOrCreate.mock.calls[0][0];

    expect(mockProjectRuntimeUser.findOrCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        defaults: expect.objectContaining({
          username: projectCreator.username,
          displayName: projectCreator.fullName,
          createdBy: projectCreator.id,
          updatedBy: projectCreator.id,
        }),
      }),
    );
    expect(await bcrypt.compare("Initial#123", createdUserCall.defaults.passwordHash)).toBe(true);
    expect(result.runtimeUser.username).toBe(projectCreator.username);
    expect(result.runtimeUser.displayName).toBe(projectCreator.fullName);
  });

  test("同一创建者改名后再次 bootstrap 会复用原账号而不是新建账号", async () => {
    const transaction = { id: "tx-2" };
    const project = { id: "project-1", name: "演示工程" };
    const creator = {
      id: "user-creator-1",
      username: "alice-new",
      fullName: "张三新",
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
      username: "alice-old",
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

    mockProjectRole.findOne.mockResolvedValue(existingRole);
    mockProjectRuntimeUser.findOne.mockImplementation(({ where }) => {
      if (Object.prototype.hasOwnProperty.call(where, "createdBy")) {
        return existingUser;
      }

      return null;
    });
    mockProjectUserRoleBinding.findOne.mockResolvedValue(existingBinding);

    const result = await ensureRuntimeAdminBootstrap({
      project,
      creator,
      initialPassword: "Initial#123",
      transaction,
    });

    expect(mockProjectRuntimeUser.findOrCreate).not.toHaveBeenCalled();
    expect(mockProjectRuntimeUser.findOne).toHaveBeenCalledTimes(1);
    expect(mockProjectRuntimeUser.findOne).toHaveBeenCalledWith(
      expect.objectContaining({
        where: expect.objectContaining({
          projectId: "project-1",
          createdBy: creator.id,
        }),
        transaction,
      }),
    );
    expect(result.runtimeUser.id).toBe(existingUser.id);
    expect(result.runtimeUser.username).toBe("alice-old");
    expect(result.runtimeUser.passwordHash).toBeUndefined();
  });

  test("ensureRuntimeAdminBootstrap 已有记录时会复用而不重复创建", async () => {
    const transaction = { id: "tx-3" };
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

    mockProjectRole.findOne.mockResolvedValue(existingRole);
    mockProjectRuntimeUser.findOne.mockImplementation(({ where }) => {
      if (Object.prototype.hasOwnProperty.call(where, "createdBy")) {
        return existingUser;
      }

      return existingUser;
    });
    mockProjectUserRoleBinding.findOne.mockResolvedValue(existingBinding);

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
    expect(mockProjectRole.findOrCreate).not.toHaveBeenCalled();
    expect(mockProjectRuntimeUser.findOrCreate).not.toHaveBeenCalled();
    expect(mockProjectUserRoleBinding.findOrCreate).not.toHaveBeenCalled();
  });

  test("ensureRuntimeAdminBootstrap 遇到同名但非创建者来源的账号时应拒绝自动提权", async () => {
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
    mockProjectRuntimeUser.findOne.mockImplementation(({ where }) => {
      if (Object.prototype.hasOwnProperty.call(where, "createdBy")) {
        return null;
      }

      return {
        id: "user-999",
        projectId: "project-1",
        username: creator.username,
        createdBy: "someone-else",
        updatedBy: "someone-else",
      };
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

  test("role 的 findOrCreate 遇到唯一冲突后会回查并复用已有记录", async () => {
    const transaction = { id: "tx-5" };
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

    mockProjectRole.findOne.mockResolvedValueOnce(null).mockResolvedValueOnce(existingRole);
    mockProjectRole.findOrCreate.mockRejectedValueOnce(createUniqueConstraintError());
    mockProjectRuntimeUser.findOne.mockImplementation(({ where }) => {
      if (Object.prototype.hasOwnProperty.call(where, "createdBy")) {
        return existingUser;
      }

      return existingUser;
    });
    mockProjectUserRoleBinding.findOne.mockResolvedValue(existingBinding);

    const result = await ensureRuntimeAdminBootstrap({
      project,
      creator,
      initialPassword: "Initial#123",
      transaction,
    });

    expect(mockProjectRole.findOrCreate).toHaveBeenCalledTimes(1);
    expect(mockProjectRole.findOne).toHaveBeenCalledTimes(2);
    expect(result.role.id).toBe(existingRole.id);
  });

  test("runtimeUser 的 findOrCreate 遇到唯一冲突后会回查并执行归属校验", async () => {
    const transaction = { id: "tx-6" };
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

    mockProjectRole.findOne.mockResolvedValue(existingRole);
    mockProjectRuntimeUser.findOne
      .mockImplementationOnce(({ where }) => {
        if (Object.prototype.hasOwnProperty.call(where, "createdBy")) {
          return null;
        }

        return null;
      })
      .mockImplementationOnce(() => null)
      .mockImplementationOnce(() => null)
      .mockImplementationOnce(() => existingUser);
    mockProjectRuntimeUser.findOrCreate.mockRejectedValueOnce(createUniqueConstraintError());
    mockProjectUserRoleBinding.findOne.mockResolvedValue(existingBinding);

    const result = await ensureRuntimeAdminBootstrap({
      project,
      creator,
      initialPassword: "Initial#123",
      transaction,
    });

    expect(mockProjectRuntimeUser.findOrCreate).toHaveBeenCalledTimes(1);
    expect(mockProjectRuntimeUser.findOne).toHaveBeenCalledTimes(4);
    expect(result.runtimeUser.id).toBe(existingUser.id);
    expect(result.runtimeUser.passwordHash).toBeUndefined();
  });

  test("binding 的 findOrCreate 遇到唯一冲突后会回查并复用已有记录", async () => {
    const transaction = { id: "tx-7" };
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

    mockProjectRole.findOne.mockResolvedValue(existingRole);
    mockProjectRuntimeUser.findOne.mockImplementation(({ where }) => {
      if (Object.prototype.hasOwnProperty.call(where, "createdBy")) {
        return existingUser;
      }

      return existingUser;
    });
    mockProjectUserRoleBinding.findOne.mockResolvedValueOnce(null).mockResolvedValueOnce(existingBinding);
    mockProjectUserRoleBinding.findOrCreate.mockRejectedValueOnce(createUniqueConstraintError());

    const result = await ensureRuntimeAdminBootstrap({
      project,
      creator,
      initialPassword: "Initial#123",
      transaction,
    });

    expect(mockProjectUserRoleBinding.findOrCreate).toHaveBeenCalledTimes(1);
    expect(mockProjectUserRoleBinding.findOne).toHaveBeenCalledTimes(2);
    expect(result.binding.id).toBe(existingBinding.id);
  });

  test("buildEffectiveRoleGrantMap 的 allowRoles 和 denyRoles 是最终生效集合", () => {
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
        localAllowRoles: ["PROJECT_RUNTIME_ADMIN"],
        localDenyRoles: [],
      }),
    );
    expect(grantMap.project["project-1"].deploy).toEqual(
      expect.objectContaining({
        allowRoles: ["PROJECT_RUNTIME_ADMIN"],
        denyRoles: ["PROJECT_RUNTIME_OPERATOR"],
        localAllowRoles: [],
        localDenyRoles: ["PROJECT_RUNTIME_OPERATOR"],
      }),
    );
    expect(grantMap.project["project-2"].deploy).toEqual(
      expect.objectContaining({
        allowRoles: ["PROJECT_RUNTIME_ADMIN", "PROJECT_RUNTIME_VIEWER"],
        denyRoles: ["PROJECT_RUNTIME_DENY"],
        localAllowRoles: ["PROJECT_RUNTIME_VIEWER"],
        localDenyRoles: ["PROJECT_RUNTIME_DENY"],
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
        allowRoles: ["ROLE_SPECIFIC_ALLOW"],
        denyRoles: ["ROLE_WILDCARD_DENY"],
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
        allowRoles: ["ROLE_WILDCARD_ALLOW"],
        denyRoles: ["ROLE_SPECIFIC_DENY"],
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
