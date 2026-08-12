import assert from 'node:assert/strict'
import test from 'node:test'

import { access, configureRuntime, createRuntimeClient, points, scenes } from './index.js'

test('points 将属性路径和操作转发给 adapter', async () => {
  const calls = []
  const unsubscribe = () => {}
  const client = createRuntimeClient({
    adapter: {
      get(path, params) {
        calls.push(['get', path, params])
        return Promise.resolve(42)
      },
      set(path, value) {
        calls.push(['set', path, value])
        return 'set-result'
      },
      subscribe(path, handler) {
        calls.push(['subscribe', path, handler])
        return unsubscribe
      },
      publish(path, payload) {
        calls.push(['publish', path, payload])
        return 'publish-result'
      },
    },
  })
  const handler = () => {}

  assert.equal(await client.points.factory.line1.temperature.get({ range: '1h' }), 42)
  assert.equal(client.points.factory.line1.temperature.set(80), 'set-result')
  assert.equal(client.points.factory.line1.temperature.sub(handler), unsubscribe)
  assert.equal(client.points.factory.events.pub({ code: 1 }), 'publish-result')
  assert.deepEqual(calls, [
    ['get', 'factory.line1.temperature', { range: '1h' }],
    ['set', 'factory.line1.temperature', 80],
    ['subscribe', 'factory.line1.temperature', handler],
    ['publish', 'factory.events', { code: 1 }],
  ])
})

test('points 不会被 Promise 当作 thenable', async () => {
  const client = createRuntimeClient()
  assert.equal(await client.points, client.points)
})

test('points 在缺少适配器方法时给出明确错误', () => {
  const client = createRuntimeClient({ adapter: {} })
  assert.throws(() => client.points.device.speed.get(), /runtime adapter 缺少 get\(\) 方法/)
  assert.throws(() => client.points.device.speed.sub('invalid'), /handler 必须是函数/)
})

test('access 基于当前角色判断权限', () => {
  const client = createRuntimeClient({ roles: ['operator', 'viewer'] })

  assert.equal(client.access.hasRole('operator'), true)
  assert.equal(client.access.hasRole('admin'), false)
  assert.equal(client.access.hasAnyRole(['admin', 'viewer']), true)
  assert.equal(client.access.hasAnyRole(['admin', 'owner']), false)
  assert.throws(() => client.access.hasAnyRole('operator'), /roles 必须是数组/)
})

test('scenes 将导航请求转发给 navigation adapter', () => {
  const calls = []
  const client = createRuntimeClient({
    navigation: {
      open2D(sceneId, options) {
        calls.push(['2d', sceneId, options])
        return '2d-result'
      },
      open3D(sceneId, options) {
        calls.push(['3d', sceneId, options])
        return '3d-result'
      },
    },
  })

  assert.equal(client.scenes.open2D('overview', { target: '_self' }), '2d-result')
  assert.equal(client.scenes.open3D('factory'), '3d-result')
  assert.deepEqual(calls, [
    ['2d', 'overview', { target: '_self' }],
    ['3d', 'factory', undefined],
  ])
})

test('configureRuntime 配置顶层导出，并保持角色配置实时可读', () => {
  const runtime = {
    roles: ['viewer'],
    adapter: {
      get(path, params) {
        return { path, params }
      },
      set() {},
      subscribe() {},
      publish() {},
    },
    navigation: {
      open2D(sceneId) {
        return sceneId
      },
      open3D(sceneId) {
        return sceneId
      },
    },
  }

  const client = configureRuntime(runtime)
  assert.equal(client.points, points)
  assert.equal(client.access, access)
  assert.equal(client.scenes, scenes)
  assert.deepEqual(points.machine.speed.get({ limit: 1 }), {
    path: 'machine.speed',
    params: { limit: 1 },
  })
  assert.equal(access.hasRole('viewer'), true)
  assert.equal(scenes.open3D('main'), 'main')

  runtime.roles = ['operator']
  assert.equal(access.hasRole('viewer'), false)
  assert.equal(access.hasRole('operator'), true)
})

test('默认配置从 window.__INDUFORGE_RUNTIME__ 读取', async () => {
  const moduleUrl = new URL(`./index.js?browser-default=${Date.now()}`, import.meta.url)
  globalThis.window = {
    __INDUFORGE_RUNTIME__: {
      roles: ['browser-role'],
      adapter: {
        get(path) {
          return path
        },
        set() {},
        subscribe() {},
        publish() {},
      },
    },
  }

  try {
    const browserSdk = await import(moduleUrl.href)
    assert.equal(browserSdk.access.hasRole('browser-role'), true)
    assert.equal(browserSdk.points.browser.value.get(), 'browser.value')
  } finally {
    delete globalThis.window
  }
})

test('配置入口拒绝无效参数', () => {
  assert.throws(() => configureRuntime(null), /runtime 必须是对象/)
  assert.throws(() => createRuntimeClient([]), /runtime 必须是对象/)
})
