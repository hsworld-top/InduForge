import type {
  DeploymentWorkload,
  DeploymentRunEvent,
  HostNode,
  OpsHealth,
  OpsLifecycle,
  OpsNodeRole,
  OpsWorkloadRole,
  ProjectDeployment,
  RuntimeCluster,
} from '@/api/ops.api'

/** 与控制面离线判定保持一致，超过该间隔的心跳不得用于调度。 */
export const NODE_HEARTBEAT_FRESH_MS = 45_000

export const nodeRoleLabel: Record<OpsNodeRole, string> = {
  runtime_linux: 'Linux 运行节点',
  collector_linux: 'Linux 采集节点',
  collector_windows: 'Windows 采集节点',
}

/**
 * 节点页只展示用户能理解的运行位置：运行节点归属资源池，
 * 采集节点的原生进程则直接在当前服务器上运行。
 */
export const nodeLocationPresentation = (
  node: Pick<HostNode, 'role' | 'runtimeClusterId' | 'runtimeClusterName'>,
) => {
  if (node.role !== 'runtime_linux') return '该服务器'
  if (node.runtimeClusterName?.trim()) return node.runtimeClusterName.trim()
  return node.runtimeClusterId ? '运行资源池' : '未分配运行位置'
}

export const nodeServicePresentation = (node: Pick<HostNode, 'role'>) =>
  node.role === 'runtime_linux' ? '运行工程服务' : '运行采集服务'

export const workloadRoleLabel: Record<OpsWorkloadRole, string> = {
  compute: '计算服务',
  alert: '报警服务',
  collector: '采集服务',
}

export const healthPresentation = (health?: OpsHealth | string) => {
  const value = health || 'unknown'
  const map: Record<OpsHealth, { label: string; type: 'success' | 'warning' | 'danger' | 'info' }> =
    {
      healthy: { label: '健康', type: 'success' },
      degraded: { label: '需关注', type: 'warning' },
      unavailable: { label: '不可用', type: 'danger' },
      maintenance: { label: '维护中', type: 'info' },
      unknown: { label: '状态待更新', type: 'info' },
    }
  // 后端滚动升级或脏数据不应导致整页渲染中断，未知值统一降级展示。
  return map[value as OpsHealth] || map.unknown
}

/** 优先展示可确认的节点状态，避免将撤销、离线和普通不可用混为一谈。 */
export const nodeHealthPresentation = (health?: OpsHealth | string, observedStatus?: string) => {
  if (observedStatus === 'revoked') return { label: '已撤销', type: 'danger' as const }
  if (observedStatus === 'offline') return { label: '未连接', type: 'danger' as const }
  return healthPresentation(health)
}

export const lifecyclePresentation = (status?: OpsLifecycle | string) => {
  const map: Record<string, string> = {
    pending: '等待中',
    starting: '启动中',
    running: '运行中',
    stopping: '停止中',
    stopped: '已停止',
    failed: '失败',
    initializing: '初始化中',
    ready: '已就绪',
    degraded: '需关注',
    offline: '已离线',
    online: '在线',
    pending_approval: '待确认',
    active: '已启用',
    revoked: '已撤销',
    maintenance: '维护中',
    disabled: '已禁用',
  }
  return map[String(status || '').toLowerCase()] || '状态待更新'
}

export const lifecycleTagType = (status?: OpsLifecycle | string) => {
  const value = String(status || '').toLowerCase()
  if (['running', 'online', 'ready', 'active'].includes(value)) return 'success' as const
  if (['failed', 'offline', 'revoked'].includes(value)) return 'danger' as const
  if (['pending', 'starting', 'stopping', 'pending_approval', 'degraded'].includes(value))
    return 'warning' as const
  return 'info' as const
}

export const deploymentHasCollectorNode = (deployment: Pick<ProjectDeployment, 'workloads'>) =>
  Boolean(
    deployment.workloads?.some(
      (workload) => workload.role === 'collector' && Boolean(workload.hostNodeId),
    ),
  )

export const deploymentCollectorNodeId = (deployment: Pick<ProjectDeployment, 'workloads'>) =>
  deployment.workloads?.find((workload) => workload.role === 'collector')?.hostNodeId || ''

export const deploymentNeedsAttention = (
  deployment: Pick<ProjectDeployment, 'workloads' | 'health'>,
  collectorNode?: Pick<HostNode, 'health'>,
  runtimeCluster?: Pick<RuntimeCluster, 'health'>,
) =>
  !deploymentHasCollectorNode(deployment) ||
  ['degraded', 'unavailable'].includes(String(collectorNode?.health || '')) ||
  ['degraded', 'unavailable'].includes(String(runtimeCluster?.health || '')) ||
  ['degraded', 'unavailable'].includes(String(deployment.health || ''))

export const deploymentStatePresentation = (
  deployment: Pick<ProjectDeployment, 'workloads' | 'health' | 'observedStatus'>,
  collectorNode?: Pick<HostNode, 'health'>,
  runtimeCluster?: Pick<RuntimeCluster, 'health'>,
) => {
  if (!deploymentHasCollectorNode(deployment)) {
    return { label: '配置不完整', detail: '采集服务不可用', type: 'warning' as const }
  }
  if (runtimeCluster?.health === 'unavailable' && collectorNode?.health === 'unavailable') {
    return { label: '需关注', detail: '运行节点和采集节点未连接', type: 'danger' as const }
  }
  if (runtimeCluster?.health === 'unavailable') {
    return { label: '需关注', detail: '运行节点未连接', type: 'danger' as const }
  }
  if (collectorNode?.health === 'unavailable') {
    return { label: '需关注', detail: '采集节点未连接', type: 'danger' as const }
  }
  if (runtimeCluster?.health === 'degraded' || collectorNode?.health === 'degraded') {
    return { label: '需关注', detail: '部分节点状态异常', type: 'warning' as const }
  }
  if (deployment.observedStatus === 'failed' || deployment.health === 'unavailable') {
    return { label: '运行失败', detail: '请查看操作记录', type: 'danger' as const }
  }
  if (deployment.health === 'degraded') {
    return { label: '需关注', detail: '部分服务状态异常', type: 'warning' as const }
  }
  if (['pending', 'starting'].includes(String(deployment.observedStatus || ''))) {
    return { label: '发布中', detail: '状态更新中', type: 'warning' as const }
  }
  if (deployment.observedStatus === 'running' && deployment.health === 'healthy') {
    return { label: '运行中', detail: '状态正常', type: 'success' as const }
  }
  return {
    label: lifecyclePresentation(deployment.observedStatus),
    detail: healthPresentation(deployment.health).label,
    type: lifecycleTagType(deployment.observedStatus),
  }
}

export const workloadStatePresentation = (
  workload: Pick<
    DeploymentWorkload,
    'role' | 'health' | 'observedStatus' | 'lastError' | 'lastMessage' | 'version'
  >,
  hostHealth?: OpsHealth | string,
) => {
  const locationLabel = workload.role === 'collector' ? '采集节点' : '运行节点'
  if (hostHealth === 'unavailable') {
    return { label: '未连接', detail: `${locationLabel}未连接`, type: 'danger' as const }
  }
  if (hostHealth === 'degraded') {
    return { label: '需关注', detail: `${locationLabel}状态异常`, type: 'warning' as const }
  }
  if (hostHealth === 'maintenance') {
    return { label: '维护中', detail: `${locationLabel}正在维护`, type: 'info' as const }
  }
  if (
    workload.lastError ||
    workload.health === 'unavailable' ||
    workload.observedStatus === 'failed'
  ) {
    return {
      label: '运行异常',
      detail: '服务运行异常，可尝试重新启动；如仍失败，请联系管理员。',
      type: 'danger' as const,
    }
  }
  if (workload.health === 'degraded') {
    return { label: '需关注', detail: '服务状态异常', type: 'warning' as const }
  }
  return {
    label: lifecyclePresentation(workload.observedStatus),
    detail: workload.lastMessage
      ? runEventMessagePresentation(workload.lastMessage)
      : workload.version
        ? `版本 ${workload.version}`
        : '等待状态更新',
    type: lifecycleTagType(workload.observedStatus),
  }
}

export const enrollmentStatusPresentation = (status?: string) => {
  const map: Record<string, string> = {
    created: '等待安装',
    downloaded: '安装包已下载',
    registered: '正在注册',
    claimed: '待确认',
    approved: '接入完成',
    rejected: '已拒绝',
    expired: '接入码已过期',
    failed: '接入失败',
  }
  return map[status || ''] || '状态待更新'
}

export const sortRunEvents = (events: DeploymentRunEvent[] = []) =>
  [...events].sort((left, right) => {
    const leftTime = new Date(left.updatedAt || left.createdAt || 0).getTime()
    const rightTime = new Date(right.updatedAt || right.createdAt || 0).getTime()
    return leftTime - rightTime
  })

export const runEventStagePresentation = (stage?: string) => {
  const map: Record<string, string> = {
    queued: '等待执行',
    dispatched: '正在执行',
    observed: '已完成',
    failed: '执行失败',
  }
  return map[String(stage || '').toLowerCase()] || '状态更新'
}

export const runEventMessagePresentation = (message?: string) => {
  const map: Record<string, string> = {
    'deployment queued': '工程发布等待执行',
    'agent fetched desired workload': '正在应用服务配置',
    'agent observed desired state': '服务状态已更新',
    'agent reported workload failure': '服务运行失败，可尝试重新启动',
    'deploy queued': '发布操作等待执行',
    'start queued': '启动操作等待执行',
    'stop queued': '停止操作等待执行',
    'restart queued': '重启操作等待执行',
  }
  return map[String(message || '').toLowerCase()] || '状态已更新'
}

/**
 * 只有控制面尚未完成、且正处于队列或状态切换阶段的任务才轮询。
 * `running` 是部署成功后的常见终态，不能据此持续轮询。
 */
export const isDeploymentRunActive = (status?: string, completedAt?: string | null) => {
  if (completedAt) return false
  return ['pending', 'queued', 'starting', 'stopping'].includes(String(status || '').toLowerCase())
}

export const isDeploymentPending = (status?: string) =>
  String(status || '').toLowerCase() === 'pending'

/** 首版发布只接受已就绪的单节点运行集群，高可用集群待调度器支持后再开放。 */
export const isDeployableRuntimeCluster = (cluster: RuntimeCluster) =>
  cluster.topology === 'single_node' && cluster.desiredStatus === 'ready'

export const isFreshNodeHeartbeat = (value?: string, now = Date.now()) => {
  if (!value) return false
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) && Math.abs(now - timestamp) <= NODE_HEARTBEAT_FRESH_MS
}

/** 仅将已审批、启用、在线且心跳新鲜的原生采集节点暴露给工程发布。 */
export const isSchedulableCollectorNode = (node: HostNode, now = Date.now()) =>
  ['collector_linux', 'collector_windows'].includes(node.role) &&
  node.approvedAt != null &&
  node.desiredStatus === 'active' &&
  node.observedStatus === 'online' &&
  node.health === 'healthy' &&
  isFreshNodeHeartbeat(node.lastHeartbeatAt, now)
