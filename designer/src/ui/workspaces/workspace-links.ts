export type DesignerWorkspaceKey = 'code' | '2d' | '3d-scene' | '3d-model'

const workspaceKeys = new Set<DesignerWorkspaceKey>(['code', '2d', '3d-scene', '3d-model'])

export function resolveWorkspaceKey(value: unknown): DesignerWorkspaceKey {
  if (typeof value !== 'string') return 'code'
  return workspaceKeys.has(value as DesignerWorkspaceKey) ? (value as DesignerWorkspaceKey) : 'code'
}

export function buildHtWorkspaceUrl(
  baseUrl: string,
  projectId: string,
  workspace: Exclude<DesignerWorkspaceKey, 'code'>,
): string {
  const entry = workspace === '2d' ? 'index.html' : 'index3d.html'
  const params = new URLSearchParams({ projectId })
  if (workspace !== '2d') {
    params.set('workspace', workspace === '3d-model' ? 'model' : 'scene')
  }
  return `${baseUrl.replace(/\/$/, '')}/ht-editor/${entry}?${params.toString()}`
}

export function sanitizeCodeServerUrl(value: unknown): string | null {
  if (typeof value !== 'string' || value.trim() === '') return null
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:' ? url.toString() : null
  } catch {
    return null
  }
}
