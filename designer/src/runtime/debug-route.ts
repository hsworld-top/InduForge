/**
 * 设计器调试路由判定。
 *
 * 正式入口和独立调试入口的行为完全不同：
 * - 开发态 `/designer/debug` 允许独立打开，不要求宿主 bootstrap；
 * - 其他正式入口路径必须走 IDE handoff + bootstrap 流程；
 * - 生产构建不注册 `/debug`，也不再把该路径视为合法调试入口。
 */

const DESIGNER_DEBUG_PATHS = new Set(['/debug', '/designer/debug'])
declare const __DESIGNER_DEBUG_ROUTE_ENABLED__: boolean

function getDefaultDebugRouteEnabled(): boolean {
  return typeof __DESIGNER_DEBUG_ROUTE_ENABLED__ !== 'undefined'
    ? __DESIGNER_DEBUG_ROUTE_ENABLED__
    : true
}

function normalizePathname(pathname: string): string {
  if (!pathname) {
    return '/'
  }

  const trimmed = pathname.trim()
  if (!trimmed || trimmed === '/') {
    return '/'
  }

  return trimmed.endsWith('/') ? trimmed.slice(0, -1) : trimmed
}

function normalizeDebugRouteEnabled(debugRouteEnabled: unknown): boolean {
  if (typeof debugRouteEnabled === 'string') {
    return debugRouteEnabled !== 'false'
  }

  return Boolean(debugRouteEnabled)
}

export function isDesignerDebugRouteEnabled(
  debugRouteEnabled = getDefaultDebugRouteEnabled(),
): boolean {
  return normalizeDebugRouteEnabled(debugRouteEnabled)
}

export function isDesignerDebugPath(
  pathname: string,
  debugRouteEnabled = getDefaultDebugRouteEnabled(),
): boolean {
  if (!isDesignerDebugRouteEnabled(debugRouteEnabled)) {
    return false
  }

  return DESIGNER_DEBUG_PATHS.has(normalizePathname(pathname))
}

/**
 * 对外暴露统一的 debug 判定 helper，避免入口守卫和 bootstrap 各自维护一套路径规则。
 */
export function shouldUseDebugMode(
  pathname: string,
  debugRouteEnabled = getDefaultDebugRouteEnabled(),
): boolean {
  return isDesignerDebugPath(pathname, debugRouteEnabled)
}
