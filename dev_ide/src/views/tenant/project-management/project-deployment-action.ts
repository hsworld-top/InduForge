export interface ProjectDeploymentPrimary {
  id: string
  environmentId: string
  environmentName: string
  mode: 'development' | 'production'
  desiredStatus: string
  observedStatus: string
  applicationVersionId?: string
  version?: string
  operationInProgress: boolean
  updating?: boolean
  placements: Partial<Record<'base' | 'compute' | 'alarm' | 'collector', string>>
  services?: Array<{ serviceType: string; nodeId: string; nodeName: string }>
  updatedAt?: string
}

export const publishAccessPort = (current?: { accessPort?: number | null } | null) =>
  current?.accessPort ?? 17800

export const publishActionContextMatches = (
  initial: Pick<ProjectDeploymentPrimary, 'mode' | 'environmentId'> | null | undefined,
  mode: 'DEV' | 'RELEASE',
  environmentId: string,
) => Boolean(initial && initial.environmentId === environmentId &&
  initial.mode === (mode === 'RELEASE' ? 'production' : 'development'))

export interface ProjectDeploymentSummary {
  deploymentCount: number
  environmentCount: number
  operationInProgress: boolean
  primarySelection: 'none' | 'unique' | 'multiple'
  primaryDeployment: ProjectDeploymentPrimary | null
  updatedAt?: string | null
}

/** 只从服务端汇总决定入口；多槽位永不猜测“最近的一套”部署。 */
export function projectDeploymentAction(summary?: ProjectDeploymentSummary | null) {
  if (!summary) return { label: '读取部署状态', kind: 'loading', icon: 'Loading', disabled: true }
  if (summary.deploymentCount === 0)
    return { label: '发布工程', kind: 'publish', icon: 'UploadFilled', disabled: false }
  if (summary.deploymentCount > 1 || summary.environmentCount > 1 || !summary.primaryDeployment)
    return { label: '管理部署', kind: 'manage', icon: 'Setting', disabled: false }
  const deployment = summary.primaryDeployment
  if (summary.operationInProgress || deployment.operationInProgress || deployment.observedStatus === 'pending')
    return { label: '查看进度', kind: 'progress', icon: 'Loading', disabled: false }
  if (['failed', 'degraded'].includes(deployment.observedStatus))
    return { label: '处理异常', kind: 'error', icon: 'Warning', disabled: false }
  if (deployment.updating)
    return { label: '查看进度', kind: 'progress', icon: 'Loading', disabled: false }
  if (deployment.observedStatus === 'stopped' && deployment.desiredStatus === 'stopped')
    return { label: '更新并启动', kind: 'resume', icon: 'VideoPlay', disabled: false }
  if (deployment.observedStatus === 'running')
    return {
      label: deployment.mode === 'development' ? '更新开发版' : '发布新版本',
      kind: 'publish',
      icon: 'UploadFilled',
      disabled: false,
    }
  return { label: '管理部署', kind: 'manage', icon: 'Setting', disabled: false }
}
