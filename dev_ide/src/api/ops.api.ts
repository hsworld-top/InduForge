import request, { type RawResponseConfig, type RequestConfig } from '@/utils/request'

export type OpsId = string

export type OpsHealth = 'healthy' | 'degraded' | 'unavailable' | 'maintenance' | 'unknown'
export type OpsLifecycle = 'starting' | 'running' | 'stopping' | 'stopped' | 'failed' | 'pending'
export type OpsNodeRole = 'runtime_linux' | 'collector_linux' | 'collector_windows'
export type OpsWorkloadRole = 'compute' | 'alert' | 'collector'
export type OpsWorkloadAction = 'start' | 'stop' | 'restart'

export interface OpsApiEnvelope<T> {
  code?: number
  msg?: string
  data: T
  reqId?: string
}

export interface RuntimeCluster {
  id: OpsId
  name: string
  code?: string
  topology?: 'single_node' | 'high_availability'
  description?: string
  health?: OpsHealth
  desiredStatus?: 'ready' | 'maintenance' | 'disabled'
  observedStatus?: 'pending' | 'initializing' | 'ready' | 'degraded' | 'offline'
  controllerStatus?: 'pending' | 'ready' | 'offline'
  nodeCount?: number
  onlineNodeCount?: number
  version?: string
  createdAt?: string
  updatedAt?: string
}

export interface HostNodeMetrics {
  cpuPercent?: number
  memoryPercent?: number
  diskPercent?: number
}

export interface HostNode {
  id: OpsId
  name: string
  displayName?: string
  role: OpsNodeRole
  runtimeClusterId?: OpsId | null
  runtimeClusterName?: string | null
  health?: OpsHealth
  desiredStatus?: OpsLifecycle | 'active' | 'revoked'
  observedStatus?: OpsLifecycle | 'online' | 'offline' | 'degraded' | 'pending_approval' | 'revoked'
  ipAddress?: string
  os?: string
  architecture?: string
  lastHeartbeatAt?: string
  /** 控制面审批完成时间；未审批节点不能作为采集调度目标。 */
  approvedAt?: string
  agentVersion?: string
  k3sStatus?: string
  collectorStatus?: string
  metrics?: HostNodeMetrics
  createdAt?: string
  updatedAt?: string
}

export interface NodePackage {
  id: OpsId
  name?: string
  role: OpsNodeRole
  version?: string
  platform?: 'linux' | 'windows'
  architecture?: string
  fileName?: string
  available?: boolean
  size?: number
}

export interface NodeEnrollment {
  id: OpsId
  role: OpsNodeRole
  displayName?: string
  runtimeClusterId?: OpsId | null
  packageId?: OpsId
  packageName?: string
  enrollmentCode?: string
  /** 接入码只在创建结果中短暂可见，后续查询应为空。 */
  expiresAt?: string
  status:
    | 'created'
    | 'downloaded'
    | 'registered'
    | 'claimed'
    | 'approved'
    | 'rejected'
    | 'expired'
    | 'failed'
  reportedHostName?: string
  machineFingerprint?: string
  ipAddress?: string
  node?: HostNode | null
  createdAt?: string
  updatedAt?: string
}

export interface DeploymentWorkload {
  role: OpsWorkloadRole
  displayName?: string
  health?: OpsHealth
  desiredStatus?: OpsLifecycle
  observedStatus?: OpsLifecycle
  version?: string
  lastError?: string | null
  lastMessage?: string | null
  updatedAt?: string
}

export interface ProjectDeployment {
  id: OpsId
  projectId: OpsId
  projectName: string
  displayName?: string
  mode: 'development' | 'production'
  deploymentMode?: 'development' | 'production'
  runtimeClusterId?: OpsId | null
  runtimeClusterName?: string | null
  hostNodeId?: OpsId | null
  version?: string
  health?: OpsHealth
  desiredStatus?: OpsLifecycle
  observedStatus?: OpsLifecycle
  progress?: number
  updatedAt?: string
  workloads?: DeploymentWorkload[]
  runId?: OpsId
  latestRunId?: OpsId
}

export interface DeploymentRunEvent {
  id?: OpsId
  stage?: string
  status?: OpsLifecycle | string
  message?: string
  createdAt?: string
  updatedAt?: string
}

export interface DeploymentRun {
  id: OpsId
  deploymentId?: OpsId
  status?: OpsLifecycle | string
  observedStatus?: OpsLifecycle | string
  progress?: number
  startedAt?: string
  /** 终态任务会由后端写入完成时间，轮询必须以它为准停止。 */
  completedAt?: string | null
  events?: DeploymentRunEvent[]
}

type EnrollmentCreateResult = { enrollment: NodeEnrollment; code?: string }
type DeploymentCreateResult = { deployment: ProjectDeployment; run?: DeploymentRun }

export interface ListParams {
  page?: number
  pageSize?: number
  keyword?: string
}

export interface ListResult<T> {
  items: T[]
  total: number
}

const unpack = <T>(payload: OpsApiEnvelope<T> | T): T =>
  payload && typeof payload === 'object' && 'data' in payload
    ? (payload as OpsApiEnvelope<T>).data
    : (payload as T)

const normalizeList = <T>(
  payload: OpsApiEnvelope<ListResult<T> | T[]> | ListResult<T> | T[],
): ListResult<T> => {
  const data = unpack(payload)
  if (Array.isArray(data)) return { items: data, total: data.length }
  return {
    items: Array.isArray(data?.items) ? data.items : [],
    total: typeof data?.total === 'number' ? data.total : data?.items?.length || 0,
  }
}

const normalizeHostNode = (node: HostNode): HostNode => ({
  ...node,
  name: node.displayName || node.name || String(node.id),
})

const normalizeDeployment = (deployment: ProjectDeployment): ProjectDeployment => ({
  ...deployment,
  projectName: deployment.displayName || deployment.projectName || String(deployment.projectId),
  mode: deployment.deploymentMode || deployment.mode || 'development',
  runId: deployment.latestRunId || deployment.runId,
})

const normalizeRun = (run: DeploymentRun): DeploymentRun => ({
  ...run,
  status: run.status || run.observedStatus,
})

/** 运维页统一在业务层展示错误，避免 HTTP 拦截器与页面 catch 重复弹窗。 */
const opsRequestConfig: RequestConfig = { skipErrorToast: true }

export const opsAPI = {
  async listRuntimeClusters(params: ListParams = {}) {
    return normalizeList<RuntimeCluster>(
      await request.get('/ops/runtime-clusters', { params, ...opsRequestConfig }),
    )
  },
  async getRuntimeCluster(id: OpsId) {
    return unpack<RuntimeCluster>(
      await request.get(`/ops/runtime-clusters/${id}`, opsRequestConfig),
    )
  },
  async createRuntimeCluster(
    payload: Pick<RuntimeCluster, 'name' | 'code' | 'description' | 'topology'>,
  ) {
    return unpack<RuntimeCluster>(
      await request.post('/ops/runtime-clusters', payload, opsRequestConfig),
    )
  },
  async listEnrollments(params: ListParams = {}) {
    return normalizeList<NodeEnrollment>(
      await request.get('/ops/node-enrollments', { params, ...opsRequestConfig }),
    )
  },
  async getEnrollment(id: OpsId) {
    return unpack<NodeEnrollment>(
      await request.get(`/ops/node-enrollments/${id}`, opsRequestConfig),
    )
  },
  async createEnrollment(payload: {
    role: OpsNodeRole
    displayName: string
    ttlMinutes: number
    runtimeClusterId?: OpsId | null
    packageId?: OpsId
  }) {
    const result = unpack<EnrollmentCreateResult>(
      await request.post('/ops/node-enrollments', payload, opsRequestConfig),
    )
    return {
      ...result.enrollment,
      enrollmentCode: result.code,
      packageId: payload.packageId,
    }
  },
  async approveEnrollment(id: OpsId) {
    return unpack<NodeEnrollment>(
      await request.post(`/ops/node-enrollments/${id}/approve`, undefined, opsRequestConfig),
    )
  },
  async rejectEnrollment(id: OpsId) {
    return unpack<NodeEnrollment>(
      await request.post(`/ops/node-enrollments/${id}/reject`, undefined, opsRequestConfig),
    )
  },
  async listHostNodes(params: ListParams = {}) {
    const result = normalizeList<HostNode>(
      await request.get('/ops/host-nodes', { params, ...opsRequestConfig }),
    )
    return { ...result, items: result.items.map(normalizeHostNode) }
  },
  async getHostNode(id: OpsId) {
    return normalizeHostNode(
      unpack<HostNode>(await request.get(`/ops/host-nodes/${id}`, opsRequestConfig)),
    )
  },
  async listNodePackages(params: ListParams = {}) {
    const result = normalizeList<NodePackage>(
      await request.get('/ops/node-packages', { params, ...opsRequestConfig }),
    )
    return {
      ...result,
      items: result.items.map((item) => ({
        ...item,
        name: item.name || item.fileName || item.role,
      })),
    }
  },
  downloadNodePackage(packageId: OpsId) {
    return request.get(`/ops/node-packages/${packageId}/download`, {
      responseType: 'blob',
      returnRawResponse: true,
      ...opsRequestConfig,
    } as RawResponseConfig)
  },
  async listProjectDeployments(params: ListParams = {}) {
    const result = normalizeList<ProjectDeployment>(
      await request.get('/ops/project-deployments', { params, ...opsRequestConfig }),
    )
    return { ...result, items: result.items.map(normalizeDeployment) }
  },
  async getProjectDeployment(id: OpsId) {
    return normalizeDeployment(
      unpack<ProjectDeployment>(
        await request.get(`/ops/project-deployments/${id}`, opsRequestConfig),
      ),
    )
  },
  async createProjectDeployment(payload: {
    projectId: OpsId
    deploymentMode: 'development' | 'production'
    runtimeClusterId: OpsId
    workloads: Array<{ role: OpsWorkloadRole; hostNodeId?: OpsId | null }>
  }) {
    const result = unpack<DeploymentCreateResult>(
      await request.post('/ops/project-deployments', payload, opsRequestConfig),
    )
    return normalizeDeployment({ ...result.deployment, runId: result.run?.id })
  },
  async runWorkloadAction(id: OpsId, role: OpsWorkloadRole, action: OpsWorkloadAction) {
    const result = unpack<DeploymentCreateResult>(
      await request.post(
        `/ops/project-deployments/${id}/workloads/${role}/${action}`,
        undefined,
        opsRequestConfig,
      ),
    )
    return result.run ? normalizeRun(result.run) : result.run
  },
  async getDeploymentRun(id: OpsId) {
    return normalizeRun(
      unpack<DeploymentRun>(await request.get(`/ops/deployment-runs/${id}`, opsRequestConfig)),
    )
  },
  async listDeploymentRunEvents(id: OpsId) {
    return normalizeList<DeploymentRunEvent>(
      await request.get(`/ops/deployment-runs/${id}/events`, opsRequestConfig),
    )
  },
}

export { normalizeList, unpack }
