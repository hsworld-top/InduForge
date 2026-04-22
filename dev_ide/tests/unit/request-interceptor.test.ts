import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { InternalAxiosRequestConfig } from 'axios'

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
})
