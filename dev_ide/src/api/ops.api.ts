import request, { type RawResponseConfig, type RequestConfig } from '@/utils/request'

export type OpsId = string
export type OpsPlatform = 'linux' | 'windows'
export type OpsCapability = 'project_entry' | 'data_runtime' | 'collector'
export type OpsServiceType = DeploymentEngine
export type DeploymentEngine = 'base' | 'compute' | 'alarm' | 'collector'
export type DeploymentMode = 'development' | 'production'
export type OpsDeploymentAction = 'start' | 'stop' | 'restart' | 'redeploy'
export type OpsHealth = 'healthy' | 'degraded' | 'unavailable' | 'unknown'
export type OpsLifecycle = 'running' | 'stopped' | 'failed' | 'pending' | 'online' | 'offline'

export interface OpsApiEnvelope<T> {
  code?: number
  msg?: string
  data: T
  reqId?: string
}
export interface OpsNode {
  id: OpsId
  name: string
  displayName?: string
  hostname?: string
  ipAddress?: string
  platform: OpsPlatform
  architecture?: string
  capabilities: OpsCapability[]
  desiredStatus?: string
  observedStatus?: string
  lastHeartbeatAt?: string
  approvedAt?: string
  agentVersion?: string
  resourceSummary?: Record<string, unknown>
  assignedDeploymentId?: OpsId
  assignedProjectId?: OpsId
  assignedProjectName?: string
  environmentId?: OpsId
  environmentName?: string
  environmentNames?: string[]
  environmentCount?: number
  nodeKind?: 'center' | 'worker'
  clusterId?: OpsId
  clusterRole?: 'server' | 'agent'
  clusterStatus?: 'pending' | 'starting' | 'ready' | 'failed' | 'not-installed' | 'removing'
  clusterDesiredAction?: 'active' | 'removing'
  clusterMessage?: string
  clusterDesiredGeneration?: number
  clusterObservedGeneration?: number
  clusterObservedAt?: string
}
export type RuntimeEnvironmentStatus = 'available' | 'attention' | 'uninitialized' | 'deleting'
export interface RuntimeEnvironment {
  id: OpsId
  name: string
  code: string
  isDefault: boolean
  status: RuntimeEnvironmentStatus
  desiredStatus: 'active' | 'maintenance' | 'deleting'
  nodeCount: number
  onlineNodeCount: number
  foundationTotal: number
  foundationHealthy: number
  projectCount: number
  runningDeploymentCount: number
  recentChange: string
  recentAt?: string | null
  recentBy: string
  createdAt?: string
  updatedAt?: string
}
export interface RuntimeEnvironmentOverview {
  environmentId: string
  snapshotAt: string
  thresholds: {
    freshnessSeconds: number
    capacityAttentionPercent: number
    capacityCriticalPercent: number
  }
  deployments: {
    total: number
    running: number
    stopped: number
    failed: number
    pending: number
    stale: number
    unknown: number
  }
  nodes: {
    total: number
    online: number
    offline: number
    fault: number
    staleMetrics: number
    unknownMetrics: number
    capacityAttention: number
    capacityCritical: number
    maxDiskUsagePercent: number | null
  }
  riskNodes: Array<{
    id: string
    name: string
    ipAddress: string
    nodeKind: string
    observedStatus: string
    clusterStatus: string
    lastHeartbeatAt: string | null
    metricsStale: boolean
    cpuPercent: number | null
    memoryPercent: number | null
    diskPercent: number | null
    health:
      | 'offline'
      | 'failed'
      | 'stale'
      | 'pending'
      | 'capacity_critical'
      | 'capacity_attention'
      | 'unknown'
      | 'healthy'
  }>
}
export interface RuntimeEnvironmentEvent {
  id: OpsId
  environmentId: OpsId
  eventType: string
  name: string
  target: string
  result: 'success' | 'failed'
  message?: string
  operator: string
  createdAt: string
}
export type FoundationServiceType =
  | 'if_realtime'
  | 'if_history'
  | 'if_timeseries'
  | 'if_message'
  | 'if_object'
  | 'nats_jetstream'
  | 'nginx'
  | 'traefik'
export interface RuntimeEnvironmentService {
  id: OpsId
  environmentId: OpsId
  nodeId: OpsId
  nodeName: string
  serviceType: FoundationServiceType
  desiredStatus: string
  observedStatus: 'pending' | 'running' | 'stopped' | 'degraded' | 'failed'
  lastMessage?: string
  desiredGeneration: number
  observedGeneration: number
  operation?: 'apply' | 'migrate'
  observedAt?: string
  /** 服务端按摘要相同口径判定观测是否过期，不使用浏览器时钟推断健康。 */
  observedStale: boolean
  updatedAt?: string
}
export interface NodePackage {
  id: OpsId
  name?: string
  platform: OpsPlatform
  architecture?: string
  version?: string
  fileName?: string
  available?: boolean
  size?: number
}
export interface NodeEnrollment {
  id: OpsId
  platform: OpsPlatform
  capabilities: OpsCapability[]
  displayName?: string
  status: string
  enrollmentCode?: string
  expiresAt?: string
  reportedHostName?: string
  machineFingerprint?: string
  ipAddress?: string
  node?: OpsNode | null
  createdAt?: string
  updatedAt?: string
}
export interface DeploymentService {
  id?: OpsId
  nodeId?: OpsId
  nodeName?: string
  serviceType: OpsServiceType
  desiredStatus?: string
  observedStatus?: string
  lastMessage?: string
  endpoint?: string | null
  desiredGeneration?: number
  observedGeneration?: number
  observedAt?: string
  updatedAt?: string
}
export interface ProjectDeployment {
  id: OpsId
  projectId: OpsId
  projectName: string
  environmentId: OpsId
  environmentName?: string
  mode?: 'development' | 'production'
  applicationVersionId?: OpsId
  /** 仅兼容旧单节点列表响应；新部署统一由 environmentId + services 表达。 */
  nodeId?: OpsId
  nodeName?: string
  accessPort: number
  version?: string
  desiredStatus?: string
  observedStatus?: string
  health?: OpsHealth
  progress?: number
  entryStatus?: string
  accessUrl?: string | null
  accessAvailable?: boolean
  /** 运维单槽摘要返回的显示节点名，不包含集群实现细节。 */
  nodeNames?: string[]
  services?: DeploymentService[]
  latestRunId?: OpsId
  latestRunOperation?: 'deploy' | 'start' | 'stop' | 'restart' | 'redeploy' | 'delete'
  updatedAt?: string
}
export interface ReleaseArtifactDescriptor {
  file?: string
  checksum?: string
}
export interface ApplicationVersionManifest {
  schemaVersion?: string
  artifacts?: {
    client?: ReleaseArtifactDescriptor
    runtime?: ReleaseArtifactDescriptor
    collector?: ReleaseArtifactDescriptor
  }
  [key: string]: unknown
}
export interface ApplicationVersion {
  id: OpsId
  projectId: OpsId
  version: string
  name?: string
  status?: string
  mode?: string
  artifactHash?: string
  manifest?: ApplicationVersionManifest | null
  capabilities?: DeploymentEngine[]
  completedAt?: string
  createdAt?: string
  restorable?: boolean
  restoreUnavailableReason?: string
  authoringProjectRevision?: string | number
}
export type DevelopmentRestoreTaskState =
  | 'queued'
  | 'staging'
  | 'restoring_workspace'
  | 'restoring_scenes'
  | 'restoring_data'
  | 'finalizing'
  | 'compensating'
  | 'succeeded'
  | 'failed'
export interface DevelopmentRestoreTask {
  taskId: OpsId
  state: DevelopmentRestoreTaskState
  versionId?: OpsId
  version?: string
  backupId?: string
  currentProjectRevision?: string | number
  startedAt?: string
  completedAt?: string
  errorMessage?: string
  rolledBack?: boolean
}
export interface DeploymentRun {
  id: OpsId
  deploymentId?: OpsId
  observedStatus?: string
  status?: string
  progress?: number
  completedAt?: string | null
  startedAt?: string
  operation?: string
  message?: string
  actorDisplayName?: string
  durationMs?: number | null
}
export interface DeploymentRunEvent {
  id?: OpsId
  stage?: string
  message?: string
  createdAt?: string
}
export interface ListParams {
  page?: number
  pageSize?: number
  keyword?: string
  projectId?: OpsId
  environmentId?: OpsId
  status?: RuntimeEnvironmentStatus
}
export interface ListResult<T> {
  items: T[]
  total: number
}
type EnrollmentCreateResult = { enrollment: NodeEnrollment; code?: string }
type DeploymentCreateResult = { deployment: ProjectDeployment; run: DeploymentRun }
type ProjectDeploymentCreatePayload = {
  projectId: OpsId
  environmentId: OpsId
  mode: DeploymentMode
  applicationVersionId?: OpsId
  accessPort?: number
  placements: Partial<Record<DeploymentEngine, OpsId>>
}
/** @deprecated 仅供历史单节点页面过渡，新增调用必须使用 ProjectDeploymentCreatePayload。 */
type LegacyProjectDeploymentCreatePayload = {
  projectId: OpsId
  nodeId: OpsId
  applicationVersionId: OpsId
  accessPort: number
  enableCollector: boolean
}
const config: RequestConfig = { skipErrorToast: true }
const normalizeParams = (value: ListParams) => {
  const { keyword, ...params } = value
  return keyword ? { ...params, search: keyword } : params
}
const unpack = <T>(payload: OpsApiEnvelope<T> | T): T =>
  payload && typeof payload === 'object' && 'data' in payload
    ? (payload as OpsApiEnvelope<T>).data
    : (payload as T)
const list = <T>(
  payload: OpsApiEnvelope<ListResult<T> | T[]> | ListResult<T> | T[],
): ListResult<T> => {
  const data = unpack(payload)
  return Array.isArray(data)
    ? { items: data, total: data.length }
    : {
        items: Array.isArray(data.items) ? data.items : [],
        total: typeof data.total === 'number' ? data.total : data.items?.length || 0,
      }
}
const node = (value: OpsNode): OpsNode => ({
  ...value,
  name: value.displayName || value.name || value.hostname || value.id,
})
const deployment = (value: ProjectDeployment): ProjectDeployment => ({
  ...value,
  projectName: value.projectName || value.projectId,
})

export interface OpsRecord {
  id: string
  sourceKind: 'deployment_run' | 'cluster_event' | 'environment_event'
  recordType: 'operation' | 'event'
  objectType: 'deployment' | 'foundation' | 'environment' | 'node' | 'cluster'
  objectId: string
  objectName: string
  environmentId: string | null
  title: string
  status: 'accepted' | 'running' | 'success' | 'failed' | 'warning' | 'recovered' | 'info'
  eventResult: string
  actorDisplayName: string
  time: string
  completedAt: string | null
  durationMs: number | null
  message: string
  taskRef: { runId: string; deploymentId: string } | null
  detailUnavailableReason: string
}
export interface OpsRecordQuery {
  page: number
  limit: number
  search?: string
  recordType?: string
  objectType?: string
  objectId?: string
  environmentId?: string
  status?: string
  from?: string
  to?: string
  sort?: 'time_desc' | 'time_asc'
}

export const opsAPI = {
  async getRuntimeEnvironmentOverview(id: string, signal?: AbortSignal) {
    return unpack<RuntimeEnvironmentOverview>(
      await request.get(`/ops/runtime-environments/${id}/overview`, { ...config, signal }),
    )
  },
  async listRecords(params: OpsRecordQuery, signal?: AbortSignal) {
    const data = unpack<{
      list: OpsRecord[]
      pagination: { page: number; limit: number; total: number }
    }>(await request.get('/ops/records', { ...config, params, signal }))
    return { items: data.list, total: data.pagination.total }
  },
  async listEnrollments(params: ListParams = {}, signal?: AbortSignal) {
    return list<NodeEnrollment>(
      await request.get('/ops/node-enrollments', {
        params: normalizeParams(params),
        ...(signal ? { signal } : {}),
        ...config,
      }),
    )
  },
  async getEnrollment(id: OpsId) {
    return unpack<NodeEnrollment>(await request.get(`/ops/node-enrollments/${id}`, config))
  },
  async createEnrollment(payload: {
    platform: OpsPlatform
    capabilities: OpsCapability[]
    displayName: string
    ttlMinutes: number
  }) {
    const result = unpack<EnrollmentCreateResult>(
      await request.post('/ops/node-enrollments', payload, config),
    )
    return { ...result.enrollment, enrollmentCode: result.code }
  },
  async approveEnrollment(id: OpsId) {
    return unpack<NodeEnrollment>(
      await request.post(`/ops/node-enrollments/${id}/approve`, undefined, config),
    )
  },
  async rejectEnrollment(id: OpsId) {
    return unpack<NodeEnrollment>(
      await request.post(`/ops/node-enrollments/${id}/reject`, undefined, config),
    )
  },
  async listNodes(params: ListParams = {}, signal?: AbortSignal) {
    const result = list<OpsNode>(
      await request.get('/ops/nodes', {
        params: normalizeParams(params),
        ...(signal ? { signal } : {}),
        ...config,
      }),
    )
    return { ...result, items: result.items.map(node) }
  },
  async getNode(id: OpsId) {
    return node(unpack<OpsNode>(await request.get(`/ops/nodes/${id}`, config)))
  },
  async removeNode(id: OpsId) {
    return unpack<{ status: 'removing' }>(await request.delete(`/ops/nodes/${id}`, config))
  },
  async listRuntimeEnvironments(params: ListParams = {}, signal?: AbortSignal) {
    return list<RuntimeEnvironment>(
      await request.get('/ops/runtime-environments', {
        ...(signal ? { signal } : {}),
        params: normalizeParams(params),
        ...config,
      }),
    )
  },
  async createRuntimeEnvironment(payload: { name: string }) {
    return unpack<RuntimeEnvironment>(
      await request.post('/ops/runtime-environments', payload, config),
    )
  },
  async updateRuntimeEnvironment(id: OpsId, payload: { name: string }) {
    return unpack<RuntimeEnvironment>(
      await request.patch(`/ops/runtime-environments/${id}`, payload, config),
    )
  },
  async deleteRuntimeEnvironment(id: OpsId, confirmationName: string) {
    return unpack<{ status: 'deleting' | 'deleted' }>(
      await request.delete(`/ops/runtime-environments/${id}`, {
        ...config,
        data: { confirmationName },
      }),
    )
  },
  async getRuntimeEnvironment(id: OpsId, signal?: AbortSignal) {
    return unpack<RuntimeEnvironment>(
      await request.get(`/ops/runtime-environments/${id}`, {
        ...config,
        ...(signal ? { signal } : {}),
      }),
    )
  },
  async listRuntimeEnvironmentNodes(id: OpsId, params: ListParams = {}, signal?: AbortSignal) {
    const result = list<OpsNode>(
      await request.get(`/ops/runtime-environments/${id}/nodes`, {
        ...(signal ? { signal } : {}),
        params: normalizeParams(params),
        ...config,
      }),
    )
    return { ...result, items: result.items.map(node) }
  },
  async addRuntimeEnvironmentNodes(id: OpsId, nodeIds: OpsId[]) {
    const result = list<OpsNode>(
      await request.post(`/ops/runtime-environments/${id}/nodes`, { nodeIds }, config),
    )
    return { ...result, items: result.items.map(node) }
  },
  async removeRuntimeEnvironmentNode(id: OpsId, nodeId: OpsId) {
    return unpack<{ removed: boolean }>(
      await request.delete(`/ops/runtime-environments/${id}/nodes/${nodeId}`, config),
    )
  },
  async listRuntimeEnvironmentEvents(id: OpsId, params: ListParams = {}, signal?: AbortSignal) {
    return list<RuntimeEnvironmentEvent>(
      await request.get(`/ops/runtime-environments/${id}/events`, {
        ...(signal ? { signal } : {}),
        params: normalizeParams(params),
        ...config,
      }),
    )
  },
  async listRuntimeEnvironmentServices(id: OpsId, signal?: AbortSignal) {
    return list<RuntimeEnvironmentService>(
      await request.get(`/ops/runtime-environments/${id}/foundation-services`, {
        ...config,
        ...(signal ? { signal } : {}),
      }),
    )
  },
  async deployRuntimeEnvironmentFoundation(
    id: OpsId,
    assignments: Array<{ serviceType: FoundationServiceType; nodeId: OpsId }>,
  ) {
    return list<RuntimeEnvironmentService>(
      await request.post(
        `/ops/runtime-environments/${id}/foundation-services/deploy`,
        { assignments },
        config,
      ),
    )
  },
  async migrateRuntimeEnvironmentFoundation(
    id: OpsId,
    assignments: Array<{ serviceType: FoundationServiceType; nodeId: OpsId }>,
  ) {
    return list<RuntimeEnvironmentService>(
      await request.post(
        `/ops/runtime-environments/${id}/foundation-services/migrate`,
        { assignments },
        config,
      ),
    )
  },
  async listNodePackages() {
    const result = list<NodePackage>(await request.get('/ops/node-packages', config))
    return {
      ...result,
      items: result.items.map((item) => ({ ...item, name: item.name || item.fileName || item.id })),
    }
  },
  downloadNodePackage(id: OpsId) {
    return request.get(`/ops/node-packages/${id}/download`, {
      responseType: 'blob',
      returnRawResponse: true,
      ...config,
    } as RawResponseConfig)
  },
  async listProjectDeployments(params: ListParams = {}, signal?: AbortSignal) {
    const result = list<ProjectDeployment>(
      await request.get('/ops/project-deployments', {
        params: normalizeParams(params),
        signal,
        ...config,
      }),
    )
    return { ...result, items: result.items.map(deployment) }
  },
  async getDevelopmentDeploymentRequirements(projectId: OpsId) {
    const data = unpack<{ engines?: DeploymentEngine[] }>(
      await request.get('/ops/project-deployments/development-requirements', {
        params: { projectId },
        ...config,
      }),
    )
    return Array.isArray(data.engines) ? data.engines : []
  },
  async listProjectVersions(projectId: OpsId, query: ListParams = {}) {
    return list<ApplicationVersion>(
      await request.get(`/publish/${projectId}/versions`, {
        params: normalizeParams(query),
        ...config,
      }),
    )
  },
  async createProjectVersion(projectId: OpsId) {
    return unpack<ApplicationVersion>(
      await request.post(`/publish/${projectId}`, {}, { ...config, timeout: 120_000 }),
    )
  },
  async restoreProjectDevelopment(versionId: OpsId) {
    return unpack<DevelopmentRestoreTask>(
      await request.post(
        `/publish/versions/${versionId}/restore-development`,
        { confirmation: 'RESTORE' },
        config,
      ),
    )
  },
  async getDevelopmentRestoreTask(taskId: OpsId) {
    return unpack<DevelopmentRestoreTask>(
      await request.get(`/publish/restore-tasks/${taskId}`, config),
    )
  },
  async getProjectDeployment(id: OpsId, signal?: AbortSignal) {
    return deployment(
      unpack<ProjectDeployment>(
        await request.get(`/ops/project-deployments/${id}`, { ...config, signal }),
      ),
    )
  },
  async createProjectDeployment(
    payload: ProjectDeploymentCreatePayload | LegacyProjectDeploymentCreatePayload,
  ) {
    const result = unpack<DeploymentCreateResult>(
      await request.post('/ops/project-deployments', payload, config),
    )
    return { ...deployment(result.deployment), latestRunId: result.run?.id }
  },
  async operateProjectDeployment(id: OpsId, action: OpsDeploymentAction) {
    const result = unpack<DeploymentCreateResult>(
      await request.post(`/ops/project-deployments/${id}/${action}`, undefined, config),
    )
    return { deployment: deployment(result.deployment), run: result.run }
  },
  async deleteProjectDeployment(id: OpsId) {
    const result = unpack<{ run: DeploymentRun }>(
      await request.delete(`/ops/project-deployments/${id}`, config),
    )
    return result.run
  },
  async getDeploymentRun(id: OpsId, signal?: AbortSignal) {
    const result = unpack<DeploymentRun>(
      await request.get(`/ops/deployment-runs/${id}`, { ...config, signal }),
    )
    return { ...result, status: result.status || result.observedStatus }
  },
  async listDeploymentRunEvents(id: OpsId, signal?: AbortSignal) {
    return list<DeploymentRunEvent>(
      await request.get(`/ops/deployment-runs/${id}/events`, { ...config, signal }),
    )
  },
  async listProjectDeploymentRuns(id: OpsId, page = 1, limit = 10, signal?: AbortSignal) {
    const data = unpack<{ list: DeploymentRun[]; pagination: { total: number } }>(
      await request.get(`/ops/project-deployments/${id}/runs`, {
        params: { page, limit },
        signal,
        ...config,
      }),
    )
    return { items: data.list, total: data.pagination.total }
  },
  async listDeploymentRunEventsPage(id: OpsId, page = 1, limit = 10, signal?: AbortSignal) {
    const data = unpack<{ list: DeploymentRunEvent[]; pagination: { total: number } }>(
      await request.get(`/ops/deployment-runs/${id}/events/page`, {
        params: { page, limit },
        signal,
        ...config,
      }),
    )
    return { items: data.list, total: data.pagination.total }
  },
}
