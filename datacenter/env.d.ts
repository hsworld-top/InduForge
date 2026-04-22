/// <reference types="vite/client" />
/// <reference types="unplugin-icons/types/vue" />

declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<
    Record<string, unknown>,
    Record<string, unknown>,
    unknown
  >;
  export default component;
}

declare const __DATACENTER_DEBUG_ROUTE_ENABLED__: boolean;
declare const __VITE_API_URL__: string | undefined;
declare const __VITE_DATA_SERVICE_URL__: string | undefined;
