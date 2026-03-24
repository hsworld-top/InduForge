/// <reference types="vite/client" />

declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<object, object, unknown>;
  export default component;
}

declare module "~icons/*" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<object, object, unknown>;
  export default component;
}

/** 尚未迁 TS 的模块，供 shell 等严格 SFC 引用 */
declare module "@/stores/editor-store" {
  import type { StoreGeneric } from "pinia";
  export function useEditorStore(): StoreGeneric;
}

declare module "@/ui/editors/page/canvas" {
  import type { DefineComponent } from "vue";
  export const CanvasContainer: DefineComponent<object, object, unknown>;
}

declare module "@/ui/editors/page/panels/left" {
  import type { DefineComponent } from "vue";
  export const OutlineTree: DefineComponent<object, object, unknown>;
  export const MaterialPanel: DefineComponent<object, object, unknown>;
  export const DataPanel: DefineComponent<object, object, unknown>;
}

declare module "@/ui/shared/panels" {
  import type { DefineComponent } from "vue";
  export const PageTree: DefineComponent<object, object, unknown>;
  export const I18nPanel: DefineComponent<object, object, unknown>;
  export const ScriptVarsPanel: DefineComponent<object, object, unknown>;
  export const RolePanel: DefineComponent<object, object, unknown>;
  export const VariablesPanel: DefineComponent<object, object, unknown>;
}

declare module "@/ui/editors/page/panels/right" {
  import type { DefineComponent } from "vue";
  export const PropertyPanel: DefineComponent<object, object, unknown>;
  export const AdvancedPanel: DefineComponent<object, object, unknown>;
}
