import assert from 'node:assert/strict'

import { ROLES } from '../src/constants/index.js'
import {
  hasRole,
  canAccessTab,
  canManageUsers,
  canApproveNodes,
  canRequestTenantStats,
  getTabAccessDeniedMessage,
} from '../src/permissions/rules.js'
import {
  buildDeployStatusSummary,
  getDeployStatusType,
  getDeployLabel,
  isFailedDeploy,
  getDeployFailureReason,
} from '../src/views/tenant/utils/ops-status.js'
import { messages } from '../src/lang/index.js'
import {
  createEmbeddedUpdateMessage,
  broadcastToEmbeddedIframes,
  createEmbeddedRegistryEntry,
  handleEmbeddedWindowMessage,
  isDesignerEmbeddedIframe,
  syncDesignerLocaleToEmbeddedIframes,
} from '../src/utils/embeddedIframeSync.js'
import { buildAppEntry, buildAppUrl } from '../src/utils/appUrl.js'
import {
  createBootstrapResponse,
  createHandoffRecord,
  buildEmbeddedAppUrl,
  loadHandoffRecord,
  resolveRestorePayload,
  saveHandoffRecord,
} from '../src/utils/embeddedAppBridge.js'

const run = async (name, fn) => {
  try {
    await fn()
    console.log(`PASS ${name}`)
  } catch (error) {
    console.error(`FAIL ${name}`)
    console.error(error)
    process.exitCode = 1
  }
}

const withMockLocalStorage = async (fn) => {
  const originalLocalStorage = globalThis.localStorage
  const store = new Map()

  globalThis.localStorage = {
    get length() {
      return store.size
    },
    key(index) {
      return [...store.keys()][index] ?? null
    },
    getItem(key) {
      return store.has(key) ? store.get(key) : null
    },
    setItem(key, value) {
      store.set(key, String(value))
    },
    removeItem(key) {
      store.delete(key)
    },
    clear() {
      store.clear()
    },
  }

  try {
    return await fn()
  } finally {
    globalThis.localStorage = originalLocalStorage
  }
}

const withMockLocation = async (origin, fn) => {
  const originalLocation = globalThis.location
  const originalWindow = globalThis.window
  const mockedLocation = { origin }

  globalThis.location = mockedLocation
  globalThis.window = {
    ...(originalWindow || {}),
    location: mockedLocation,
  }

  try {
    return await fn()
  } finally {
    globalThis.location = originalLocation
    globalThis.window = originalWindow
  }
}

await run('权限规则：hasRole', () => {
  assert.equal(hasRole(ROLES.SYSTEM_ADMIN, [ROLES.SYSTEM_ADMIN, ROLES.OPS_ADMIN]), true)
  assert.equal(hasRole(ROLES.USER, [ROLES.SYSTEM_ADMIN]), false)
  assert.equal(hasRole('', [ROLES.SYSTEM_ADMIN]), false)
})

await run('权限规则：canAccessTab', () => {
  assert.equal(canAccessTab('tenant-management', ROLES.SUPER_ADMIN), true)
  assert.equal(canAccessTab('tenant-management', ROLES.SYSTEM_ADMIN), false)
  assert.equal(canAccessTab('dashboard', ROLES.USER), true)
})

await run('权限规则：业务判定函数', () => {
  assert.equal(canManageUsers(ROLES.USER_ADMIN), true)
  assert.equal(canManageUsers(ROLES.OPS_ADMIN), false)
  assert.equal(canApproveNodes(ROLES.SYSTEM_ADMIN), true)
  assert.equal(canApproveNodes(ROLES.USER_ADMIN), false)
  assert.equal(canRequestTenantStats(ROLES.SUPER_ADMIN), true)
  assert.equal(canRequestTenantStats(ROLES.SYSTEM_ADMIN), false)
  assert.equal(getTabAccessDeniedMessage('system-settings'), '只有系统管理员才能访问系统设置')
})

await run('运维状态：聚合统计', () => {
  const summary = buildDeployStatusSummary([
    { deployments: [{ status: 'running' }, { status: 'deploying' }, { status: 'error' }] },
    { deployments: [{ status: 'pending' }, { status: 'stopped' }, { status: 'failed' }] },
  ])
  assert.deepEqual(summary, { running: 1, deploying: 2, stopped: 1, failed: 2 })
})

await run('运维状态：映射与失败判定', () => {
  assert.equal(getDeployStatusType('running'), 'success')
  assert.equal(getDeployStatusType('failed'), 'danger')
  assert.equal(getDeployLabel('pending'), '等待中')
  assert.equal(isFailedDeploy({ status: 'error' }), true)
  assert.equal(getDeployFailureReason({ errorMessage: 'network error' }), 'network error')
})

await run('国际化：核心键值存在性', () => {
  assert.equal(messages.zh.dashboard.title, '仪表盘')
  assert.equal(messages.en.dashboard.title, 'Dashboard')
  assert.equal(messages.zh.profile.uploadAvatar, '上传头像')
  assert.equal(messages.en.profile.uploadAvatar, 'Upload Avatar')
  assert.equal(messages.zh.auth.tenantCode, '租户代码')
  assert.equal(messages.en.auth.tenantCode, 'Tenant Code')
})

await run('应用地址：designer 透传当前语言与主题', () => {
  return withMockLocalStorage(() => {
    globalThis.localStorage.setItem('auth_token', 'token-1')
    globalThis.localStorage.setItem('refresh_token', 'refresh-1')
    globalThis.localStorage.setItem('theme', JSON.stringify('dark'))
    globalThis.localStorage.setItem('language', JSON.stringify('en'))

    const url = buildAppUrl('designer', { id: 'p-1', tenantId: 't-1' })
    const builtUrl = new URL(url, 'http://localhost')
    const handoffId = builtUrl.searchParams.get('handoffId')

    assert.equal(builtUrl.pathname, '/designer/')
    assert.ok(handoffId)
    assert.equal(builtUrl.searchParams.get('pid'), null)
    assert.equal(builtUrl.searchParams.get('tenant'), null)
    assert.equal(builtUrl.searchParams.get('token'), null)
    assert.equal(builtUrl.searchParams.get('refreshToken'), null)
    assert.equal(builtUrl.searchParams.get('theme'), null)
    assert.equal(builtUrl.searchParams.get('locale'), null)
    assert.equal(builtUrl.searchParams.get('type'), null)

    const record = loadHandoffRecord(handoffId)
    assert.equal(record?.projectId, 'p-1')
    assert.equal(record?.tenantId, 't-1')
  })
})

await run('应用地址：datacenter 不追加语言参数', () => {
  return withMockLocalStorage(() => {
    globalThis.localStorage.setItem('theme', JSON.stringify('light'))
    globalThis.localStorage.setItem('language', JSON.stringify('zh'))

    const url = buildAppUrl('datacenter', { id: 'p-2', tenantId: 't-2' })
    const builtUrl = new URL(url, 'http://localhost')
    const handoffId = builtUrl.searchParams.get('handoffId')

    assert.equal(builtUrl.pathname, '/datacenter/')
    assert.ok(handoffId)
    assert.equal(builtUrl.searchParams.get('theme'), null)
    assert.equal(builtUrl.searchParams.get('locale'), null)

    const record = loadHandoffRecord(handoffId)
    assert.equal(record?.projectId, 'p-2')
    assert.equal(record?.tenantId, 't-2')
  })
})

await run('应用入口：相对 handoff URL 场景返回真实 origin', () => {
  return withMockLocalStorage(() =>
    withMockLocation('https://ide.example.com', () => {
      const entry = buildAppEntry('designer', { id: 'p-origin', tenantId: 't-origin' })

      assert.equal(entry.url.startsWith('/designer/?handoffId='), true)
      assert.equal(entry.origin, 'https://ide.example.com')
    })
  )
})

await run('应用地址：未知类型显式失败', () => {
  assert.throws(() => buildEmbeddedAppUrl('unknown-app', 'handoff-1'), /未知的应用类型|Unsupported app type/)
})

// 以下用例只验证桥接工具函数级契约，不代表 EmbeddedApp / Dashboard 的真实接入链路。
await run('宿主桥接：bootstrap 的 projectId 和 appType 以函数参数为准', () => {
  return withMockLocalStorage(() => {
    const response = createBootstrapResponse('designer', {
      appType: 'datacenter',
      pid: 'p-5',
      tenantId: 't-5',
      token: 'token-5',
      refreshToken: 'refresh-5',
    })

    assert.equal(response.appType, 'designer')
    assert.equal(response.projectId, 'p-5')
    assert.equal(response.pid, undefined)
    assert.equal(response.url, buildEmbeddedAppUrl('designer', response.handoffId))
    assert.equal(loadHandoffRecord(response.handoffId)?.appType, 'designer')
  })
})

await run('宿主桥接：bootstrap URL 只下发 opaque handoff id，不把敏感字段放进查询串', () => {
  return withMockLocalStorage(() => {
    const response = createBootstrapResponse('designer', {
      pid: 'p-1',
      tenantId: 't-1',
      token: 'token-1',
      refreshToken: 'refresh-1',
      theme: 'dark',
      locale: 'en',
    })

    assert.ok(response.handoffId)
    assert.ok(response.url.includes('handoffId='))
    assert.ok(!response.url.includes('token='))
    assert.ok(!response.url.includes('pid='))
    assert.ok(!response.url.includes('projectId='))
  })
})

await run('宿主桥接：bootstrap 持久化失败时显式失败', () => {
  return withMockLocalStorage(() => {
    const originalSetItem = globalThis.localStorage.setItem
    const originalWarn = console.warn
    globalThis.localStorage.setItem = () => {
      throw new Error('persist failed')
    }
    console.warn = () => {}

    try {
      assert.throws(
        () =>
          createBootstrapResponse('designer', {
            pid: 'p-1',
            tenantId: 't-1',
            token: 'token-1',
            refreshToken: 'refresh-1',
          }),
        (error) => {
          assert.match(error.message, /保存 handoff 票据失败/)
          assert.equal(error.cause?.message, 'persist failed')
          return true
        }
      )
    } finally {
      globalThis.localStorage.setItem = originalSetItem
      console.warn = originalWarn
    }
  })
})

await run('宿主桥接：无效票据不会被保存', () => {
  return withMockLocalStorage(() => {
    assert.throws(
      () =>
        saveHandoffRecord({
          handoffId: 'handoff-invalid',
          projectId: 'p-9',
          issuedAt: 1700000000000,
          expiresAt: 1700000600000,
        }),
      /appType|issuedAt|expiresAt/
    )
  })
})

await run('宿主桥接：未知 appType 保存时显式失败', () => {
  return withMockLocalStorage(() => {
    assert.throws(
      () =>
        saveHandoffRecord({
          handoffId: 'handoff-unknown',
          appType: 'unknown-app',
          projectId: 'p-10',
          issuedAt: 1700000000000,
          expiresAt: 1700000600000,
        }),
      /appType|unknown-app|支持的嵌入类型/
    )
  })
})

await run('宿主桥接：过期记录不可恢复', () => {
  return withMockLocalStorage(() => {
    const originalNow = Date.now
    const now = 1700000000000
    const staleNow = now + 2000
    Date.now = () => now
    const record = createHandoffRecord(
      {
        appType: 'designer',
        pid: 'p-2',
        tenantId: 't-2',
        token: 'token-2',
        refreshToken: 'refresh-2',
        theme: 'light',
        locale: 'zh',
      },
      { now, ttlMs: 1000 }
    )

    try {
      saveHandoffRecord(record)
      globalThis.localStorage.setItem(
        `embedded_app_handoff:${record.handoffId}`,
        JSON.stringify({
          ...record,
          expiresAt: staleNow,
        })
      )
      Date.now = () => staleNow
      assert.equal(loadHandoffRecord(record.handoffId), null)
      assert.equal(
        globalThis.localStorage.getItem(`embedded_app_handoff:${record.handoffId}`),
        null
      )
    } finally {
      Date.now = originalNow
    }
  })
})

await run('宿主桥接：未知 appType 读取时视为无效记录', () => {
  return withMockLocalStorage(() => {
    const rawRecord = {
      handoffId: 'handoff-unknown',
      appType: 'unknown-app',
      projectId: 'p-11',
      tenantId: 't-11',
      issuedAt: 1700000000000,
      expiresAt: 1700000600000,
    }
    globalThis.localStorage.setItem(
      'embedded_app_handoff:handoff-unknown',
      JSON.stringify(rawRecord)
    )

    assert.equal(loadHandoffRecord('handoff-unknown'), null)
  })
})

await run('宿主桥接：bootstrap 响应包含完整宿主态字段', () => {
  return withMockLocalStorage(() => {
    const response = createBootstrapResponse('designer', {
      pid: 'p-3',
      tenantId: 't-3',
      token: 'token-3',
      refreshToken: 'refresh-3',
      theme: 'dark',
      locale: 'en',
    })

    assert.equal(response.appType, 'designer')
    assert.equal(response.projectId, 'p-3')
    assert.equal(response.pid, undefined)
    assert.equal(response.tenantId, 't-3')
    assert.equal(response.token, 'token-3')
    assert.equal(response.refreshToken, 'refresh-3')
    assert.equal(response.theme, 'dark')
    assert.equal(response.locale, 'en')
    assert.equal(typeof response.handoffId, 'string')
    assert.equal(buildEmbeddedAppUrl('designer', response.handoffId), response.url)
  })
})

await run('宿主桥接：对象态恢复同样去敏且校验过期', () => {
  return withMockLocalStorage(() => {
    const activeRecord = {
      handoffId: 'handoff-active',
      appType: 'designer',
      projectId: 'p-6',
      tenantId: 't-6',
      token: 'token-6',
      refreshToken: 'refresh-6',
      issuedAt: 1700000000000,
      expiresAt: 1700000600000,
    }
    const expiredRecord = {
      handoffId: 'handoff-expired',
      appType: 'designer',
      projectId: 'p-7',
      tenantId: 't-7',
      token: 'token-7',
      refreshToken: 'refresh-7',
      issuedAt: 1700000000000,
      expiresAt: 1700000000001,
    }

    const originalNow = Date.now
    Date.now = () => 1700000100000

    try {
      assert.deepEqual(resolveRestorePayload(activeRecord), {
        handoffId: 'handoff-active',
        appType: 'designer',
        projectId: 'p-6',
        tenantId: 't-6',
      })
      assert.equal(resolveRestorePayload({ handoffId: 'broken' }), null)
      assert.equal(resolveRestorePayload(expiredRecord), null)
    } finally {
      Date.now = originalNow
    }
  })
})

await run('宿主桥接：未知 appType 对象态恢复直接失败', () => {
  const unknownRecord = {
    handoffId: 'handoff-unknown',
    appType: 'unknown-app',
    projectId: 'p-12',
    tenantId: 't-12',
    issuedAt: 1700000000000,
    expiresAt: 1700000600000,
  }

  assert.equal(resolveRestorePayload(unknownRecord), null)
})

await run('宿主桥接：恢复载荷去敏且不回传 token', () => {
  return withMockLocalStorage(() => {
    const record = createHandoffRecord({
      appType: 'designer',
      pid: 'p-4',
      tenantId: 't-4',
      token: 'token-4',
      refreshToken: 'refresh-4',
      theme: 'light',
      locale: 'zh',
    })

    saveHandoffRecord(record)
    assert.equal(loadHandoffRecord(record.handoffId)?.token, undefined)
    assert.equal(loadHandoffRecord(record.handoffId)?.refreshToken, undefined)
    assert.deepEqual(resolveRestorePayload(record.handoffId), {
      handoffId: record.handoffId,
      appType: 'designer',
      projectId: 'p-4',
      tenantId: 't-4',
    })
  })
})

await run('宿主桥接：前缀扫描兼容历史 key', () => {
  return withMockLocalStorage(() => {
    const record = createHandoffRecord({
      appType: 'designer',
      pid: 'p-8',
      tenantId: 't-8',
    })

    const legacyKey = 'embedded_app_handoff:legacy-slot'
    globalThis.localStorage.setItem(legacyKey, JSON.stringify(record))
    globalThis.localStorage.removeItem(`embedded_app_handoff:${record.handoffId}`)

    assert.deepEqual(loadHandoffRecord(record.handoffId), {
      handoffId: record.handoffId,
      appType: 'designer',
      projectId: 'p-8',
      tenantId: 't-8',
      issuedAt: record.issuedAt,
      expiresAt: record.expiresAt,
    })
  })
})

await run('测试辅助：withMockLocalStorage 等待异步完成后再恢复', () => {
  const originalLocalStorage = globalThis.localStorage
  return withMockLocalStorage(async () => {
    const mockedStorage = globalThis.localStorage
    await Promise.resolve()
    assert.equal(globalThis.localStorage, mockedStorage)
  }).then(() => {
    assert.equal(globalThis.localStorage, originalLocalStorage)
  })
})

await run('嵌入同步：生成主题更新消息', () => {
  assert.deepEqual(createEmbeddedUpdateMessage('THEME_UPDATE', 'theme', 'dark'), {
    type: 'THEME_UPDATE',
    theme: 'dark',
  })
})

await run('嵌入同步：设计中心 iframe 判定', () => {
  assert.equal(isDesignerEmbeddedIframe({ src: '/designer/?pid=1' }), true)
  assert.equal(isDesignerEmbeddedIframe({ src: '/designer?pid=1' }), true)
  assert.equal(isDesignerEmbeddedIframe({ src: 'http://host/designer/?pid=1' }), true)
  assert.equal(isDesignerEmbeddedIframe({ src: '/datacenter/?pid=1' }), false)
  assert.equal(isDesignerEmbeddedIframe({ src: '' }), false)
  assert.equal(isDesignerEmbeddedIframe({}), false)
  assert.equal(isDesignerEmbeddedIframe({ src: '::bad::url' }), false)
})

await run('嵌入同步：广播使用注册项 origin', () => {
  const events = []
  const entry = {
    origin: 'https://designer.example.com',
    iframe: {
      src: 'https://designer.example.com/designer/?handoffId=handoff-1',
      contentWindow: {
        postMessage(message, targetOrigin) {
          events.push([message, targetOrigin])
        },
      },
    },
  }

  broadcastToEmbeddedIframes([entry], createEmbeddedUpdateMessage('THEME_UPDATE', 'theme', 'dark'))

  assert.deepEqual(events, [[{ type: 'THEME_UPDATE', theme: 'dark' }, 'https://designer.example.com']])
})

await run('嵌入同步：仅通过 Dashboard 链路同步 designer 语言', () => {
  const events = []
  const designerEntry = {
    appType: 'designer',
    origin: 'https://designer.example.com',
    iframe: {
      src: 'https://designer.example.com/designer/?handoffId=handoff-1',
      contentWindow: {
        postMessage(message, targetOrigin) {
          events.push(['designer', message, targetOrigin])
        },
      },
    },
  }
  const datacenterEntry = {
    appType: 'datacenter',
    origin: 'https://datacenter.example.com',
    iframe: {
      src: 'https://datacenter.example.com/datacenter/?handoffId=handoff-2',
      contentWindow: {
        postMessage(message, targetOrigin) {
          events.push(['datacenter', message, targetOrigin])
        },
      },
    },
  }

  syncDesignerLocaleToEmbeddedIframes([designerEntry, datacenterEntry], 'en')

  assert.deepEqual(events, [
    ['designer', { type: 'LOCALE_UPDATE', locale: 'en' }, 'https://designer.example.com'],
  ])
})

await run('嵌入同步：bootstrap 仅给已注册 iframe 回包', () => {
  return withMockLocalStorage(() => {
    const registry = new Map()
    const sourceWindow = { label: 'designer-window' }
    const unmatchedWindow = { label: 'other-window' }
    const responses = []

    const entry = createEmbeddedRegistryEntry({
      iframe: {
        src: 'https://designer.example.com/designer/?handoffId=handoff-3',
        contentWindow: sourceWindow,
      },
      appType: 'designer',
      project: { id: 'p-3', tenantId: 't-3' },
      tabKey: 'design-center-p-3',
    })
    registry.set(entry.iframe, entry)

    const handled = handleEmbeddedWindowMessage({
      event: {
        data: { type: 'APP_BOOTSTRAP_REQUEST' },
        origin: 'https://designer.example.com',
        source: sourceWindow,
      },
      registry,
      resolveBootstrapState(targetEntry) {
        return {
          token: 'token-3',
          refreshToken: 'refresh-3',
          theme: 'dark',
          locale: 'en',
          tenantId: targetEntry.project.tenantId,
        }
      },
      postMessage(targetEntry, message) {
        responses.push([targetEntry.tabKey, message, targetEntry.origin])
      },
    })

    const ignored = handleEmbeddedWindowMessage({
      event: {
        data: { type: 'APP_BOOTSTRAP_REQUEST' },
        origin: 'https://designer.example.com',
        source: unmatchedWindow,
      },
      registry,
      resolveBootstrapState() {
        return {
          token: 'ignored',
        }
      },
      postMessage(targetEntry, message) {
        responses.push([targetEntry.tabKey, message, targetEntry.origin])
      },
    })

    assert.equal(handled.handled, true)
    assert.equal(ignored.handled, false)
    assert.equal(responses.length, 1)
    assert.equal(responses[0][0], 'design-center-p-3')
    assert.equal(responses[0][1].type, 'APP_BOOTSTRAP_RESPONSE')
    assert.equal(responses[0][2], 'https://designer.example.com')
    assert.equal(responses[0][1].payload.projectId, 'p-3')
    assert.equal(responses[0][1].payload.tenantId, 't-3')
    assert.equal(typeof responses[0][1].payload.handoffId, 'string')
  })
})

await run('嵌入同步：相对 handoff URL 也能走完整注册与 bootstrap 链路', () => {
  return withMockLocalStorage(() =>
    withMockLocation('https://ide.example.com', () => {
      const registry = new Map()
      const sourceWindow = { label: 'designer-window-relative' }
      const appEntry = buildAppEntry('designer', { id: 'p-relative', tenantId: 't-relative' })
      const entry = createEmbeddedRegistryEntry({
        iframe: {
          src: appEntry.url,
          contentWindow: sourceWindow,
        },
        origin: appEntry.origin,
        appType: 'designer',
        project: { id: 'p-relative', tenantId: 't-relative' },
        tabKey: 'design-center-p-relative',
      })

      registry.set(entry.iframe, entry)

      const result = handleEmbeddedWindowMessage({
        event: {
          data: { type: 'APP_BOOTSTRAP_REQUEST' },
          origin: 'https://ide.example.com',
          source: sourceWindow,
        },
        registry,
        resolveBootstrapState() {
          return {
            token: 'token-relative',
            refreshToken: 'refresh-relative',
            theme: 'dark',
            locale: 'zh',
          }
        },
      })

      assert.equal(result.handled, true)
      assert.equal(result.entry.origin, 'https://ide.example.com')
      assert.equal(result.responseMessage.payload.projectId, 'p-relative')
    })
  )
})

await run('嵌入同步：AUTH_REFRESHED 会更新宿主 token', () => {
  const registry = new Map()
  const sourceWindow = { label: 'designer-window' }
  const entry = createEmbeddedRegistryEntry({
    iframe: {
      src: 'https://designer.example.com/designer/?handoffId=handoff-4',
      contentWindow: sourceWindow,
    },
    appType: 'designer',
    project: { id: 'p-4', tenantId: 't-4' },
    tabKey: 'design-center-p-4',
  })
  registry.set(entry.iframe, entry)

  const hostAuth = {
    token: 'token-old',
    refreshToken: 'refresh-old',
  }

  const handled = handleEmbeddedWindowMessage({
    event: {
      data: {
        type: 'AUTH_REFRESHED',
        payload: {
          token: 'token-new',
          refreshToken: 'refresh-new',
        },
      },
      origin: 'https://designer.example.com',
      source: sourceWindow,
    },
    registry,
    onAuthRefreshed(payload) {
      hostAuth.token = payload.token
      hostAuth.refreshToken = payload.refreshToken
    },
  })

  assert.equal(handled.handled, true)
  assert.equal(hostAuth.token, 'token-new')
  assert.equal(hostAuth.refreshToken, 'refresh-new')
})

await run('嵌入同步：未知来源不会被处理', () => {
  const registry = new Map()
  const sourceWindow = { label: 'designer-window' }
  const entry = createEmbeddedRegistryEntry({
    iframe: {
      src: 'https://designer.example.com/designer/?handoffId=handoff-5',
      contentWindow: sourceWindow,
    },
    appType: 'designer',
    project: { id: 'p-5', tenantId: 't-5' },
    tabKey: 'design-center-p-5',
  })
  registry.set(entry.iframe, entry)

  let refreshed = false
  let expired = false

  const refreshedResult = handleEmbeddedWindowMessage({
    event: {
      data: { type: 'AUTH_REFRESHED', payload: { token: 'token-ignored' } },
      origin: 'https://unknown.example.com',
      source: sourceWindow,
    },
    registry,
    onAuthRefreshed() {
      refreshed = true
    },
  })

  const expiredResult = handleEmbeddedWindowMessage({
    event: {
      data: { type: 'AUTH_EXPIRED' },
      origin: 'https://designer.example.com',
      source: { label: 'unknown-window' },
    },
    registry,
    onAuthExpired() {
      expired = true
    },
  })

  assert.equal(refreshedResult.handled, false)
  assert.equal(expiredResult.handled, false)
  assert.equal(refreshed, false)
  assert.equal(expired, false)
})

await run('嵌入同步：兼容 document 查询链路时仍使用显式 origin', () => {
  return withMockLocation('https://ide.example.com', () => {
    const events = []
    const designerIframe = {
      src: '/designer/?handoffId=handoff-6',
      contentWindow: {
        postMessage(message, targetOrigin) {
          events.push(['designer', message, targetOrigin])
        },
      },
    }

    const queries = []
    const documentLike = {
      querySelectorAll(selector) {
        queries.push(selector)
        return [designerIframe]
      },
    }

    syncDesignerLocaleToEmbeddedIframes(documentLike, 'en')

    assert.deepEqual(queries, ['iframe.embedded-iframe'])
    assert.deepEqual(events, [['designer', { type: 'LOCALE_UPDATE', locale: 'en' }, 'https://ide.example.com']])
  })
})

if (process.exitCode !== 1) {
  console.log('ALL TESTS PASSED')
}
