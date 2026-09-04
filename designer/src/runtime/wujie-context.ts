import { getEditorUiStore } from '@/stores/editor-ui-store'

export type MicroAppContext = {
  instanceName?: string | undefined
  projectId?: string | undefined
  projectName?: string | undefined
  tenantId?: string | undefined
  authoringEpoch?: string | undefined
  theme?: 'light' | 'dark' | undefined
  locale?: 'zh' | 'en' | undefined
  onRefreshAuth?: (() => Promise<boolean>) | undefined
  onAuthExpired?: (() => void) | undefined
  onAuthoringStale?: ((payload: {
    projectId: string
    currentAuthoringEpoch?: string
  }) => void) | undefined
  onStateChange?: ((payload: { title?: string; dirty?: boolean }) => void) | undefined
  onOpenWorkspace?: ((request: WorkspaceOpenRequest) => void) | undefined
  onCloseWorkspace?: ((request: WorkspaceCloseRequest) => void) | undefined
}

export type WorkspaceOpenTarget = '2d' | '3d'

export interface WorkspaceOpenRequest {
  type: 'WORKSPACE_OPEN_REQUEST'
  projectId: string
  target: WorkspaceOpenTarget
  sceneId: string
  sceneName: string
}

export interface WorkspaceCloseRequest {
  type: 'WORKSPACE_CLOSE_REQUEST'
  projectId: string
  target: WorkspaceOpenTarget
  sceneId: string
}

export interface SceneCommittedEvent {
  projectId: string
  target: WorkspaceOpenTarget
  sceneId: string
  revision: number
}

let currentContext: MicroAppContext | null = null
let contextListenerBound = false
const sceneCommittedListeners = new Set<(event: SceneCommittedEvent) => void>()

const dispatchSceneCommitted = (value: unknown): void => {
  if (!value || typeof value !== 'object') return
  const event = value as Record<string, unknown>
  if (
    event.projectId !== currentContext?.projectId ||
    !['2d', '3d'].includes(String(event.target)) ||
    typeof event.sceneId !== 'string' ||
    !event.sceneId.trim() ||
    typeof event.revision !== 'number'
  ) {
    return
  }
  sceneCommittedListeners.forEach((listener) => listener(event as unknown as SceneCommittedEvent))
}

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
    authoringEpoch:
      typeof input.authoringEpoch === 'string' && input.authoringEpoch.trim()
        ? input.authoringEpoch.trim()
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
    onOpenWorkspace:
      typeof input.onOpenWorkspace === 'function'
        ? (input.onOpenWorkspace as MicroAppContext['onOpenWorkspace'])
        : undefined,
    onCloseWorkspace:
      typeof input.onCloseWorkspace === 'function'
        ? (input.onCloseWorkspace as MicroAppContext['onCloseWorkspace'])
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
      window.$wujie.bus.$on(`micro-app:${instanceName}:scene-committed`, dispatchSceneCommitted)
      window.$wujie.bus.$emit(`micro-app:${instanceName}:context-ready`)
    }
  }
  return currentContext
}

export const getMicroAppContext = (): MicroAppContext | null => currentContext
export const getCurrentProjectId = (): string | null => currentContext?.projectId ?? null
export const getCurrentProjectName = (): string | null => currentContext?.projectName ?? null
export const getCurrentTenantId = (): string | null => currentContext?.tenantId ?? null
export const getCurrentAuthoringEpoch = (): string | null =>
  currentContext?.authoringEpoch ?? null

export const reportAuthoringStale = (payload: {
  projectId: string
  currentAuthoringEpoch?: string
}): void => {
  currentContext?.onAuthoringStale?.(payload)
}

export const subscribeSceneCommitted = (
  listener: (event: SceneCommittedEvent) => void,
): (() => void) => {
  sceneCommittedListeners.add(listener)
  return () => sceneCommittedListeners.delete(listener)
}

export const reportMicroAppState = (payload: { title?: string; dirty?: boolean }): void => {
  currentContext?.onStateChange?.(payload)
}

export const requestWorkspaceOpen = (
  target: WorkspaceOpenTarget,
  sceneId: string,
  sceneName: string,
): boolean => {
  if (
    !['2d', '3d'].includes(target) ||
    !currentContext?.projectId ||
    !sceneId.trim() ||
    !sceneName.trim() ||
    !currentContext.onOpenWorkspace
  ) {
    return false
  }
  currentContext.onOpenWorkspace({
    type: 'WORKSPACE_OPEN_REQUEST',
    projectId: currentContext.projectId,
    target,
    sceneId: sceneId.trim(),
    sceneName: sceneName.trim(),
  })
  return true
}

export const requestWorkspaceClose = (target: WorkspaceOpenTarget, sceneId: string): boolean => {
  if (
    !['2d', '3d'].includes(target) ||
    !currentContext?.projectId ||
    !sceneId.trim() ||
    !currentContext.onCloseWorkspace
  ) {
    return false
  }
  currentContext.onCloseWorkspace({
    type: 'WORKSPACE_CLOSE_REQUEST',
    projectId: currentContext.projectId,
    target,
    sceneId: sceneId.trim(),
  })
  return true
}
