import { validateFrontendLinuxProxyTarget } from './frontend-linux-env.mjs'

const WORKSPACE_SERVICE_PATTERN = '(ai|code|preview|preview-control)'
const UUID_PATTERN = '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}'
const DNS_SUFFIX_PATTERN = /^workspace(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$/

/** frontend-linux 工作区入口的域名后缀，例如 workspace.172.16.125.129.nip.io。 */
export function validateFrontendWorkspaceProxySuffix(value) {
  const suffix = String(value || '')
    .trim()
    .toLowerCase()
  if (!suffix) return { valid: false, message: 'IF_FRONTEND_WORKSPACE_PROXY_SUFFIX 不能为空。' }
  if (suffix.includes('://') || suffix.includes('/') || suffix.includes(':')) {
    return {
      valid: false,
      message:
        'IF_FRONTEND_WORKSPACE_PROXY_SUFFIX 必须是 workspace. 开头的纯域名，不能包含协议、端口或路径。',
    }
  }
  if (!DNS_SUFFIX_PATTERN.test(suffix)) {
    return {
      valid: false,
      message: 'IF_FRONTEND_WORKSPACE_PROXY_SUFFIX 必须是合法的 workspace. 域名后缀。',
    }
  }
  return { valid: true, suffix }
}

function readPort(value, name) {
  const port = Number(value)
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`${name} 必须是 1 到 65535 的端口号。`)
  }
  return port
}

/**
 * 仅将四类、带 UUID 的 localhost 工作区 Host 映射到远端固定 workspace 域。
 * 其余 Host 都返回 null，调用方必须绕过而不是尝试代理。
 */
export function resolveLocalWorkspaceRequest(host, localPort, suffix, target) {
  const normalizedHost = String(host || '')
    .trim()
    .toLowerCase()
  const port = readPort(localPort, '本地前端端口')
  const match = normalizedHost.match(
    new RegExp(`^${WORKSPACE_SERVICE_PATTERN}-(${UUID_PATTERN})\\.localhost:${port}$`),
  )
  if (!match) return null

  const center = new URL(target)
  const remoteHost = `${match[1]}-${match[2]}.${suffix}${center.port ? `:${center.port}` : ''}`
  return {
    localOrigin: `http://${normalizedHost}`,
    remoteHost,
    remoteOrigin: `${center.protocol}//${remoteHost}`,
  }
}

export function resolveFrontendWorkspaceProxyConfig(
  env = {},
  { requireConfig = false, localPort } = {},
) {
  const rawSuffix = String(env.IF_FRONTEND_WORKSPACE_PROXY_SUFFIX || '').trim()
  if (!rawSuffix) {
    if (requireConfig)
      throw new Error('frontend-linux 模式必须配置 IF_FRONTEND_WORKSPACE_PROXY_SUFFIX。')
    return null
  }

  const suffixValidation = validateFrontendWorkspaceProxySuffix(rawSuffix)
  if (!suffixValidation.valid) {
    throw new Error(`前端 Linux 工作区代理配置无效：${suffixValidation.message}`)
  }
  const targetValidation = validateFrontendLinuxProxyTarget(env.IF_FRONTEND_PROXY_TARGET)
  if (!targetValidation.valid) {
    throw new Error(`前端 Linux 工作区代理配置无效：${targetValidation.message}`)
  }
  return {
    localPort: readPort(localPort, '本地前端端口'),
    suffix: suffixValidation.suffix,
    target: targetValidation.target,
  }
}

function rewriteProxyOrigin(proxyRequest, request, workspace) {
  if (String(request.headers.origin || '').toLowerCase() === workspace.localOrigin) {
    proxyRequest.setHeader('origin', workspace.remoteOrigin)
  }
  proxyRequest.setHeader('host', workspace.remoteHost)
}

/**
 * 使用中心入口作为 TCP 目标、工作区域名作为 Host 路由目标。bypass 严格拒绝非工作区 Host，
 * 因此此规则不会把 Vite 变成任意 Host 的转发器。
 */
export function createFrontendWorkspaceProxy(env = {}, options = {}) {
  const config = resolveFrontendWorkspaceProxyConfig(env, options)
  if (!config) return {}
  const resolveRequest = (request) =>
    resolveLocalWorkspaceRequest(
      request.headers.host,
      config.localPort,
      config.suffix,
      config.target,
    )

  return {
    '^/.*': {
      target: config.target,
      changeOrigin: false,
      secure: false,
      ws: true,
      bypass(request) {
        // 返回原路径让 Vite 继续正常处理；绝不把未匹配 Host 发送到 target。
        return resolveRequest(request) ? undefined : request.url
      },
      configure(proxy) {
        proxy.on('proxyReq', (proxyRequest, request) => {
          const workspace = resolveRequest(request)
          if (workspace) rewriteProxyOrigin(proxyRequest, request, workspace)
        })
        proxy.on('proxyReqWs', (proxyRequest, request) => {
          const workspace = resolveRequest(request)
          if (workspace) rewriteProxyOrigin(proxyRequest, request, workspace)
        })
        proxy.on('proxyRes', (proxyResponse, request) => {
          const workspace = resolveRequest(request)
          if (!workspace) return
          const allowOrigin = proxyResponse.headers['access-control-allow-origin']
          if (String(allowOrigin || '').toLowerCase() === workspace.remoteOrigin) {
            proxyResponse.headers['access-control-allow-origin'] = workspace.localOrigin
          }
        })
      },
    },
  }
}
