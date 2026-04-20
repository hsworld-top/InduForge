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
  isDesignerEmbeddedIframe,
  syncDesignerLocaleToEmbeddedIframes,
} from '../src/utils/embeddedIframeSync.js'
import { buildAppUrl } from '../src/utils/appUrl.js'

const run = (name, fn) => {
  try {
    fn()
    console.log(`PASS ${name}`)
  } catch (error) {
    console.error(`FAIL ${name}`)
    console.error(error)
    process.exitCode = 1
  }
}

run('权限规则：hasRole', () => {
  assert.equal(hasRole(ROLES.SYSTEM_ADMIN, [ROLES.SYSTEM_ADMIN, ROLES.OPS_ADMIN]), true)
  assert.equal(hasRole(ROLES.USER, [ROLES.SYSTEM_ADMIN]), false)
  assert.equal(hasRole('', [ROLES.SYSTEM_ADMIN]), false)
})

run('权限规则：canAccessTab', () => {
  assert.equal(canAccessTab('tenant-management', ROLES.SUPER_ADMIN), true)
  assert.equal(canAccessTab('tenant-management', ROLES.SYSTEM_ADMIN), false)
  assert.equal(canAccessTab('dashboard', ROLES.USER), true)
})

run('权限规则：业务判定函数', () => {
  assert.equal(canManageUsers(ROLES.USER_ADMIN), true)
  assert.equal(canManageUsers(ROLES.OPS_ADMIN), false)
  assert.equal(canApproveNodes(ROLES.SYSTEM_ADMIN), true)
  assert.equal(canApproveNodes(ROLES.USER_ADMIN), false)
  assert.equal(canRequestTenantStats(ROLES.SUPER_ADMIN), true)
  assert.equal(canRequestTenantStats(ROLES.SYSTEM_ADMIN), false)
  assert.equal(getTabAccessDeniedMessage('system-settings'), '只有系统管理员才能访问系统设置')
})

run('运维状态：聚合统计', () => {
  const summary = buildDeployStatusSummary([
    { deployments: [{ status: 'running' }, { status: 'deploying' }, { status: 'error' }] },
    { deployments: [{ status: 'pending' }, { status: 'stopped' }, { status: 'failed' }] },
  ])
  assert.deepEqual(summary, { running: 1, deploying: 2, stopped: 1, failed: 2 })
})

run('运维状态：映射与失败判定', () => {
  assert.equal(getDeployStatusType('running'), 'success')
  assert.equal(getDeployStatusType('failed'), 'danger')
  assert.equal(getDeployLabel('pending'), '等待中')
  assert.equal(isFailedDeploy({ status: 'error' }), true)
  assert.equal(getDeployFailureReason({ errorMessage: 'network error' }), 'network error')
})

run('国际化：核心键值存在性', () => {
  assert.equal(messages.zh.dashboard.title, '仪表盘')
  assert.equal(messages.en.dashboard.title, 'Dashboard')
  assert.equal(messages.zh.profile.uploadAvatar, '上传头像')
  assert.equal(messages.en.profile.uploadAvatar, 'Upload Avatar')
  assert.equal(messages.zh.auth.tenantCode, '租户代码')
  assert.equal(messages.en.auth.tenantCode, 'Tenant Code')
})

run('应用地址：designer 透传当前语言与主题', () => {
  const originalLocalStorage = globalThis.localStorage
  globalThis.localStorage = {
    getItem(key) {
      const values = {
        auth_token: 'token-1',
        refresh_token: 'refresh-1',
        theme: JSON.stringify('dark'),
        language: JSON.stringify('en'),
      }
      return values[key] ?? null
    },
    setItem() {},
    removeItem() {},
    clear() {},
  }

  try {
    const url = buildAppUrl('designer', { id: 'p-1', tenantId: 't-1' })
    assert.ok(url.includes('pid=p-1'))
    assert.ok(url.includes('tenant=t-1'))
    assert.ok(url.includes('token=token-1'))
    assert.ok(url.includes('refreshToken=refresh-1'))
    assert.ok(url.includes('theme=dark'))
    assert.ok(url.includes('locale=en'))
    assert.ok(url.includes('type=app'))
  } finally {
    globalThis.localStorage = originalLocalStorage
  }
})

run('应用地址：datacenter 不追加语言参数', () => {
  const originalLocalStorage = globalThis.localStorage
  globalThis.localStorage = {
    getItem(key) {
      const values = {
        theme: JSON.stringify('light'),
        language: JSON.stringify('zh'),
      }
      return values[key] ?? null
    },
    setItem() {},
    removeItem() {},
    clear() {},
  }

  try {
    const url = buildAppUrl('datacenter', { id: 'p-2', tenantId: 't-2' })
    assert.ok(url.includes('/datacenter/?'))
    assert.ok(url.includes('theme=light'))
    assert.ok(!url.includes('locale='))
  } finally {
    globalThis.localStorage = originalLocalStorage
  }
})

run('嵌入同步：生成主题更新消息', () => {
  assert.deepEqual(createEmbeddedUpdateMessage('THEME_UPDATE', 'theme', 'dark'), {
    type: 'THEME_UPDATE',
    theme: 'dark',
  })
})

run('嵌入同步：设计中心 iframe 判定', () => {
  assert.equal(isDesignerEmbeddedIframe({ src: '/designer/?pid=1' }), true)
  assert.equal(isDesignerEmbeddedIframe({ src: '/designer?pid=1' }), true)
  assert.equal(isDesignerEmbeddedIframe({ src: 'http://host/designer/?pid=1' }), true)
  assert.equal(isDesignerEmbeddedIframe({ src: '/datacenter/?pid=1' }), false)
  assert.equal(isDesignerEmbeddedIframe({ src: '' }), false)
  assert.equal(isDesignerEmbeddedIframe({}), false)
  assert.equal(isDesignerEmbeddedIframe({ src: '::bad::url' }), false)
})

run('嵌入同步：仅通过 Dashboard 链路同步 designer 语言', () => {
  const events = []
  const designerIframe = {
    src: '/designer/?pid=p-1',
    contentWindow: {
      postMessage(message, targetOrigin) {
        events.push(['designer', message, targetOrigin])
      },
    },
  }
  const datacenterIframe = {
    src: '/datacenter/?pid=p-1',
    contentWindow: {
      postMessage(message, targetOrigin) {
        events.push(['datacenter', message, targetOrigin])
      },
    },
  }
  const malformedIframe = {
    contentWindow: {
      postMessage(message, targetOrigin) {
        events.push(['malformed', message, targetOrigin])
      },
    },
  }

  const queries = []
  const documentLike = {
    querySelectorAll(selector) {
      queries.push(selector)
      return [designerIframe, datacenterIframe, malformedIframe, null]
    },
  }

  syncDesignerLocaleToEmbeddedIframes(
    documentLike,
    createEmbeddedUpdateMessage('LOCALE_UPDATE', 'locale', 'en').locale
  )

  assert.deepEqual(queries, ['iframe.embedded-iframe'])
  assert.deepEqual(events, [['designer', { type: 'LOCALE_UPDATE', locale: 'en' }, '*']])
})

if (process.exitCode !== 1) {
  console.log('ALL TESTS PASSED')
}
