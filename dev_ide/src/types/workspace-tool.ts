export type WorkspaceToolTarget = '2d' | '3d'

export interface WorkspaceOpenRequest {
  type: 'WORKSPACE_OPEN_REQUEST'
  projectId: string
  target: WorkspaceToolTarget
}

export interface WorkspaceToolProject {
  id: string
  name?: string
  tenantId?: string
}

export interface WorkspaceToolTabProps {
  target: WorkspaceToolTarget
  project: WorkspaceToolProject
}

export function isWorkspaceOpenRequest(value: unknown): value is WorkspaceOpenRequest {
  if (!value || typeof value !== 'object') return false
  const request = value as Record<string, unknown>
  return (
    request.type === 'WORKSPACE_OPEN_REQUEST' &&
    typeof request.projectId === 'string' &&
    ['2d', '3d'].includes(String(request.target))
  )
}

export function workspaceToolTabKey(projectId: string, target: WorkspaceToolTarget): string {
  return `${projectId}:${target}`
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

export function buildWorkspaceToolUrl(target: WorkspaceToolTarget, projectId: string): string {
  const params = new URLSearchParams({ projectId })
  if (target === '3d') params.set('workspace', 'scene')
  const entry = target === '2d' ? 'index.html' : 'index3d.html'
  return `/designer/ht-editor/${entry}?${params.toString()}`
}
