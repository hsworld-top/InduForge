export interface PublishDeploymentState {
  mode?: 'development' | 'production'
  desiredStatus?: string
  observedStatus?: string
  applicationVersionId?: string
}

export function projectPublishCta(
  deployment: PublishDeploymentState | null | undefined,
  mode: 'DEV' | 'RELEASE',
  applicationVersionId: string,
) {
  if (!deployment)
    return { label: mode === 'DEV' ? '部署开发版' : '发布并部署', disabled: false }
  if (deployment.observedStatus === 'pending' || ['deleting', 'deletion_requested'].includes(deployment.desiredStatus || ''))
    return { label: '部署状态已变化，请稍后重试', disabled: true }
  if (['failed', 'degraded'].includes(deployment.observedStatus || ''))
    return { label: '部署异常，请先处理', disabled: true }
  const targetMode = mode === 'RELEASE' ? 'production' : 'development'
  const sameMode = deployment.mode === targetMode
  const stopped = deployment.observedStatus === 'stopped'
  if (!sameMode)
    return { label: stopped ? '切换并启动' : mode === 'DEV' ? '切换为开发部署' : '切换为生产部署', disabled: false }
  if (stopped) return { label: '更新并启动', disabled: false }
  if (mode === 'DEV') return { label: '更新开发版', disabled: false }
  if (deployment.applicationVersionId && deployment.applicationVersionId === applicationVersionId)
    return { label: '当前版本已部署', disabled: true }
  return { label: '部署新版本', disabled: false }
}
