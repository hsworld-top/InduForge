import request from '@/utils/request'

export type StoragePolicyStatus = 'enabled' | 'disabled' | 'error'
export type StoragePolicyWriteMode = 'every_sample' | 'on_change' | 'periodic_snapshot'
export type StoragePolicyBindingMode = 'static' | 'dynamic'
export type StorageTargetCapability = 'timeseriesAppend' | 'relationalAppend'
export type StorageTargetTableMode = 'auto_create' | 'existing_mapping'

export interface StorageDataPointFilter {
  type?: string
  status?: string
  search?: string
  accessSourceId?: string
  sourceId?: string
  sourceIds?: string[]
  tags?: string[]
}

export interface StoragePolicyDiagnostic {
  severity: 'error' | 'warning' | 'info' | string
  type: string
  message: string
  suggest?: string
}

export interface StoragePolicyEstimate {
  matchedDataPointCount: number
  eventsPerSecond: number
  rowsPerDay: number
  rowsByRetention: number
  assumption: string
}

export interface StorageTarget {
  id: string
  name: string
  type: string
  category?: string
  status: string
  capabilities: StorageTargetCapability[]
}

export interface StoragePolicyBinding {
  id: string
  datapointId: string
  datapointPath: string
  datapointName: string
  dataType: string
  status: string
  createdAt?: string
}

export interface StoragePolicy {
  id: string
  projectId: string
  name: string
  description?: string | null
  target: StorageTarget
  targetCapability: StorageTargetCapability
  bindingMode: StoragePolicyBindingMode
  bindingFilter: StorageDataPointFilter
  writeMode: StoragePolicyWriteMode
  minIntervalMs?: number | null
  deadband?: number | null
  snapshotIntervalMs?: number | null
  includeQualities: string[]
  retentionDays: number
  targetTableMode: StorageTargetTableMode
  targetTableConfig: Record<string, unknown>
  status: StoragePolicyStatus
  diagnostics: StoragePolicyDiagnostic[]
  bindingCount: number
  estimate: StoragePolicyEstimate
  createdAt?: string
  updatedAt?: string
}

export interface StoragePolicyDetail extends StoragePolicy {
  bindings: StoragePolicyBinding[]
}

export interface StoragePolicySummary {
  enabledCount: number
  errorCount: number
  totalBindingCount: number
  estimatedRowsPerDay: number
  estimatedEventsSecond: number
}

export interface StoragePolicyPagination {
  page: number
  pageSize: number
  total: number
  totalPages?: number
}

export interface StoragePolicyListResult {
  list: StoragePolicy[]
  pagination?: StoragePolicyPagination
  summary?: StoragePolicySummary
}

export interface StoragePolicyListParams {
  page?: number
  pageSize?: number
  search?: string
  status?: string
  targetConnectionId?: string
  writeMode?: string
  bindingMode?: string
}

export interface StoragePolicySavePayload {
  name: string
  description?: string | null
  targetConnectionId: string
  targetCapability?: StorageTargetCapability | ''
  bindingMode: StoragePolicyBindingMode
  bindingFilter?: StorageDataPointFilter
  datapointIds?: string[]
  writeMode: StoragePolicyWriteMode
  minIntervalMs?: number | null
  deadband?: number | null
  snapshotIntervalMs?: number | null
  includeQualities?: string[]
  retentionDays: number
  targetTableMode?: StorageTargetTableMode
  targetTableConfig?: Record<string, unknown>
  status?: StoragePolicyStatus
}

export interface StoragePolicyCoverage {
  datapointId?: string
  datapointPath?: string
  matchedPolicyCount: number
}

const unwrapData = <T>(value: unknown): T => {
  if (value && typeof value === 'object' && 'data' in value) {
    return (value as { data?: T }).data as T
  }
  return value as T
}

export async function listStoragePolicies(
  projectId: string,
  params: StoragePolicyListParams = {},
): Promise<StoragePolicyListResult> {
  const res = await request({
    url: `/data/projects/${projectId}/data-storage-policies`,
    method: 'get',
    params,
  })
  return unwrapData<StoragePolicyListResult>(res)
}

export async function getStoragePolicy(
  projectId: string,
  policyId: string,
): Promise<StoragePolicyDetail> {
  const res = await request({
    url: `/data/projects/${projectId}/data-storage-policies/${policyId}`,
    method: 'get',
  })
  return unwrapData<StoragePolicyDetail>(res)
}

export async function createStoragePolicy(
  projectId: string,
  payload: StoragePolicySavePayload,
): Promise<StoragePolicyDetail> {
  const res = await request({
    url: `/data/projects/${projectId}/data-storage-policies`,
    method: 'post',
    data: payload,
  })
  return unwrapData<StoragePolicyDetail>(res)
}

export async function updateStoragePolicy(
  projectId: string,
  policyId: string,
  payload: StoragePolicySavePayload,
): Promise<StoragePolicyDetail> {
  const res = await request({
    url: `/data/projects/${projectId}/data-storage-policies/${policyId}`,
    method: 'put',
    data: payload,
  })
  return unwrapData<StoragePolicyDetail>(res)
}

export async function deleteStoragePolicy(projectId: string, policyId: string): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/data-storage-policies/${policyId}`,
    method: 'delete',
  })
}

export async function listStorageTargets(projectId: string): Promise<StorageTarget[]> {
  const res = await request({
    url: `/data/projects/${projectId}/data-storage-policies/targets`,
    method: 'get',
  })
  const data = unwrapData<{ targets?: StorageTarget[] }>(res)
  return data.targets || []
}

export async function estimateStoragePolicy(
  projectId: string,
  payload: StoragePolicySavePayload,
): Promise<StoragePolicyEstimate> {
  const res = await request({
    url: `/data/projects/${projectId}/data-storage-policies/estimate`,
    method: 'post',
    data: payload,
  })
  return unwrapData<StoragePolicyEstimate>(res)
}

export async function getStoragePolicyCoverage(
  projectId: string,
  params: { datapointId?: string; path?: string },
): Promise<StoragePolicyCoverage> {
  const res = await request({
    url: `/data/projects/${projectId}/data-storage-policies/coverage`,
    method: 'get',
    params,
  })
  return unwrapData<StoragePolicyCoverage>(res)
}
