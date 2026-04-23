import { afterEach, describe, expect, it } from 'vitest'
import { buildAppEntry, buildAppUrl } from '@/utils/appUrl'
import { loadHandoffRecord } from '@/utils/embeddedAppBridge'

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

describe('appUrl', () => {
  it('designer 入口 URL 只保留 handoffId，不泄露敏感查询参数', () => {
    Object.defineProperty(globalThis, 'localStorage', {
      configurable: true,
      value: createMockStorage(),
    })

    localStorage.setItem('auth_token', 'token-1')
    localStorage.setItem('refresh_token', 'refresh-1')
    localStorage.setItem('theme', JSON.stringify('dark'))
    localStorage.setItem('language', JSON.stringify('en'))

    const url = buildAppUrl('designer', { id: 'p-1', tenantId: 't-1' })
    const parsed = new URL(url, 'http://localhost')
    const handoffId = parsed.searchParams.get('handoffId')

    expect(parsed.pathname).toBe('/designer/')
    expect(handoffId).toBeTruthy()
    expect(parsed.searchParams.get('pid')).toBeNull()
    expect(parsed.searchParams.get('projectId')).toBeNull()
    expect(parsed.searchParams.get('token')).toBeNull()
    expect(parsed.searchParams.get('refreshToken')).toBeNull()
    expect(parsed.searchParams.get('theme')).toBeNull()
    expect(parsed.searchParams.get('locale')).toBeNull()

    const record = loadHandoffRecord(handoffId as string)
    expect(record).toMatchObject({
      appType: 'designer',
      projectId: 'p-1',
      tenantId: 't-1',
    })
  })

  it('buildAppEntry 返回宿主 origin，供 iframe 注册链路使用', () => {
    const entry = buildAppEntry('designer', {
      id: 'p-origin',
      tenantId: 't-origin',
    })

    expect(entry.url.startsWith('/designer/?handoffId=')).toBe(true)
    expect(entry.origin).toBe(window.location.origin)
  })
})
