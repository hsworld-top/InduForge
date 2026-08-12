import request, { type ApiResponsePayload } from '@/utils/request'

export type CodeWorkspaceStatus = 'running' | 'stopped' | 'missing' | 'starting' | 'error'

export interface CodeWorkspaceOnlineUser {
  id: string
  name: string
  isCurrent: boolean
}

export interface CodeWorkspaceState {
  status: CodeWorkspaceStatus
  url: string | null
  hostPort: number | null
  containerName: string | null
  onlineUsers: CodeWorkspaceOnlineUser[]
}

type CodeWorkspaceEnvelope = ApiResponsePayload<CodeWorkspaceState>

function unwrapWorkspace(response: CodeWorkspaceEnvelope): CodeWorkspaceState {
  if (!response.data) {
    throw new Error('代码工作区接口未返回 data')
  }
  return response.data
}

function workspacePath(projectId: string): string {
  return `/projects/${encodeURIComponent(projectId)}/code-workspace`
}

export const codeWorkspaceApi = {
  async get(projectId: string): Promise<CodeWorkspaceState> {
    return unwrapWorkspace(await request.get<CodeWorkspaceEnvelope>(workspacePath(projectId)))
  },

  async start(projectId: string): Promise<CodeWorkspaceState> {
    return unwrapWorkspace(
      await request.post<CodeWorkspaceEnvelope>(`${workspacePath(projectId)}/start`),
    )
  },

  async stop(projectId: string): Promise<CodeWorkspaceState> {
    return unwrapWorkspace(
      await request.post<CodeWorkspaceEnvelope>(`${workspacePath(projectId)}/stop`),
    )
  },

  async rebuild(projectId: string): Promise<CodeWorkspaceState> {
    return unwrapWorkspace(
      await request.post<CodeWorkspaceEnvelope>(`${workspacePath(projectId)}/rebuild`),
    )
  },
}
