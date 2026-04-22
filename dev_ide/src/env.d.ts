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
  export const META_SERVICE: Record<string, string>
  export const META_APP: Record<string, string>
}

declare module '@vueuse/core' {
  export function useBroadcastChannel<T = unknown>(options: { name: string }): {
    post: (value: T) => void
  }
}
