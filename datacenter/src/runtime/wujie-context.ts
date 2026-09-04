import { setDatacenterLocale } from '@/i18n/runtime'
import { applyDatacenterTheme } from '@/theme/runtime'

export type MicroAppContext = {
  instanceName?: string | undefined
  projectId?: string | undefined
  tenantId?: string | undefined
  authoringEpoch?: string | undefined
  theme?: 'light' | 'dark' | undefined
  locale?: 'zh' | 'en' | undefined
  onRefreshAuth?: (() => Promise<boolean>) | undefined
  onAuthExpired?: (() => void) | undefined
  onAuthoringStale?:
    | ((payload: { projectId: string; currentAuthoringEpoch?: string; action: 'reload' }) => void)
    | undefined
  onStateChange?: ((payload: { title?: string; dirty?: boolean }) => void) | undefined
}

let currentContext: MicroAppContext | null = null
let contextListenerBound = false

const normalizeContext = (value: unknown): MicroAppContext | null => {
  if (!value || typeof value !== 'object') return null
  const input = value as Record<string, unknown>
  const projectId = typeof input.projectId === 'string' ? input.projectId.trim() : ''
  if (!projectId) return null

  return {
    instanceName: typeof input.instanceName === 'string' ? input.instanceName : undefined,
    projectId,
    tenantId: typeof input.tenantId === 'string' ? input.tenantId : undefined,
    authoringEpoch:
      typeof input.authoringEpoch === 'string'
        ? input.authoringEpoch.trim() || undefined
        : undefined,
    theme: input.theme === 'dark' ? 'dark' : 'light',
    locale: input.locale === 'en' ? 'en' : 'zh',
    onRefreshAuth:
      typeof input.onRefreshAuth === 'function'
        ? (input.onRefreshAuth as MicroAppContext['onRefreshAuth'])
        : undefined,
    onAuthExpired:
      typeof input.onAuthExpired === 'function'
        ? (input.onAuthExpired as MicroAppContext['onAuthExpired'])
        : undefined,
    onAuthoringStale:
      typeof input.onAuthoringStale === 'function'
        ? (input.onAuthoringStale as MicroAppContext['onAuthoringStale'])
        : undefined,
    onStateChange:
      typeof input.onStateChange === 'function'
        ? (input.onStateChange as MicroAppContext['onStateChange'])
        : undefined,
  }
}

export const isWujieMicroApp = (): boolean => Boolean(window.__POWERED_BY_WUJIE__)

export const applyMicroAppContext = (value: unknown): MicroAppContext | null => {
  const nextContext = normalizeContext(value)
  if (!nextContext) return currentContext

  currentContext = nextContext
  applyDatacenterTheme(nextContext.theme)
  setDatacenterLocale(nextContext.locale ?? 'zh')
  return currentContext
}

export const initializeWujieContext = (): MicroAppContext | null => {
  if (!isWujieMicroApp()) return null

  applyMicroAppContext(window.$wujie?.props)
  if (!contextListenerBound && window.$wujie?.bus) {
    contextListenerBound = true
    const instanceName = currentContext?.instanceName
    if (instanceName) {
      window.$wujie.bus.$on(`micro-app:${instanceName}:context`, applyMicroAppContext)
    }
  }
  return currentContext
}

export const getMicroAppContext = (): MicroAppContext | null => currentContext
export const getCurrentProjectId = (): string | null => currentContext?.projectId ?? null
export const getCurrentTenantId = (): string | null => currentContext?.tenantId ?? null
export const getCurrentAuthoringEpoch = (): string | null => currentContext?.authoringEpoch ?? null

export const reportMicroAppState = (payload: { title?: string; dirty?: boolean }): void => {
  currentContext?.onStateChange?.(payload)
}
