type Translate = (key: string, params?: Record<string, unknown>) => string

interface DashboardTabTitleLike {
  title?: string
  titleKey?: string | null
  titlePrefix?: string | null
  titleParams?: Record<string, unknown> | null
}

/**
 * 统一解析 Dashboard 标签标题。
 *
 * 设计中心 / 数据中心这类工程嵌入页在打开时不能把标题拍平成静态字符串，
 * 否则宿主语言切换后只能刷新整页才能拿到新文案。
 *
 * @param {object|null|undefined} tab - 标签配置
 * @param {(key: string, params?: Record<string, unknown>) => string} translate - i18n 翻译函数
 * @returns {string} 最终标题
 */
export const resolveDashboardTabTitle = (
  tab: DashboardTabTitleLike | null | undefined,
  translate: Translate
): string => {
  if (!tab || typeof translate !== 'function') return ''

  const titleKey = typeof tab.titleKey === 'string' ? tab.titleKey : ''
  const titleParams = tab?.titleParams && typeof tab.titleParams === 'object' ? tab.titleParams : undefined
  const translatedTitle = titleKey ? translate(titleKey, titleParams) : (tab.title || '')
  const prefix = typeof tab.titlePrefix === 'string' ? tab.titlePrefix.trim() : ''

  if (!translatedTitle) {
    return prefix
  }

  return prefix ? `${prefix} - ${translatedTitle}` : translatedTitle
}
