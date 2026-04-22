type StatusLike = string | null | undefined

interface DeployLike {
  status?: StatusLike
  errorMessage?: string | null
  lastError?: string | null
  message?: string | null
}

interface NodeLike {
  deployments?: DeployLike[] | null
}

export const getNodeStatusType = (status: StatusLike): string =>
  status === 'online' ? 'success' : status === 'offline' ? 'info' : 'danger'

export const getNodeStatusLabel = (status: StatusLike): string => {
  const map: Record<string, string> = { online: '在线', offline: '离线', error: '监控异常' }
  return (status && map[status]) || status || ''
}

export const getDeployStatusType = (status: StatusLike): string => {
  const map: Record<string, string> = {
    running: 'success',
    stopped: 'info',
    deploying: 'warning',
    error: 'danger',
    failed: 'danger',
    pending: 'warning',
  }
  return (status && map[status]) || 'info'
}

export const getDeployLabel = (status: StatusLike): string => {
  const map: Record<string, string> = {
    running: '运行中',
    stopped: '已停止',
    deploying: '部署中',
    error: '故障',
    failed: '失败',
    pending: '等待中',
  }
  return (status && map[status]) || status || ''
}

export const isFailedDeploy = (deploy: DeployLike | null | undefined): boolean =>
  ['error', 'failed'].includes(deploy?.status || '')

export const getDeployFailureReason = (deploy: DeployLike | null | undefined): string =>
  deploy?.errorMessage || deploy?.lastError || deploy?.message || '未提供失败原因'

export const buildDeployStatusSummary = (
  nodes: Array<NodeLike | null | undefined> = []
): { running: number; deploying: number; stopped: number; failed: number } => {
  const summary = { running: 0, deploying: 0, stopped: 0, failed: 0 }
  nodes.forEach((node) => {
    const deployments = Array.isArray(node?.deployments) ? node.deployments : []
    deployments.forEach((deploy) => {
      const status = deploy?.status
      if (status === 'running') summary.running++
      else if (status === 'deploying' || status === 'pending') summary.deploying++
      else if (status === 'stopped') summary.stopped++
      else if (status === 'error' || status === 'failed') summary.failed++
    })
  })
  return summary
}

export const getProgressColor = (percentage: number): string => {
  if (percentage < 60) return 'var(--if-color-success-500)'
  if (percentage < 85) return 'var(--if-color-warning-500)'
  return 'var(--if-color-danger-500)'
}
