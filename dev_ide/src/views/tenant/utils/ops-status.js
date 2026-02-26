export const getNodeStatusType = (status) =>
  status === 'online' ? 'success' : status === 'offline' ? 'info' : 'danger'

export const getNodeStatusLabel = (status) => {
  const map = { online: '在线', offline: '离线', error: '监控异常' }
  return map[status] || status
}

export const getDeployStatusType = (status) => {
  const map = { running: 'success', stopped: 'info', deploying: 'warning', error: 'danger', failed: 'danger', pending: 'warning' }
  return map[status] || 'info'
}

export const getDeployLabel = (status) => {
  const map = {
    running: '运行中',
    stopped: '已停止',
    deploying: '部署中',
    error: '故障',
    failed: '失败',
    pending: '等待中',
  }
  return map[status] || status
}

export const isFailedDeploy = (deploy) => ['error', 'failed'].includes(deploy?.status)

export const getDeployFailureReason = (deploy) =>
  deploy?.errorMessage || deploy?.lastError || deploy?.message || '未提供失败原因'

export const buildDeployStatusSummary = (nodes = []) => {
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

export const getProgressColor = (percentage) => {
  if (percentage < 60) return 'var(--if-color-success-500)'
  if (percentage < 85) return 'var(--if-color-warning-500)'
  return 'var(--if-color-danger-500)'
}
