interface Window {
  __POWERED_BY_WUJIE__?: boolean
  $wujie?: {
    props?: unknown
    bus?: {
      $on(event: string, handler: (payload: unknown) => void): void
      $emit(event: string, payload?: unknown): void
    }
  }
}
