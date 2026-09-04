import { describe, expect, it, vi } from 'vitest'
import {
  createFrontendWorkspaceProxy,
  resolveFrontendWorkspaceProxyConfig,
  resolveFrontendWorkspaceProxyPort,
  resolveLocalWorkspaceRequest,
  validateFrontendWorkspaceProxySuffix,
} from '../../../scripts/dev/frontend-workspace-proxy.mjs'

const target = 'http://172.16.125.129:18080'
const suffix = 'workspace.172.16.125.129.nip.io'
const uuid = '123e4567-e89b-12d3-a456-426614174000'

describe('frontend-linux 工作区 Host 代理', () => {
  it('只接受四个服务的 UUID.localhost Host，并构造固定远端 Host', () => {
    expect(
      resolveLocalWorkspaceRequest(`preview-${uuid}.localhost:18604`, 18604, suffix, target),
    ).toEqual({
      localOrigin: `http://preview-${uuid}.localhost:18604`,
      remoteHost: `preview-${uuid}.${suffix}:18080`,
      remoteOrigin: `http://preview-${uuid}.${suffix}:18080`,
    })
    for (const host of [
      `admin-${uuid}.localhost:18604`,
      `preview-${uuid}.localhost:18603`,
      `preview-not-a-uuid.localhost:18604`,
      `preview-${uuid}.example.com:18604`,
    ]) {
      expect(resolveLocalWorkspaceRequest(host, 18604, suffix, target)).toBeNull()
    }
  })

  it('工作区代理端口默认为 18604，显式值必须是合法端口', () => {
    expect(resolveFrontendWorkspaceProxyPort()).toBe(18604)
    expect(resolveFrontendWorkspaceProxyPort('19604')).toBe(19604)
    expect(() => resolveFrontendWorkspaceProxyPort('not-a-port')).toThrow(
      'IF_FRONTEND_WORKSPACE_PROXY_PORT',
    )
  })

  it('仅 frontend-linux 需要受控 workspace 后缀，拒绝协议、端口和非 workspace 域', () => {
    expect(validateFrontendWorkspaceProxySuffix(suffix)).toEqual({ valid: true, suffix })
    for (const value of [
      '',
      'https://workspace.example.com',
      'workspace.example.com:18080',
      'proxy.example.com',
    ]) {
      expect(validateFrontendWorkspaceProxySuffix(value).valid).toBe(false)
    }
    expect(() =>
      resolveFrontendWorkspaceProxyConfig(
        { IF_FRONTEND_PROXY_TARGET: target },
        { requireConfig: true, localPort: 18601 },
      ),
    ).toThrow('IF_FRONTEND_WORKSPACE_PROXY_SUFFIX')
  })

  it('命中时改写 Host 与同 workspace Origin，IDE localhost Origin 保持不变，CORS 只回写受控 self-origin', () => {
    const proxy = createFrontendWorkspaceProxy(
      { IF_FRONTEND_PROXY_TARGET: target, IF_FRONTEND_WORKSPACE_PROXY_SUFFIX: suffix },
      { requireConfig: true, localPort: 18604 },
    )['^/.*']
    const handlers: Record<string, Function> = {}
    proxy.configure({ on: (name: string, handler: Function) => (handlers[name] = handler) })
    const request = {
      headers: { host: `ai-${uuid}.localhost:18604`, origin: `http://ai-${uuid}.localhost:18604` },
    }
    const proxyRequest = { setHeader: vi.fn() }

    expect(proxy.bypass(request)).toBeUndefined()
    handlers.proxyReq(proxyRequest, request)
    expect(proxyRequest.setHeader).toHaveBeenCalledWith(
      'origin',
      `http://ai-${uuid}.${suffix}:18080`,
    )
    expect(proxyRequest.setHeader).toHaveBeenCalledWith('host', `ai-${uuid}.${suffix}:18080`)

    const ideOriginRequest = {
      headers: { host: `ai-${uuid}.localhost:18604`, origin: 'http://localhost:18601' },
    }
    const ideProxyRequest = { setHeader: vi.fn() }
    handlers.proxyReqWs(ideProxyRequest, ideOriginRequest)
    expect(ideProxyRequest.setHeader).not.toHaveBeenCalledWith('origin', expect.anything())
    expect(ideProxyRequest.setHeader).toHaveBeenCalledWith('host', `ai-${uuid}.${suffix}:18080`)

    const response = {
      headers: { 'access-control-allow-origin': `http://ai-${uuid}.${suffix}:18080` },
    }
    handlers.proxyRes(response, request)
    expect(response.headers['access-control-allow-origin']).toBe(
      `http://ai-${uuid}.localhost:18604`,
    )
    const external = { headers: { host: 'evil.example:18604' }, url: '/anything' }
    expect(proxy.bypass(external)).toBe(false)
  })

  it('独立工作区代理对 /api 和 /socket.io 也按 Host 转发，不匹配 Host 一律拒绝', () => {
    const proxy = createFrontendWorkspaceProxy(
      { IF_FRONTEND_PROXY_TARGET: target, IF_FRONTEND_WORKSPACE_PROXY_SUFFIX: suffix },
      { requireConfig: true, localPort: 18604 },
    )['^/.*']
    for (const url of ['/api/v1/project', '/socket.io/?EIO=4', '/designer/assets/app.js']) {
      expect(
        proxy.bypass({ headers: { host: `code-${uuid}.localhost:18604` }, url }),
      ).toBeUndefined()
    }
    expect(proxy.bypass({ headers: { host: `code-${uuid}.localhost:18601` }, url: '/api' })).toBe(
      false,
    )
  })
})
