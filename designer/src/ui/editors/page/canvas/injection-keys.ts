/**
 * 画布子树 provide/inject 的 InjectionKey，避免字符串 key 在 Volar 下无法推断类型。
 */
import type { InjectionKey, Ref } from 'vue'
import type { RuntimeAccessContext } from './runtime-access'

export interface PreviewRenderBoundsContext {
  width: Ref<number>
  height: Ref<number>
}

/** 设计画布缩放比（用于拖放落点、坐标换算） */
export const canvasZoomKey: InjectionKey<Ref<number>> = Symbol('designer.canvasZoom')

/** 设计画布吸附开关（用于节点拖拽对齐吸附） */
export const canvasSnapEnabledKey: InjectionKey<Ref<boolean>> = Symbol('designer.canvasSnapEnabled')

export const runtimeAccessContextKey: InjectionKey<RuntimeAccessContext> = Symbol(
  'designer.runtimeAccessContext',
)

/** 预览/运行态页面边界：根节点下完全落在边界外的工作区素材不参与渲染。 */
export const previewRenderBoundsKey: InjectionKey<PreviewRenderBoundsContext> = Symbol(
  'designer.previewRenderBounds',
)
