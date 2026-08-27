import assert from 'node:assert/strict'
import { afterEach, test } from 'node:test'
import { JSDOM } from 'jsdom'

const dom = new JSDOM('<!doctype html><html><body></body></html>', {
  url: 'https://app.induforge.test/project/',
  pretendToBeVisual: true,
})

Object.assign(globalThis, {
  window: dom.window,
  document: dom.window.document,
  HTMLElement: dom.window.HTMLElement,
  customElements: dom.window.customElements,
  CustomEvent: dom.window.CustomEvent,
})

let resolveScene
window.__INDUFORGE_RUNTIME__ = {
  sceneResolver: {
    resolve(input) {
      return resolveScene(input)
    },
  },
}

const { SceneRuntimeError } = await import(`./scene-elements.js?test=${Date.now()}`)

const contract = {
  description: '测试场景',
  parameters: [
    { name: 'deviceId', required: true, schema: { type: 'string' } },
  ],
  events: [
    {
      name: 'selected',
      schema: {
        type: 'object',
        properties: { id: { type: 'string' } },
        required: ['id'],
        additionalProperties: false,
      },
    },
  ],
  commands: [
    {
      name: 'focusDevice',
      inputSchema: {
        type: 'object',
        properties: { id: { type: 'string' } },
        required: ['id'],
        additionalProperties: false,
      },
      outputSchema: {
        type: 'object',
        properties: { focused: { type: 'boolean' } },
        required: ['focused'],
        additionalProperties: false,
      },
    },
  ],
}

function session(revision = 3) {
  return {
    sessionId: `session-${revision}`,
    sceneId: 'scene-a',
    kind: '2d',
    revision,
    url: `https://app.induforge.test/designer/scene-studio/display.html?viewerSessionId=session-${revision}`,
    expiresAt: '2026-08-18T08:00:00Z',
    contract,
  }
}

function nextTurn() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

async function createReadyElement() {
  const calls = []
  resolveScene = async (input) => {
    calls.push(input)
    return session()
  }
  const element = document.createElement('induforge-scene-2d')
  element.setAttribute('scene-id', 'scene-a')
  element.params = { deviceId: 'A01' }
  document.body.append(element)
  await nextTurn()
  const frame = element.shadowRoot.querySelector('iframe')
  assert.ok(frame)
  const posted = []
  Object.defineProperty(frame, 'contentWindow', {
    configurable: true,
    value: { postMessage: (message, origin) => posted.push({ message, origin }) },
  })
  dispatchViewerMessage(frame, { type: 'READY' })
  return { element, frame, posted, calls }
}

function dispatchViewerMessage(frame, data) {
  window.dispatchEvent(new window.MessageEvent('message', {
    source: frame.contentWindow,
    origin: 'https://app.induforge.test',
    data: {
      channel: 'induforge-scene-runtime',
      version: 1,
      sceneId: 'scene-a',
      ...data,
    },
  }))
}

afterEach(() => {
  document.body.replaceChildren()
})

test('注册 2D/3D 组件并通过受控解析器加载已提交 revision', async () => {
  assert.ok(customElements.get('induforge-scene-2d'))
  assert.ok(customElements.get('induforge-scene-3d'))

  const { element, posted, calls } = await createReadyElement()

  assert.deepEqual(calls, [{ kind: '2d', sceneId: 'scene-a' }])
  assert.equal(element.shadowRoot.querySelector('iframe').style.opacity, '1')
  assert.deepEqual(posted[0], {
    message: {
      channel: 'induforge-scene-runtime',
      version: 1,
      type: 'INITIALIZE',
      sceneId: 'scene-a',
      params: { deviceId: 'A01' },
    },
    origin: 'https://app.induforge.test',
  })
})

test('Viewer READY 前拒绝命令，清空 scene-id 时移除运行会话', async () => {
  resolveScene = async () => session()
  const element = document.createElement('induforge-scene-2d')
  element.setAttribute('scene-id', 'scene-a')
  element.params = { deviceId: 'A01' }
  document.body.append(element)
  await nextTurn()

  await assert.rejects(element.invoke('focusDevice', { id: 'A02' }), /场景尚未就绪/)
  element.removeAttribute('scene-id')
  assert.equal(element.shadowRoot.querySelector('iframe'), null)
  assert.equal(element.shadowRoot.querySelector('.empty').hidden, false)
})

test('严格校验参数和事件，并只转发有效场景事件', async () => {
  const { element, frame, posted } = await createReadyElement()
  const events = []
  const errors = []
  element.addEventListener('scene-event', (event) => events.push(event.detail))
  element.addEventListener('scene-error', (event) => errors.push(event.detail))

  await element.setParams({ deviceId: 'A02' })
  assert.equal(posted.at(-1).message.type, 'PARAMS_CHANGED')
  assert.deepEqual(posted.at(-1).message.params, { deviceId: 'A02' })
  await assert.rejects(element.setParams({ unknown: true }), (error) => error.code === 26003)

  dispatchViewerMessage(frame, { type: 'EVENT', name: 'selected', payload: { id: 'A02' } })
  dispatchViewerMessage(frame, { type: 'EVENT', name: 'selected', payload: { id: 2 } })
  assert.equal(events.length, 1)
  assert.equal(events[0].name, 'selected')
  assert.equal(errors.length, 2)
  assert.equal(errors[0].code, 26003)
})

test('命令调用校验声明、输入、Viewer 返回和输出', async () => {
  const { element, frame, posted } = await createReadyElement()
  const errors = []
  element.addEventListener('scene-error', (event) => errors.push(event.detail))

  await assert.rejects(element.invoke('missing', {}), (error) => error.code === 26004)
  await assert.rejects(element.invoke('focusDevice', { id: 1 }), (error) => error.code === 26003)

  const success = element.invoke('focusDevice', { id: 'A02' })
  const request = posted.at(-1).message
  dispatchViewerMessage(frame, {
    type: 'COMMAND_RESULT',
    requestId: request.requestId,
    result: { focused: true },
  })
  assert.deepEqual(await success, { focused: true })

  const notRegistered = element.invoke('focusDevice', { id: 'A02' })
  const failedRequest = posted.at(-1).message
  dispatchViewerMessage(frame, {
    type: 'COMMAND_RESULT',
    requestId: failedRequest.requestId,
    error: { code: 26005, msg: '场景未注册命令 focusDevice' },
  })
  await assert.rejects(notRegistered, (error) => error.code === 26005)

  const invalidOutput = element.invoke('focusDevice', { id: 'A02' })
  const invalidRequest = posted.at(-1).message
  dispatchViewerMessage(frame, {
    type: 'COMMAND_RESULT',
    requestId: invalidRequest.requestId,
    result: { focused: 'yes' },
  })
  await assert.rejects(invalidOutput, (error) => error.code === 26003)
  assert.deepEqual(errors.map((error) => error.code), [26004, 26003, 26005, 26003])
})

test('命令超时会拒绝调用并派发统一错误事件', async () => {
  const { element } = await createReadyElement()
  const errors = []
  element.addEventListener('scene-error', (event) => errors.push(event.detail))
  const originalSetTimeout = window.setTimeout
  window.setTimeout = (handler) => {
    queueMicrotask(handler)
    return 1
  }
  try {
    await assert.rejects(
      element.invoke('focusDevice', { id: 'A02' }),
      (error) => error instanceof SceneRuntimeError && error.code === 26006,
    )
  } finally {
    window.setTimeout = originalSetTimeout
  }
  assert.equal(errors.at(-1).code, 26006)
})

test('reload 保留旧 Viewer 到新 Viewer 就绪，并取消旧命令', async () => {
  const { element, frame, posted } = await createReadyElement()
  const pending = element.invoke('focusDevice', { id: 'A02' })
  const rejected = assert.rejects(pending, /场景正在重新加载/)
  resolveScene = async () => session(4)

  await element.reload()
  await rejected
  const frames = element.shadowRoot.querySelectorAll('iframe')
  assert.equal(frames.length, 2)
  assert.equal(frame.isConnected, true)
  const replacement = frames[1]
  Object.defineProperty(replacement, 'contentWindow', {
    configurable: true,
    value: { postMessage() {} },
  })
  dispatchViewerMessage(replacement, { type: 'READY' })
  assert.equal(frame.isConnected, false)
  assert.equal(element.shadowRoot.querySelector('iframe'), replacement)
  assert.equal(posted.some(({ message }) => message.type === 'COMMAND_REQUEST'), true)
})

test('Viewer 会话绝对过期后自动重新申请会话', async () => {
  let revision = 3
  const { element, frame } = await createReadyElement()
  resolveScene = async () => session(++revision)

  dispatchViewerMessage(frame, {
    type: 'ERROR',
    code: 10003,
    msg: 'Viewer 会话已过期',
    requestId: 'heartbeat-1',
  })
  await nextTurn()

  const frames = element.shadowRoot.querySelectorAll('iframe')
  assert.equal(frames.length, 2)
  assert.match(frames[1].src, /viewerSessionId=session-4/)
})

test('销毁组件会清理 iframe 并拒绝未完成命令', async () => {
  const { element } = await createReadyElement()
  const pending = element.invoke('focusDevice', { id: 'A02' })
  const rejected = assert.rejects(pending, /场景组件已销毁/)
  element.remove()
  await rejected
  assert.equal(element.shadowRoot.querySelector('iframe'), null)
})
