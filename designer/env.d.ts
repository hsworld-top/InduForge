/// <reference types="vite/client" />

/** 部分构建脚本注入的全局（与 getApiBase 一致） */
declare const __VITE_API_URL__: string | undefined;

/** DataService 仅声明预览运行时用到的方法，完整实现见 src/data */
declare module "@/data" {
  export class DataService {
    constructor(options: { baseUrl?: string });
    connect(url: string): Promise<unknown> | undefined;
    destroy(): void;
    subscribe(path: string, cb: (payload: { value?: unknown }) => void): void;
    getValue(path: string): unknown;
  }
}

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
