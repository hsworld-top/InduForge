export interface ProjectAuthoringContext {
  projectId: string
  authoringEpoch: string
}

export interface AuthoringStaleDetail {
  projectId: string
  currentAuthoringEpoch?: string
  action: 'reload'
}

export const AUTHORING_STALE_EVENT = 'induforge:authoring-stale'

const contexts = new Map<string, ProjectAuthoringContext>()

export const getCachedAuthoringContext = (projectId: string): ProjectAuthoringContext | null =>
  contexts.get(String(projectId)) ?? null

export const cacheAuthoringContext = (context: ProjectAuthoringContext): void => {
  const projectId = String(context.projectId || '').trim()
  const authoringEpoch = String(context.authoringEpoch || '').trim()
  if (!projectId || !authoringEpoch) return
  contexts.set(projectId, { projectId, authoringEpoch })
}

export const invalidateAuthoringContext = (projectId: string): void => {
  contexts.delete(String(projectId))
}

export const notifyAuthoringStale = (detail: AuthoringStaleDetail): void => {
  invalidateAuthoringContext(detail.projectId)
  window.dispatchEvent(new CustomEvent<AuthoringStaleDetail>(AUTHORING_STALE_EVENT, { detail }))
}
