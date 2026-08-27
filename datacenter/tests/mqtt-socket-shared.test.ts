import { test } from 'vitest'
import assert from 'node:assert/strict'

import { createMqttSocketSharedRegistry } from '../src/composables/mqtt-socket-shared'

type SocketHandler = (...args: any[]) => void

function createFakeSocket(id: string) {
  const listeners = new Map<string, SocketHandler[]>()
  const anyListeners = new Set<(event: string, ...args: any[]) => void>()

  const socket = {
    id,
    connected: false,
    emitted: [] as Array<{ event: string; payload: unknown }>,
    disconnectCalls: 0,
    on(event: string, handler: SocketHandler) {
      const handlers = listeners.get(event) || []
      handlers.push(handler)
      listeners.set(event, handlers)
      return socket
    },
    onAny(handler: (event: string, ...args: any[]) => void) {
      anyListeners.add(handler)
      return socket
    },
    emit(event: string, payload: unknown) {
      socket.emitted.push({ event, payload })
    },
    disconnect() {
      socket.disconnectCalls += 1
      socket.connected = false
    },
    trigger(event: string, ...args: any[]) {
      if (event === 'connect') {
        socket.connected = true
      }
      if (event === 'disconnect') {
        socket.connected = false
      }

      ;(listeners.get(event) || []).forEach((handler) => handler(...args))
      anyListeners.forEach((handler) => handler(event, ...args))
    },
  }

  return socket
}

function createSilentLogger() {
  return {
    log() {},
    warn() {},
    error() {},
  }
}

test('共享注册表会复用同一条连接，并对订阅做引用计数', () => {
  const sockets: ReturnType<typeof createFakeSocket>[] = []
  let socketOptions: Record<string, unknown> | undefined
  const registry = createMqttSocketSharedRegistry({
    ioFactory: (_url, options) => {
      socketOptions = options
      const socket = createFakeSocket(`socket-${sockets.length + 1}`)
      sockets.push(socket)
      return socket
    },
    getApiUrl: () => 'http://localhost:19601',
    logger: createSilentLogger(),
  })

  const first = registry.acquire({
    projectId: 'project-1',
    previewSessionId: 'session-1',
  })
  const second = registry.acquire({
    projectId: 'project-1',
    previewSessionId: 'session-1',
  })

  assert.ok(first)
  assert.ok(second)
  assert.equal(first.key, second.key)
  assert.equal(sockets.length, 1)
  assert.deepEqual(socketOptions?.auth, {
    projectId: 'project-1',
    previewSessionId: 'session-1',
  })

  registry.subscribeSubscription(first.key, 'sub-1')
  registry.subscribeSubscription(first.key, 'sub-1')
  registry.subscribeTag(first.key, 'tag-1')
  registry.subscribeTag(first.key, 'tag-1')

  assert.deepEqual(sockets[0].emitted, [])

  sockets[0].trigger('connect')

  assert.deepEqual(sockets[0].emitted, [
    {
      event: 'mqtt:subscribe',
      payload: { subscriptionId: 'sub-1' },
    },
    {
      event: 'mqtt:tag:subscribe',
      payload: { tagId: 'tag-1' },
    },
  ])

  registry.unsubscribeSubscription(first.key, 'sub-1')
  registry.unsubscribeTag(first.key, 'tag-1')

  assert.deepEqual(sockets[0].emitted, [
    {
      event: 'mqtt:subscribe',
      payload: { subscriptionId: 'sub-1' },
    },
    {
      event: 'mqtt:tag:subscribe',
      payload: { tagId: 'tag-1' },
    },
  ])

  registry.unsubscribeSubscription(first.key, 'sub-1')
  registry.unsubscribeTag(first.key, 'tag-1')

  assert.deepEqual(sockets[0].emitted, [
    {
      event: 'mqtt:subscribe',
      payload: { subscriptionId: 'sub-1' },
    },
    {
      event: 'mqtt:tag:subscribe',
      payload: { tagId: 'tag-1' },
    },
    {
      event: 'mqtt:unsubscribe',
      payload: { subscriptionId: 'sub-1' },
    },
    {
      event: 'mqtt:tag:unsubscribe',
      payload: { tagId: 'tag-1' },
    },
  ])

  registry.release(first.key)
  assert.equal(sockets[0].disconnectCalls, 0)

  registry.release(second.key)
  assert.equal(sockets[0].disconnectCalls, 1)
})

test('共享注册表会使用传入的数据服务地址建立 socket 连接', () => {
  const captured: Array<{ url: string; options: any }> = []
  const registry = createMqttSocketSharedRegistry({
    ioFactory: (url, options) => {
      captured.push({ url, options })
      return createFakeSocket('socket-1')
    },
    getApiUrl: () => 'http://localhost:19602/',
    logger: createSilentLogger(),
  })

  const result = registry.acquire({
    projectId: 'project-1',
    previewSessionId: 'session-1',
  })

  assert.ok(result)
  assert.equal(captured.length, 1)
  assert.equal(captured[0].url, 'http://localhost:19602')
  assert.equal(captured[0].options.path, '/socket.io/')
  assert.deepEqual(captured[0].options.auth, {
    projectId: 'project-1',
    previewSessionId: 'session-1',
  })
  assert.equal(captured[0].options.withCredentials, true)
})
