export type WorkspaceToolTarget = '2d' | '3d'

export interface WorkspaceOpenRequest {
  type: 'WORKSPACE_OPEN_REQUEST'
  projectId: string
  target: WorkspaceToolTarget
  sceneId: string
  sceneName: string
}

export interface WorkspaceCloseRequest {
  type: 'WORKSPACE_CLOSE_REQUEST'
  projectId: string
  target: WorkspaceToolTarget
  sceneId: string
}

export interface WorkspaceSceneCommittedEvent {
  projectId: string
  target: WorkspaceToolTarget
  sceneId: string
  revision: number
}

export interface WorkspaceToolProject {
  id: string
  name?: string
  tenantId?: string
}

export interface WorkspaceToolTabProps {
  target: WorkspaceToolTarget
  project: WorkspaceToolProject
  sceneId: string
  sceneName: string
}

export function isWorkspaceOpenRequest(value: unknown): value is WorkspaceOpenRequest {
  if (!value || typeof value !== 'object') return false
  const request = value as Record<string, unknown>
  return (
    request.type === 'WORKSPACE_OPEN_REQUEST' &&
    typeof request.projectId === 'string' &&
    typeof request.sceneId === 'string' &&
    request.sceneId.trim().length > 0 &&
    typeof request.sceneName === 'string' &&
    request.sceneName.trim().length > 0 &&
    ['2d', '3d'].includes(String(request.target))
  )
}

export function isWorkspaceCloseRequest(value: unknown): value is WorkspaceCloseRequest {
  if (!value || typeof value !== 'object') return false
  const request = value as Record<string, unknown>
  return (
    request.type === 'WORKSPACE_CLOSE_REQUEST' &&
    typeof request.projectId === 'string' &&
    typeof request.sceneId === 'string' &&
    request.sceneId.trim().length > 0 &&
    ['2d', '3d'].includes(String(request.target))
  )
}

export function workspaceToolTabKey(
  projectId: string,
  target: WorkspaceToolTarget,
  sceneId: string,
): string {
  return `${projectId}:${target}:${sceneId}`
}

export function workspaceToolTitle(target: WorkspaceToolTarget, sceneName: string): string {
  return `${sceneName.trim()} · ${target === '2d' ? '2D' : '3D'}`
}

export function matchesWorkspaceRequestProject(
  request: WorkspaceOpenRequest | WorkspaceCloseRequest,
  projectId: string,
): boolean {
  return Boolean(projectId) && request.projectId === projectId
}
