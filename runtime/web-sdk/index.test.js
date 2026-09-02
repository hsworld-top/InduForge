import assert from 'node:assert/strict'
import test from 'node:test'

import {
  access,
  alarms,
  computes,
  configureRuntime,
  createHttpRuntime,
  createRuntimeClient,
  points,
  scenes,
} from './index.js'

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
  assert.deepEqual(await client.points.factory.events.pub({ code: 1 }), {
    code: 0,
    msg: 'ok',
    data: 'published',
  })
  assert.deepEqual(
    calls.map((item) => item.slice(0, 2)),
    [
      ['get', 'factory.line1.temperature'],
      ['read', 'factory.line1.temperature'],
      ['set', 'factory.line1.temperature'],
      ['subscribe', 'factory.line1.temperature'],
      ['publish', 'factory.events'],
    ],
  )
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
  assert.equal(
    (await client.alarms.actions.acknowledge('current-1', { comment: '已处理' })).code,
    0,
  )
  assert.deepEqual((await client.alarms.history.list({ page: 1 })).data, [])
  assert.deepEqual(
    calls.map((item) => item[0]),
    ['listItems', 'listCurrent', 'subscribeChanges', 'acknowledge', 'listHistory'],
  )
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
  assert.equal(
    (await client.computes.temperatureConvert.run({ value: 10 })).data.ref,
    'temperatureConvert',
  )
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

test('预览 iframe 默认通过宿主桥接读取并订阅数据点', async () => {
  const listeners = new Map()
  const requests = []
  const parent = {
    postMessage(message, origin) {
      requests.push({ message, origin })
      if (message.type !== 'REQUEST') return
      queueMicrotask(() => {
        listeners.get('message')?.({
          source: parent,
          data: {
            channel: 'induforge-page-runtime',
            version: 1,
            type: 'RESULT',
            requestId: message.requestId,
            result: {
              code: 0,
              msg: 'ok',
              data:
                message.operation === 'read'
                  ? { value: 26.5, quality: 'good' }
                  : { path: message.path, subscribed: message.operation === 'subscribe' },
            },
          },
        })
      })
    },
  }
  globalThis.window = {
    parent,
    document: { referrer: 'https://designer.test/workspace' },
    addEventListener(type, handler) {
      listeners.set(type, handler)
    },
    removeEventListener(type) {
      listeners.delete(type)
    },
  }

  try {
    const moduleUrl = new URL(`./index.js?preview-bridge=${Date.now()}`, import.meta.url)
    const browserSdk = await import(moduleUrl.href)
    const point = browserSdk.points.byPath('db.IF关系库.demo.temperature')

    assert.deepEqual(await point.read(), {
      code: 0,
      msg: 'ok',
      data: { value: 26.5, quality: 'good' },
    })
    assert.equal(requests[0].origin, 'https://designer.test')
    assert.equal(requests[0].message.operation, 'read')

    let pushedValue = null
    const subscription = await point.subscribe((value) => {
      pushedValue = value
    })
    const subscribeRequest = requests.find(({ message }) => message.operation === 'subscribe')
    listeners.get('message')({
      source: parent,
      data: {
        channel: 'induforge-page-runtime',
        version: 1,
        type: 'EVENT',
        subscriptionId: subscribeRequest.message.subscriptionId,
        data: { path: 'db.IF关系库.demo.temperature', value: 27.1 },
      },
    })

    assert.deepEqual(pushedValue, { path: 'db.IF关系库.demo.temperature', value: 27.1 })
    assert.equal(typeof subscription.data, 'function')
    subscription.data()
    assert.equal(requests.at(-1).message.operation, 'unsubscribe')
  } finally {
    delete globalThis.window
  }
})

test('配置入口拒绝无效参数', () => {
  assert.throws(() => configureRuntime(null), /runtime 必须是对象/)
  assert.throws(() => createRuntimeClient([]), /runtime 必须是对象/)
})

function runtimeResponse(data, { code = 0, msg = 'ok', reqId = 'req-test', status = 200 } = {}) {
  return new Response(JSON.stringify({ code, msg, data, reqId }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

test('HTTP Runtime 适配器使用会话、当前值、历史、报警和计算目录', async () => {
  const requests = []
  const fetch = async (url, init) => {
    requests.push({ url, init })
    const target = String(url)
    if (target.endsWith('/session') && init.method === 'POST') {
      return runtimeResponse({
        subjectId: 'user-1',
        roles: ['viewer'],
        expiresAt: '2026-09-01T00:00:00Z',
      })
    }
    if (target.endsWith('/points/line.temperature')) {
      return runtimeResponse({ path: 'line.temperature', value: 42.5, quality: 'good' })
    }
    if (target.includes('/points/line.temperature/history')) {
      return runtimeResponse({
        items: [{ path: 'line.temperature', value: 41, quality: 'good' }],
        total: 1,
      })
    }
    if (target.endsWith('/alarms/current?limit=20'))
      return runtimeResponse({ items: [{ id: 'alarm-1' }], total: 1 })
    if (target.endsWith('/catalog')) {
      return runtimeResponse({
        schemaVersion: 'runtime-project-artifact.v1',
        artifactDigest: 'sha256:test',
        points: [{ path: 'line.temperature', id: 'point-1', unit: 'C' }],
        computes: [{ id: 'compute-1', name: 'temperatureConvert', enabled: true, revision: 1 }],
        alarms: [],
      })
    }
    if (
      target.endsWith('/points/line.temperature/write') ||
      target.endsWith('/computes/compute-1/run')
    ) {
      return runtimeResponse(null, {
        code: 50031,
        msg: '当前 Runtime V1 未启用受控写入/人工计算命令链路',
        status: 501,
      })
    }
    throw new Error(`unexpected URL ${target}`)
  }
  const runtime = createHttpRuntime({
    baseUrl: 'https://gateway.test/api/v1/runtime',
    accessToken: 'runtime-token',
    identity: { deploymentId: 'deployment-1', projectId: 'project-1' },
    fetch,
  })
  const client = createRuntimeClient(runtime)

  assert.equal((await runtime.session.establish()).data.subjectId, 'user-1')
  assert.equal(client.access.hasRole('viewer'), true)
  assert.equal(client.access.hasRole('operator'), false)
  assert.equal(requests[0].init.headers.get('Authorization'), 'Bearer runtime-token')
  assert.equal(requests[0].init.headers.get('X-InduForge-Deployment-Id'), 'deployment-1')
  assert.equal((await client.points.line.temperature.get()).data, 42.5)
  assert.equal((await client.points.line.temperature.read()).data.quality, 'good')
  assert.deepEqual((await client.points.line.temperature.history({ limit: 10 })).data, [
    { path: 'line.temperature', value: 41, quality: 'good' },
  ])
  assert.equal((await client.alarms.current.list({ limit: 20 })).data.items[0].id, 'alarm-1')
  assert.equal((await runtime.catalog.get()).data.computes[0].id, 'compute-1')
  assert.equal(client.points.line.temperature.unit, 'C')
  assert.equal((await client.computes.temperatureConvert.describe()).data.id, 'compute-1')
  assert.equal((await client.points.line.temperature.set(60)).code, 50031)
  assert.equal((await client.computes.byRef('compute-1').run({})).code, 50031)
  assert.equal(
    requests.find(({ url }) => String(url).endsWith('/points/line.temperature/write')).init.body,
    '{"value":60}',
  )
  assert.equal(
    requests.find(({ url }) => String(url).endsWith('/computes/compute-1/run')).init.body,
    '{"input":{}}',
  )
})

test('HTTP Runtime 适配器将取消和非标准响应归一为 SDKResult', async () => {
  const runtime = createHttpRuntime({
    fetch: (_url, init) =>
      new Promise((_resolve, reject) => {
        init.signal.addEventListener(
          'abort',
          () => reject(new DOMException('cancelled', 'AbortError')),
          { once: true },
        )
      }),
  })
  const controller = new AbortController()
  const pending = runtime.session.query({ signal: controller.signal })
  controller.abort()
  const result = await pending
  assert.equal(result.code, 50031)
  assert.match(result.msg, /取消/)
})

test('HTTP Runtime 适配器通过正式接口确认报警', async () => {
  let request
  const runtime = createHttpRuntime({
    baseUrl: 'https://gateway.test/api/v1/runtime',
    fetch: async (url, init) => {
      request = { url: String(url), init }
      return runtimeResponse({ version: 8, acknowledgedAt: '2026-08-31T10:00:00Z' })
    },
  })

  const result = await runtime.alarmAdapter.acknowledge('alarm-1', {
    expectedVersion: 7,
    comment: '现场已确认',
  })
  assert.equal(result.code, 0)
  assert.equal(result.data.version, 8)
  assert.equal(request.url, 'https://gateway.test/api/v1/runtime/alarms/alarm-1/acknowledge')
  assert.equal(request.init.method, 'POST')
  assert.equal(request.init.body, '{"expectedVersion":7,"comment":"现场已确认"}')
})

test('HTTP Runtime WebSocket 订阅交付实时点位并传播关闭和错误', async () => {
  let socket
  class FakeWebSocket {
    constructor(url) {
      this.url = url
      socket = this
      queueMicrotask(() => this.onopen?.())
    }

    send(payload) {
      this.sent = JSON.parse(payload)
      queueMicrotask(() =>
        this.onmessage?.({ data: JSON.stringify({ type: 'subscribed', paths: this.sent.paths }) }),
      )
    }

    close(code = 1000, reason = '') {
      this.onclose?.({ code, reason, wasClean: code === 1000 })
    }
  }
  const received = []
  const errors = []
  const closes = []
  const runtime = createHttpRuntime({
    baseUrl: 'https://gateway.test/api/v1/runtime',
    WebSocket: FakeWebSocket,
  })
  const subscription = await runtime.adapter.subscribe(
    'line.temperature',
    (event) => received.push(event),
    {
      onError: (error) => errors.push(error),
      onClose: (event) => closes.push(event),
    },
  )

  assert.equal(subscription.code, 0)
  assert.equal(socket.url, 'wss://gateway.test/ws/v1/points')
  assert.deepEqual(socket.sent, { action: 'subscribe', paths: ['line.temperature'] })
  socket.onmessage({
    data: JSON.stringify({ type: 'point', data: { path: 'line.temperature', value: 42.5 } }),
  })
  assert.deepEqual(received, [{ path: 'line.temperature', value: 42.5 }])
  socket.onmessage({ data: JSON.stringify({ type: 'error', code: 50031, msg: '订阅不可用' }) })
  assert.equal(errors[0].message, '订阅不可用')
  assert.equal(closes[0].wasClean, true)
})

test('HTTP Runtime WebSocket 订阅报警变更并传播关闭', async () => {
  let socket
  class FakeWebSocket {
    constructor(url) { this.url = url; socket = this; queueMicrotask(() => this.onopen?.()) }
    send(payload) { this.sent = JSON.parse(payload); queueMicrotask(() => this.onmessage?.({ data: JSON.stringify({ type: 'subscribed' }) })) }
    close(code = 1000, reason = '') { this.onclose?.({ code, reason, wasClean: code === 1000 }) }
  }
  const received = []
  const closes = []
  const runtime = createHttpRuntime({ baseUrl: 'https://gateway.test/api/v1/runtime', WebSocket: FakeWebSocket })
  const subscription = await runtime.alarmAdapter.subscribeChanges((event) => received.push(event), { onClose: (event) => closes.push(event) })
  assert.equal(subscription.code, 0)
  assert.equal(socket.url, 'wss://gateway.test/ws/v1/alarms')
  assert.deepEqual(socket.sent, { action: 'subscribe' })
  socket.onmessage({ data: JSON.stringify({ type: 'alarm', data: { operation: 'RAISE', alarmItemId: 'alarm-1' } }) })
  assert.equal(received[0].operation, 'RAISE')
  subscription.data()
  assert.equal(closes[0].wasClean, true)
})

test('HTTP Runtime 报警订阅在异常关闭后重连', async () => {
  const sockets = []
  class FakeWebSocket {
    constructor() { sockets.push(this); queueMicrotask(() => this.onopen?.()) }
    send() { queueMicrotask(() => this.onmessage?.({ data: JSON.stringify({ type: 'subscribed' }) })) }
    close(code = 1000, reason = '') { this.onclose?.({ code, reason, wasClean: code === 1000 }) }
  }
  const runtime = createHttpRuntime({ WebSocket: FakeWebSocket })
  const subscription = await runtime.alarmAdapter.subscribeChanges(() => {}, { reconnectDelayMs: 0 })
  sockets[0].onclose({ code: 1006, reason: 'network', wasClean: false })
  await new Promise((resolve) => setTimeout(resolve, 0))
  assert.equal(sockets.length, 2)
  subscription.data()
})
