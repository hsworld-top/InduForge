/**
 * 画布子树 provide/inject 的 InjectionKey，避免字符串 key 在 Volar 下无法推断类型。
 */
import type { InjectionKey, Ref } from "vue";
import type { RuntimeAccessContext } from "./runtime-access";

/** 设计画布缩放比（用于拖放落点、坐标换算） */
export const canvasZoomKey: InjectionKey<Ref<number>> = Symbol("designer.canvasZoom");

/** 设计画布吸附开关（用于节点拖拽对齐吸附） */
export const canvasSnapEnabledKey: InjectionKey<Ref<boolean>> = Symbol(
  "designer.canvasSnapEnabled",
);

export const runtimeAccessContextKey: InjectionKey<RuntimeAccessContext> = Symbol(
  "designer.runtimeAccessContext",
);
