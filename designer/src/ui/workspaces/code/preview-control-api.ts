export type PreviewProcessStatus = 'starting' | 'running' | 'stopping' | 'stopped' | 'error'
export type PreviewProcessOwnership = 'managed' | 'external' | null
export type WorkspaceInitializationStatus =
  | 'uninitialized'
  | 'initializing'
  | 'initialized'
  | 'error'

export interface PreviewControlState {
  status: PreviewProcessStatus
  ownership: PreviewProcessOwnership
  port: number
  updatedAt: string
  message: string | null
}

export interface WorkspaceInitializationState {
  status: WorkspaceInitializationStatus
  templateId: string | null
  initializedAt: string | null
  message: string | null
}

export interface WorkspaceTemplate {
  id: string
  name: string
  description: string
  framework: 'vue' | 'react'
  language: 'javascript' | 'typescript'
}

export interface WorkspaceTemplateCatalog {
  version: number
  generator: string
  templates: WorkspaceTemplate[]
}

export interface WorkspaceInitializationResult {
  workspace: WorkspaceInitializationState
  preview: PreviewControlState
}

interface PreviewControlEnvelope {
  code: number
  msg: string
  data?: unknown
  reqId?: string
}

function objectValue(value: unknown, errorMessage: string): Record<string, unknown> {
  if (!value || typeof value !== 'object') throw new Error(errorMessage)
  return value as Record<string, unknown>
}

function parsePreviewControlState(value: unknown): PreviewControlState {
  const input = objectValue(value, '预览进程状态不符合契约')
  if (!['starting', 'running', 'stopping', 'stopped', 'error'].includes(String(input.status))) {
    throw new Error('预览进程 status 不符合契约')
  }
  if (![null, 'managed', 'external'].includes(input.ownership as null | string)) {
    throw new Error('预览进程 ownership 不符合契约')
  }
  if (!Number.isInteger(input.port) || Number(input.port) <= 0 || Number(input.port) > 65535) {
    throw new Error('预览进程 port 不符合契约')
  }
  if (typeof input.updatedAt !== 'string' || !input.updatedAt) {
    throw new Error('预览进程 updatedAt 不符合契约')
  }
  if (input.message !== null && typeof input.message !== 'string') {
    throw new Error('预览进程 message 不符合契约')
  }
  return {
    status: input.status as PreviewProcessStatus,
    ownership: input.ownership as PreviewProcessOwnership,
    port: Number(input.port),
    updatedAt: input.updatedAt,
    message: input.message as string | null,
  }
}

function parseWorkspaceState(value: unknown): WorkspaceInitializationState {
  const input = objectValue(value, '工程初始化状态不符合契约')
  if (!['uninitialized', 'initializing', 'initialized', 'error'].includes(String(input.status))) {
    throw new Error('工程初始化 status 不符合契约')
  }
  if (input.templateId !== null && typeof input.templateId !== 'string') {
    throw new Error('工程初始化 templateId 不符合契约')
  }
  if (input.initializedAt !== null && typeof input.initializedAt !== 'string') {
    throw new Error('工程初始化 initializedAt 不符合契约')
  }
  if (input.message !== null && typeof input.message !== 'string') {
    throw new Error('工程初始化 message 不符合契约')
  }
  return {
    status: input.status as WorkspaceInitializationStatus,
    templateId: input.templateId as string | null,
    initializedAt: input.initializedAt as string | null,
    message: input.message as string | null,
  }
}

function parseTemplate(value: unknown): WorkspaceTemplate {
  const input = objectValue(value, '工程模板不符合契约')
  if (
    typeof input.id !== 'string' ||
    typeof input.name !== 'string' ||
    typeof input.description !== 'string' ||
    !['vue', 'react'].includes(String(input.framework)) ||
    !['javascript', 'typescript'].includes(String(input.language))
  ) {
    throw new Error('工程模板字段不符合契约')
  }
  return input as unknown as WorkspaceTemplate
}

function parseTemplateCatalog(value: unknown): WorkspaceTemplateCatalog {
  const input = objectValue(value, '工程模板目录不符合契约')
  if (!Number.isInteger(input.version) || typeof input.generator !== 'string') {
    throw new Error('工程模板目录字段不符合契约')
  }
  if (!Array.isArray(input.templates)) throw new Error('工程模板列表不符合契约')
  return {
    version: Number(input.version),
    generator: input.generator,
    templates: input.templates.map(parseTemplate),
  }
}

function parseInitializationResult(value: unknown): WorkspaceInitializationResult {
  const input = objectValue(value, '工程初始化结果不符合契约')
  return {
    workspace: parseWorkspaceState(input.workspace),
    preview: parsePreviewControlState(input.preview),
  }
}

async function requestPreviewControl(
  controlUrl: string,
  path: string,
  options: RequestInit = {},
): Promise<unknown> {
  const base = new URL(controlUrl)
  const target = new URL(path, base)
  // 正式环境的工作区 URL 携带 30-60 秒一次性票据。根路径 API 解析会
  // 丢弃 base 查询参数，因此首次请求需显式转交票据；网关以 303 换取
  // host-only HttpOnly 会话并从后续 URL 中移除它。
  const workspaceTicket = base.searchParams.get('__if_workspace_ticket')
  if (workspaceTicket) target.searchParams.set('__if_workspace_ticket', workspaceTicket)
  const response = await fetch(target, {
    ...options,
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...options.headers,
    },
  })
  const payload = (await response.json()) as PreviewControlEnvelope
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.msg || `预览控制请求失败 (${response.status})`)
  }
  return payload.data
}

export const previewControlApi = {
  async status(controlUrl: string): Promise<PreviewControlState> {
    return parsePreviewControlState(
      await requestPreviewControl(controlUrl, '/api/v1/preview/status'),
    )
  },
  async start(controlUrl: string): Promise<PreviewControlState> {
    return parsePreviewControlState(
      await requestPreviewControl(controlUrl, '/api/v1/preview/start', { method: 'POST' }),
    )
  },
  async stop(controlUrl: string): Promise<PreviewControlState> {
    return parsePreviewControlState(
      await requestPreviewControl(controlUrl, '/api/v1/preview/stop', { method: 'POST' }),
    )
  },
  async restart(controlUrl: string): Promise<PreviewControlState> {
    return parsePreviewControlState(
      await requestPreviewControl(controlUrl, '/api/v1/preview/restart', { method: 'POST' }),
    )
  },
  async workspaceStatus(controlUrl: string): Promise<WorkspaceInitializationState> {
    return parseWorkspaceState(await requestPreviewControl(controlUrl, '/api/v1/workspace/status'))
  },
  async templates(controlUrl: string): Promise<WorkspaceTemplateCatalog> {
    return parseTemplateCatalog(
      await requestPreviewControl(controlUrl, '/api/v1/workspace/templates'),
    )
  },
  async initialize(controlUrl: string, templateId: string): Promise<WorkspaceInitializationResult> {
    return parseInitializationResult(
      await requestPreviewControl(controlUrl, '/api/v1/workspace/initialize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ templateId }),
      }),
    )
  },
}
