import request from '@/utils/request'
import {
  HistoryStorageBatchResultSchema,
  HistoryStorageDatapointDetailSchema,
  HistoryStorageSourceDetailSchema,
  HistoryStorageSourceListSchema,
  HistoryStorageTargetsSchema,
  type HistoryStorageBehavior,
  type HistoryStorageConfiguration,
  type HistoryStorageScopeType,
} from './schemas/history-storage.schema'

const unwrapData = (value: unknown) => {
  if (value && typeof value === 'object' && 'code' in value && 'data' in value) {
    return (value as { data?: unknown }).data
  }
  return value
}

export type HistoryStorageConfigurationPayload = Omit<HistoryStorageConfiguration, 'targets'> & {
  targets: Array<{
    connectionId: string
    isPrimary: boolean
    sortOrder: number
    retentionDays: number | null
  }>
}

export type HistoryStorageSavePayload = {
  behavior: HistoryStorageBehavior
  configuration?: HistoryStorageConfigurationPayload | null
}

export type HistoryStorageSourceListParams = {
  page?: number
  pageSize?: number
  search?: string
  scopeType?: HistoryStorageScopeType | ''
  historyState?: 'enabled' | 'disabled' | ''
}

export async function listHistoryStorageSources(
  projectId: string,
  params: HistoryStorageSourceListParams,
) {
  const response = await request({
    url: `/data/projects/${projectId}/history-storage/sources`,
    method: 'get',
    params,
  })
  return HistoryStorageSourceListSchema.parse(unwrapData(response))
}

export async function getHistoryStorageSource(
  projectId: string,
  scopeType: HistoryStorageScopeType,
  scopeId: string,
) {
  const response = await request({
    url: `/data/projects/${projectId}/history-storage/sources/${scopeType}/${scopeId}`,
    method: 'get',
  })
  return HistoryStorageSourceDetailSchema.parse(unwrapData(response))
}

export async function saveHistoryStorageSource(
  projectId: string,
  scopeType: HistoryStorageScopeType,
  scopeId: string,
  payload: HistoryStorageSavePayload,
) {
  const response = await request({
    url: `/data/projects/${projectId}/history-storage/sources/${scopeType}/${scopeId}`,
    method: 'put',
    data: payload,
  })
  return HistoryStorageSourceDetailSchema.parse(unwrapData(response))
}

export async function listHistoryStorageTargets(projectId: string) {
  const response = await request({
    url: `/data/projects/${projectId}/history-storage/targets`,
    method: 'get',
  })
  return HistoryStorageTargetsSchema.parse(unwrapData(response)).targets
}

export async function getDatapointHistoryStorage(projectId: string, datapointId: string) {
  const response = await request({
    url: `/data/projects/${projectId}/history-storage/datapoints/${datapointId}`,
    method: 'get',
  })
  return HistoryStorageDatapointDetailSchema.parse(unwrapData(response))
}

export async function saveDatapointHistoryStorage(
  projectId: string,
  datapointId: string,
  payload: HistoryStorageSavePayload,
) {
  const response = await request({
    url: `/data/projects/${projectId}/history-storage/datapoints/${datapointId}`,
    method: 'put',
    data: payload,
  })
  return HistoryStorageDatapointDetailSchema.parse(unwrapData(response))
}

export type HistoryStorageBulkSelection =
  | { mode: 'ids'; datapointIds: string[] }
  | {
      mode: 'filtered'
      filters: Record<string, unknown>
      excludeDatapointIds?: string[]
    }

export async function batchConfigureDatapointHistoryStorage(
  projectId: string,
  selection: HistoryStorageBulkSelection,
  payload: HistoryStorageSavePayload,
) {
  const response = await request({
    url: `/data/projects/${projectId}/history-storage/datapoints/batch-configure`,
    method: 'post',
    data: { selection, ...payload },
  })
  return HistoryStorageBatchResultSchema.parse(unwrapData(response))
}
