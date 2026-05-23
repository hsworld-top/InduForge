/**
 * 加载工程首屏（从 editor-store 拆出）
 */

import type { Ref, ShallowRef } from 'vue'
import type { PageListEntry } from './page-crud-actions'
import type { PageContentApi, ResolveProjectSchemaFn } from './page-load-save-actions'
import type { PagesRefreshResult } from './pages-sync-types'
import type { DocumentModel } from '@/editor-core/document/DocumentModel.ts'
import type { ProjectSchema } from '@/editor-core/document/types'
import { applyEntryPatchIfPresent } from './entry-config-helpers'
import { fetchResolvedProjectSchemaForPage } from './page-load-save-actions'

export interface LoadProjectStoreContext {
  projectId: Ref<string>
  projectLoadRequestSerial: Ref<number>
  projectLoadInvalidatorProjectIds: Ref<Record<number, string>>
  isLoading: Ref<boolean>
  error: Ref<string>
  doc: ShallowRef<DocumentModel | null>
  currentPageId: Ref<string>
  loadProjectSettings: (canCommit?: () => boolean) => Promise<void>
  loadProjectRuntimeRoles: (projectId: string) => void | Promise<unknown>
  releasePageLock: () => Promise<void>
  refreshPages: (canCommit?: () => boolean) => Promise<PagesRefreshResult>
  createHomePage: (
    pid: string,
    canCommit?: () => boolean,
    shouldRollbackStaleRemote?: () => boolean,
  ) => Promise<{ ok: boolean; pageId?: string; error?: Error }>
  initEditor: (schema: ProjectSchema) => void
  createBaseSchema: (projectId: string) => ProjectSchema
  applyRuntimeRoleCodesToSchema: (schema: ProjectSchema) => ProjectSchema
  resolveLandingPageId: (
    pageList: PageListEntry[],
    entryConfig: { homePageId?: string } | null | undefined,
  ) => string | null
  projectApi: PageContentApi
  resolveProjectSchema: ResolveProjectSchemaFn
}

export async function loadProjectForStore(
  id: string,
  ctx: LoadProjectStoreContext,
): Promise<{ ok: boolean; error?: Error }> {
  const previousRequestSerial = ctx.projectLoadRequestSerial.value
  if (
    previousRequestSerial > 0 &&
    !ctx.projectLoadInvalidatorProjectIds.value[previousRequestSerial]
  ) {
    ctx.projectLoadInvalidatorProjectIds.value = {
      ...ctx.projectLoadInvalidatorProjectIds.value,
      [previousRequestSerial]: id,
    }
  }

  const requestSerial = previousRequestSerial + 1
  ctx.projectLoadRequestSerial.value = requestSerial
  ctx.projectId.value = id || ''
  ctx.isLoading.value = true
  ctx.error.value = ''
  ctx.loadProjectRuntimeRoles(id)
  const canCommit = () =>
    ctx.projectLoadRequestSerial.value === requestSerial && ctx.projectId.value === id
  const shouldRollbackStaleRemote = () => {
    const invalidatorProjectId = ctx.projectLoadInvalidatorProjectIds.value[requestSerial]
    return Boolean(invalidatorProjectId) && invalidatorProjectId !== id
  }

  try {
    await ctx.loadProjectSettings(canCommit)
    if (!canCommit()) {
      return { ok: true }
    }
    await ctx.releasePageLock()
    if (!canCommit()) {
      return { ok: true }
    }
    const { pages: pageList, entryConfig: entryConfigResp } = await ctx.refreshPages(canCommit)
    if (!canCommit()) {
      return { ok: true }
    }

    if (!pageList.length) {
      const homePageResult = await ctx.createHomePage(id, canCommit, shouldRollbackStaleRemote)
      if (!homePageResult.ok) {
        if (canCommit()) {
          ctx.initEditor(ctx.createBaseSchema(id))
        }
      }
      return { ok: homePageResult.ok }
    }

    const targetPageId = ctx.resolveLandingPageId(pageList, entryConfigResp)

    if (!targetPageId) {
      const homePageResult = await ctx.createHomePage(id, canCommit, shouldRollbackStaleRemote)
      if (!homePageResult.ok) {
        if (canCommit()) {
          ctx.initEditor(ctx.createBaseSchema(id))
        }
      }
      return { ok: homePageResult.ok }
    }

    const nextSchema = await fetchResolvedProjectSchemaForPage(
      ctx.projectApi,
      id,
      targetPageId,
      ctx.resolveProjectSchema,
    )
    if (!canCommit()) {
      return { ok: true }
    }
    ctx.initEditor(ctx.applyRuntimeRoleCodesToSchema(nextSchema))

    if (!canCommit()) {
      return { ok: true }
    }
    applyEntryPatchIfPresent(ctx.doc.value, entryConfigResp)

    ctx.currentPageId.value = targetPageId
    return { ok: true }
  } catch (cause) {
    if (!canCommit()) {
      return { ok: true }
    }
    const nextError = cause instanceof Error ? cause : new Error('加载工程失败')
    ctx.error.value = nextError.message
    ctx.initEditor(ctx.createBaseSchema(id))
    return { ok: false, error: nextError }
  } finally {
    if (canCommit()) {
      ctx.isLoading.value = false
    }
  }
}

export type LoadProjectForStoreResult = Awaited<ReturnType<typeof loadProjectForStore>>
