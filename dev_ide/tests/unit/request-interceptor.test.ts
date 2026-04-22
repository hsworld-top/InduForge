import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { InternalAxiosRequestConfig } from 'axios'
import { Storage } from '@/utils/storage'
import { ElMessage } from 'element-plus'

const { createMock, requestUseMock, responseUseMock } = vi.hoisted(() => ({
  createMock: vi.fn(),
  requestUseMock: vi.fn(),
  responseUseMock: vi.fn(),
}))

vi.mock('axios', () => {
  createMock.mockImplementation(() => ({
    interceptors: {
      request: { use: requestUseMock },
      response: { use: responseUseMock },
    },
  }))

  return {
    default: {
      create: createMock,
    },
  }
})

vi.mock('element-plus', () => ({
  ElMessage: {
    error: vi.fn(),
  },
}))

vi.mock('@/utils/storage', () => ({
  Storage: {
    getToken: vi.fn(() => null),
    getTenantId: vi.fn(() => null),
    getRefreshToken: vi.fn(() => null),
    setToken: vi.fn(),
    setRefreshToken: vi.fn(),
    remove: vi.fn(),
  },
}))

vi.mock('@/api/auth.api', () => ({
  authAPI: {
    refreshToken: vi.fn(),
  },
}))

describe('request interceptor', () => {
  beforeEach(() => {
    createMock.mockClear()
    requestUseMock.mockClear()
    responseUseMock.mockClear()
    vi.resetModules()
    vi.mocked(Storage.remove).mockClear()
    vi.mocked(Storage.getRefreshToken).mockReturnValue(null)
    vi.mocked(ElMessage.error).mockClear()
  })

  it('FormData 请求应删除 Content-Type 头', async () => {
    await import('@/utils/request')

    const requestFulfilled = requestUseMock.mock.calls[0]?.[0] as
      | ((config: InternalAxiosRequestConfig) => InternalAxiosRequestConfig)
      | undefined

    expect(typeof requestFulfilled).toBe('function')

    const deleteMock = vi.fn()
    const config = {
      data: new FormData(),
      headers: {
        delete: deleteMock,
      },
    } as unknown as InternalAxiosRequestConfig

    const nextConfig = requestFulfilled!(config)

    expect(deleteMock).toHaveBeenCalledWith('Content-Type')
    expect(nextConfig).toBe(config)
  })

  it('401 且缺少 refreshToken 时应清理登录态', async () => {
    await import('@/utils/request')

    const responseRejected = responseUseMock.mock.calls[0]?.[1] as
      | ((error: unknown) => Promise<unknown>)
      | undefined

    expect(typeof responseRejected).toBe('function')

    const error = {
      response: {
        status: 401,
        data: {},
      },
      config: {
        url: '/users',
        headers: {},
      },
    }

    await expect(responseRejected!(error)).rejects.toBe(error)

    expect(Storage.remove).toHaveBeenCalled()
    expect(Storage.remove).toHaveBeenCalledTimes(4)
  })

  it('403 且开启权限提示时应调用 ElMessage.error', async () => {
    await import('@/utils/request')

    const responseRejected = responseUseMock.mock.calls[0]?.[1] as
      | ((error: unknown) => Promise<unknown>)
      | undefined

    expect(typeof responseRejected).toBe('function')

    const error = {
      response: {
        status: 403,
        data: {
          message: '没有权限访问此资源',
        },
      },
      config: {
        url: '/users',
        forcePermissionToast: true,
        skipPermissionToast: false,
      },
    }

    await expect(responseRejected!(error)).rejects.toBe(error)
    expect(ElMessage.error).toHaveBeenCalledWith('没有权限访问此资源')
  })

  it('403 且 skipPermissionToast=true 时不应调用 ElMessage.error', async () => {
    await import('@/utils/request')

    const responseRejected = responseUseMock.mock.calls[0]?.[1] as
      | ((error: unknown) => Promise<unknown>)
      | undefined

    expect(typeof responseRejected).toBe('function')

    const error = {
      response: {
        status: 403,
        data: {
          message: '没有权限访问此资源',
        },
      },
      config: {
        url: '/users',
        forcePermissionToast: true,
        skipPermissionToast: true,
      },
    }

    await expect(responseRejected!(error)).rejects.toBe(error)
    expect(ElMessage.error).not.toHaveBeenCalled()
  })
})
