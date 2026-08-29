import type {
  DeploymentRunEvent,
  HostNode,
  OpsHealth,
  OpsLifecycle,
  OpsNodeRole,
  OpsWorkloadRole,
  RuntimeCluster,
} from '@/api/ops.api'

/** 与控制面离线判定保持一致，超过该间隔的心跳不得用于调度。 */
export const NODE_HEARTBEAT_FRESH_MS = 45_000

export const nodeRoleLabel: Record<OpsNodeRole, string> = {
  runtime_linux: 'Linux 运行节点',
  collector_linux: 'Linux 采集节点',
  collector_windows: 'Windows 采集节点',
}

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
      unknown: { label: '待确认', type: 'info' },
    }
  // 后端滚动升级或脏数据不应导致整页渲染中断，未知值统一降级展示。
  return map[value as OpsHealth] || map.unknown
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
    maintenance: '维护中',
    disabled: '已禁用',
  }
  return map[String(status || '').toLowerCase()] || '待确认'
}

export const lifecycleTagType = (status?: OpsLifecycle | string) => {
  const value = String(status || '').toLowerCase()
  if (value === 'running') return 'success' as const
  if (value === 'failed') return 'danger' as const
  if (['pending', 'starting', 'stopping'].includes(value)) return 'warning' as const
  return 'info' as const
}

export const enrollmentStatusPresentation = (status?: string) => {
  const map: Record<string, string> = {
    created: '等待安装',
    downloaded: '已下载安装包',
    registered: '已注册，等待确认',
    claimed: '待确认',
    approved: '接入完成',
    rejected: '已拒绝',
    expired: '接入码已过期',
    failed: '接入失败',
  }
  return map[status || ''] || '等待安装'
}

export const sortRunEvents = (events: DeploymentRunEvent[] = []) =>
  [...events].sort((left, right) => {
    const leftTime = new Date(left.updatedAt || left.createdAt || 0).getTime()
    const rightTime = new Date(right.updatedAt || right.createdAt || 0).getTime()
    return leftTime - rightTime
  })

export const runEventStagePresentation = (stage?: string) => {
  const map: Record<string, string> = {
    queued: '任务已提交',
    dispatched: '命令已下发',
    observed: '目标状态已生效',
    failed: '执行失败',
  }
  return map[String(stage || '').toLowerCase()] || stage || '状态更新'
}

export const runEventMessagePresentation = (message?: string) => {
  const map: Record<string, string> = {
    'deployment queued': '工程部署已进入执行队列',
    'agent fetched desired workload': '节点已领取工作负载目标',
    'agent observed desired state': '节点已上报目标状态',
    'agent reported workload failure': '节点上报工作负载执行失败',
    'deploy queued': '部署任务已进入执行队列',
    'start queued': '启动任务已进入执行队列',
    'stop queued': '停止任务已进入执行队列',
    'restart queued': '重启任务已进入执行队列',
  }
  return map[String(message || '').toLowerCase()] || message || '状态已更新'
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
