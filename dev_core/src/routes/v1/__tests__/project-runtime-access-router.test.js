const ErrorCodes = require('../../../constants/errorCodes')
const { invokeRoute } = require('../test-utils/route-test-helpers')

const mockTransaction = { id: 'tx-project-create' }
const mockProject = {
  sequelize: {
    transaction: jest.fn(async (callback) => callback(mockTransaction)),
  },
  create: jest.fn(),
  findByPk: jest.fn(),
}
const mockDesignPage = {
  create: jest.fn(),
}

const mockService = {
  ensureRuntimeAdminBootstrap: jest.fn(),
  listRuntimeUsers: jest.fn(),
  createRuntimeUser: jest.fn(),
  updateRuntimeUserStatus: jest.fn(),
  deleteRuntimeUser: jest.fn(),
  resetRuntimeUserPassword: jest.fn(),
  bindRuntimeUserRoles: jest.fn(),
  listRuntimeRoles: jest.fn(),
  createRuntimeRole: jest.fn(),
  updateRuntimeRole: jest.fn(),
  deleteRuntimeRole: jest.fn(),
}

const mockUpsertProjectSettings = jest.fn()
const mockReplaceProjectSnapshot = jest.fn()
const mockLogger = { error: jest.fn(), warn: jest.fn() }
let mockCurrentUserRole = 'SYSTEM_ADMIN'
const mockRequireResourceOwnership = jest.fn(() => (req, res, next) => {
  if (req.method === 'PATCH') {
    return res.status(403).json({
      success: false,
      errorCode: 'A1002',
      message: 'legacy project permission denied',
    })
  }
  return next()
})

jest.mock('crypto', () => {
  const actualCrypto = jest.requireActual('crypto')
  return {
    ...actualCrypto,
    randomUUID: jest.fn(() => 'uuid-1'),
  }
})

jest.mock('../../../models', () => ({
  Project: mockProject,
  Tenant: {},
  User: {},
  DesignPage: mockDesignPage,
  NodeDeployment: { findAll: jest.fn() },
}))

jest.mock('../../../middlewares/auth', () => ({
  authenticateToken: (req, res, next) => {
    req.user = {
      id: 'user-1',
      username: 'owner',
      fullName: '工程创建者',
      tenantId: 'tenant-1',
      role: mockCurrentUserRole,
    }
    next()
  },
  requireResourceOwnership: (...args) => mockRequireResourceOwnership(...args),
  hasCapability: () => true,
}))

jest.mock('../../../middlewares/validate', () => ({
  validate: () => (req, res, next) => next(),
}))

jest.mock('../../../utils/logger', () => ({
  logger: mockLogger,
}))

jest.mock('../../../services/projectRuntimeAccessService', () => mockService)

jest.mock('../../../services/projectSettingsStore', () => ({
  upsertProjectSettings: mockUpsertProjectSettings,
  getProjectSettingsRow: jest.fn(),
}))

jest.mock('../../../services/dataDomainClient', () => ({
  dataDomainClient: {
    replaceProjectSnapshot: mockReplaceProjectSnapshot,
    getProjectArtifact: jest.fn(),
  },
}))

jest.mock('../../../services/deploymentService', () => ({
  undeploy: jest.fn(),
}))

jest.mock('../../../services/designAssetService', () => ({
  deleteAssetsByProject: jest.fn(),
  deleteFoldersByProject: jest.fn(),
}))

const router = require('../project')

describe('project runtime access router', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockCurrentUserRole = 'SYSTEM_ADMIN'
    mockProject.sequelize.transaction.mockImplementation(async (callback) =>
      callback(mockTransaction),
    )
    mockUpsertProjectSettings.mockResolvedValue(undefined)
    mockReplaceProjectSnapshot.mockResolvedValue(undefined)
    mockService.ensureRuntimeAdminBootstrap.mockResolvedValue({})
    mockDesignPage.create.mockResolvedValue(undefined)
    mockProject.findByPk.mockResolvedValue({
      id: 'project-1',
      tenantId: 'tenant-1',
      creator: { id: 'user-1', username: 'owner', fullName: '工程创建者' },
    })
  })

  test('创建工程后会在事务内初始化默认工程管理员账号并传入固定 initialPassword', async () => {
    mockProject.create.mockResolvedValue({ id: 'project-1', tenantId: 'tenant-1' })
    mockService.ensureRuntimeAdminBootstrap.mockResolvedValue({})

    const response = await invokeRoute(router, '/', 'post', {
      body: { name: '新工程' },
    })

    expect(response.status).toBe(201)
    expect(mockProject.sequelize.transaction).toHaveBeenCalledTimes(1)
    expect(mockProject.create).toHaveBeenCalledWith(
      expect.objectContaining({
        name: '新工程',
        tenantId: 'tenant-1',
        createdBy: 'user-1',
      }),
      expect.objectContaining({
        transaction: mockTransaction,
      }),
    )
    expect(mockService.ensureRuntimeAdminBootstrap).toHaveBeenCalledWith(
      expect.objectContaining({
        project: expect.objectContaining({ id: 'project-1' }),
        creator: expect.objectContaining({
          id: 'user-1',
          username: 'owner',
          fullName: '工程创建者',
        }),
        transaction: mockTransaction,
        initialPassword: 'admin',
      }),
    )
    expect(mockUpsertProjectSettings).toHaveBeenCalledTimes(1)
    expect(JSON.stringify(response.body)).not.toContain('admin')
  })

  test('创建工程时如果 bootstrap 失败则返回 500，且不会继续查询工程详情', async () => {
    mockProject.create.mockResolvedValue({ id: 'project-1', tenantId: 'tenant-1' })
    mockService.ensureRuntimeAdminBootstrap.mockRejectedValue(new Error('bootstrap failed'))

    const response = await invokeRoute(router, '/', 'post', {
      body: { name: '新工程' },
    })

    expect(response.status).toBe(500)
    expect(mockProject.sequelize.transaction).toHaveBeenCalledTimes(1)
    expect(mockProject.findByPk).not.toHaveBeenCalled()
    expect(mockUpsertProjectSettings).not.toHaveBeenCalled()
  })

  test('创建工程在 post-commit settings 初始化失败时会补偿删除工程并返回 500', async () => {
    const destroy = jest.fn().mockResolvedValue(undefined)
    const createdProject = { id: 'project-1', tenantId: 'tenant-1', destroy }
    mockProject.create.mockResolvedValue(createdProject)
    mockProject.findByPk.mockResolvedValue(createdProject)
    mockService.ensureRuntimeAdminBootstrap.mockResolvedValue({})
    mockUpsertProjectSettings.mockRejectedValue(new Error('settings failed'))

    const response = await invokeRoute(router, '/', 'post', {
      body: { name: '新工程' },
    })

    expect(response.status).toBe(500)
    expect(destroy).toHaveBeenCalledTimes(1)
  })

  test('导入工程后也会初始化默认工程管理员账号并传入固定 initialPassword', async () => {
    const projectUpdate = jest.fn().mockResolvedValue(undefined)
    const destroy = jest.fn().mockResolvedValue(undefined)
    mockProject.create.mockResolvedValue({
      id: 'project-import-1',
      tenantId: 'tenant-1',
      update: projectUpdate,
      destroy,
    })
    mockService.ensureRuntimeAdminBootstrap.mockResolvedValue({})
    mockDesignPage.create.mockResolvedValue(undefined)

    const response = await invokeRoute(router, '/import', 'post', {
      body: {
        name: '导入工程',
        payload: {
          project: {
            name: '旧工程',
          },
          pages: [],
          datacenter: {},
        },
      },
    })

    expect(response.status).toBe(201)
    expect(mockService.ensureRuntimeAdminBootstrap).toHaveBeenCalledWith(
      expect.objectContaining({
        project: expect.objectContaining({ id: 'project-import-1' }),
        creator: expect.objectContaining({
          id: 'user-1',
          username: 'owner',
          fullName: '工程创建者',
        }),
        initialPassword: 'admin',
      }),
    )
    expect(JSON.stringify(response.body)).not.toContain('admin')
  })

  test('导入工程在 snapshot 初始化失败时会补偿删除工程并返回 500', async () => {
    const projectUpdate = jest.fn().mockResolvedValue(undefined)
    const destroy = jest.fn().mockResolvedValue(undefined)
    const importedProject = {
      id: 'project-import-1',
      tenantId: 'tenant-1',
      update: projectUpdate,
      destroy,
    }
    mockProject.create.mockResolvedValue(importedProject)
    mockProject.findByPk.mockResolvedValue(importedProject)
    mockService.ensureRuntimeAdminBootstrap.mockResolvedValue({})
    mockDesignPage.create.mockResolvedValue(undefined)
    mockReplaceProjectSnapshot.mockRejectedValue(new Error('snapshot failed'))

    const response = await invokeRoute(router, '/import', 'post', {
      body: {
        name: '导入工程',
        payload: {
          project: { name: '旧工程' },
          pages: [],
          datacenter: {},
        },
      },
    })

    expect(response.status).toBe(500)
    expect(destroy).toHaveBeenCalledTimes(1)
  })

  test('GET /projects/:id/runtime-users 返回 runtimeUsers', async () => {
    mockService.listRuntimeUsers.mockResolvedValue([
      { id: 'runtime-user-1', username: 'operator', roleIds: ['role-1'] },
    ])

    const response = await invokeRoute(router, '/:id/runtime-users', 'get', {
      params: { id: 'project-1' },
    })

    expect(response.status).toBe(200)
    expect(response.body.data.runtimeUsers).toEqual([
      expect.objectContaining({ username: 'operator', roleIds: ['role-1'] }),
    ])
  })

  test('POST /projects/:id/runtime-users 可创建用户并绑定角色', async () => {
    mockService.createRuntimeUser.mockResolvedValue({
      id: 'runtime-user-1',
      projectId: 'project-1',
      username: 'operator',
      displayName: '值班员',
      status: 'active',
      roleIds: ['role-1', 'role-2'],
      roles: [
        { id: 'role-1', code: 'OPERATOR', name: '值班员' },
        { id: 'role-2', code: 'VIEWER', name: '观察员' },
      ],
    })

    const response = await invokeRoute(router, '/:id/runtime-users', 'post', {
      params: { id: 'project-1' },
      body: {
        username: 'operator',
        displayName: '值班员',
        initialPassword: 'Initial#123',
        roleIds: ['role-1', 'role-2'],
      },
    })

    expect(response.status).toBe(201)
    expect(mockService.createRuntimeUser).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: 'project-1',
        actorId: 'user-1',
        username: 'operator',
        displayName: '值班员',
        initialPassword: 'Initial#123',
        roleIds: ['role-1', 'role-2'],
      }),
    )
    expect(response.body.data.runtimeUser).toEqual(
      expect.objectContaining({
        username: 'operator',
        roleIds: ['role-1', 'role-2'],
      }),
    )
  })

  test('运行态管理写接口不会再被旧的 project PATCH 权限判断误拦截', async () => {
    mockService.updateRuntimeUserStatus.mockResolvedValue({
      id: 'runtime-user-1',
      username: 'operator',
      status: 'disabled',
      roleIds: [],
    })

    const response = await invokeRoute(
      router,
      '/:id/runtime-users/:runtimeUserId/status',
      'patch',
      {
        params: { id: 'project-1', runtimeUserId: 'runtime-user-1' },
        body: { status: 'disabled' },
      },
    )

    expect(response.status).toBe(200)
    expect(response.body.data.runtimeUser).toEqual(
      expect.objectContaining({
        status: 'disabled',
      }),
    )
  })

  test('DELETE /projects/:id/runtime-users/:runtimeUserId 会删除运行态用户', async () => {
    mockService.deleteRuntimeUser.mockResolvedValue({
      id: 'runtime-user-1',
      projectId: 'project-1',
      deleted: true,
    })

    const response = await invokeRoute(router, '/:id/runtime-users/:runtimeUserId', 'delete', {
      params: { id: 'project-1', runtimeUserId: 'runtime-user-1' },
    })

    expect(response.status).toBe(200)
    expect(mockService.deleteRuntimeUser).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: 'project-1',
        runtimeUserId: 'runtime-user-1',
        actorId: 'user-1',
      }),
    )
    expect(response.body.data.runtimeUser).toEqual(
      expect.objectContaining({
        id: 'runtime-user-1',
        deleted: true,
      }),
    )
  })

  test('OPS_ADMIN 无权调用运行态管理写接口', async () => {
    mockCurrentUserRole = 'OPS_ADMIN'

    const response = await invokeRoute(router, '/:id/runtime-users', 'post', {
      params: { id: 'project-1' },
      body: {
        username: 'operator',
        displayName: '值班员',
        initialPassword: 'Initial#123',
        roleIds: [],
      },
    })

    expect(response.status).toBe(200)
    expect(response.body).toEqual({
      code: ErrorCodes.toPublicCode(ErrorCodes.PERMISSION_INSUFFICIENT),
      msg: expect.any(String),
      data: null,
      reqId: 'req-route-test',
    })
    expect(mockService.createRuntimeUser).not.toHaveBeenCalled()
  })

  test('GET /projects/:id/runtime-roles 返回 runtimeRoles', async () => {
    mockService.listRuntimeRoles.mockResolvedValue([
      { id: 'role-1', code: 'OPERATOR', name: '值班员', bindingCount: 2, grantCount: 3 },
    ])

    const response = await invokeRoute(router, '/:id/runtime-roles', 'get', {
      params: { id: 'project-1' },
    })

    expect(response.status).toBe(200)
    expect(response.body.data.runtimeRoles).toEqual([
      expect.objectContaining({ code: 'OPERATOR', bindingCount: 2, grantCount: 3 }),
    ])
  })

  test('DELETE /projects/:id/runtime-roles/:roleId 会把业务错误透传为统一响应', async () => {
    mockService.deleteRuntimeRole.mockRejectedValue(
      Object.assign(new Error('系统内置角色不能删除'), {
        errorCode: 'B0001',
        statusCode: 400,
        options: { message: '系统内置角色不能删除' },
      }),
    )

    const response = await invokeRoute(router, '/:id/runtime-roles/:roleId', 'delete', {
      params: { id: 'project-1', roleId: 'role-admin' },
    })

    expect(response.status).toBe(200)
    expect(response.body).toEqual({
      code: ErrorCodes.toPublicCode('B0001'),
      msg: '系统内置角色不能删除',
      data: null,
      reqId: 'req-route-test',
    })
  })

  test('PATCH /projects/:id/runtime-users/:runtimeUserId/status 只走 active 或 disabled 契约', async () => {
    mockService.updateRuntimeUserStatus.mockResolvedValue({
      id: 'runtime-user-1',
      username: 'operator',
      status: 'disabled',
      roleIds: [],
    })

    const response = await invokeRoute(
      router,
      '/:id/runtime-users/:runtimeUserId/status',
      'patch',
      {
        params: { id: 'project-1', runtimeUserId: 'runtime-user-1' },
        body: { status: 'disabled' },
      },
    )

    expect(response.status).toBe(200)
    expect(mockService.updateRuntimeUserStatus).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: 'project-1',
        runtimeUserId: 'runtime-user-1',
        status: 'disabled',
        actorId: 'user-1',
      }),
    )
    expect(response.body.data.runtimeUser).toEqual(
      expect.objectContaining({
        status: 'disabled',
      }),
    )
  })
})
