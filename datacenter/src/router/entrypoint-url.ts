/**
 * 根据目标路由拼出当前浏览器里实际会出现的 pathname。
 * 这样即便当前 href 仍停留在正式入口，程序化跳到 `/debug` 时也不会被旧 pathname 误判。
 */
export const resolveRoutePathname = (routePath: string) => {
  if (!routePath || routePath === '/') {
    return '/datacenter/'
  }

  return `/datacenter${routePath}`.replace(/\/{2,}/g, '/')
}

const hasHandoffSearch = (search: string) => {
  const params = new URLSearchParams(search)
  return Boolean(params.get('handoff') || params.get('handoffId'))
}

/**
 * 用目标路由生成入口判断 URL。
 * 首次进入根路由时必须保留入口 handoff，否则 iframe 会复用旧工程缓存。
 * 内部模块切换仍丢弃旧 query，避免重复消费已经完成的 handoff。
 */
export const buildRouteRuntimeUrl = (currentUrl: string, routePath: string) => {
  const source = new URL(currentUrl)
  const target = new URL(resolveRoutePathname(routePath), source)
  if ((routePath === '/' || routePath === '') && hasHandoffSearch(source.search)) {
    target.search = source.search
  }
  return target
}
