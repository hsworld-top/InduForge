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
  DEFAULT_RUNTIME_ADMIN_PASSWORD,
  DEFAULT_RUNTIME_ADMIN_USERNAME,
  DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
  ensureRuntimeAdminBootstrap,
  buildEffectiveRoleGrantMap,
} = require("../projectRuntimeAccessService");

describe("projectRuntimeAccessService", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("ensureRuntimeAdminBootstrap 会创建默认管理员角色、用户与绑定，并对密码进行哈希", async () => {
    const hashedPassword = "bcrypt-hash-runtime-admin";
    bcrypt.hash.mockResolvedValue(hashedPassword);
    bcrypt.compare.mockResolvedValue(true);

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
      username: DEFAULT_RUNTIME_ADMIN_USERNAME,
      passwordHash: hashedPassword,
    });

    mockProjectUserRoleBinding.findOne.mockResolvedValue(null);
    mockProjectUserRoleBinding.create.mockResolvedValue({
      id: "binding-1",
      projectId: "project-1",
      runtimeUserId: "user-1",
      roleId: "role-1",
    });

    const result = await ensureRuntimeAdminBootstrap("project-1");

    expect(bcrypt.hash).toHaveBeenCalledWith(DEFAULT_RUNTIME_ADMIN_PASSWORD, 12);
    expect(mockProjectRole.create).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: "project-1",
        code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
      }),
    );
    expect(mockProjectRuntimeUser.create).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: "project-1",
        username: DEFAULT_RUNTIME_ADMIN_USERNAME,
        passwordHash: hashedPassword,
      }),
    );
    expect(mockProjectUserRoleBinding.create).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: "project-1",
        runtimeUserId: "user-1",
        roleId: "role-1",
      }),
    );
    expect(
      await bcrypt.compare(DEFAULT_RUNTIME_ADMIN_PASSWORD, result.runtimeUser.passwordHash),
    ).toBe(true);
    expect(result.role.code).toBe(DEFAULT_RUNTIME_ADMIN_ROLE_CODE);
    expect(result.runtimeUser.username).toBe(DEFAULT_RUNTIME_ADMIN_USERNAME);
  });

  test("buildEffectiveRoleGrantMap 会合并多角色授权并让 deny 覆盖 allow", () => {
    const grantMap = buildEffectiveRoleGrantMap([
      {
        roleCode: "PROJECT_RUNTIME_ADMIN",
        resourceType: "project",
        action: "deploy",
        effect: "allow",
      },
      {
        roleCode: "PROJECT_RUNTIME_OPERATOR",
        resourceType: "project",
        action: "deploy",
        effect: "allow",
      },
      {
        roleCode: "PROJECT_RUNTIME_OPERATOR",
        resourceType: "project",
        action: "deploy",
        effect: "deny",
      },
      {
        roleCode: "PROJECT_RUNTIME_VIEWER",
        resourceType: "project",
        action: "deploy",
        effect: "deny",
      },
    ]);

    expect(grantMap).toEqual({
      project: {
        deploy: {
          allowRoles: ["PROJECT_RUNTIME_ADMIN"],
          denyRoles: ["PROJECT_RUNTIME_OPERATOR", "PROJECT_RUNTIME_VIEWER"],
        },
      },
    });
  });
});
