import request, { type RawResponseConfig, type RequestConfig } from '@/utils/request'

export type OpsId = string
export type OpsPlatform = 'linux' | 'windows'
export type OpsCapability = 'project_entry' | 'data_runtime' | 'collector'
export type OpsServiceType = OpsCapability
export type OpsServiceAction = 'start' | 'stop' | 'restart'
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
  nodeId: OpsId
  nodeName?: string
  applicationVersionId: OpsId
  version?: string
  desiredStatus?: string
  observedStatus?: string
  health?: OpsHealth
  progress?: number
  entryStatus?: string
  accessUrl?: string | null
  services?: DeploymentService[]
  latestRunId?: OpsId
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
  artifactHash?: string
  manifest?: ApplicationVersionManifest | null
  completedAt?: string
  createdAt?: string
}
export interface DeploymentRun {
  id: OpsId
  deploymentId?: OpsId
  observedStatus?: string
  status?: string
  progress?: number
  completedAt?: string | null
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
}
export interface ListResult<T> {
  items: T[]
  total: number
}
type EnrollmentCreateResult = { enrollment: NodeEnrollment; code?: string }
type DeploymentCreateResult = { deployment: ProjectDeployment; run: DeploymentRun }
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

export const opsAPI = {
  async listEnrollments(params: ListParams = {}) {
    return list<NodeEnrollment>(
      await request.get('/ops/node-enrollments', { params: normalizeParams(params), ...config }),
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
  async listNodes(params: ListParams = {}) {
    const result = list<OpsNode>(
      await request.get('/ops/nodes', { params: normalizeParams(params), ...config }),
    )
    return { ...result, items: result.items.map(node) }
  },
  async getNode(id: OpsId) {
    return node(unpack<OpsNode>(await request.get(`/ops/nodes/${id}`, config)))
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
  async listProjectDeployments(params: ListParams = {}) {
    const result = list<ProjectDeployment>(
      await request.get('/ops/project-deployments', { params: normalizeParams(params), ...config }),
    )
    return { ...result, items: result.items.map(deployment) }
  },
  async listProjectVersions(projectId: OpsId, query: ListParams = {}) {
    return list<ApplicationVersion>(
      await request.get(`/publish/${projectId}/versions`, {
        params: normalizeParams(query),
        ...config,
      }),
    )
  },
  async getProjectDeployment(id: OpsId) {
    return deployment(
      unpack<ProjectDeployment>(await request.get(`/ops/project-deployments/${id}`, config)),
    )
  },
  async createProjectDeployment(payload: {
    projectId: OpsId
    nodeId: OpsId
    applicationVersionId: OpsId
    enableCollector: boolean
  }) {
    const result = unpack<DeploymentCreateResult>(
      await request.post('/ops/project-deployments', payload, config),
    )
    return { ...deployment(result.deployment), latestRunId: result.run?.id }
  },
  async runServiceAction(id: OpsId, service: OpsServiceType, action: OpsServiceAction) {
    const result = unpack<DeploymentCreateResult>(
      await request.post(
        `/ops/project-deployments/${id}/services/${service}/${action}`,
        undefined,
        config,
      ),
    )
    return result.run
  },
  async getDeploymentRun(id: OpsId) {
    const result = unpack<DeploymentRun>(await request.get(`/ops/deployment-runs/${id}`, config))
    return { ...result, status: result.status || result.observedStatus }
  },
  async listDeploymentRunEvents(id: OpsId) {
    return list<DeploymentRunEvent>(await request.get(`/ops/deployment-runs/${id}/events`, config))
  },
}
