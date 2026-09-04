export interface FrontendLinuxTargetValidation {
  valid: boolean
  target?: string
  message?: string
}

export function readFrontendLinuxProxyTarget(content: string): string
export function validateFrontendLinuxProxyTarget(value: string): FrontendLinuxTargetValidation
