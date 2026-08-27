import { beforeEach, describe, test, vi } from 'vitest'
import assert from 'node:assert/strict'

vi.mock('element-plus', () => ({
  ElMessage: {
    error: vi.fn(),
  },
}))

import request, {
  ApiBusinessError,
  getApiErrorMessage,
  resolveApiError,
} from '../src/utils/request'

const createSuccessAdapter =
  (data: unknown, status = 200) =>
  async (config: any) => ({
    data,
    status,
    statusText: 'OK',
    headers: {},
    config,
    request: {},
  })

const createHttpErrorAdapter =
  (status: number, data: unknown, message = 'Request failed') =>
  async (config: any) => {
    return Promise.reject({
      config,
      message,
      isAxiosError: true,
      response: {
        status,
        data,
        headers: {},
        config,
      },
    })
  }

describe('request 响应包络识别与错误契约', () => {
  beforeEach(() => {
    globalThis.localStorage = {
      length: 0,
      key: vi.fn(() => null),
      getItem: vi.fn(() => null),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    }
  })

  test('code=0 且缺省 data 时，仍按统一包络返回成功结果', async () => {
    const result = (await request({
      url: '/__test__/success-without-data',
      method: 'get',
      adapter: createSuccessAdapter({
        code: 0,
        msg: 'ok',
        reqId: 'req-success-1',
      }),
    })) as any

    assert.equal(result.code, 0)
    assert.equal(result.msg, 'ok')
    assert.equal(result.reqId, 'req-success-1')
    assert.equal(result.data, undefined)
  })

  test('code!=0 且缺省 data 时，必须 reject 为 ApiBusinessError', async () => {
    await assert.rejects(
      () =>
        request({
          url: '/__test__/business-error-without-data',
          method: 'get',
          adapter: createSuccessAdapter({
            code: '60001',
            msg: '业务失败',
            reqId: 'req-business-1',
          }),
        }),
      (error: any) => {
        assert.equal(error instanceof ApiBusinessError, true)
        assert.equal(error.code, 60001)
        assert.equal(error.message, '业务失败')
        assert.equal(error.reqId, 'req-business-1')
        assert.equal(error.data, undefined)

        const resolved = resolveApiError(error, '请求失败')
        assert.equal(resolved.code, 60001)
        assert.equal(resolved.msg, '业务失败')
        assert.equal(resolved.reqId, 'req-business-1')
        assert.equal(resolved.status, 200)
        return true
      },
    )
  })

  test('code!=0 且无 msg，但存在 reqId/data 包络标记时也必须 reject', async () => {
    await assert.rejects(
      () =>
        request({
          url: '/__test__/business-error-envelope-marker',
          method: 'get',
          adapter: createSuccessAdapter({
            code: 70001,
            reqId: 'req-business-2',
            data: { reason: 'invalid_state' },
          }),
        }),
      (error: any) => {
        assert.equal(error instanceof ApiBusinessError, true)
        assert.equal(error.code, 70001)
        assert.equal(error.message, '请求失败')
        assert.equal(error.reqId, 'req-business-2')
        assert.deepEqual(error.data, { reason: 'invalid_state' })
        return true
      },
    )
  })

  test('仅有 code 且无包络字段时，不强制按业务包络处理', async () => {
    const raw = (await request({
      url: '/__test__/non-envelope',
      method: 'get',
      adapter: createSuccessAdapter({
        code: 12345,
      }),
    })) as any

    assert.deepEqual(raw, { code: 12345 })
  })

  test('技术异常仍按 HTTP 状态与 msg 解析', async () => {
    await assert.rejects(
      () =>
        request({
          url: '/__test__/http-error',
          method: 'get',
          adapter: createHttpErrorAdapter(500, {
            code: '50001',
            msg: '服务器繁忙',
            reqId: 'req-http-1',
          }),
        }),
      (error: any) => {
        const resolved = resolveApiError(error, '请求失败')
        assert.equal(resolved.code, 50001)
        assert.equal(resolved.msg, '服务器繁忙')
        assert.equal(resolved.reqId, 'req-http-1')
        assert.equal(resolved.status, 500)
        assert.equal(getApiErrorMessage(error, '请求失败'), '服务器繁忙')
        return true
      },
    )
  })

  test('访问令牌过期时应刷新会话并重试原请求', async () => {
    const refresh = vi.fn().mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue({ code: 0, msg: 'ok', data: {} }),
    })
    vi.stubGlobal('fetch', refresh)
    let attempts = 0

    const result = (await request({
      url: '/__test__/auth-renewal',
      method: 'get',
      adapter: async (config: any) => {
        attempts += 1
        if (attempts === 1) {
          return Promise.reject({
            config,
            isAxiosError: true,
            response: { status: 401, data: {}, headers: {}, config },
          })
        }
        return {
          data: { code: 0, msg: 'ok', data: { renewed: true } },
          status: 200,
          statusText: 'OK',
          headers: {},
          config,
          request: {},
        }
      },
    })) as any

    assert.equal(refresh.mock.calls.length, 1)
    assert.equal(attempts, 2)
    assert.deepEqual(result.data, { renewed: true })
  })
})
