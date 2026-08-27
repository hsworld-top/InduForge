import assert from 'node:assert/strict'
import test from 'node:test'

import { access, alarms, computes, configureRuntime, createRuntimeClient, points, scenes } from './index.js'

test('points 转发完整数据点方法并统一返回 SDKResult', async () => {
  const calls = []
  const client = createRuntimeClient({
    adapter: {
      get(path, options) {
        calls.push(['get', path, options])
        return 42
      },
      read(path) {
        calls.push(['read', path])
        return { code: 0, msg: 'ok', data: { path, value: 42, quality: 'good' } }
      },
      set(path, value) {
        calls.push(['set', path, value])
        return { accepted: true }
      },
      subscribe(path, handler) {
        calls.push(['subscribe', path, handler])
        return () => {}
      },
      publish(path, payload) {
        calls.push(['publish', path, payload])
        return 'published'
      },
    },
  })
  const handler = () => {}
  const point = client.points.factory.line1.temperature

  assert.deepEqual(await point.get({ range: '1h' }), { code: 0, msg: 'ok', data: 42 })
  assert.equal((await point.read()).data.quality, 'good')
  assert.deepEqual(await point.set(80), { code: 0, msg: 'ok', data: { accepted: true } })
  assert.equal(typeof (await point.sub(handler)).data, 'function')
  assert.deepEqual(await client.points.factory.events.pub({ code: 1 }), { code: 0, msg: 'ok', data: 'published' })
  assert.deepEqual(calls.map((item) => item.slice(0, 2)), [
    ['get', 'factory.line1.temperature'],
    ['read', 'factory.line1.temperature'],
    ['set', 'factory.line1.temperature'],
    ['subscribe', 'factory.line1.temperature'],
    ['publish', 'factory.events'],
  ])
})

test('points 暴露宿主注入的数据点稳定属性', () => {
  const client = createRuntimeClient({
    pointContracts: {
      'factory.line1.temperature': {
        id: 'point-1',
        name: '温度',
        dataType: 'float64',
        sourceType: 'collector.point',
        unit: '℃',
        tags: ['生产'],
        attributes: { area: 'A' },
        capabilities: { get: true, read: true },
      },
    },
  })
  const point = client.points.factory.line1.temperature

  assert.equal(point.id, 'point-1')
  assert.equal(point.path, 'factory.line1.temperature')
  assert.equal(point.displayName, '温度')
  assert.equal(point.source.type, 'collector.point')
  assert.deepEqual(point.tags, ['生产'])
  assert.deepEqual(point.attributes, { area: 'A' })
  assert.equal(client.points.byPath('factory.line1.temperature').unit, '℃')
})

test('缺少适配器能力时返回统一失败结果', async () => {
  const client = createRuntimeClient({ adapter: {} })
  const result = await client.points.device.speed.history({ limit: 10 })

  assert.equal(result.code, 50031)
  assert.equal(result.data, null)
  assert.match(result.msg, /history/)
  assert.throws(() => client.points.device.speed.subscribe('invalid'), /handler 必须是函数/)
})

test('alarms 区分配置、当前状态、变化、动作和历史', async () => {
  const calls = []
  const client = createRuntimeClient({
    alarmAdapter: {
      listItems(query) {
        calls.push(['listItems', query])
        return [{ id: 'item-1' }]
      },
      listCurrent(query) {
        calls.push(['listCurrent', query])
        return [{ id: 'current-1' }]
      },
      subscribeChanges(handler, options) {
        calls.push(['subscribeChanges', handler, options])
        return () => {}
      },
      acknowledge(id, input) {
        calls.push(['acknowledge', id, input])
        return { code: 0, msg: '已确认', data: { id } }
      },
      listHistory(query) {
        calls.push(['listHistory', query])
        return []
      },
    },
  })
  const handler = () => {}

  assert.equal((await client.alarms.items.list({ enabled: true })).data[0].id, 'item-1')
  assert.equal((await client.alarms.current.list({ status: 'active' })).data[0].id, 'current-1')
  assert.equal(typeof (await client.alarms.changes.subscribe(handler)).data, 'function')
  assert.equal((await client.alarms.actions.acknowledge('current-1', { comment: '已处理' })).code, 0)
  assert.deepEqual((await client.alarms.history.list({ page: 1 })).data, [])
  assert.deepEqual(calls.map((item) => item[0]), [
    'listItems', 'listCurrent', 'subscribeChanges', 'acknowledge', 'listHistory',
  ])
})

test('computes 转发运行和描述请求', async () => {
  const client = createRuntimeClient({
    computeAdapter: {
      run(ref, input) {
        return { ref, input }
      },
      describe(ref) {
        return { ref, language: 'js' }
      },
    },
  })
  assert.equal((await client.computes.temperatureConvert.run({ value: 10 })).data.ref, 'temperatureConvert')
  assert.equal((await client.computes.byRef('folder.compute').describe()).data.language, 'js')
})

test('points 和 computes 不会被 Promise 当作 thenable', async () => {
  const client = createRuntimeClient()
  assert.equal(await client.points, client.points)
  assert.equal(await client.computes, client.computes)
})

test('access 与 scenes 保持现有行为', () => {
  const calls = []
  const client = createRuntimeClient({
    roles: ['operator', 'viewer'],
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
  assert.equal(client.access.hasRole('operator'), true)
  assert.equal(client.access.hasAnyRole(['admin', 'viewer']), true)
  assert.equal(client.scenes.open2D('overview', { target: '_self' }), '2d-result')
  assert.equal(client.scenes.open3D('factory'), '3d-result')
  assert.equal(calls.length, 2)
})

test('configureRuntime 配置顶层全部领域导出', async () => {
  const runtime = {
    roles: ['viewer'],
    adapter: { get: (path) => path },
    alarmAdapter: { listCurrent: () => [] },
    computeAdapter: { run: (ref) => ref },
    navigation: { open2D: (id) => id, open3D: (id) => id },
  }
  const client = configureRuntime(runtime)

  assert.equal(client.points, points)
  assert.equal(client.alarms, alarms)
  assert.equal(client.computes, computes)
  assert.equal(client.access, access)
  assert.equal(client.scenes, scenes)
  assert.equal((await points.machine.speed.get()).data, 'machine.speed')
  assert.deepEqual((await alarms.current.list()).data, [])
  assert.equal((await computes.demo.run()).data, 'demo')
})

test('默认配置从 window.__INDUFORGE_RUNTIME__ 读取', async () => {
  const moduleUrl = new URL(`./index.js?browser-default=${Date.now()}`, import.meta.url)
  globalThis.window = {
    __INDUFORGE_RUNTIME__: {
      roles: ['browser-role'],
      adapter: { get: (path) => path },
    },
  }
  try {
    const browserSdk = await import(moduleUrl.href)
    assert.equal(browserSdk.access.hasRole('browser-role'), true)
    assert.equal((await browserSdk.points.browser.value.get()).data, 'browser.value')
  } finally {
    delete globalThis.window
  }
})

test('配置入口拒绝无效参数', () => {
  assert.throws(() => configureRuntime(null), /runtime 必须是对象/)
  assert.throws(() => createRuntimeClient([]), /runtime 必须是对象/)
})
