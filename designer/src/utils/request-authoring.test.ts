import type { InternalAxiosRequestConfig } from 'axios'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { createMock, requestUseMock, responseUseMock, messageMock } = vi.hoisted(() => ({
  createMock: vi.fn(),
  requestUseMock: vi.fn(),
  responseUseMock: vi.fn(),
  messageMock: vi.fn(),
}))

vi.mock('axios', () => {
  createMock.mockImplementation(() => ({
    interceptors: {
      request: { use: requestUseMock },
      response: { use: responseUseMock },
    },
  }))
  return { default: { create: createMock } }
})

vi.mock('element-plus', () => ({ ElMessage: messageMock }))

describe('Designer authoring epoch 请求约束', () => {
  beforeEach(() => {
    requestUseMock.mockClear()
    responseUseMock.mockClear()
    messageMock.mockClear()
    vi.resetModules()
    vi.stubGlobal('fetch', vi.fn())
  })

  it('Wujie 模式使用宿主上下文，而不是从 URL 推断工程', async () => {
    const context = await import('@/runtime/wujie-context')
    context.applyMicroAppContext({ projectId: 'project-explicit', authoringEpoch: 'epoch-7' })
    await import('./request')
    const requestFulfilled = requestUseMock.mock.calls[0]?.[0] as (
      config: InternalAxiosRequestConfig & { authoringProjectId?: string },
    ) => Promise<InternalAxiosRequestConfig>
    const config = {
      method: 'post',
      url: '/projects/project-from-url/scenes',
      authoringProjectId: 'project-explicit',
      headers: {},
    } as InternalAxiosRequestConfig & { authoringProjectId?: string }

    await requestFulfilled(config)

    expect(config.headers['X-InduForge-Authoring-Epoch']).toBe('epoch-7')
    expect(fetch).not.toHaveBeenCalled()
  })

  it('独立模式先读取 authoring-context，再发送工程写请求', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: vi.fn().mockResolvedValue({ code: 0, msg: 'ok', data: { authoringEpoch: 'epoch-9' } }),
    } as unknown as Response)
    await import('./request')
    const requestFulfilled = requestUseMock.mock.calls[0]?.[0] as (
      config: InternalAxiosRequestConfig & { authoringProjectId?: string },
    ) => Promise<InternalAxiosRequestConfig>
    const config = {
      method: 'put',
      authoringProjectId: 'project/standalone',
      headers: {},
    } as InternalAxiosRequestConfig & { authoringProjectId?: string }

    await requestFulfilled(config)

    expect(fetch).toHaveBeenCalledWith('/api/v1/projects/project%2Fstandalone/authoring-context', {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    expect(config.headers['X-InduForge-Authoring-Epoch']).toBe('epoch-9')
  })

  it('409 reload 通知宿主且不自动重放写请求', async () => {
    const context = await import('@/runtime/wujie-context')
    const onAuthoringStale = vi.fn()
    context.applyMicroAppContext({
      projectId: 'project-1',
      authoringEpoch: 'epoch-7',
      onAuthoringStale,
    })
    await import('./request')
    const responseRejected = responseUseMock.mock.calls[0]?.[1] as (
      error: unknown,
    ) => Promise<unknown>
    const error = {
      response: {
        status: 409,
        data: {
          code: 40901,
          msg: '工程已恢复',
          data: { action: 'reload', currentAuthoringEpoch: 'epoch-8' },
        },
      },
      config: { method: 'put', authoringProjectId: 'project-1', headers: {} },
    }

    await expect(responseRejected(error)).rejects.toBe(error)

    expect(onAuthoringStale).toHaveBeenCalledWith({
      projectId: 'project-1',
      currentAuthoringEpoch: 'epoch-8',
    })
    expect(messageMock).toHaveBeenCalledOnce()
  })
})
