/**
 * 画布子树 provide/inject 的 InjectionKey，避免字符串 key 在 Volar 下无法推断类型。
 */
import type { InjectionKey, Ref } from "vue";

/** 设计画布缩放比（用于拖放落点、坐标换算） */
export const canvasZoomKey: InjectionKey<Ref<number>> = Symbol("designer.canvasZoom");
