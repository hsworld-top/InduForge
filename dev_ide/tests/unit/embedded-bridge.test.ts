import { afterEach, describe, expect, it } from 'vitest'

import { createBootstrapResponse, loadHandoffRecord } from '@/utils/embeddedAppBridge'
import {
  createEmbeddedRegistryEntry,
  handleEmbeddedWindowMessage,
} from '@/utils/embeddedIframeSync'

const createMockStorage = (): globalThis.Storage => {
  const bucket = new Map<string, string>()

  return {
    get length() {
      return bucket.size
    },
    clear() {
      bucket.clear()
    },
    getItem(key: string) {
      return bucket.has(key) ? bucket.get(key)! : null
    },
    key(index: number) {
      return [...bucket.keys()][index] ?? null
    },
    removeItem(key: string) {
      bucket.delete(key)
    },
    setItem(key: string, value: string) {
      bucket.set(key, String(value))
    },
  }
}

const originalLocalStorage = globalThis.localStorage

afterEach(() => {
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: originalLocalStorage,
  })
})

describe('embedded bridge', () => {
  it('createBootstrapResponse 只通过 handoffId 暴露上下文，不在 URL 查询串泄露敏感字段', () => {
    Object.defineProperty(globalThis, 'localStorage', {
      configurable: true,
      value: createMockStorage(),
    })

    const response = createBootstrapResponse('designer', {
      pid: 'project-1',
      tenantId: 'tenant-1',
      token: 'token-1',
      refreshToken: 'refresh-1',
      theme: 'dark',
      locale: 'en',
    })

    expect(response.url).toContain('handoffId=')
    expect(response.url).not.toContain('token=')
    expect(response.url).not.toContain('refreshToken=')
    expect(response.url).not.toContain('projectId=')
    expect(response.url).not.toContain('pid=')

    const record = loadHandoffRecord(response.handoffId as string)
    expect(record).toMatchObject({
      handoffId: response.handoffId,
      appType: 'designer',
      projectId: 'project-1',
      tenantId: 'tenant-1',
    })
  })

  it('handleEmbeddedWindowMessage 只处理已登记 source+origin 的 bootstrap 请求', () => {
    Object.defineProperty(globalThis, 'localStorage', {
      configurable: true,
      value: createMockStorage(),
    })

    const registry = new Map()
    const sourceWindow = { label: 'designer-window' }
    const unmatchedWindow = { label: 'unknown-window' }
    const responses: Array<{
      target: unknown
      message: Record<string, unknown>
    }> = []

    const entry = createEmbeddedRegistryEntry({
      iframe: {
        src: 'https://designer.example.com/designer/?handoffId=handoff-1',
        contentWindow: sourceWindow,
      },
      origin: 'https://designer.example.com',
      appType: 'designer',
      project: { id: 'project-2', tenantId: 'tenant-2' },
      tabKey: 'design-center-project-2',
    })

    expect(entry).not.toBeNull()
    registry.set(entry!.iframe, entry)

    const handled = handleEmbeddedWindowMessage({
      event: {
        data: {
          type: 'APP_BOOTSTRAP_REQUEST',
          payload: { requestId: 'req-1' },
        },
        source: sourceWindow,
        origin: 'https://designer.example.com',
      },
      registry,
      resolveBootstrapState() {
        return {
          token: 'token-2',
          refreshToken: 'refresh-2',
          theme: 'light',
          locale: 'zh',
        }
      },
      postMessage(target, message) {
        responses.push({
          target: target.tabKey,
          message,
        })
      },
    })

    const ignored = handleEmbeddedWindowMessage({
      event: {
        data: { type: 'APP_BOOTSTRAP_REQUEST' },
        source: unmatchedWindow,
        origin: 'https://designer.example.com',
      },
      registry,
    })

    expect(handled.handled).toBe(true)
    expect(ignored.handled).toBe(false)
    if (ignored.handled) {
      throw new Error('unmatched source should not be handled')
    }
    expect(ignored.reason).toBe('unregistered-source')
    expect(responses).toHaveLength(1)
    expect(responses[0]).toMatchObject({
      target: 'design-center-project-2',
      message: {
        type: 'APP_BOOTSTRAP_RESPONSE',
      },
    })
  })
})
