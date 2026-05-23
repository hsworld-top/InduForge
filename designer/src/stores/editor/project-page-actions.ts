/**
 * 工程页面列表刷新（从 editor-store 拆出）
 */

import type { EntryConfigFromApi, PagesRefreshResult } from './pages-sync-types'
import { unwrapApiData } from '@/types/api'
import { normalizePageList } from './normalize-schema'
import { toPageListEntries } from './pages-sync-types'

export interface PagesListApi {
  getPages: (projectId: string) => Promise<unknown>
}

export async function fetchNormalizedPageList(
  projectId: string,
  api: PagesListApi,
): Promise<PagesRefreshResult & { responseData: Record<string, unknown> }> {
  const pagesResponse = await api.getPages(projectId)
  const responseData = unwrapApiData(pagesResponse)
  if (!responseData || typeof responseData !== 'object') {
    throw new Error('页面列表响应无效')
  }
  const rd = responseData as Record<string, unknown>
  const rawList = normalizePageList(rd.pages)
  const pageList = toPageListEntries(rawList)
  const ec = rd.entryConfig
  const entryConfig: EntryConfigFromApi =
    ec && typeof ec === 'object' ? (ec as EntryConfigFromApi) : ({} as EntryConfigFromApi)
  return { pages: pageList, entryConfig, responseData: rd }
}
