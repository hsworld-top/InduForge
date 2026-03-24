// @ts-nocheck
/**
 * 页面 CRUD 辅助（从 editor-store 拆出，保持列表与文档同步的纯函数）
 */

/**
 * 重命名后更新内存中的 pages 列表项
 */
export function mapPagesAfterRename(
  list: unknown[],
  pageId: string,
  name: string,
  path: string | undefined,
): unknown[] {
  return list.map((page: { id?: string; path?: string } & Record<string, unknown>) =>
    page.id === pageId
      ? {
          ...page,
          name,
          path: path !== undefined ? path : page.path,
        }
      : page,
  );
}
