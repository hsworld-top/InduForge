const bcrypt = require("bcryptjs");

const mockProjectRole = {
  findOne: jest.fn(),
  create: jest.fn(),
};

const mockProjectRuntimeUser = {
  findOne: jest.fn(),
  create: jest.fn(),
};

const mockProjectUserRoleBinding = {
  findOne: jest.fn(),
  create: jest.fn(),
};

const mockProjectRoleGrant = {
  findAll: jest.fn(),
};

jest.mock("bcryptjs", () => ({
  hash: jest.fn(),
  compare: jest.fn(),
}));

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

  test("ensureRuntimeAdminBootstrap 会把创建者用户名作为默认账号并创建角色、用户与绑定", async () => {
    const hashedPassword = "bcrypt-hash-runtime-admin";
    bcrypt.hash.mockResolvedValue(hashedPassword);
    bcrypt.compare.mockResolvedValue(true);
    const transaction = { id: "tx-1" };
    const project = { id: "project-1", name: "演示工程" };
    const creator = {
      id: "user-creator-1",
      username: "alice",
      fullName: "张三",
    };

    mockProjectRole.findOne.mockResolvedValue(null);
    mockProjectRole.create.mockResolvedValue({
      id: "role-1",
      projectId: "project-1",
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
      name: "运行态管理员",
    });

    mockProjectRuntimeUser.findOne.mockResolvedValue(null);
    mockProjectRuntimeUser.create.mockResolvedValue({
      id: "user-1",
      projectId: "project-1",
      username: creator.username,
      passwordHash: hashedPassword,
    });

    mockProjectUserRoleBinding.findOne.mockResolvedValue(null);
    mockProjectUserRoleBinding.create.mockResolvedValue({
      id: "binding-1",
      projectId: "project-1",
      runtimeUserId: "user-1",
      roleId: "role-1",
    });

    const result = await ensureRuntimeAdminBootstrap({
      project,
      creator,
      initialPassword: "Initial#123",
      transaction,
    });

    expect(bcrypt.hash).toHaveBeenCalledWith("Initial#123", 12);
    expect(mockProjectRole.create).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: "project-1",
        code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
        createdBy: creator.id,
        updatedBy: creator.id,
      }),
      expect.objectContaining({ transaction }),
    );
    expect(mockProjectRuntimeUser.create).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: "project-1",
        username: creator.username,
        passwordHash: hashedPassword,
        createdBy: creator.id,
        updatedBy: creator.id,
      }),
      expect.objectContaining({ transaction }),
    );
    expect(mockProjectUserRoleBinding.create).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: "project-1",
        runtimeUserId: "user-1",
        roleId: "role-1",
        createdBy: creator.id,
      }),
      expect.objectContaining({ transaction }),
    );
    expect(
      await bcrypt.compare("Initial#123", result.runtimeUser.passwordHash),
    ).toBe(true);
    expect(result.role.code).toBe(DEFAULT_RUNTIME_ADMIN_ROLE_CODE);
    expect(result.runtimeUser.username).toBe(creator.username);
    expect(mockProjectRole.findOne).toHaveBeenCalledWith(
      expect.objectContaining({ transaction }),
    );
    expect(mockProjectRuntimeUser.findOne).toHaveBeenCalledWith(
      expect.objectContaining({ transaction }),
    );
    expect(mockProjectUserRoleBinding.findOne).toHaveBeenCalledWith(
      expect.objectContaining({ transaction }),
    );
  });

  test("buildEffectiveRoleGrantMap 会按 resourceId 粒度聚合并让 deny 覆盖 allow", () => {
    const grantMap = buildEffectiveRoleGrantMap([
      {
        roleCode: "PROJECT_RUNTIME_ADMIN",
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

    expect(grantMap).toEqual({
      project: {
        "project-1": {
          deploy: {
            allowRoles: ["PROJECT_RUNTIME_ADMIN"],
            denyRoles: ["PROJECT_RUNTIME_OPERATOR"],
          },
        },
        "project-2": {
          deploy: {
            allowRoles: ["PROJECT_RUNTIME_VIEWER"],
            denyRoles: ["PROJECT_RUNTIME_DENY"],
          },
        },
      },
    });
  });
});
