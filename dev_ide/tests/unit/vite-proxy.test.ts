import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'

describe('dev_ide vite proxy', () => {
  it('三个前端共用同一套同源代理配置', () => {
    const source = readFileSync('vite.config.ts', 'utf8')

    expect(source).toContain("createFrontendProxy(env)")
  })

  it('远程中心地址只由 Vite 进程读取，HTTP 和两个 Socket 路径都走同一个入口', async () => {
    const { createFrontendProxy, resolveFrontendProxyTargets } = await import(
      '../../../scripts/dev/frontend-proxy.mjs'
    )
    const remoteTarget = 'http://172.16.125.129:18080'
    const proxy = createFrontendProxy({
      IF_FRONTEND_PROXY_TARGET: remoteTarget,
      VITE_API_URL: 'http://localhost:18101',
      VITE_DATA_SERVICE_URL: 'http://localhost:18102',
    })

    expect(resolveFrontendProxyTargets({ IF_FRONTEND_PROXY_TARGET: remoteTarget })).toEqual({
      control: remoteTarget,
      data: remoteTarget,
    })
    expect(proxy['/api/v1/data'].target).toBe(remoteTarget)
    expect(proxy['/api'].target).toBe(remoteTarget)
    expect(proxy['/control-socket.io']).toMatchObject({ target: remoteTarget, ws: true })
    expect(proxy['/socket.io']).toMatchObject({ target: remoteTarget, ws: true })
  })

  it('未启用远程模式时，保留控制面和数据域的本地独立目标', async () => {
    const { resolveFrontendProxyTargets } = await import('../../../scripts/dev/frontend-proxy.mjs')

    expect(
      resolveFrontendProxyTargets({
        VITE_API_URL: 'http://localhost:18101',
        VITE_DATA_SERVICE_URL: 'http://localhost:18102',
      }),
    ).toEqual({
      control: 'http://localhost:18101',
      data: 'http://localhost:18102',
    })
  })
})
