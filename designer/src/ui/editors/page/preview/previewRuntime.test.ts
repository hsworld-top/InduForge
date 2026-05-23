import { afterEach, describe, expect, it, vi } from 'vitest'
import { clearPreviewRuntime, getPreviewRuntime, initPreviewRuntime } from './previewRuntime'

const mocks = vi.hoisted(() => {
  const sockets: Array<{
    options: Record<string, unknown>
    emit: ReturnType<typeof vi.fn>
    disconnect: ReturnType<typeof vi.fn>
    once: (event: string, handler: (...args: unknown[]) => void) => unknown
    on: (event: string, handler: (...args: unknown[]) => void) => unknown
    trigger: (event: string, ...args: unknown[]) => void
  }> = []

  const createPreviewSession = vi.fn()
  const heartbeatPreviewSession = vi.fn()
  const deletePreviewSession = vi.fn()
  const getDataPoints = vi.fn()
  const storageGetToken = vi.fn()

  const io = vi.fn((url: string, options: Record<string, unknown>) => {
    const listeners = new Map<string, Array<(...args: unknown[]) => void>>()
    const socket = {
      url,
      options,
      emit: vi.fn(),
      disconnect: vi.fn(),
      once(event: string, handler: (...args: unknown[]) => void) {
        const list = listeners.get(event) || []
        list.push(handler)
        listeners.set(event, list)
        return socket
      },
      on(event: string, handler: (...args: unknown[]) => void) {
        const list = listeners.get(event) || []
        list.push(handler)
        listeners.set(event, list)
        return socket
      },
      trigger(event: string, ...args: unknown[]) {
        const list = listeners.get(event) || []
        listeners.delete(event)
        if (event === 'connect') {
          ;(socket as { connected?: boolean }).connected = true
        }
        list.forEach((handler) => handler(...args))
      },
    }
    ;(socket as { connected?: boolean }).connected = false
    sockets.push(socket)
    queueMicrotask(() => {
      socket.trigger('connect')
    })
    return socket
  })

  return {
    sockets,
    createPreviewSession,
    heartbeatPreviewSession,
    deletePreviewSession,
    getDataPoints,
    storageGetToken,
    io,
  }
})

mocks.deletePreviewSession.mockResolvedValue(undefined)

vi.mock('@/services', () => ({
  datacenterApi: {
    createPreviewSession: mocks.createPreviewSession,
    heartbeatPreviewSession: mocks.heartbeatPreviewSession,
    deletePreviewSession: mocks.deletePreviewSession,
    getConnections: vi.fn(),
    getQueries: vi.fn(),
    getDatapoints: vi.fn(),
    getDataPoints: mocks.getDataPoints,
    getDatapointValues: vi.fn(),
    executeQuery: vi.fn(),
  },
}))

vi.mock('@/utils/storage', () => ({
  Storage: {
    getToken: mocks.storageGetToken,
  },
}))

vi.mock('socket.io-client', () => ({
  io: mocks.io,
}))

async function flushPromises() {
  await Promise.resolve()
  await Promise.resolve()
}

async function waitFor(predicate: () => boolean, rounds = 6) {
  for (let i = 0; i < rounds; i += 1) {
    if (predicate()) return
    await flushPromises()
  }
}

function createRuntime() {
  return initPreviewRuntime({
    projectId: 'proj-1',
    projectVariables: {
      previewTag: {
        mapped: true,
        source: {
          type: 'dataCenter',
          sourceType: 'tag',
          sourceId: 'tag-1',
          path: 'mqtt.previewTag',
        },
      },
    },
    globalScripts: {},
  })
}

describe('previewRuntime', () => {
  afterEach(async () => {
    clearPreviewRuntime()
    await flushPromises()
    mocks.sockets.length = 0
    vi.clearAllMocks()
  })

  it('creates preview session on start, binds socket auth, and cleans up idempotently', async () => {
    mocks.storageGetToken.mockReturnValue('preview-token')
    mocks.createPreviewSession.mockResolvedValue({ data: { sessionId: 'session-1' } })
    mocks.getDataPoints.mockResolvedValue({
      datapoints: [
        {
          path: 'mqtt.previewTag',
          sourceType: 'tag',
          sourceId: 'tag-1',
          id: 'dp-1',
        },
      ],
      pagination: { totalPages: 1 },
    })

    const rt = createRuntime()

    await expect(rt.start?.()).resolves.toBeUndefined()
    await rt.runCode?.('return $global.previewTag;', null, null, null)
    await waitFor(() => mocks.io.mock.calls.length > 0)

    expect(mocks.createPreviewSession).toHaveBeenCalledTimes(1)
    expect(mocks.io).toHaveBeenCalledTimes(1)
    expect(mocks.sockets[0]?.options).toMatchObject({
      path: '/socket.io',
      transports: ['websocket'],
      auth: {
        token: 'preview-token',
        projectId: 'proj-1',
        previewSessionId: 'session-1',
      },
    })

    await expect(rt.stop?.()).resolves.toBeUndefined()
    await expect(rt.stop?.()).resolves.toBeUndefined()
    clearPreviewRuntime()
    clearPreviewRuntime()
    await flushPromises()

    expect(mocks.deletePreviewSession).toHaveBeenCalledTimes(1)
    expect(mocks.deletePreviewSession).toHaveBeenCalledWith('session-1')
  })

  it('falls back when preview session creation fails', async () => {
    mocks.storageGetToken.mockReturnValue('preview-token')
    mocks.createPreviewSession.mockRejectedValueOnce(new Error('boom'))
    mocks.getDataPoints.mockResolvedValue({
      datapoints: [
        {
          path: 'mqtt.previewTag',
          sourceType: 'tag',
          sourceId: 'tag-1',
          id: 'dp-1',
        },
      ],
      pagination: { totalPages: 1 },
    })

    const rt = createRuntime()

    await expect(rt.start?.()).resolves.toBeUndefined()
    await rt.runCode?.('return $global.previewTag;', null, null, null)
    await flushPromises()

    expect(mocks.io).not.toHaveBeenCalled()
    expect(mocks.deletePreviewSession).not.toHaveBeenCalled()
  })

  it('retries preview session creation after an earlier failure', async () => {
    mocks.storageGetToken.mockReturnValue('preview-token')
    mocks.createPreviewSession
      .mockRejectedValueOnce(new Error('boom'))
      .mockResolvedValueOnce({ data: { sessionId: 'session-2' } })
    mocks.getDataPoints.mockResolvedValue({
      datapoints: [
        {
          path: 'mqtt.previewTag',
          sourceType: 'tag',
          sourceId: 'tag-1',
          id: 'dp-1',
        },
      ],
      pagination: { totalPages: 1 },
    })

    const rt = createRuntime()

    await expect(rt.start?.()).resolves.toBeUndefined()
    await rt.runCode?.('return $global.previewTag;', null, null, null)
    await flushPromises()
    expect(mocks.io).not.toHaveBeenCalled()

    await rt.runCode?.('return $global.previewTag;', null, null, null)
    await waitFor(() => mocks.io.mock.calls.length > 0)

    expect(mocks.createPreviewSession).toHaveBeenCalledTimes(2)
    expect(mocks.io).toHaveBeenCalledTimes(1)
    expect(mocks.sockets[0]?.options).toMatchObject({
      auth: {
        token: 'preview-token',
        projectId: 'proj-1',
        previewSessionId: 'session-2',
      },
    })
  })

  it('getPreviewRuntime returns null after clear', () => {
    clearPreviewRuntime()
    expect(getPreviewRuntime()).toBeNull()
  })
})
