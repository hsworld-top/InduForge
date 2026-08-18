export type WorkspaceToolTarget = '2d' | '3d'

export interface WorkspaceOpenRequest {
  type: 'WORKSPACE_OPEN_REQUEST'
  projectId: string
  target: WorkspaceToolTarget
  sceneId: string
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
}

export function isWorkspaceOpenRequest(value: unknown): value is WorkspaceOpenRequest {
  if (!value || typeof value !== 'object') return false
  const request = value as Record<string, unknown>
  return (
    request.type === 'WORKSPACE_OPEN_REQUEST' &&
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

export function workspaceToolTitle(target: WorkspaceToolTarget): string {
  if (target === '2d') return '2D'
  return '3D'
}

export function matchesWorkspaceRequestProject(
  request: WorkspaceOpenRequest,
  projectId: string,
): boolean {
  return Boolean(projectId) && request.projectId === projectId
}
