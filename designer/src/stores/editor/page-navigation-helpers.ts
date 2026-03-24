/**
 * 工程打开时选择首屏页面 ID（从 editor-store 拆出的纯函数）
 */

export type PageIdEntry = { id: string };

/**
 * 根据 entry.homePageId 与页面列表决定加载哪一页；无有效页时返回 null
 */
export function resolveLandingPageId(
  pageList: PageIdEntry[],
  entryConfig: { homePageId?: string } | null | undefined,
): string | null {
  if (!pageList.length) return null;
  const homePageId = entryConfig?.homePageId;
  const target =
    homePageId && pageList.some((p) => p.id === homePageId)
      ? homePageId
      : pageList[0]?.id;
  return target ? String(target) : null;
}
