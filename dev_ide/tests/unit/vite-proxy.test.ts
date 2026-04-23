import { describe, expect, it } from 'vitest'

import createViteConfig from '../../vite.config'

describe('dev_ide vite proxy', () => {
  it('数据域请求必须优先代理到 data_service，而不是被泛化 /api 规则吞掉', () => {
    const config = createViteConfig({
      command: 'serve',
      mode: 'development',
      isPreview: false,
    })

    const proxy = config.server?.proxy as Record<string, { target?: string }>
    const proxyKeys = Object.keys(proxy)

    expect(proxy['/api/v1/data']).toBeDefined()
    expect(proxy['/api/v1/data']?.target).toBe('http://localhost:19602')
    expect(proxy['/api']?.target).toBe('http://localhost:19601')
    expect(proxyKeys.indexOf('/api/v1/data')).toBeGreaterThanOrEqual(0)
    expect(proxyKeys.indexOf('/api')).toBeGreaterThanOrEqual(0)
    expect(proxyKeys.indexOf('/api/v1/data')).toBeLessThan(proxyKeys.indexOf('/api'))
  })
})
