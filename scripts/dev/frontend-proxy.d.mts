export interface FrontendProxyTargets {
  control: string
  data: string
}

export interface FrontendProxyOption {
  target: string
  changeOrigin: boolean
  secure: boolean
  ws?: boolean
  rewriteWsOrigin?: boolean
}

export function resolveFrontendProxyTargets(
  env?: Record<string, string | undefined>,
): FrontendProxyTargets

export function createFrontendProxy(
  env?: Record<string, string | undefined>,
): Record<string, FrontendProxyOption>
