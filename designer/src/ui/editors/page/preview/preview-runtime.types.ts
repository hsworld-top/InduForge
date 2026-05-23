/**
 * 预览运行时对外协议（initPreviewRuntime / getPreviewRuntime / 页面级生命周期）
 * 与 PreviewView、NodeRenderer/usePreview、previewRuntime 实现约定一致。
 */

import type { ComputedRef, Ref } from 'vue'

/** 脚本表单项（custom / variableChanges / timers 等 { items } 段落） */
export interface PreviewScriptItem {
  name?: string
  variable?: string
  code?: unknown
  enabled?: boolean
  interval?: number
  time?: number
  params?: unknown
  args?: unknown
  [key: string]: unknown
}

/** 仅接受规范形态 { items: T[] }；items 在 store 侧常为 unknown[]，运行时由 itemsFromScriptSection 收窄 */
export interface ScriptSectionWithItems {
  items?: unknown[]
  [key: string]: unknown
}

/** 全局脚本 system.startup / system.shutdown */
export interface PreviewScriptSystemBlock {
  startup?: { code?: unknown; [key: string]: unknown }
  shutdown?: { code?: unknown; [key: string]: unknown }
  [key: string]: unknown
}

/** 工程级 globalScripts（与 normalizeGlobalScripts / initPreviewRuntime 消费一致） */
export interface PreviewGlobalScriptsShape {
  custom?: ScriptSectionWithItems
  variableChanges?: ScriptSectionWithItems
  timers?: ScriptSectionWithItems
  system?: PreviewScriptSystemBlock
  [key: string]: unknown
}

/** 页面生命周期：脚本段落 + onMounted/onUnmounted 等处理器数组 */
export type PreviewLifecycleHandler = string | PreviewScriptItem

export type PreviewPageLifecycleShape = {
  variableChanges?: ScriptSectionWithItems
  timers?: ScriptSectionWithItems
  onMounted?: PreviewLifecycleHandler[]
  onUnmounted?: PreviewLifecycleHandler[]
} & Record<string, unknown>

/** 画布注册到预览运行时的组件 ref（最小字典形） */
export type PreviewComponentRefInfo = Record<string, unknown>

/**
 * initPreviewRuntime 入参（PreviewView onMounted、其他入口需与此对齐）
 */
export interface PreviewRuntimeInitOptions {
  projectId?: string | null
  projectVariables?: Record<string, unknown>
  globalScripts?: PreviewGlobalScriptsShape
  pageLifecycle?: PreviewPageLifecycleShape
  pageVariables?: Record<string, unknown>
  /** 传给页面生命周期脚本（variableChanges 等）的页面标识 */
  pageId?: string | null
}

/**
 * initPreviewRuntime 返回值；单例由 getPreviewRuntime 读取
 */
export interface PreviewRuntimeHandle {
  globals?: object
  customScripts?: Record<string, (...args: unknown[]) => unknown>
  runCode?: (code: string, event?: unknown, context?: unknown, pageId?: string | null) => unknown
  start?: () => void | Promise<void>
  stop?: () => void | Promise<void>
  registerComponentRef?: (pageIdValue: string, name: string, refInfo: unknown) => void
  unregisterComponentRef?: (pageIdValue: string, name: string, refInfo: unknown) => void
}

/** 组件脚本 Location 桩（读为 0，写则排队 setStyle） */
export interface PreviewComponentStubLocation {
  X: number
  Y: number
}

/** 组件脚本 Size 桩（读为 0，写则排队 setStyle） */
export interface PreviewComponentStubSize {
  Width: number
  Height: number
}

/**
 * 未注册真实 ref 前的组件脚本占位对象（脚本可写属性/调方法，真实挂载后由 applyPendingCalls 回放）
 */
export type PreviewComponentStubApi = Record<string, unknown> & {
  readonly Name: string
  readonly Comment: string
  readonly Location: PreviewComponentStubLocation
  readonly Size: PreviewComponentStubSize
  Visible: boolean
  Enable: boolean
  Caption: string
  Image: string
}

/** NodeRenderer / 画布侧传入 usePreview 的数据中心 API（最小形状） */
export interface UsePreviewDatacenterApi {
  getConnections: (...args: unknown[]) => Promise<unknown>
  getQueries: (...args: unknown[]) => Promise<unknown>
  executeQuery: (...args: unknown[]) => Promise<unknown>
  getDatapointValues: (...args: unknown[]) => Promise<unknown>
}

export interface UsePreviewNodeShape {
  id?: string
  type?: string
  props?: Record<string, unknown>
  events?: Record<string, unknown[]>
}

/**
 * usePreview(deps) 依赖（与 NodeRenderer 传入字段一致）
 */
export interface UsePreviewDeps {
  node: Ref<UsePreviewNodeShape | null | undefined>
  doc: Ref<unknown>
  currentPage: Ref<{ name?: string; id?: string } | null | undefined>
  projectVariables: Ref<Record<string, unknown> | null | undefined>
  projectId: Ref<string | null | undefined>
  docVersion: Ref<unknown>
  globalScripts: Ref<PreviewGlobalScriptsShape | null | undefined>
  datacenterApi: UsePreviewDatacenterApi
  buildRefInfo: () => unknown
  readonly: Ref<boolean>
  detailConfigText?: ComputedRef<string | undefined>
  resolveMenuConfigFromContent?: (code: string) => unknown
  applyMenuDslConfig?: (config: unknown) => void
}

export interface UsePreviewReturn {
  runPreviewScript: (eventName: string, event: unknown) => Promise<unknown>
  buildPreviewGlobals: () => object
  buildPreviewCustomScripts: (
    globals: object,
  ) => Record<string, (...args: unknown[]) => Promise<unknown>>
  runDetailConfigScript: (code: string) => Promise<void>
  scheduleDetailConfig: (force?: boolean) => void
  isRunningDetailConfig: () => boolean
}
