// @ts-nocheck
import { afterEach, beforeEach, test } from 'vitest'
import assert from 'node:assert/strict'

import {
  APP_BOOTSTRAP_TIMEOUT_MS,
  applyBootstrapPayload,
  buildIdeRestoreUrl,
  createBootstrapRequest,
  initializeHostBootstrap,
  isTrustedHostMessage,
  postAppBootstrapRequest,
  postMessageToHost,
  resetHostBootstrapSessionForTests,
  shouldRedirectTopLevelToIde,
  shouldUseDebugMode,
  waitForHostBootstrap,
} from '../src/runtime/host-bootstrap'

function createLocalStorageMock() {
  const store = new Map()
  return {
    clear() {
      store.clear()
    },
    getItem(key) {
      return store.has(key) ? store.get(key) : null
    },
    key(index) {
      return [...store.keys()][index] ?? null
    },
    removeItem(key) {
      store.delete(key)
    },
    setItem(key, value) {
      store.set(key, String(value))
    },
  }
}

function installBrowserMocks() {
  const localStorage = createLocalStorageMock()
  const location = {
    href: 'http://datacenter.example/datacenter/?handoff=handoff-default',
    origin: 'http://datacenter.example',
    hostname: 'datacenter.example',
    protocol: 'http:',
  }
  const selfWindow = {
    clearTimeout: globalThis.clearTimeout,
    location,
    parent: null,
    postMessage() {},
    setTimeout: globalThis.setTimeout,
  }
  selfWindow.parent = selfWindow
  const documentMock = {
    referrer: '',
    documentElement: {
      classList: {
        toggle() {},
      },
    },
  }

  globalThis.localStorage = localStorage
  globalThis.window = selfWindow
  globalThis.document = documentMock

  return {
    documentMock,
    localStorage,
    selfWindow,
  }
}

beforeEach(() => {
  installBrowserMocks()
  resetHostBootstrapSessionForTests()
})

afterEach(() => {
  resetHostBootstrapSessionForTests()
  delete globalThis.document
  delete globalThis.localStorage
  delete globalThis.window
})

test('shouldRedirectTopLevelToIde 会区分正式入口和 debug 入口', () => {
  assert.equal(shouldRedirectTopLevelToIde('/datacenter/', true), true)
  assert.equal(shouldRedirectTopLevelToIde('/datacenter/debug', true), false)
  assert.equal(shouldUseDebugMode('/datacenter/debug'), true)
  assert.equal(shouldUseDebugMode('/datacenter/debug', false), false)
  assert.equal(shouldUseDebugMode('/datacenter/debug/query'), true)
  assert.equal(shouldUseDebugMode('/datacenter/'), false)
})

test('createBootstrapRequest 会生成 handoff、requestId 与 URL 上下文', () => {
  const message = createBootstrapRequest({
    handoff: 'handoff-1',
    requestId: 'request-1',
    url: 'http://datacenter.example/datacenter/?handoff=handoff-1',
  })

  assert.equal(message.type, 'APP_BOOTSTRAP_REQUEST')
  assert.equal(message.app, 'datacenter')
  assert.equal(message.requestId, 'request-1')
  assert.equal(message.handoff, 'handoff-1')
  assert.equal(message.requestedPath, '/datacenter/')
  assert.equal(message.requestedQuery, '?handoff=handoff-1')
})

test('postAppBootstrapRequest 会按顶层 bootstrap 契约发送消息', () => {
  const parentWindow = {
    postMessageCalls: [],
    postMessage(message, origin) {
      this.postMessageCalls.push({ message, origin })
    },
  }

  initializeHostBootstrap({
    currentUrl: 'http://datacenter.example/datacenter/?handoff=handoff-request',
    isTopLevelWindow: false,
    parentWindow,
    referrer: 'http://ide.example/dashboard',
    selfWindow: globalThis.window,
  })

  assert.equal(postAppBootstrapRequest(), true)
  assert.deepEqual(parentWindow.postMessageCalls, [
    {
      message: {
        type: 'APP_BOOTSTRAP_REQUEST',
        app: 'datacenter',
        requestId: parentWindow.postMessageCalls[0].message.requestId,
        handoff: 'handoff-request',
        requestedPath: '/datacenter/',
        requestedQuery: '?handoff=handoff-request',
      },
      origin: 'http://ide.example',
    },
  ])
})

test('buildIdeRestoreUrl 会复用 handoff，没有 handoff 时回 IDE 首页', () => {
  assert.equal(
    buildIdeRestoreUrl('handoff-restore', 'http://ide.example'),
    'http://ide.example/?handoff=handoff-restore',
  )
  assert.equal(buildIdeRestoreUrl(null, 'http://ide.example'), 'http://ide.example/')
})

test('applyBootstrapPayload 会写入 token、refreshToken、tenantId、projectId、theme 与 locale', () => {
  applyBootstrapPayload({
    token: 'bootstrap-token',
    refreshToken: 'bootstrap-refresh',
    tenantId: 'tenant-1',
    projectId: 'project-1',
    theme: 'dark',
    locale: 'en',
  })

  assert.equal(globalThis.localStorage.getItem('auth_token'), 'bootstrap-token')
  assert.equal(globalThis.localStorage.getItem('refresh_token'), 'bootstrap-refresh')
  assert.equal(globalThis.localStorage.getItem('tenant_id'), JSON.stringify('tenant-1'))
  assert.equal(globalThis.localStorage.getItem('project_id'), JSON.stringify('project-1'))
  assert.equal(globalThis.localStorage.getItem('theme'), JSON.stringify('dark'))
  assert.equal(globalThis.localStorage.getItem('language'), JSON.stringify('en'))
})

test('trusted origin/source 校验要求两者同时匹配', () => {
  const trustedSource = {}
  const trustedOrigins = new Set(['http://ide.example'])
  const trustedSources = [trustedSource]

  assert.equal(
    isTrustedHostMessage(
      { origin: 'http://ide.example', source: trustedSource },
      { trustedOrigins, trustedSources },
    ),
    true,
  )
  assert.equal(
    isTrustedHostMessage(
      { origin: 'http://evil.example', source: trustedSource },
      { trustedOrigins, trustedSources },
    ),
    false,
  )
  assert.equal(
    isTrustedHostMessage(
      { origin: 'http://ide.example', source: {} },
      { trustedOrigins, trustedSources },
    ),
    false,
  )
})

test('postMessageToHost 只会向可信 origin 发送消息', () => {
  const parentWindow = {
    postMessageCalls: [],
    postMessage(message, origin) {
      this.postMessageCalls.push({ message, origin })
    },
  }

  initializeHostBootstrap({
    currentUrl: 'http://datacenter.example/datacenter/?handoff=handoff-post',
    isTopLevelWindow: false,
    parentWindow,
    referrer: 'http://ide.example/dashboard',
    selfWindow: globalThis.window,
  })

  const posted = postMessageToHost({
    type: 'AUTH_REFRESHED',
    app: 'datacenter',
  })
  assert.equal(posted, true)
  assert.deepEqual(parentWindow.postMessageCalls, [
    {
      message: { type: 'AUTH_REFRESHED', app: 'datacenter' },
      origin: 'http://ide.example',
    },
  ])
})

test('bootstrap 等待超时会收敛，不会无限挂起', async () => {
  initializeHostBootstrap({
    currentUrl: 'http://datacenter.example/datacenter/?handoff=handoff-timeout',
    isTopLevelWindow: false,
    parentWindow: {
      postMessage() {},
    },
    referrer: 'http://ide.example/dashboard',
    selfWindow: globalThis.window,
  })

  const bootstrapResult = await Promise.race([
    waitForHostBootstrap(),
    new Promise((resolve) => {
      setTimeout(() => resolve('timeout-guard'), APP_BOOTSTRAP_TIMEOUT_MS + 200)
    }),
  ])

  assert.equal(bootstrapResult, false)
})

test('顶层正式入口在已有本地会话时不应强制回跳 IDE', () => {
  globalThis.localStorage.setItem('auth_token', 'cached-token')
  globalThis.localStorage.setItem('project_id', JSON.stringify('project-cached'))

  const plan = initializeHostBootstrap({
    currentUrl: 'http://datacenter.example/datacenter/',
    isTopLevelWindow: true,
    referrer: '',
    selfWindow: globalThis.window,
  })

  assert.equal(plan.shouldRedirectToIde, false)
  assert.equal(plan.shouldWaitForBootstrap, false)
  assert.equal(plan.ideRedirectUrl, null)
})

test('顶层正式入口即使已有本地会话，只要携带新的 handoff 仍应回跳 IDE 恢复指定工程', () => {
  globalThis.localStorage.setItem('auth_token', 'cached-token')
  globalThis.localStorage.setItem('project_id', JSON.stringify('project-cached'))

  const plan = initializeHostBootstrap({
    currentUrl: 'http://datacenter.example/datacenter/?handoff=handoff-top-cache',
    isTopLevelWindow: true,
    referrer: '',
    selfWindow: globalThis.window,
  })

  assert.equal(plan.shouldRedirectToIde, true)
  assert.equal(plan.shouldWaitForBootstrap, false)
  assert.equal(plan.ideRedirectUrl, 'http://datacenter.example:18601/?handoff=handoff-top-cache')
})

test('嵌入正式入口即使已有缓存会话，只要携带新的 handoff 也必须等待 bootstrap 覆盖旧工程', () => {
  globalThis.localStorage.setItem('auth_token', 'cached-token')
  globalThis.localStorage.setItem('project_id', JSON.stringify('project-cached'))

  const parentWindow = {
    postMessage() {},
  }

  const plan = initializeHostBootstrap({
    currentUrl: 'http://datacenter.example/datacenter/?handoff=handoff-iframe-cache',
    isTopLevelWindow: false,
    parentWindow,
    referrer: 'http://ide.example/dashboard',
    selfWindow: globalThis.window,
  })

  assert.equal(plan.shouldRedirectToIde, false)
  assert.equal(plan.shouldWaitForBootstrap, true)
})
