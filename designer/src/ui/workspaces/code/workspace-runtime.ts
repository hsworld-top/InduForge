import { codeWorkspaceApi, type CodeWorkspaceState } from './code-workspace-api'

interface WorkspaceRuntimeEnvironment {
  DEV: boolean
  VITE_DESIGNER_AI_URL?: string
  VITE_DESIGNER_CODE_URL?: string
  VITE_DESIGNER_PREVIEW_URL?: string
  VITE_DESIGNER_PREVIEW_CONTROL_URL?: string
  VITE_DESIGNER_WORKSPACE_INSTANCE_ID?: string
}

interface WorkspaceRuntimeDependencies {
  get: typeof codeWorkspaceApi.get
  start: typeof codeWorkspaceApi.start
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
    throw new Error(
      '开发环境必须配置 Pi Web、code-server、Vite Preview 和 Preview Control 地址',
    )
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
  if (environment.DEV) return createDevelopmentWorkspaceState(environment)

  let workspace = await dependencies.get(projectId)
  if (autoStart && ['missing', 'stopped'].includes(workspace.status)) {
    workspace = await dependencies.start(projectId)
  }
  return workspace
}
