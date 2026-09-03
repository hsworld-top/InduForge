import type { MicroAppType } from '@/types/micro-app'

/**
 * Wujie 会通过此 fetch 拉取子应用入口和入口引用的静态资产。入口 HTML 不带构建 hash，
 * 浏览器若复用旧 HTML，会继续请求已经删除的 hash 资产并造成子应用白屏；仅入口强制
 * no-store，hash 资产仍按服务端的长期缓存策略加载。
 */
export const createMicroAppEntryFetch = (
  appType: MicroAppType,
  fetcher: typeof window.fetch = window.fetch.bind(window),
): typeof window.fetch => {
  const entryPath = `/${appType}/`
  const indexPath = `${entryPath}index.html`

  return (input, init) => {
    const rawUrl =
      typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    const pathname = new URL(rawUrl, window.location.origin).pathname
    const isEntry = pathname === entryPath || pathname === indexPath

    return fetcher(input, isEntry ? { ...init, cache: 'no-store' } : init)
  }
}
