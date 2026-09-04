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

export interface ResolveFrontendProxyOptions {
  requireCenterTarget?: boolean
}

export function resolveFrontendProxyTargets(
  env?: Record<string, string | undefined>,
  options?: ResolveFrontendProxyOptions,
): FrontendProxyTargets

export function createFrontendProxy(
  env?: Record<string, string | undefined>,
  options?: ResolveFrontendProxyOptions,
): Record<string, FrontendProxyOption>
