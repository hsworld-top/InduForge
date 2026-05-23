/**
 * 工程页面列表与入口配置刷新 — 与 Pinia / API 对齐的共享类型
 */

import type { PageListEntry } from './page-crud-actions'
import type { EntryConfig } from '@/editor-core/document/types'

/**
 * 列表接口返回的 entry 片段（Partial 内核字段 + 允许 API 扩展键）
 */
export type EntryConfigFromApi = Partial<EntryConfig> & Record<string, unknown>

export interface PagesRefreshResult {
  pages: PageListEntry[]
  entryConfig: EntryConfigFromApi
}

/**
 * 在 normalizePageList 之后做一次列表项形状校验，避免在各 consumer 重复断言
 */
export function toPageListEntries(list: unknown[]): PageListEntry[] {
  for (let i = 0; i < list.length; i++) {
    const item = list[i]
    if (!item || typeof item !== 'object') {
      throw new Error(`页面列表第 ${i} 项无效：期望对象`)
    }
    const id = (item as { id?: unknown }).id
    if (typeof id !== 'string' || !id) {
      throw new Error(`页面列表第 ${i} 项缺少有效 id`)
    }
  }
  return list as PageListEntry[]
}
