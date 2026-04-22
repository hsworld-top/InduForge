import { beforeEach, describe, expect, it, vi } from 'vitest'

const { postMock } = vi.hoisted(() => ({ postMock: vi.fn() }))

vi.mock('@/utils/request', () => ({
  default: {
    post: postMock,
  },
}))

import { authAPI } from '@/api/auth.api'

describe('authAPI.login', () => {
  beforeEach(() => {
    postMock.mockReset()
  })

  it('应调用登录接口并透传登录参数', async () => {
    const credentials = {
      username: 'tester',
      password: 'secret',
      captchaKey: 'captcha-key',
      captchaCode: '1234',
      tenantCode: 'tenant-a',
    }

    postMock.mockResolvedValueOnce({ success: true })

    const result = await authAPI.login(credentials)

    expect(postMock).toHaveBeenCalledTimes(1)
    expect(postMock).toHaveBeenCalledWith('/auth/login', {
      username: 'tester',
      password: 'secret',
      tenantCode: 'tenant-a',
      captchaKey: 'captcha-key',
      captchaCode: '1234',
    })
    expect(result).toEqual({ success: true })
  })
})
