/**
 * DesignerView 用类型与默认配置（从 DesignerView.vue 抽出）
 */
import type { PageConfig } from '@/editor-core/document/types'
import type { EditorRouteProjectMeta } from '@/stores/editor-store.types'

/** 路由 meta 中的工程信息（由路由守卫注入） */
export interface DesignerRouteProjectMeta {
  project?: EditorRouteProjectMeta
}

/** store.pages 中单页列表项的最小形状 */
export interface DesignerStorePageRow {
  id: string
  name?: string
  type?: string
  parentId?: string | null
  config?: {
    width?: number
    height?: number
    showGrid?: boolean
    enableSnap?: boolean
  }
  rootNodeId?: string
}

export interface DesignerPageTab {
  id: string
  name: string
  isDirty: boolean
}

export const DESIGNER_DEFAULT_PAGE_CONFIG_DIMS: Pick<PageConfig, 'width' | 'height'> = {
  width: 1366,
  height: 768,
}
