const mockDesignPage = {
  findAll: jest.fn(),
  findByPk: jest.fn(),
  findOne: jest.fn(),
  max: jest.fn(),
  create: jest.fn(),
}

const mockProject = {
  findByPk: jest.fn(),
}

const mockTransaction = jest.fn(async (callback) => callback({}))

jest.mock('../../models', () => ({
  DesignPage: mockDesignPage,
  Project: mockProject,
}))

jest.mock('../../config/database', () => ({
  sequelize: {
    transaction: mockTransaction,
  },
}))

const designService = require('../designService')

describe('designService path sync', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('createPage 会持久化显式传入的 path', async () => {
    mockProject.findByPk.mockResolvedValue({ id: 'project-1' })
    mockDesignPage.max.mockResolvedValue(0)
    mockDesignPage.create.mockResolvedValue({
      id: 'page-1',
      projectId: 'project-1',
      parentId: null,
      name: '登录页',
      path: '/login',
      type: 'page',
      sortOrder: 1,
      createdAt: new Date('2026-03-28T00:00:00Z'),
      updatedAt: new Date('2026-03-28T00:00:00Z'),
    })

    const result = await designService.createPage(
      'project-1',
      {
        name: '登录页',
        type: 'page',
        parentId: null,
        path: '/login',
        schemaContent: {
          page: {
            path: '/login',
          },
        },
      },
      'user-1',
    )

    expect(mockDesignPage.create).toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/login',
      }),
    )
    expect(result.path).toBe('/login')
  })

  test('getPages 优先返回模型 path，缺失时回退到 schema path', async () => {
    mockProject.findByPk.mockResolvedValue({
      id: 'project-1',
      entryConfig: { loginPageId: 'page-1' },
    })
    mockDesignPage.findAll.mockResolvedValue([
      {
        get: () => ({
          id: 'page-1',
          name: '登录页',
          path: '/login-direct',
          type: 'page',
          parentId: null,
          sortOrder: 1,
          lockedBy: null,
          lockedAt: null,
          createdAt: new Date('2026-03-28T00:00:00Z'),
          updatedAt: new Date('2026-03-28T00:00:00Z'),
          schemaContent: {
            page: {
              path: '/login-schema',
            },
          },
        }),
      },
      {
        get: () => ({
          id: 'page-2',
          name: '登出页',
          path: null,
          type: 'page',
          parentId: null,
          sortOrder: 2,
          lockedBy: null,
          lockedAt: null,
          createdAt: new Date('2026-03-28T00:00:00Z'),
          updatedAt: new Date('2026-03-28T00:00:00Z'),
          schemaContent: {
            page: {
              path: '/logout',
            },
          },
        }),
      },
    ])

    const result = await designService.getPages('project-1')

    expect(result.pages[0].path).toBe('/login-direct')
    expect(result.pages[1].path).toBe('/logout')
    expect(result.entryConfig).toEqual({ loginPageId: 'page-1' })
  })

  test('updatePage 会把 schema 中的 path 同步到主字段', async () => {
    const update = jest.fn().mockResolvedValue(undefined)
    mockDesignPage.findByPk.mockResolvedValue({
      id: 'page-1',
      type: 'page',
      update,
    })

    await designService.updatePage(
      'page-1',
      {
        page: {
          path: '/login',
        },
      },
      'user-1',
    )

    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/login',
      }),
    )
  })

  test('renamePage 会同步更新 path 与 schemaContent.page.path', async () => {
    const update = jest.fn().mockResolvedValue(undefined)
    mockDesignPage.findByPk.mockResolvedValue({
      id: 'page-1',
      schemaContent: {
        page: {
          path: '/old-login',
        },
        meta: {},
      },
      update,
    })

    await designService.renamePage('page-1', '登录页', 'user-1', '/login')

    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/login',
        schemaContent: expect.objectContaining({
          page: expect.objectContaining({
            path: '/login',
          }),
        }),
      }),
    )
  })

  test('movePage 会同步更新 path 与 schemaContent.page.path', async () => {
    const update = jest.fn().mockResolvedValue(undefined)
    mockDesignPage.findByPk.mockResolvedValue({
      id: 'page-1',
      projectId: 'project-1',
      schemaContent: {
        page: {
          path: '/old-login',
        },
      },
      update,
    })

    await designService.movePage(
      'page-1',
      {
        parentId: null,
        path: '/login',
      },
      'user-1',
    )

    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/login',
        schemaContent: expect.objectContaining({
          page: expect.objectContaining({
            path: '/login',
          }),
        }),
      }),
    )
  })

  test('deletePage 删除普通页面时会清理 entryConfig 残留绑定', async () => {
    const destroy = jest.fn().mockResolvedValue(undefined)
    const update = jest.fn().mockResolvedValue(undefined)
    mockDesignPage.findByPk.mockResolvedValue({
      id: 'login-page-id',
      projectId: 'project-1',
      type: 'page',
      destroy,
    })
    mockProject.findByPk.mockResolvedValue({
      id: 'project-1',
      entryConfig: {
        homePageId: 'home-page-id',
        loginPageId: 'login-page-id',
        logoutPageId: 'logout-page-id',
      },
      update,
    })

    await designService.deletePage('login-page-id')

    expect(destroy).toHaveBeenCalledWith(
      expect.objectContaining({
        transaction: expect.any(Object),
      }),
    )
    expect(update).toHaveBeenCalledWith(
      {
        entryConfig: {
          homePageId: 'home-page-id',
          loginPageId: null,
          logoutPageId: 'logout-page-id',
        },
      },
      expect.objectContaining({
        transaction: expect.any(Object),
      }),
    )
  })

  test('deletePage 级联删除子页面时会清理对应 entryConfig 绑定', async () => {
    const rootDestroy = jest.fn().mockResolvedValue(undefined)
    const childDestroy = jest.fn().mockResolvedValue(undefined)
    const update = jest.fn().mockResolvedValue(undefined)
    mockDesignPage.findByPk.mockResolvedValue({
      id: 'folder-1',
      projectId: 'project-1',
      type: 'folder',
      destroy: rootDestroy,
    })
    mockDesignPage.findAll.mockResolvedValue([
      {
        id: 'login-child-page',
        parentId: 'folder-1',
        type: 'page',
        destroy: childDestroy,
      },
    ])
    mockProject.findByPk.mockResolvedValue({
      id: 'project-1',
      entryConfig: {
        loginPageId: 'login-child-page',
        logoutPageId: 'logout-page-id',
      },
      update,
    })

    await designService.deletePage('folder-1', 'cascade')

    expect(childDestroy).toHaveBeenCalledWith(
      expect.objectContaining({
        transaction: expect.any(Object),
      }),
    )
    expect(rootDestroy).toHaveBeenCalledWith(
      expect.objectContaining({
        transaction: expect.any(Object),
      }),
    )
    expect(update).toHaveBeenCalledWith(
      {
        entryConfig: {
          loginPageId: null,
          logoutPageId: 'logout-page-id',
        },
      },
      expect.objectContaining({
        transaction: expect.any(Object),
      }),
    )
  })
})
