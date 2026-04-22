import { afterEach, describe, expect, it } from 'vitest'

import { STORAGE_KEYS } from '@/constants'
import { Storage } from '@/utils/storage'

const createMockStorage = (): Storage => {
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

describe('utils/storage', () => {
  it('Storage.set/get 支持 JSON 读写', () => {
    Object.defineProperty(globalThis, 'localStorage', {
      configurable: true,
      value: createMockStorage(),
    })

    const payload = {
      id: 'project-1',
      permissions: ['read', 'write'],
      enabled: true,
    }

    Storage.set(STORAGE_KEYS.USER_INFO, payload)

    expect(Storage.get(STORAGE_KEYS.USER_INFO)).toEqual(payload)
  })

  it('getTheme 在未设置时返回默认值 light', () => {
    Object.defineProperty(globalThis, 'localStorage', {
      configurable: true,
      value: createMockStorage(),
    })

    expect(Storage.getTheme()).toBe('light')
  })
})