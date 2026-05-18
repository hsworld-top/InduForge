import type { useEditorStore } from "./editor-store";
/**
 * 编辑器 Pinia store 与壳层/预览共用的轻量类型（不手抄整份 store 形状）。
 *
 * - `EditorStore`：`defineStore` 的 ReturnType；随 editor-store.ts 收紧而变精。
 * - `EditorRouteProjectMeta`：路由 meta 注入，避免各处在 `route.meta as any`。
 * - `EditorStorePreviewShell`：预览视图等仅需的字段窄化。
 * - 不依赖 store 本体的契约见 `./editor-store.contracts.ts`（避免与本文件循环引用）。
 */
import type { EditorReadonlyState, PageLockState } from "@/editor-core/document/types";

export type EditorStore = ReturnType<typeof useEditorStore>;

/** 锁状态快照（与 PageLockManager.getLockState 一致） */
export type EditorLockStateSnapshot = PageLockState;

/** 只读态快照（与 PageLockManager.getReadonlyState 一致） */
export type EditorReadonlySnapshot = EditorReadonlyState;

export type {
  CreatePagePayload,
  EditorClipboardItem,
  EditorClipboardRefValue,
  EditorCreateHomePageResult,
  EditorDeletePageMode,
  EditorEntryConfigPatch,
  EditorLoadPageResult,
  EditorPageDraftEntry,
  EditorPageDraftsMap,
  EditorPageSchemaPayload,
  EditorPageTabItem,
  EditorPageTabState,
  EditorPasteTargetPosition,
  EditorSaveProjectSettingsResult,
  EditorUpdateCurrentPagePatch,
  LockResult,
  PageMutationProjectApi,
  PageMutationStoreContext,
  TogglePageLockResult,
} from "./editor-store.contracts";
export { normalizeEditorPageTabsForState } from "./editor-store.contracts";
export type { CreatePageForStoreResult } from "./editor/page-create-actions";

export type { LoadProjectForStoreResult } from "./editor/project-load-actions";

/** 路由守卫注入的 `meta.project`（DesignerView / PreviewView loadProject） */
export interface EditorRouteProjectMeta {
  id: string;
}

/** PreviewView 等：仅依赖工程加载与预览 runtime 初始化相关 state/action */
export type EditorStorePreviewShell = Pick<
  EditorStore,
  | "loadProject"
  | "projectId"
  | "currentPage"
  | "currentPageId"
  | "doc"
  | "docVersion"
  | "projectVariables"
  | "globalScripts"
  | "projectI18n"
  | "projectRuntimeLocale"
>;

/** Designer 壳层常用窄接口（随实际引用可继续扩展 Pick） */
export type EditorStoreDesignerShell = Pick<
  EditorStore,
  | "loadProject"
  | "projectId"
  | "projectName"
  | "currentPageId"
  | "currentPage"
  | "doc"
  | "selection"
  | "history"
  | "ensureEditable"
  | "isReadonly"
  | "readonlyState"
  | "initEditor"
>;
