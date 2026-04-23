import { beforeEach, describe, expect, it, vi } from 'vitest'

const { postMock, getMock } = vi.hoisted(() => ({
  postMock: vi.fn(),
  getMock: vi.fn(),
}))

vi.mock('@/utils/request', () => ({
  default: {
    post: postMock,
    get: getMock,
  },
}))

import { authAPI } from '@/api/auth.api'

describe('authAPI', () => {
  beforeEach(() => {
    postMock.mockReset()
    getMock.mockReset()
  })

  it('login 应调用登录接口并透传登录参数', async () => {
    const credentials = {
      username: 'tester',
      password: 'secret',
      captchaKey: 'captcha-key',
      captchaCode: '1234',
      tenantCode: 'tenant-a',
    }

    postMock.mockResolvedValueOnce({
      code: 0,
      msg: 'ok',
      data: { token: 'token' },
      reqId: 'req_1',
    })

    const result = await authAPI.login(credentials)

    expect(postMock).toHaveBeenCalledTimes(1)
    expect(postMock).toHaveBeenCalledWith('/auth/login', {
      username: 'tester',
      password: 'secret',
      tenantCode: 'tenant-a',
      captchaKey: 'captcha-key',
      captchaCode: '1234',
    })
    expect(result).toEqual({
      code: 0,
      msg: 'ok',
      data: { token: 'token' },
      reqId: 'req_1',
    })
  })

  it('getConfig 传入 undefined 时不应下发 params', async () => {
    getMock.mockResolvedValueOnce({ data: {} })

    await authAPI.getConfig(undefined)

    expect(getMock).toHaveBeenCalledTimes(1)
    expect(getMock).toHaveBeenCalledWith('/auth/config')
  })
})
