/**
 * 根据目标路由拼出当前浏览器里实际会出现的 pathname。
 * 这样即便当前 href 仍停留在正式入口，程序化跳到 `/debug` 时也不会被旧 pathname 误判。
 */
export const resolveRoutePathname = (routePath: string) => {
  if (!routePath || routePath === "/") {
    return "/datacenter/";
  }

  return `/datacenter${routePath}`.replace(/\/{2,}/g, "/");
};

/**
 * 用目标路由生成入口判断 URL。
 * 只使用 currentUrl 的 origin，不继承旧 query，避免内部模块切换重复消费 handoff。
 */
export const buildRouteRuntimeUrl = (currentUrl: string, routePath: string) =>
  new URL(resolveRoutePathname(routePath), currentUrl);
