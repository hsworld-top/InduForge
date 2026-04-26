const fs = require("fs");
const path = require("path");
const bcrypt = require("bcryptjs");

const mockProjectRole = {
  findOne: jest.fn(),
  findOrCreate: jest.fn(),
  create: jest.fn(),
  findAll: jest.fn(),
  findByPk: jest.fn(),
};

const mockProjectRuntimeUser = {
  findOne: jest.fn(),
  findOrCreate: jest.fn(),
  create: jest.fn(),
  findAll: jest.fn(),
};

const mockProjectUserRoleBinding = {
  findOne: jest.fn(),
  findOrCreate: jest.fn(),
  create: jest.fn(),
  findAll: jest.fn(),
  destroy: jest.fn(),
};

const mockProjectRoleGrant = {
  findAll: jest.fn(),
};

const mockProjectTransaction = jest.fn(async (callback) => callback({ id: "tx-service" }));

jest.mock("../../models", () => ({
  ProjectRole: mockProjectRole,
  ProjectRuntimeUser: mockProjectRuntimeUser,
  ProjectUserRoleBinding: mockProjectUserRoleBinding,
  ProjectRoleGrant: mockProjectRoleGrant,
  Project: {
    sequelize: {
      transaction: mockProjectTransaction,
    },
  },
}));

const {
  DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
  DEFAULT_RUNTIME_ADMIN_NAME,
  ensureRuntimeAdminBootstrap,
  buildEffectiveRoleGrantMap,
  listRuntimeUsers,
  createRuntimeUser,
  updateRuntimeUserStatus,
  resetRuntimeUserPassword,
  bindRuntimeUserRoles,
  listRuntimeRoles,
  createRuntimeRole,
  updateRuntimeRole,
  deleteRuntimeRole,
  normalizeRuntimeRoleCode,
} = require("../projectRuntimeAccessService");

const createUniqueConstraintError = () =>
  Object.assign(new Error("unique constraint violation"), {
    name: "SequelizeUniqueConstraintError",
  });

describe("projectRuntimeAccessService", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockProjectTransaction.mockImplementation(async (callback) => callback({ id: "tx-service" }));
  });

  test("normalizeRuntimeRoleCode 会统一补齐 PROJECT_ 前缀", () => {
    expect(normalizeRuntimeRoleCode("operator role")).toBe("PROJECT_OPERATOR_ROLE");
    expect(normalizeRuntimeRoleCode("project_viewer")).toBe("PROJECT_VIEWER");
    expect(normalizeRuntimeRoleCode("")).toBe("");
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
          name: DEFAULT_RUNTIME_ADMIN_NAME,
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
    expect(result.role.name).toBe(DEFAULT_RUNTIME_ADMIN_NAME);
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
        roleCode: "PROJECT_ADMIN",
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
        allowRoles: ["PROJECT_ADMIN"],
        denyRoles: [],
        localAllowRoles: ["PROJECT_ADMIN"],
        localDenyRoles: [],
      }),
    );
    expect(grantMap.project["project-1"].deploy).toEqual(
      expect.objectContaining({
        allowRoles: ["PROJECT_ADMIN"],
        denyRoles: ["PROJECT_RUNTIME_OPERATOR"],
        localAllowRoles: [],
        localDenyRoles: ["PROJECT_RUNTIME_OPERATOR"],
      }),
    );
    expect(grantMap.project["project-2"].deploy).toEqual(
      expect.objectContaining({
        allowRoles: ["PROJECT_ADMIN", "PROJECT_RUNTIME_VIEWER"],
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

  test("listRuntimeUsers 会返回脱敏后的用户列表并附带 roleIds", async () => {
    mockProjectRuntimeUser.findAll.mockResolvedValue([
      {
        id: "runtime-user-1",
        projectId: "project-1",
        username: "operator",
        displayName: "值班员",
        status: "inactive",
        passwordHash: "hash-1",
        roleBindings: [
          { roleId: "role-1", role: { id: "role-1", code: "OPERATOR", name: "值班员" } },
          { roleId: "role-2", role: { id: "role-2", code: "VIEWER", name: "观察员" } },
        ],
      },
    ]);

    const result = await listRuntimeUsers({ projectId: "project-1" });

    expect(mockProjectRuntimeUser.findAll).toHaveBeenCalledWith(
      expect.objectContaining({
        where: { projectId: "project-1" },
      }),
    );
    expect(result[0].passwordHash).toBeUndefined();
    expect(result[0].status).toBe("disabled");
    expect(result[0].roleIds).toEqual(["role-1", "role-2"]);
    expect(result[0].roles).toEqual([
      expect.objectContaining({ id: "role-1", code: "OPERATOR" }),
      expect.objectContaining({ id: "role-2", code: "VIEWER" }),
    ]);
  });

  test("createRuntimeUser 会创建用户并绑定角色，且返回对象不含 passwordHash", async () => {
    mockProjectRole.findAll.mockResolvedValue([
      { id: "role-1", projectId: "project-1", code: "OPERATOR", name: "值班员" },
      { id: "role-2", projectId: "project-1", code: "VIEWER", name: "观察员" },
    ]);
    mockProjectRuntimeUser.findOne
      .mockResolvedValueOnce(null)
      .mockResolvedValueOnce({
        id: "runtime-user-1",
        projectId: "project-1",
        username: "operator",
        displayName: "值班员",
        status: "active",
        passwordHash: "hash-operator",
        roleBindings: [
          { roleId: "role-1", role: { id: "role-1", code: "OPERATOR", name: "值班员" } },
          { roleId: "role-2", role: { id: "role-2", code: "VIEWER", name: "观察员" } },
        ],
      });
    mockProjectRuntimeUser.create.mockResolvedValue({
      id: "runtime-user-1",
      projectId: "project-1",
      username: "operator",
      displayName: "值班员",
      status: "active",
      passwordHash: "hash-operator",
      toJSON() {
        return {
          id: this.id,
          projectId: this.projectId,
          username: this.username,
          displayName: this.displayName,
          status: this.status,
          passwordHash: this.passwordHash,
        };
      },
    });
    mockProjectUserRoleBinding.create
      .mockResolvedValueOnce({ id: "binding-1", roleId: "role-1" })
      .mockResolvedValueOnce({ id: "binding-2", roleId: "role-2" });

    const result = await createRuntimeUser({
      projectId: "project-1",
      actorId: "user-1",
      username: "operator",
      displayName: "值班员",
      initialPassword: "Initial#123",
      roleIds: ["role-1", "role-2"],
    });

    expect(mockProjectRuntimeUser.create).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: "project-1",
        username: "operator",
        displayName: "值班员",
      }),
      expect.objectContaining({
        transaction: expect.any(Object),
      }),
    );
    expect(mockProjectUserRoleBinding.create).toHaveBeenCalledTimes(2);
    expect(result.passwordHash).toBeUndefined();
    expect(result.roleIds).toEqual(["role-1", "role-2"]);
    expect(result.roles).toEqual([
      expect.objectContaining({ id: "role-1", code: "OPERATOR" }),
      expect.objectContaining({ id: "role-2", code: "VIEWER" }),
    ]);
    expect(result.status).toBe("active");
  });

  test("createRuntimeUser 遇到并发唯一键冲突时会翻译成 409 业务错误", async () => {
    mockProjectRole.findAll.mockResolvedValue([]);
    mockProjectRuntimeUser.findOne.mockResolvedValue(null);
    mockProjectRuntimeUser.create.mockRejectedValue(
      Object.assign(new Error("unique conflict"), {
        name: "SequelizeUniqueConstraintError",
      }),
    );

    await expect(
      createRuntimeUser({
        projectId: "project-1",
        actorId: "user-1",
        username: "operator",
        displayName: "值班员",
        initialPassword: "Initial#123",
        roleIds: [],
      }),
    ).rejects.toMatchObject({
      errorCode: "B1002",
      statusCode: 409,
      options: expect.objectContaining({
        message: "运行态用户名已存在",
      }),
    });
  });

  test("updateRuntimeUserStatus 只接受 active 或 disabled，并对外返回 active 或 disabled", async () => {
    const runtimeUserRecord = {
      update: jest.fn().mockResolvedValue(undefined),
    };
    mockProjectRuntimeUser.findOne
      .mockResolvedValueOnce(runtimeUserRecord)
      .mockResolvedValueOnce({
        id: "runtime-user-1",
        projectId: "project-1",
        username: "operator",
        displayName: "值班员",
        status: "inactive",
        passwordHash: "hash-operator",
        roleBindings: [
          {
            roleId: "role-1",
            role: {
              id: "role-1",
              code: "VIEWER",
              name: "观察员",
              status: "inactive",
            },
          },
        ],
      });

    const result = await updateRuntimeUserStatus({
      projectId: "project-1",
      runtimeUserId: "runtime-user-1",
      status: "disabled",
      actorId: "user-1",
    });

    expect(runtimeUserRecord.update).toHaveBeenCalledWith({
      status: "inactive",
      updatedBy: "user-1",
    });
    expect(result.status).toBe("disabled");
    expect(result.passwordHash).toBeUndefined();
    expect(result.roles).toEqual([
      expect.objectContaining({
        id: "role-1",
        status: "disabled",
      }),
    ]);
  });

  test("updateRuntimeUserStatus 会拒绝 inactive 或 suspended 这样的旧枚举输入", async () => {
    await expect(
      updateRuntimeUserStatus({
        projectId: "project-1",
        runtimeUserId: "runtime-user-1",
        status: "inactive",
        actorId: "user-1",
      }),
    ).rejects.toMatchObject({
      errorCode: "B0001",
      statusCode: 400,
      options: expect.objectContaining({
        message: "状态值不合法",
      }),
    });
  });

  test("updateRuntimeUserStatus 不允许禁用唯一 active 的运行态管理员", async () => {
    const runtimeUserRecord = {
      id: "runtime-user-1",
      projectId: "project-1",
      status: "active",
      update: jest.fn().mockResolvedValue(undefined),
    };
    mockProjectRuntimeUser.findOne.mockResolvedValue(runtimeUserRecord);
    mockProjectRole.findOne.mockResolvedValue({
      id: "role-admin",
      projectId: "project-1",
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
      isSystem: true,
    });
    mockProjectUserRoleBinding.findAll
      .mockResolvedValueOnce([{ roleId: "role-admin" }])
      .mockResolvedValueOnce([{ runtimeUserId: "runtime-user-1" }]);
    mockProjectRuntimeUser.findAll.mockResolvedValue([]);

    await expect(
      updateRuntimeUserStatus({
        projectId: "project-1",
        runtimeUserId: "runtime-user-1",
        status: "disabled",
        actorId: "user-1",
      }),
    ).rejects.toMatchObject({
      errorCode: "B0001",
      statusCode: 400,
      options: expect.objectContaining({
        message: "至少保留一个启用中的运行态管理员",
      }),
    });
    expect(runtimeUserRecord.update).not.toHaveBeenCalled();
  });

  test("updateRuntimeUserStatus 在还有其他 active 管理员时允许禁用当前管理员", async () => {
    const runtimeUserRecord = {
      id: "runtime-user-1",
      projectId: "project-1",
      status: "active",
      update: jest.fn().mockResolvedValue(undefined),
    };
    mockProjectRuntimeUser.findOne
      .mockResolvedValueOnce(runtimeUserRecord)
      .mockResolvedValueOnce({
        id: "runtime-user-1",
        projectId: "project-1",
        username: "operator",
        displayName: "值班员",
        status: "inactive",
        passwordHash: "hash-operator",
        roleBindings: [],
      });
    mockProjectRole.findOne.mockResolvedValue({
      id: "role-admin",
      projectId: "project-1",
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
      isSystem: true,
    });
    mockProjectUserRoleBinding.findAll
      .mockResolvedValueOnce([{ roleId: "role-admin" }])
      .mockResolvedValueOnce([
        { runtimeUserId: "runtime-user-1" },
        { runtimeUserId: "runtime-user-2" },
      ]);
    mockProjectRuntimeUser.findAll.mockResolvedValue([
      { id: "runtime-user-2", status: "active" },
    ]);

    const result = await updateRuntimeUserStatus({
      projectId: "project-1",
      runtimeUserId: "runtime-user-1",
      status: "disabled",
      actorId: "user-1",
    });

    expect(runtimeUserRecord.update).toHaveBeenCalledWith({
      status: "inactive",
      updatedBy: "user-1",
    });
    expect(result.status).toBe("disabled");
  });

  test("bindRuntimeUserRoles 不允许把工程里的最后一个管理员解绑干净", async () => {
    mockProjectRole.findAll.mockResolvedValue([
      { id: "role-viewer" },
    ]);
    mockProjectRole.findOne.mockResolvedValue({
      id: "role-admin",
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
      isSystem: true,
    });
    mockProjectRuntimeUser.findOne.mockResolvedValue({
      id: "runtime-user-1",
      projectId: "project-1",
      status: "active",
    });
    mockProjectUserRoleBinding.findAll
      .mockResolvedValueOnce([{ roleId: "role-admin" }])
      .mockResolvedValueOnce([{ runtimeUserId: "runtime-user-1" }]);

    await expect(
      bindRuntimeUserRoles({
        projectId: "project-1",
        runtimeUserId: "runtime-user-1",
        roleIds: ["role-viewer"],
        actorId: "user-1",
      }),
    ).rejects.toMatchObject({
      errorCode: "B0001",
      statusCode: 400,
      options: expect.objectContaining({
        message: "至少保留一个启用中的运行态管理员",
      }),
    });
  });

  test("bindRuntimeUserRoles 在剩余管理员都已 disabled 时不允许移除当前 active 管理员角色", async () => {
    mockProjectRole.findAll.mockResolvedValue([
      { id: "role-viewer" },
    ]);
    mockProjectRole.findOne.mockResolvedValue({
      id: "role-admin",
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
      isSystem: true,
    });
    mockProjectRuntimeUser.findOne.mockResolvedValue({
      id: "runtime-user-1",
      projectId: "project-1",
      status: "active",
    });
    mockProjectUserRoleBinding.findAll
      .mockResolvedValueOnce([{ roleId: "role-admin" }])
      .mockResolvedValueOnce([
        { runtimeUserId: "runtime-user-1" },
        { runtimeUserId: "runtime-user-2" },
      ]);
    mockProjectRuntimeUser.findAll.mockResolvedValue([]);

    await expect(
      bindRuntimeUserRoles({
        projectId: "project-1",
        runtimeUserId: "runtime-user-1",
        roleIds: ["role-viewer"],
        actorId: "user-1",
      }),
    ).rejects.toMatchObject({
      errorCode: "B0001",
      statusCode: 400,
      options: expect.objectContaining({
        message: "至少保留一个启用中的运行态管理员",
      }),
    });
  });

  test("listRuntimeRoles 会返回 bindingCount 和 grantCount", async () => {
    mockProjectRole.findAll.mockResolvedValue([
      {
        id: "role-1",
        projectId: "project-1",
        code: "OPERATOR",
        name: "值班员",
        isSystem: false,
        userBindings: [
          { runtimeUserId: "runtime-user-1" },
          { runtimeUserId: "runtime-user-2" },
        ],
        grants: [{ id: "grant-1" }, { id: "grant-2" }, { id: "grant-3" }],
      },
    ]);

    const result = await listRuntimeRoles({ projectId: "project-1" });

    expect(result).toEqual([
      expect.objectContaining({
        code: "OPERATOR",
        bindingCount: 2,
        grantCount: 3,
      }),
    ]);
    expect(result[0].userCount).toBeUndefined();
  });

  test("createRuntimeRole 会规范化角色编码后再保存", async () => {
    mockProjectRole.findOne.mockResolvedValue(null);
    mockProjectRole.create.mockResolvedValue({
      id: "role-operator",
      projectId: "project-1",
      code: "PROJECT_OPERATOR_ROLE",
      name: "值班员",
      description: null,
      isSystem: false,
      status: "active",
      userBindings: [],
      grants: [],
    });

    const result = await createRuntimeRole({
      projectId: "project-1",
      actorId: "user-1",
      code: "operator role",
      name: "值班员",
    });

    expect(mockProjectRole.findOne).toHaveBeenCalledWith(
      expect.objectContaining({
        where: expect.objectContaining({
          projectId: "project-1",
          code: "PROJECT_OPERATOR_ROLE",
        }),
      }),
    );
    expect(mockProjectRole.create).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: "project-1",
        code: "PROJECT_OPERATOR_ROLE",
        name: "值班员",
        createdBy: "user-1",
        updatedBy: "user-1",
      }),
    );
    expect(result).toEqual(expect.objectContaining({ code: "PROJECT_OPERATOR_ROLE" }));
  });

  test("updateRuntimeRole 会规范化角色编码并检查重复", async () => {
    const roleRecord = {
      id: "role-operator",
      projectId: "project-1",
      code: "PROJECT_OPERATOR",
      isSystem: false,
      status: "active",
      update: jest.fn().mockResolvedValue(undefined),
    };
    mockProjectRole.findOne
      .mockResolvedValueOnce(roleRecord)
      .mockResolvedValueOnce(null)
      .mockResolvedValueOnce({
        id: "role-operator",
        projectId: "project-1",
        code: "PROJECT_VIEWER",
        name: "观察员",
        description: null,
        isSystem: false,
        status: "active",
        userBindings: [],
        grants: [],
      });

    const result = await updateRuntimeRole({
      projectId: "project-1",
      roleId: "role-operator",
      actorId: "user-1",
      code: "viewer",
      name: "观察员",
    });

    expect(mockProjectRole.findOne).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        where: expect.objectContaining({
          projectId: "project-1",
          code: "PROJECT_VIEWER",
        }),
      }),
    );
    expect(roleRecord.update).toHaveBeenCalledWith(
      expect.objectContaining({
        code: "PROJECT_VIEWER",
        name: "观察员",
        updatedBy: "user-1",
      }),
    );
    expect(result).toEqual(expect.objectContaining({ code: "PROJECT_VIEWER" }));
  });

  test("deleteRuntimeRole 会拒绝删除系统内置角色", async () => {
    mockProjectRole.findOne.mockResolvedValue({
      id: "role-admin",
      projectId: "project-1",
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
      isSystem: true,
    });

    await expect(
      deleteRuntimeRole({
        projectId: "project-1",
        roleId: "role-admin",
        actorId: "user-1",
      }),
    ).rejects.toMatchObject({
      errorCode: "B0001",
      statusCode: 400,
      options: expect.objectContaining({
        message: "系统内置角色不能删除",
      }),
    });
  });

  test("updateRuntimeRole 不允许编辑系统角色", async () => {
    mockProjectRole.findOne.mockResolvedValue({
      id: "role-admin",
      projectId: "project-1",
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
      isSystem: true,
      status: "active",
    });

    await expect(
      updateRuntimeRole({
        projectId: "project-1",
        roleId: "role-admin",
        actorId: "user-1",
        code: "OTHER_CODE",
        status: "inactive",
      }),
    ).rejects.toMatchObject({
      errorCode: "B0001",
      statusCode: 400,
      options: expect.objectContaining({
        message: "系统内置角色不允许编辑",
      }),
    });
  });

  test("deleteRuntimeRole 会拒绝删除已绑定用户的角色", async () => {
    mockProjectRole.findOne.mockResolvedValue({
      id: "role-operator",
      projectId: "project-1",
      code: "OPERATOR",
      isSystem: false,
      userBindings: [{ runtimeUserId: "runtime-user-1" }],
      grants: [],
    });

    await expect(
      deleteRuntimeRole({
        projectId: "project-1",
        roleId: "role-operator",
        actorId: "user-1",
      }),
    ).rejects.toMatchObject({
      errorCode: "B0001",
      statusCode: 400,
      options: expect.objectContaining({
        message: "角色存在绑定或授权记录，无法删除",
        bindingCount: 1,
        grantCount: 0,
      }),
    });
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
