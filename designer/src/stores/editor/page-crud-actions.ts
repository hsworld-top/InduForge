/**
 * 页面 CRUD 辅助（从 editor-store 拆出，保持列表与文档同步的纯函数）
 */

/** 工程页面列表项（与 Pinia 中 pages 数组元素一致的最小形状） */
export type PageListEntry = {
  id: string
  name?: string
  path?: string
  type?: string
} & Record<string, unknown>

/**
 * 重命名后更新内存中的 pages 列表项
 */
export function mapPagesAfterRename(
  list: PageListEntry[],
  pageId: string,
  name: string,
  path: string | undefined,
): PageListEntry[] {
  return list.map((page) => {
    if (page.id !== pageId) return page
    const next: PageListEntry = { ...page, name }
    if (path !== undefined) {
      next.path = path
    }
    return next
  })
}
