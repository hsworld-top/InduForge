import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'

describe('dev_ide vite proxy', () => {
  it('数据域请求必须优先代理到 data_service，而不是被泛化 /api 规则吞掉', () => {
    const source = readFileSync('vite.config.ts', 'utf8')
    const dataProxyIndex = source.indexOf("'/api/v1/data'")
    const apiProxyIndex = source.indexOf("'/api':")

    expect(dataProxyIndex).toBeGreaterThanOrEqual(0)
    expect(apiProxyIndex).toBeGreaterThanOrEqual(0)
    expect(dataProxyIndex).toBeLessThan(apiProxyIndex)
    expect(source).toContain('target: env.VITE_DATA_SERVICE_URL')
    expect(source).toContain('target: env.VITE_API_URL')
  })
})
