/**
 * 页面加载 / 保存与 API 交互（从 editor-store 拆出）
 */

import type { ProjectSchema } from '@/editor-core/document/types'
import { unwrapApiData } from '@/types/api'

export interface PageContentApi {
  getPage: (projectId: string, pageId: string) => Promise<unknown>
}

export type ResolveProjectSchemaFn = (
  payload: unknown,
  projectId: string,
  fallbackPageId: string,
) => ProjectSchema

/**
 * 拉取单页并解析为工程级 Schema（unwrap + resolveProjectSchema）
 */
export async function fetchResolvedProjectSchemaForPage(
  api: PageContentApi,
  projectId: string,
  pageId: string,
  resolveProjectSchema: ResolveProjectSchemaFn,
): Promise<ProjectSchema> {
  const pageResponse = await api.getPage(projectId, pageId)
  const pagePayload = unwrapApiData(pageResponse)
  if (pagePayload == null || typeof pagePayload !== 'object') {
    throw new Error('页面数据无效')
  }
  return resolveProjectSchema(pagePayload, projectId, pageId)
}

/**
 * 将当前文档中的页面级变量合并进待保存的 export payload
 */
export function mergePageVariablesIntoPayload(
  payload: Record<string, unknown>,
  pageId: string,
  pageVars: unknown,
): void {
  if (!pageVars || typeof pageVars !== 'object') return
  const existingVars =
    payload.vars && typeof payload.vars === 'object'
      ? (payload.vars as Record<string, unknown>)
      : {}
  const existingPages =
    existingVars.pages && typeof existingVars.pages === 'object'
      ? (existingVars.pages as Record<string, unknown>)
      : {}
  payload.vars = {
    ...existingVars,
    pages: {
      ...existingPages,
      [pageId]: pageVars,
    },
  }
}
