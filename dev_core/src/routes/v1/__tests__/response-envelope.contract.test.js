jest.mock('../../../models', () => ({
  User: {
    findByPk: jest.fn(),
  },
  Tenant: {},
  Project: {
    findByPk: jest.fn(),
  },
}))

jest.mock('../../../utils/token', () => ({
  isAccessTokenBlacklisted: jest.fn(),
  verifyAccessToken: jest.fn(),
}))

jest.mock('../../../services/storageService', () => ({
  statObject: jest.fn(),
  getObjectStream: jest.fn(),
}))

const ErrorCodes = require('../../../constants/errorCodes')
const TokenManager = require('../../../utils/token')
const storageService = require('../../../services/storageService')
const { authenticateToken } = require('../../../middlewares/auth')
const tenantRouter = require('../tenant')

/**
 * 构造最小响应对象，直接验证统一信封契约。
 */
function createMockResponse(req) {
  const state = {
    statusCode: 200,
    payload: null,
    headers: {},
  }

  const res = {
    req,
    locals: {},
    setHeader: jest.fn((key, value) => {
      state.headers[key] = value
    }),
    status: jest.fn((code) => {
      state.statusCode = code
      return res
    }),
    json: jest.fn((payload) => {
      state.payload = payload
      return res
    }),
  }

  return { res, state }
}

/**
 * 提取租户资产路由的最终处理函数（跳过 validate 中间件，直连业务分支）。
 */
function getTenantAssetsHandler() {
  const routeLayer = tenantRouter.stack.find(
    (layer) => layer.route && layer.route.path === '/assets' && layer.route.methods.get,
  )
  if (!routeLayer) {
    throw new Error('未找到租户资产路由处理器')
  }
  return routeLayer.route.stack[routeLayer.route.stack.length - 1].handle
}

describe('代表路由响应信封契约', () => {
  const originalNodeEnv = process.env.NODE_ENV

  beforeEach(() => {
    jest.clearAllMocks()
    process.env.NODE_ENV = 'test'
  })

  afterAll(() => {
    process.env.NODE_ENV = originalNodeEnv
  })

  test('业务失败应返回 200 + 非 0 code（缺少 token）', async () => {
    const req = {
      headers: {},
      cookies: {},
      language: 'zh-CN',
      requestId: 'req-business-fail',
    }
    const { res, state } = createMockResponse(req)
    const next = jest.fn()

    await authenticateToken(req, res, next)

    expect(state.statusCode).toBe(200)
    expect(state.payload).toEqual({
      code: ErrorCodes.toPublicCode(ErrorCodes.AUTH_TOKEN_REQUIRED),
      msg: expect.any(String),
      data: null,
      reqId: 'req-business-fail',
    })
    expect(state.payload.code).not.toBe(0)
    expect(Object.keys(state.payload).sort()).toEqual(['code', 'data', 'msg', 'reqId'])
    expect(next).not.toHaveBeenCalled()
  })

  test('技术异常应返回 500 + 统一信封字段（黑名单检查异常）', async () => {
    TokenManager.isAccessTokenBlacklisted.mockRejectedValueOnce(new Error('mock redis error'))
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {})

    const req = {
      headers: {
        authorization: 'Bearer token-for-test',
      },
      cookies: {},
      language: 'zh-CN',
      requestId: 'req-technical-error',
    }
    const { res, state } = createMockResponse(req)
    const next = jest.fn()

    try {
      await authenticateToken(req, res, next)

      expect(state.statusCode).toBe(500)
      expect(state.payload).toEqual({
        code: ErrorCodes.toPublicCode(ErrorCodes.INTERNAL_SERVER_ERROR),
        msg: expect.any(String),
        data: null,
        reqId: 'req-technical-error',
      })
      expect(Object.keys(state.payload).sort()).toEqual(['code', 'data', 'msg', 'reqId'])
      expect(next).not.toHaveBeenCalled()
    } finally {
      consoleErrorSpy.mockRestore()
    }
  })

  test('文件流接口成功响应允许直出流（不包裹为 ApiResponse）', async () => {
    const assetsHandler = getTenantAssetsHandler()
    const pipeResult = Symbol('stream_pipe_result')
    const mockStream = {
      pipe: jest.fn(() => pipeResult),
    }

    storageService.statObject.mockResolvedValueOnce({
      metaData: {
        'content-type': 'image/webp',
      },
    })
    storageService.getObjectStream.mockResolvedValueOnce(mockStream)

    const req = {
      query: { key: 'tenant-assets/sample-logo.webp' },
      language: 'zh-CN',
      requestId: 'req-assets-stream',
    }
    const { res, state } = createMockResponse(req)

    const result = await assetsHandler(req, res)

    expect(result).toBe(pipeResult)
    expect(mockStream.pipe).toHaveBeenCalledWith(res)
    expect(state.headers['Content-Type']).toBe('image/webp')
    expect(state.headers['Cache-Control']).toBe('public, max-age=86400')
    expect(res.status).not.toHaveBeenCalled()
    expect(res.json).not.toHaveBeenCalled()
    expect(state.payload).toBeNull()
  })

  test('文件流接口技术异常必须返回 4xx/5xx + 统一信封', async () => {
    const assetsHandler = getTenantAssetsHandler()
    const storageError = new Error('storage unavailable')
    storageError.code = 'ECONNRESET'

    storageService.statObject.mockRejectedValueOnce(storageError)
    storageService.getObjectStream.mockResolvedValueOnce({
      pipe: jest.fn(),
    })

    const req = {
      query: { key: 'tenant-assets/sample-logo.webp' },
      language: 'zh-CN',
      requestId: 'req-assets-error',
    }
    const { res, state } = createMockResponse(req)

    await assetsHandler(req, res)

    expect(state.statusCode).toBe(500)
    expect(state.payload).toEqual({
      code: ErrorCodes.toPublicCode(ErrorCodes.EXTERNAL_SERVICE_ERROR),
      msg: expect.any(String),
      data: null,
      reqId: 'req-assets-error',
    })
    expect(Object.keys(state.payload).sort()).toEqual(['code', 'data', 'msg', 'reqId'])
  })

  test('文件流接口的业务失败可继续走 200 + 非 0 code（非法路径）', async () => {
    const assetsHandler = getTenantAssetsHandler()
    const req = {
      query: { key: 'invalid-path/logo.webp' },
      language: 'zh-CN',
      requestId: 'req-assets-invalid-key',
    }
    const { res, state } = createMockResponse(req)

    await assetsHandler(req, res)

    expect(state.statusCode).toBe(200)
    expect(state.payload).toEqual({
      code: ErrorCodes.toPublicCode(ErrorCodes.VALIDATION_FAILED),
      msg: '非法资源路径',
      data: null,
      reqId: 'req-assets-invalid-key',
    })
    expect(storageService.statObject).not.toHaveBeenCalled()
    expect(storageService.getObjectStream).not.toHaveBeenCalled()
  })
})
