/**
 * 前端开发服务器的后端代理约定。
 *
 * 浏览器始终请求当前 Vite origin 下的 /api/v1、/control-socket.io 和 /socket.io；
 * 只有 Vite 进程读取此处解析出的目标地址。因此远程中心不需要为开发机端口放开 CORS。
 */
const DEFAULT_CONTROL_TARGET = 'http://localhost:18101'
const DEFAULT_DATA_TARGET = 'http://localhost:18102'

export function resolveFrontendProxyTargets(env = {}) {
  const centerTarget = String(env.IF_FRONTEND_PROXY_TARGET || '').trim()
  if (centerTarget) {
    return {
      control: centerTarget,
      data: centerTarget,
    }
  }

  return {
    control: String(env.VITE_API_URL || DEFAULT_CONTROL_TARGET).trim(),
    data: String(env.VITE_DATA_SERVICE_URL || DEFAULT_DATA_TARGET).trim(),
  }
}

export function createFrontendProxy(env = {}) {
  const targets = resolveFrontendProxyTargets(env)
  const httpProxy = (target) => ({
    target,
    changeOrigin: true,
    secure: false,
  })
  const socketProxy = (target) => ({
    ...httpProxy(target),
    ws: true,
    rewriteWsOrigin: true,
  })

  return {
    // 必须排在 /api 前，避免数据域请求被控制面泛化路由吞掉。
    '/api/v1/data': httpProxy(targets.data),
    '/control-socket.io': socketProxy(targets.control),
    '/socket.io': socketProxy(targets.data),
    '/api': httpProxy(targets.control),
  }
}
