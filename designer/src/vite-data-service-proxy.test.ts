import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('vite data_service proxy', () => {
  it('复用前端同源代理配置，确保预览 Socket 也可通过 Designer 开发服务器访问', async () => {
    const configSource = readFileSync(resolve(__dirname, '../vite.config.js'), 'utf-8')

    expect(configSource).toContain("createFrontendProxy(env)")
  })
})
