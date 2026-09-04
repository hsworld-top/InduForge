import { codeWorkspaceApi, type CodeWorkspaceState } from './code-workspace-api'

interface WorkspaceRuntimeEnvironment {
  DEV: boolean
  MODE?: string
  VITE_DESIGNER_AI_URL?: string
  VITE_DESIGNER_CODE_URL?: string
  VITE_DESIGNER_PREVIEW_URL?: string
  VITE_DESIGNER_PREVIEW_CONTROL_URL?: string
  VITE_DESIGNER_WORKSPACE_INSTANCE_ID?: string
}

/**
 * `frontend-linux` 仍属于 Vite 开发态，但浏览器必须通过中心 API 访问实际工程工作区，
 * 不能使用 Mac 的 WSL/127.0.0.1 映射地址。
 */
function usesLocalDevelopmentWorkspace(environment: WorkspaceRuntimeEnvironment): boolean {
  return environment.DEV && environment.MODE !== 'frontend-linux'
}

interface WorkspaceRuntimeDependencies {
  get: typeof codeWorkspaceApi.get
  start: typeof codeWorkspaceApi.start
}

declare const __FRONTEND_WORKSPACE_PROXY_SUFFIX__: string

const WORKSPACE_SERVICE_PATTERN = '(ai|code|preview|preview-control)'
const UUID_PATTERN = '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}'

function escapeRegex(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * frontend-linux 时工作区服务经当前 Vite 端口反向代理。仅接受中心契约中的四类 UUID Host，
 * 避免把后端返回的任意 URL 重写到 localhost。
 */
export function rewriteFrontendLinuxWorkspaceUrl(
  value: string | null,
  localOrigin: string,
  workspaceSuffix: string,
): string | null {
  if (!value || !workspaceSuffix) return value
  try {
    const remote = new URL(value)
    const local = new URL(localOrigin)
    if (
      !['http:', 'https:'].includes(remote.protocol) ||
      local.protocol !== 'http:' ||
      local.hostname !== 'localhost' ||
      !local.port
    ) {
      return value
    }
    const match = remote.hostname.match(
      new RegExp(
        `^${WORKSPACE_SERVICE_PATTERN}-(${UUID_PATTERN})\\.${escapeRegex(workspaceSuffix)}$`,
        'i',
      ),
    )
    if (!match) return value
    const [, service, instanceId] = match
    if (!service || !instanceId) return value
    return `http://${service.toLowerCase()}-${instanceId.toLowerCase()}.localhost:${local.port}${remote.pathname}${remote.search}${remote.hash}`
  } catch {
    return value
  }
}

function currentBrowserOrigin(): string {
  return typeof window === 'undefined' ? '' : window.location.origin
}

export function rewriteFrontendLinuxWorkspaceState(
  workspace: CodeWorkspaceState,
  localOrigin = currentBrowserOrigin(),
  workspaceSuffix = typeof __FRONTEND_WORKSPACE_PROXY_SUFFIX__ === 'string'
    ? __FRONTEND_WORKSPACE_PROXY_SUFFIX__
    : '',
): CodeWorkspaceState {
  const rewrite = (url: string | null) =>
    rewriteFrontendLinuxWorkspaceUrl(url, localOrigin, workspaceSuffix)
  return {
    ...workspace,
    services: {
      ...workspace.services,
      ai: { ...workspace.services.ai, url: rewrite(workspace.services.ai.url) },
      code: { ...workspace.services.code, url: rewrite(workspace.services.code.url) },
      preview: { ...workspace.services.preview, url: rewrite(workspace.services.preview.url) },
      previewControl: {
        ...workspace.services.previewControl,
        url: rewrite(workspace.services.previewControl.url),
      },
    },
  }
}

function parseDevelopmentUrl(value: string | undefined, serviceName: string): string | null {
  const normalized = value?.trim() || ''
  if (!normalized) return null
  try {
    const url = new URL(normalized)
    if (!['http:', 'https:'].includes(url.protocol)) throw new Error('invalid protocol')
    return url.toString()
  } catch {
    throw new Error(`开发环境 ${serviceName} 地址不符合契约`)
  }
}

function serviceState(url: string | null): { url: string | null; hostPort: number | null } {
  if (!url) return { url: null, hostPort: null }
  const port = new URL(url).port
  return { url, hostPort: port ? Number(port) : null }
}

/**
 * 开发模式直接连接本机映射的 WSL 工作空间，不查询或启动工程容器。
 */
export function createDevelopmentWorkspaceState(
  environment: WorkspaceRuntimeEnvironment,
): CodeWorkspaceState {
  const aiUrl = parseDevelopmentUrl(environment.VITE_DESIGNER_AI_URL, 'Pi Web')
  const codeUrl = parseDevelopmentUrl(environment.VITE_DESIGNER_CODE_URL, 'code-server')
  const previewUrl = parseDevelopmentUrl(environment.VITE_DESIGNER_PREVIEW_URL, 'Vite Preview')
  const previewControlUrl = parseDevelopmentUrl(
    environment.VITE_DESIGNER_PREVIEW_CONTROL_URL,
    'Preview Control',
  )
  if (!aiUrl || !codeUrl || !previewUrl || !previewControlUrl) {
    throw new Error('开发环境必须配置 Pi Web、code-server、Vite Preview 和 Preview Control 地址')
  }

  return {
    status: 'running',
    containerName:
      environment.VITE_DESIGNER_WORKSPACE_INSTANCE_ID?.trim() || 'designer-wsl-development',
    onlineUsers: [],
    services: {
      ai: serviceState(aiUrl),
      code: serviceState(codeUrl),
      preview: serviceState(previewUrl),
      previewControl: serviceState(previewControlUrl),
    },
  }
}

export async function resolveWorkspaceState(
  projectId: string,
  autoStart: boolean,
  environment: WorkspaceRuntimeEnvironment = import.meta.env,
  dependencies: WorkspaceRuntimeDependencies = codeWorkspaceApi,
): Promise<CodeWorkspaceState> {
  if (usesLocalDevelopmentWorkspace(environment))
    return createDevelopmentWorkspaceState(environment)

  let workspace = await dependencies.get(projectId)
  if (autoStart && ['missing', 'stopped'].includes(workspace.status)) {
    workspace = await dependencies.start(projectId)
  }
  return environment.MODE === 'frontend-linux'
    ? rewriteFrontendLinuxWorkspaceState(workspace)
    : workspace
}
