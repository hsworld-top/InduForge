const ErrorCodes = require('../../../constants/errorCodes');
const { invokeRoute } = require('../test-utils/route-test-helpers');

const mockTransaction = { id: 'tx-tags-groups' };
const mockCurrentUser = {
  id: 'user-1',
  tenantId: 'tenant-1',
  role: 'PROJECT_ADMIN',
  username: 'admin',
  fullName: '工程管理员',
};

const mockProject = {
  sequelize: {
    transaction: jest.fn(async (callback) => callback(mockTransaction)),
  },
  findByPk: jest.fn(),
  create: jest.fn(),
};

const mockProjectTag = {
  findAll: jest.fn(),
  findOne: jest.fn(),
  create: jest.fn(),
};

const mockProjectTagBinding = {
  destroy: jest.fn(),
  bulkCreate: jest.fn(),
};

const mockProjectGroup = {
  findAll: jest.fn(),
  findOne: jest.fn(),
  create: jest.fn(),
};

const mockProjectGroupMember = {
  destroy: jest.fn(),
  findAll: jest.fn(),
  upsert: jest.fn(),
};

const mockProjectOverviewService = {
  listProjectOverviews: jest.fn(),
};

jest.mock('../../../models', () => ({
  Project: mockProject,
  Tenant: {},
  User: {},
  ProjectTag: mockProjectTag,
  ProjectTagBinding: mockProjectTagBinding,
  ProjectGroup: mockProjectGroup,
  ProjectGroupMember: mockProjectGroupMember,
  DesignPage: {
    findAll: jest.fn(),
    create: jest.fn(),
  },
  NodeDeployment: {
    findAll: jest.fn(),
  },
}));

jest.mock('../../../middlewares/auth', () => ({
  authenticateToken: (req, res, next) => {
    req.user = { ...mockCurrentUser };
    next();
  },
  requireResourceOwnership: () => (req, res, next) => next(),
  hasCapability: () => true,
}));

jest.mock('../../../middlewares/validate', () => ({
  validate: () => (req, res, next) => next(),
}));

jest.mock('../../../utils/logger', () => ({
  logger: {
    error: jest.fn(),
    warn: jest.fn(),
  },
}));

jest.mock('../../../services/projectOverviewService', () => mockProjectOverviewService);

jest.mock('../../../services/projectRuntimeAccessService', () => ({
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
}));

jest.mock('../../../services/projectSettingsStore', () => ({
  upsertProjectSettings: jest.fn(),
  getProjectSettingsRow: jest.fn(),
}));

jest.mock('../../../services/dataDomainClient', () => ({
  dataDomainClient: {
    getProjectArtifact: jest.fn(),
    replaceProjectSnapshot: jest.fn(),
  },
}));

jest.mock('../../../services/deploymentService', () => ({
  undeploy: jest.fn(),
}));

jest.mock('../../../services/designAssetService', () => ({
  deleteAssetsByProject: jest.fn(),
  deleteFoldersByProject: jest.fn(),
}));

const router = require('../project');

describe('project tags/groups router', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    Object.assign(mockCurrentUser, {
      id: 'user-1',
      tenantId: 'tenant-1',
      role: 'PROJECT_ADMIN',
      username: 'admin',
      fullName: '工程管理员',
    });
    mockProject.sequelize.transaction.mockImplementation(async (callback) => callback(mockTransaction));
    mockProjectTag.findAll.mockResolvedValue([]);
    mockProjectGroup.findAll.mockResolvedValue([]);
    mockProjectGroupMember.findAll.mockResolvedValue([]);
  });

  test('GET /projects/tags 返回租户标签列表', async () => {
    mockProjectTag.findAll.mockResolvedValue([
      { id: 'tag-1', name: '核心', sortOrder: 1 },
      { id: 'tag-2', name: '生产', sortOrder: 2 },
    ]);

    const response = await invokeRoute(router, '/tags', 'get', {
      query: { keyword: '核' },
    });

    expect(response.status).toBe(200);
    expect(response.body.data.tags).toEqual([
      expect.objectContaining({ id: 'tag-1', name: '核心' }),
      expect.objectContaining({ id: 'tag-2', name: '生产' }),
    ]);
    expect(mockProjectTag.findAll).toHaveBeenCalledWith(
      expect.objectContaining({
        where: expect.objectContaining({
          tenantId: 'tenant-1',
          name: expect.any(Object),
        }),
      }),
    );
  });

  test('GET /projects/groups 返回租户分组列表', async () => {
    mockProjectGroup.findAll.mockResolvedValue([
      { id: 'group-1', name: '重点项目', sortOrder: 1 },
    ]);
    mockProjectGroupMember.findAll.mockResolvedValue([
      { groupId: 'group-1', projectId: 'project-1' },
      { groupId: 'group-1', projectId: 'project-2' },
    ]);

    const response = await invokeRoute(router, '/groups', 'get');

    expect(response.status).toBe(200);
    expect(response.body.data.groups).toEqual([
      expect.objectContaining({ id: 'group-1', name: '重点项目', projectCount: 2 }),
    ]);
    expect(mockProjectGroup.findAll).toHaveBeenCalledWith(
      expect.objectContaining({
        where: expect.objectContaining({ tenantId: 'tenant-1' }),
      }),
    );
    expect(mockProjectGroupMember.findAll).toHaveBeenCalledWith(
      expect.objectContaining({
        where: expect.objectContaining({
          tenantId: 'tenant-1',
          groupId: expect.any(Object),
        }),
        attributes: ['groupId', 'projectId'],
      }),
    );
  });

  test('POST /projects/tags 可创建标签', async () => {
    mockProjectTag.findOne.mockResolvedValue(null);
    mockProjectTag.create.mockResolvedValue({
      id: 'tag-created',
      name: '新标签',
      tenantId: 'tenant-1',
    });

    const response = await invokeRoute(router, '/tags', 'post', {
      body: {
        name: '新标签',
        description: '描述',
        sortOrder: 3,
      },
    });

    expect(response.status).toBe(201);
    expect(response.body.data.tag).toEqual(
      expect.objectContaining({
        id: 'tag-created',
        name: '新标签',
      }),
    );
    expect(mockProjectTag.create).toHaveBeenCalledWith(
      expect.objectContaining({
        tenantId: 'tenant-1',
        name: '新标签',
        createdBy: 'user-1',
      }),
    );
  });

  test('PUT /projects/:id/tags 可完成标签绑定替换', async () => {
    mockProject.findByPk.mockResolvedValue({
      id: 'project-1',
      tenantId: 'tenant-1',
    });
    mockProjectTag.findAll.mockResolvedValue([
      { id: 'tag-1' },
      { id: 'tag-2' },
    ]);
    mockProjectTagBinding.destroy.mockResolvedValue(2);
    mockProjectTagBinding.bulkCreate.mockResolvedValue([]);

    const response = await invokeRoute(router, '/:id/tags', 'put', {
      params: { id: 'project-1' },
      body: { tagIds: ['tag-1', 'tag-2'] },
    });

    expect(response.status).toBe(200);
    expect(response.body.data).toEqual(
      expect.objectContaining({
        projectId: 'project-1',
        tagIds: ['tag-1', 'tag-2'],
      }),
    );
    expect(mockProject.sequelize.transaction).toHaveBeenCalledTimes(1);
    expect(mockProjectTagBinding.destroy).toHaveBeenCalledWith(
      expect.objectContaining({
        where: { projectId: 'project-1' },
        transaction: mockTransaction,
      }),
    );
    expect(mockProjectTagBinding.bulkCreate).toHaveBeenCalledWith(
      expect.arrayContaining([
        expect.objectContaining({
          projectId: 'project-1',
          tagId: 'tag-1',
          tenantId: 'tenant-1',
        }),
        expect.objectContaining({
          projectId: 'project-1',
          tagId: 'tag-2',
          tenantId: 'tenant-1',
        }),
      ]),
      expect.objectContaining({ transaction: mockTransaction }),
    );
  });

  test('PUT /projects/:id/tags 缺失 tagIds 时返回失败且不会误清空绑定', async () => {
    const response = await invokeRoute(router, '/:id/tags', 'put', {
      params: { id: 'project-1' },
      body: {},
    });

    expect(response.status).toBe(200);
    expect(response.body.code).toBe(ErrorCodes.toPublicCode(ErrorCodes.VALIDATION_FAILED));
    expect(mockProject.findByPk).not.toHaveBeenCalled();
    expect(mockProject.sequelize.transaction).not.toHaveBeenCalled();
    expect(mockProjectTagBinding.destroy).not.toHaveBeenCalled();
  });

  test('PUT /projects/:id/tags 会拒绝跨租户标签写入', async () => {
    mockProject.findByPk.mockResolvedValue({
      id: 'project-1',
      tenantId: 'tenant-1',
    });
    mockProjectTag.findAll.mockResolvedValue([]);

    const response = await invokeRoute(router, '/:id/tags', 'put', {
      params: { id: 'project-1' },
      body: { tagIds: ['tag-cross-tenant'] },
    });

    expect(response.status).toBe(200);
    expect(response.body.code).toBe(ErrorCodes.toPublicCode(ErrorCodes.VALIDATION_FAILED));
    expect(mockProject.sequelize.transaction).not.toHaveBeenCalled();
    expect(mockProjectTagBinding.destroy).not.toHaveBeenCalled();
    expect(mockProjectTagBinding.bulkCreate).not.toHaveBeenCalled();
  });

  test('PUT /projects/:id/group 传 null 时会清空分组绑定', async () => {
    mockProject.findByPk.mockResolvedValue({
      id: 'project-2',
      tenantId: 'tenant-1',
    });
    mockProjectGroupMember.destroy.mockResolvedValue(1);

    const response = await invokeRoute(router, '/:id/group', 'put', {
      params: { id: 'project-2' },
      body: { groupId: null },
    });

    expect(response.status).toBe(200);
    expect(response.body.data).toEqual(
      expect.objectContaining({
        projectId: 'project-2',
        groupId: null,
      }),
    );
    expect(mockProjectGroupMember.destroy).toHaveBeenCalledWith({
      where: { projectId: 'project-2' },
    });
  });

  test('PUT /projects/:id/group 会拒绝跨租户分组写入', async () => {
    mockProject.findByPk.mockResolvedValue({
      id: 'project-3',
      tenantId: 'tenant-1',
    });
    mockProjectGroup.findOne.mockResolvedValue(null);

    const response = await invokeRoute(router, '/:id/group', 'put', {
      params: { id: 'project-3' },
      body: { groupId: 'group-cross-tenant' },
    });

    expect(response.status).toBe(200);
    expect(response.body.code).toBe(ErrorCodes.toPublicCode(ErrorCodes.VALIDATION_FAILED));
    expect(mockProjectGroupMember.upsert).not.toHaveBeenCalled();
  });

  test('标签/分组写接口在无管理权限时返回统一权限错误', async () => {
    mockCurrentUser.role = 'OPS_ADMIN';

    const response = await invokeRoute(router, '/tags', 'post', {
      body: { name: '无权限标签' },
    });

    expect(response.status).toBe(200);
    expect(response.body.code).toBe(ErrorCodes.toPublicCode(ErrorCodes.PERMISSION_INSUFFICIENT));
    expect(mockProjectTag.create).not.toHaveBeenCalled();
  });
});
