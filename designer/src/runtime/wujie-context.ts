import { getEditorUiStore } from '@/stores/editor-ui-store'

export type MicroAppContext = {
  instanceName?: string | undefined
  projectId?: string | undefined
  projectName?: string | undefined
  tenantId?: string | undefined
  theme?: 'light' | 'dark' | undefined
  locale?: 'zh' | 'en' | undefined
  onStateChange?: ((payload: { title?: string; dirty?: boolean }) => void) | undefined
  onOpenWorkspace?: ((request: WorkspaceOpenRequest) => void) | undefined
}

export type WorkspaceOpenTarget = '2d' | '3d'

export interface WorkspaceOpenRequest {
  type: 'WORKSPACE_OPEN_REQUEST'
  projectId: string
  target: WorkspaceOpenTarget
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
    projectName: typeof input.projectName === 'string' ? input.projectName.trim() : undefined,
    tenantId: typeof input.tenantId === 'string' ? input.tenantId : undefined,
    theme: input.theme === 'dark' ? 'dark' : 'light',
    locale: input.locale === 'en' ? 'en' : 'zh',
    onStateChange:
      typeof input.onStateChange === 'function'
        ? (input.onStateChange as MicroAppContext['onStateChange'])
        : undefined,
    onOpenWorkspace:
      typeof input.onOpenWorkspace === 'function'
        ? (input.onOpenWorkspace as MicroAppContext['onOpenWorkspace'])
        : undefined,
  }
}

export const isWujieMicroApp = (): boolean => Boolean(window.__POWERED_BY_WUJIE__)

export const applyMicroAppContext = (value: unknown): MicroAppContext | null => {
  const nextContext = normalizeContext(value)
  if (!nextContext) return currentContext

  currentContext = nextContext
  getEditorUiStore().initFromRuntime(nextContext)
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
      window.$wujie.bus.$emit(`micro-app:${instanceName}:context-ready`)
    }
  }
  return currentContext
}

export const getMicroAppContext = (): MicroAppContext | null => currentContext
export const getCurrentProjectId = (): string | null => currentContext?.projectId ?? null
export const getCurrentProjectName = (): string | null => currentContext?.projectName ?? null
export const getCurrentTenantId = (): string | null => currentContext?.tenantId ?? null

export const reportMicroAppState = (payload: { title?: string; dirty?: boolean }): void => {
  currentContext?.onStateChange?.(payload)
}

export const requestWorkspaceOpen = (target: WorkspaceOpenTarget): boolean => {
  if (
    !['2d', '3d'].includes(target) ||
    !currentContext?.projectId ||
    !currentContext.onOpenWorkspace
  ) {
    return false
  }
  currentContext.onOpenWorkspace({
    type: 'WORKSPACE_OPEN_REQUEST',
    projectId: currentContext.projectId,
    target,
  })
  return true
}
