export interface WorkspaceProxySuffixValidation {
  valid: boolean
  suffix?: string
  message?: string
}

export interface WorkspaceProxyConfig {
  localPort: number
  suffix: string
  target: string
}

export interface WorkspaceRequestRoute {
  localOrigin: string
  remoteHost: string
  remoteOrigin: string
}

export function validateFrontendWorkspaceProxySuffix(value: string): WorkspaceProxySuffixValidation
export function resolveLocalWorkspaceRequest(
  host: string | undefined,
  localPort: number,
  suffix: string,
  target: string,
): WorkspaceRequestRoute | null
export function resolveFrontendWorkspaceProxyConfig(
  env?: Record<string, string | undefined>,
  options?: { requireConfig?: boolean; localPort?: number },
): WorkspaceProxyConfig | null
export function createFrontendWorkspaceProxy(
  env?: Record<string, string | undefined>,
  options?: { requireConfig?: boolean; localPort?: number },
): Record<string, any>
