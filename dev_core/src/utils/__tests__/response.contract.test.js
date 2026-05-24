const ApiResponse = require('../response')
const ErrorCodes = require('../../constants/errorCodes')

/**
 * 构造最小化的 Express 响应对象桩，仅用于校验响应契约字段。
 */
function createMockResponse({ language = 'zh-CN', requestId = 'req-test-001' } = {}) {
  const state = {
    statusCode: 200,
    payload: null,
    headers: {},
  }

  const res = {
    locals: {
      language,
      requestId,
    },
    req: {
      language,
      requestId,
    },
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

describe('ApiResponse 响应契约', () => {
  test('success 仅输出 code/msg/data/reqId 且成功码固定为 0', () => {
    const { res, state } = createMockResponse({ requestId: 'req-success' })
    const data = { id: 1, name: 'alice' }

    ApiResponse.success(res, data)

    expect(state.statusCode).toBe(200)
    expect(state.payload).toEqual({
      code: 0,
      msg: expect.any(String),
      data,
      reqId: 'req-success',
    })
    expect(Object.keys(state.payload).sort()).toEqual(['code', 'data', 'msg', 'reqId'])
    expect(state.payload.success).toBeUndefined()
    expect(state.payload.errorCode).toBeUndefined()
    expect(state.payload.message).toBeUndefined()
    expect(state.payload.requestId).toBeUndefined()
  })

  test('error 输出整数 code，data 固定为 null，且不包含旧字段', () => {
    const { res, state } = createMockResponse({ requestId: 'req-error' })

    ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: '参数错误' }, 400)

    expect(state.statusCode).toBe(400)
    expect(state.payload).toEqual({
      code: ErrorCodes.toPublicCode(ErrorCodes.VALIDATION_FAILED),
      msg: '参数错误',
      data: null,
      reqId: 'req-error',
    })
    expect(Object.keys(state.payload).sort()).toEqual(['code', 'data', 'msg', 'reqId'])
    expect(state.payload.success).toBeUndefined()
    expect(state.payload.errorCode).toBeUndefined()
    expect(state.payload.message).toBeUndefined()
    expect(state.payload.requestId).toBeUndefined()
  })

  test('paginated 仍遵循四字段契约，分页信息放在 data 内', () => {
    const { res, state } = createMockResponse({ requestId: 'req-page' })
    const list = [{ id: 1 }, { id: 2 }]
    const pagination = {
      page: 1,
      pageSize: 20,
      total: 2,
      totalPages: 1,
    }

    ApiResponse.paginated(res, list, pagination)

    expect(state.statusCode).toBe(200)
    expect(state.payload).toEqual({
      code: 0,
      msg: expect.any(String),
      data: {
        list,
        pagination,
      },
      reqId: 'req-page',
    })
    expect(Object.keys(state.payload).sort()).toEqual(['code', 'data', 'msg', 'reqId'])
  })
})

describe('ErrorCodes 编码映射边界', () => {
  test('toPublicCode 支持 legacy 字符串码与数字字符串', () => {
    expect(ErrorCodes.toPublicCode(ErrorCodes.SUCCESS)).toBe(0)
    expect(ErrorCodes.toPublicCode('A0002')).toBe(10002)
    expect(ErrorCodes.toPublicCode('B0001')).toBe(20001)
    expect(ErrorCodes.toPublicCode('C0001')).toBe(30001)
    expect(ErrorCodes.toPublicCode('20001')).toBe(20001)
  })

  test('toI18nCode 对可识别整数码回映射，对未知码稳定回退', () => {
    expect(ErrorCodes.toI18nCode(10002)).toBe(ErrorCodes.AUTH_TOKEN_INVALID)
    expect(ErrorCodes.toI18nCode(20001)).toBe(ErrorCodes.VALIDATION_FAILED)
    expect(ErrorCodes.toI18nCode(30001)).toBe(ErrorCodes.INTERNAL_SERVER_ERROR)
    expect(ErrorCodes.toI18nCode(30000)).toBe(ErrorCodes.INTERNAL_SERVER_ERROR)
    expect(ErrorCodes.toI18nCode(99999)).toBe(ErrorCodes.INTERNAL_SERVER_ERROR)
    expect(ErrorCodes.toI18nCode('not-a-code')).toBe(ErrorCodes.INTERNAL_SERVER_ERROR)
  })
})
