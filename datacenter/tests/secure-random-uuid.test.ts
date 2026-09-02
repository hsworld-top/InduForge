import { afterEach, describe, expect, it, vi } from 'vitest'
import { secureRandomUUID } from '@/utils/secure-random-uuid'

const originalCrypto = globalThis.crypto

afterEach(() => {
  Object.defineProperty(globalThis, 'crypto', { configurable: true, value: originalCrypto })
})

describe('secureRandomUUID', () => {
  it('uses randomUUID when the browser provides it', () => {
    Object.defineProperty(globalThis, 'crypto', {
      configurable: true,
      value: { randomUUID: vi.fn(() => 'native-id') },
    })
    expect(secureRandomUUID()).toBe('native-id')
  })

  it('uses secure getRandomValues fallback when randomUUID is unavailable', () => {
    Object.defineProperty(globalThis, 'crypto', {
      configurable: true,
      value: {
        getRandomValues: (bytes: Uint8Array) => {
          bytes.fill(0)
          return bytes
        },
      },
    })
    expect(secureRandomUUID()).toBe('00000000-0000-4000-8000-000000000000')
  })

  it('fails closed without a secure random source', () => {
    Object.defineProperty(globalThis, 'crypto', { configurable: true, value: undefined })
    expect(() => secureRandomUUID()).toThrow('安全随机数')
  })
})
