import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'

describe('dev_ide vite proxy', () => {
  it('三个前端共用同一套同源代理配置', () => {
    const source = readFileSync('vite.config.ts', 'utf8')
    const datacenterSource = readFileSync('../datacenter/vite.config.js', 'utf8')
    const designerSource = readFileSync('../designer/vite.config.js', 'utf8')

    expect(source).toContain('createFrontendProxy(env,')
    expect(source).toContain("requireCenterTarget: mode === 'frontend-linux'")
    expect(datacenterSource).toContain("requireCenterTarget: mode === 'frontend-linux'")
    expect(designerSource).toContain("requireCenterTarget: mode === 'frontend-linux'")
    // IDE 仍然用本机的两个 Vite 子应用承载 Wujie，统一启动命令必须包含它们。
    expect(source).toContain("'/datacenter'")
    expect(source).toContain("'/designer'")
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

  it('frontend-linux 模式缺少或填写错误中心地址时立即失败，不会回落到本机后端', async () => {
    const { resolveFrontendProxyTargets } = await import('../../../scripts/dev/frontend-proxy.mjs')

    expect(() => resolveFrontendProxyTargets({}, { requireCenterTarget: true })).toThrow(
      'frontend-linux 模式必须配置 IF_FRONTEND_PROXY_TARGET。',
    )
    expect(() =>
      resolveFrontendProxyTargets(
        { IF_FRONTEND_PROXY_TARGET: 'http://your-center-host:18080' },
        { requireCenterTarget: true },
      ),
    ).toThrow('前端 Linux 中心代理配置无效')
  })

  it('根前端命令默认连接 Linux 中心，并在新机器上先准备端口配置', () => {
    const packageJson = JSON.parse(readFileSync('../package.json', 'utf8'))
    const scripts = packageJson.scripts as Record<string, string>

    expect(scripts['dev:frontend-linux:env']).toContain('pnpm dev:env')
    for (const name of ['dev:ide', 'dev:datacenter', 'dev:designer']) {
      expect(scripts[name]).toContain('vite --mode frontend-linux')
      expect(scripts[`${name}:local`]).toContain('pnpm dev:env')
    }
    expect(scripts['dev:frontend']).toBe('pnpm dev:frontend:linux')
  })
})
