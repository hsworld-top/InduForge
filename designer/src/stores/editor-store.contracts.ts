/**
 * 编辑器 store 与壳层共用的契约类型（不依赖 ./editor-store，避免与 editor-store.types 循环引用）。
 *
 * 页面 mutation、草稿、标签页、锁等对外形状集中在此，便于壳层 import 单一路径。
 */

import type { ExportedPagePayload } from '@/editor-core/document/Serializer'

import type {
  ComponentNode,
  EntryConfig,
  GraphicNode,
  LockResult,
  PageNode,
} from '@/editor-core/document/types'

/** 与 Serializer.exportPage / updatePage 载荷一致 */
export type EditorPageSchemaPayload = ExportedPagePayload

export type { CreatePagePayload } from './editor/page-create-actions'
export type {
  PageMutationProjectApi,
  PageMutationStoreContext,
} from './editor/page-mutation-actions'

/** 再导出：acquirePageLock 等与 PageLockManager 对齐 */
export type { LockResult }

/** 删除页面 API / store 统一模式 */
export type EditorDeletePageMode = 'single' | 'folder-only' | 'cascade'

/**
 * 底部/壳层页面标签项（与 DesignerView PageTab、setPageTabState 写入形状对齐）
 */
export interface EditorPageTabItem {
  id: string
  name?: string
  isDirty?: boolean
  [key: string]: unknown
}

/** store.pageTabState 快照 */
export interface EditorPageTabState {
  tabs: EditorPageTabItem[]
  activeId: string
}

/**
 * 将壳层传入的标签列表规范为可写入 store 的快照（浅拷贝每项）。
 */
export function normalizeEditorPageTabsForState(tabs: EditorPageTabItem[]): EditorPageTabItem[] {
  if (!Array.isArray(tabs)) return []
  return tabs.map((item) =>
    item && typeof item === 'object'
      ? ({ ...(item as Record<string, unknown>) } as EditorPageTabItem)
      : ({} as EditorPageTabItem),
  )
}

/**
 * togglePageLock 联合返回（含获取锁失败时附带 action: acquire）
 */
export type TogglePageLockResult =
  | { success: true; action: 'release' }
  | { success: true; action: 'acquire' }
  | { success: false; reason: 'error'; error: Error }
  | (Extract<LockResult, { success: false }> & { action: 'acquire' })

/** 更新当前页 patch（与 UpdatePageCommand / updateCurrentPage 一致） */
export type EditorUpdateCurrentPagePatch = Partial<PageNode>

/** 单页草稿内容（与 Serializer.exportPage 输出一致） */
export type EditorPageDraftEntry = EditorPageSchemaPayload

/** 页面草稿槽位（内存态，键为 pageId） */
export type EditorPageDraftsMap = Record<string, EditorPageDraftEntry>

/** 内存剪贴板单条（与 copyNodes / pasteNodes 一致） */
export type EditorClipboardItem =
  | { kind: 'node'; data: ComponentNode }
  | { kind: 'graphic'; data: GraphicNode }

/** 剪贴板 ref 取值 */
export type EditorClipboardRefValue = EditorClipboardItem[] | null

/** 粘贴到画布时的目标坐标（设计器坐标系） */
export interface EditorPasteTargetPosition {
  x: number
  y: number
}

/*
 * persistEntry / saveCurrentPage：现均为 Promise<void>；失败时 saveCurrentPage 抛 Error。
 * 结构化 { ok, error } 若引入，应在此文件补对应联合类型并与 store 同步。
 */

/** 更新入口配置（UpdateEntryCommand / store.entryConfig） */
export type EditorEntryConfigPatch = Partial<EntryConfig> & Record<string, unknown>

/** loadPage 结果 */
export type EditorLoadPageResult = { ok: true } | { ok: false; error: Error }

/** createHomePage 结果 */
export type EditorCreateHomePageResult = { ok: true; pageId?: string } | { ok: false; error: Error }

/** saveProjectSettings 结果（与 project-settings-actions 一致） */
export interface EditorSaveProjectSettingsResult {
  ok: boolean
  error?: Error
}
