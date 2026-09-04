const PLACEHOLDER_HOSTS = new Set(['your-center-host', 'center-host', 'example.com'])

/** 从单个 .env 内容中读取目标，供启动前校验与单测复用。 */
export function readFrontendLinuxProxyTarget(content) {
  const match = String(content).match(/^\s*IF_FRONTEND_PROXY_TARGET\s*=\s*(.*?)\s*(?:#.*)?$/m)
  if (!match) return ''

  const value = match[1].trim()
  if (
    (value.startsWith('"') && value.endsWith('"')) ||
    (value.startsWith("'") && value.endsWith("'"))
  ) {
    return value.slice(1, -1).trim()
  }
  return value
}

/**
 * Linux 中心必须是完整 HTTP(S) origin，不能把占位值、凭据或子路径带入代理目标。
 */
export function validateFrontendLinuxProxyTarget(value) {
  const raw = String(value || '').trim()
  if (!raw) return { valid: false, message: 'IF_FRONTEND_PROXY_TARGET 不能为空。' }
  if (raw.includes('<') || raw.includes('>')) {
    return { valid: false, message: 'IF_FRONTEND_PROXY_TARGET 不能使用占位符。' }
  }

  let target
  try {
    target = new URL(raw)
  } catch {
    return { valid: false, message: 'IF_FRONTEND_PROXY_TARGET 必须是完整的 http:// 或 https:// 地址。' }
  }

  if (!['http:', 'https:'].includes(target.protocol)) {
    return { valid: false, message: 'IF_FRONTEND_PROXY_TARGET 只允许 http 或 https 协议。' }
  }
  if (target.username || target.password) {
    return { valid: false, message: 'IF_FRONTEND_PROXY_TARGET 不能包含账号或密码。' }
  }
  if (target.pathname !== '/' || target.search || target.hash) {
    return { valid: false, message: 'IF_FRONTEND_PROXY_TARGET 必须是纯 origin，不能包含路径、查询参数或片段。' }
  }
  if (PLACEHOLDER_HOSTS.has(target.hostname.toLowerCase()) || target.hostname.endsWith('.example')) {
    return { valid: false, message: 'IF_FRONTEND_PROXY_TARGET 不能使用占位主机名。' }
  }

  return { valid: true, target: target.origin }
}
