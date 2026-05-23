import { afterEach, describe, expect, it, vi } from 'vitest'
import { Storage } from './storage'

const store: Record<string, string> = {}

describe('storage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    Object.keys(store).forEach((k) => {
      delete store[k]
    })
  })

  it('get returns default when missing', () => {
    vi.stubGlobal('localStorage', {
      getItem: () => null,
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    })
    expect(Storage.get('missing', 'd')).toBe('d')
  })

  it('set and get round-trip JSON', () => {
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => store[key] ?? null,
      setItem: (key: string, value: string) => {
        store[key] = value
      },
      removeItem: (key: string) => {
        delete store[key]
      },
      clear: () => {
        Object.keys(store).forEach((k) => {
          delete store[k]
        })
      },
    })
    Storage.set('k', { a: 1 })
    expect(Storage.get<{ a: number }>('k', null)).toEqual({ a: 1 })
  })

  it('theme and language helpers round-trip JSON values', () => {
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => store[key] ?? null,
      setItem: (key: string, value: string) => {
        store[key] = value
      },
      removeItem: (key: string) => {
        delete store[key]
      },
      clear: () => {
        Object.keys(store).forEach((k) => {
          delete store[k]
        })
      },
    })

    Storage.setTheme('dark')
    Storage.setLanguage('en')

    expect(Storage.getTheme()).toBe('dark')
    expect(Storage.getLanguage()).toBe('en')
    expect(store.theme).toBe(JSON.stringify('dark'))
    expect(store.language).toBe(JSON.stringify('en'))
  })

  it('theme and language helpers fall back to defaults on invalid values', () => {
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => {
        if (key === 'theme') {
          return JSON.stringify('solarized')
        }
        if (key === 'language') {
          return JSON.stringify('jp')
        }
        return null
      },
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    })

    expect(Storage.getTheme()).toBe('light')
    expect(Storage.getLanguage()).toBe('zh')
  })
})
