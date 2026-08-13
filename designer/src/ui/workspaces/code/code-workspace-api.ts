import request, { type ApiResponsePayload } from '@/utils/request'

export type CodeWorkspaceStatus = 'running' | 'stopped' | 'missing' | 'starting' | 'error'

export interface CodeWorkspaceOnlineUser {
  id: string
  name: string
  isCurrent: boolean
}

export interface WorkspaceServiceState {
  url: string | null
  hostPort: number | null
}

export interface CodeWorkspaceState {
  status: CodeWorkspaceStatus
  containerName: string | null
  onlineUsers: CodeWorkspaceOnlineUser[]
  services: {
    ai: WorkspaceServiceState
    code: WorkspaceServiceState
    preview: WorkspaceServiceState
    previewControl: WorkspaceServiceState
  }
}

type CodeWorkspaceEnvelope = ApiResponsePayload<unknown>

const workspaceStatuses = new Set<CodeWorkspaceStatus>([
  'running',
  'stopped',
  'missing',
  'starting',
  'error',
])

function parseHostPort(value: unknown, serviceName: string): number | null {
  if (value === null) return null
  if (typeof value === 'number' && Number.isInteger(value) && value > 0 && value <= 65535) {
    return value
  }
  throw new Error(`代码工作区 services.${serviceName}.hostPort 不符合契约`)
}

function parseUrl(value: unknown, serviceName: string): string | null {
  if (value === null) return null
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error(`代码工作区 services.${serviceName}.url 不符合契约`)
  }
  try {
    const url = new URL(value)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') throw new Error('invalid protocol')
    return url.toString()
  } catch {
    throw new Error(`代码工作区 services.${serviceName}.url 不符合契约`)
  }
}

function parseService(value: unknown, serviceName: string): WorkspaceServiceState {
  if (!value || typeof value !== 'object') {
    throw new Error(`代码工作区缺少 services.${serviceName}`)
  }
  const service = value as Record<string, unknown>
  return {
    url: parseUrl(service.url, serviceName),
    hostPort: parseHostPort(service.hostPort, serviceName),
  }
}

/**
 * 严格解析工程工作空间状态。开发阶段只接受当前三服务契约，禁止字段兜底或地址推导。
 */
export function parseCodeWorkspaceState(value: unknown): CodeWorkspaceState {
  if (!value || typeof value !== 'object') throw new Error('代码工作区 data 不符合契约')
  const input = value as Record<string, unknown>
  if (typeof input.status !== 'string' || !workspaceStatuses.has(input.status as CodeWorkspaceStatus)) {
    throw new Error('代码工作区 status 不符合契约')
  }
  if (input.containerName !== null && typeof input.containerName !== 'string') {
    throw new Error('代码工作区 containerName 不符合契约')
  }
  if (!Array.isArray(input.onlineUsers)) throw new Error('代码工作区 onlineUsers 不符合契约')
  const onlineUsers = input.onlineUsers.map((user) => {
    if (
      !user ||
      typeof user !== 'object' ||
      typeof (user as CodeWorkspaceOnlineUser).id !== 'string' ||
      typeof (user as CodeWorkspaceOnlineUser).name !== 'string' ||
      typeof (user as CodeWorkspaceOnlineUser).isCurrent !== 'boolean'
    ) {
      throw new Error('代码工作区 onlineUsers 项不符合契约')
    }
    return user as CodeWorkspaceOnlineUser
  })
  if (!input.services || typeof input.services !== 'object') {
    throw new Error('代码工作区缺少 services')
  }
  const services = input.services as Record<string, unknown>

  return {
    status: input.status as CodeWorkspaceStatus,
    containerName: input.containerName as string | null,
    onlineUsers,
    services: {
      ai: parseService(services.ai, 'ai'),
      code: parseService(services.code, 'code'),
      preview: parseService(services.preview, 'preview'),
      previewControl: parseService(services.previewControl, 'previewControl'),
    },
  }
}

function unwrapWorkspace(response: CodeWorkspaceEnvelope): CodeWorkspaceState {
  if (!response.data) {
    throw new Error('代码工作区接口未返回 data')
  }
  return parseCodeWorkspaceState(response.data)
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
