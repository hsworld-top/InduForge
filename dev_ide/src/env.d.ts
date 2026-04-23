/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, never>, Record<string, never>, unknown>
  export default component
}

declare module '@opentiny/tiny-engine' {
  export const HttpService: {
    apis: {
      setOptions: (options: unknown) => void
    }
  }
}

declare module '@opentiny/tiny-engine-utils' {
  export const constants: {
    BROADCAST_CHANNEL: {
      Notify: string
    }
  }
}

declare module '@opentiny/tiny-engine-meta-register' {
  interface MetaServiceMap {
    Http: string
  }

  interface MetaAppMap {
    Layout: string
    Page: string
    State: string
    OutlineTree: string
    Materials: string
    Schema: string
    Help: string
    Save: string
    GenerateCode: string
    Lang: string
    ViewSetting: string
    Preview: string
  }

  export const META_SERVICE: MetaServiceMap
  export const META_APP: MetaAppMap
}

declare module '@vueuse/core' {
  export function useBroadcastChannel<T = unknown>(options: {
    name: string
  }): {
    post: (value: T) => void
  }
}
