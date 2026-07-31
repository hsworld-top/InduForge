import request from '@/utils/request'
import type { AxiosError, AxiosRequestConfig } from 'axios'
import { z } from 'zod'
import {
  CollectorConnectionPageSchema,
  CollectorConnectionSchema,
  CollectorDriverDetailSchema,
  CollectorDriverPageSchema,
  CollectorImportPreviewSchema,
  CollectorPointBatchResultSchema,
  CollectorPointGroupSchema,
  CollectorPointPageSchema,
  CollectorPointSchema,
  CollectorTaskSchema,
  type CollectorConnection,
  type CollectorDriverDetail,
  type CollectorDriverSummary,
  type CollectorImportPreview,
  type CollectorPoint,
  type CollectorPointBatchResult,
  type CollectorPointGroup,
  type CollectorTask,
} from './schemas/collector.schema'

export type PageResult<T> = {
  list: T[]
  pagination: { page: number; pageSize: number; total: number; totalPages: number }
}

const unwrapData = (value: unknown) => {
  if (value && typeof value === 'object' && 'code' in value && 'data' in value) {
    return (value as { data?: unknown }).data
  }
  return value
}

const requestData = async (config: AxiosRequestConfig): Promise<unknown> =>
  unwrapData(await request(config))
export async function listCollectorDrivers(
  params: Record<string, unknown>,
): Promise<PageResult<CollectorDriverSummary>> {
  return CollectorDriverPageSchema.parse(
    await requestData({ url: '/data/collector/drivers', method: 'get', params }),
  )
}

const collectorDriverCatalogPageSize = 100

export async function listAllCollectorDrivers(): Promise<CollectorDriverSummary[]> {
  const firstPage = await listCollectorDrivers({ page: 1, pageSize: collectorDriverCatalogPageSize })
  const remainingPages = await Promise.all(
    Array.from({ length: Math.max(0, firstPage.pagination.totalPages - 1) }, (_, index) =>
      listCollectorDrivers({ page: index + 2, pageSize: collectorDriverCatalogPageSize }),
    ),
  )
  return [firstPage, ...remainingPages].flatMap((page) => page.list)
}

export async function getCollectorDriver(driverId: string): Promise<CollectorDriverDetail> {
  return CollectorDriverDetailSchema.parse(
    await requestData({ url: `/data/collector/drivers/${driverId}`, method: 'get' }),
  )
}

export async function listCollectorConnections(
  projectId: string,
  params: Record<string, unknown>,
): Promise<PageResult<CollectorConnection>> {
  return CollectorConnectionPageSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector/connections`,
      method: 'get',
      params,
    }),
  )
}

export async function getCollectorConnection(
  projectId: string,
  connectionId: string,
): Promise<CollectorConnection> {
  return CollectorConnectionSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}`,
      method: 'get',
    }),
  )
}

export async function createCollectorConnection(
  projectId: string,
  data: Record<string, unknown>,
): Promise<CollectorConnection> {
  return CollectorConnectionSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector/connections`,
      method: 'post',
      data,
    }),
  )
}

export async function updateCollectorConnection(
  projectId: string,
  connectionId: string,
  data: Record<string, unknown>,
): Promise<CollectorConnection> {
  return CollectorConnectionSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}`,
      method: 'put',
      data,
    }),
  )
}

export async function deleteCollectorConnection(
  projectId: string,
  connectionId: string,
): Promise<void> {
  await requestData({
    url: `/data/projects/${projectId}/collector/connections/${connectionId}`,
    method: 'delete',
  })
}

export async function listCollectorPointGroups(
  projectId: string,
  connectionId: string,
  parentId?: string | null,
): Promise<CollectorPointGroup[]> {
  const data = await requestData({
    url: `/data/projects/${projectId}/collector/connections/${connectionId}/point-groups`,
    method: 'get',
    params: parentId ? { parentId } : {},
  })
  return CollectorPointGroupSchema.array().parse(data)
}

export async function createCollectorPointGroup(
  projectId: string,
  connectionId: string,
  data: Record<string, unknown>,
): Promise<CollectorPointGroup> {
  return CollectorPointGroupSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}/point-groups`,
      method: 'post',
      data,
    }),
  )
}

export async function updateCollectorPointGroup(
  projectId: string,
  connectionId: string,
  groupId: string,
  data: { name: string },
): Promise<CollectorPointGroup> {
  return CollectorPointGroupSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}/point-groups/${groupId}`,
      method: 'put',
      data,
    }),
  )
}

export async function deleteCollectorPointGroup(
  projectId: string,
  connectionId: string,
  groupId: string,
): Promise<void> {
  await requestData({
    url: `/data/projects/${projectId}/collector/connections/${connectionId}/point-groups/${groupId}`,
    method: 'delete',
  })
}

export async function listCollectorPoints(
  projectId: string,
  connectionId: string,
  params: Record<string, unknown>,
): Promise<PageResult<CollectorPoint>> {
  return CollectorPointPageSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}/points`,
      method: 'get',
      params,
    }),
  )
}

export async function checkCollectorPointAddresses(
  projectId: string,
  connectionId: string,
  addresses: Record<string, unknown>[],
): Promise<number[]> {
  const response = z.object({ indexes: z.array(z.number().int().nonnegative()) }).parse(
    await requestData({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}/points/check-addresses`,
      method: 'post',
      data: { addresses },
    }),
  )
  return response.indexes
}

async function postPointBatch(
  projectId: string,
  connectionId: string,
  action: string,
  data: unknown,
): Promise<CollectorPoint[]> {
  const response = await requestData({
    url: `/data/projects/${projectId}/collector/connections/${connectionId}/points/${action}`,
    method: 'post',
    data,
  })
  return z.object({ list: z.array(CollectorPointSchema) }).parse(response).list
}

export async function createCollectorPointsBatch(
  projectId: string,
  connectionId: string,
  points: Record<string, unknown>[],
): Promise<CollectorPointBatchResult> {
  return CollectorPointBatchResultSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}/points/batch`,
      method: 'post',
      data: { points },
    }),
  )
}
export const updateCollectorPointsBatch = (
  projectId: string,
  connectionId: string,
  points: Record<string, unknown>[],
) => postPointBatch(projectId, connectionId, 'update-batch', { points })

export type CollectorPointExportRequest = {
  format: 'csv'
  scope: 'group' | 'current_page' | 'selected' | 'pages'
  groupId?: string | null
  includeChildren?: boolean
  search?: string
  dataType?: string
  enabled?: boolean | null
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
  page: number
  pageSize: number
  pages?: number[]
  pointIds?: string[]
}

export async function exportCollectorPoints(
  projectId: string,
  connectionId: string,
  data: CollectorPointExportRequest,
): Promise<Blob> {
  try {
    return (await request({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}/points/export`,
      method: 'post',
      responseType: 'blob',
      data,
    })) as Blob
  } catch (error) {
    const response = (error as AxiosError<Blob>)?.response
    if (response?.data instanceof Blob) {
      const payload = await response.data.text()
      let message = ''
      try {
        message = (JSON.parse(payload) as { msg?: string }).msg || ''
      } catch {
        message = ''
      }
      if (message) throw new Error(message)
    }
    throw error
  }
}

export async function deleteCollectorPointsBatch(
  projectId: string,
  connectionId: string,
  pointIds: string[],
): Promise<void> {
  await requestData({
    url: `/data/projects/${projectId}/collector/connections/${connectionId}/points/delete-batch`,
    method: 'post',
    data: { pointIds },
  })
}
export async function moveCollectorPointsBatch(
  projectId: string,
  connectionId: string,
  pointIds: string[],
  groupId: string | null,
): Promise<void> {
  await requestData({
    url: `/data/projects/${projectId}/collector/connections/${connectionId}/points/move-batch`,
    method: 'post',
    data: { pointIds, groupId },
  })
}

export async function previewCollectorPointImport(
  projectId: string,
  connectionId: string,
  file: File,
  page = 1,
  pageSize = 50,
): Promise<CollectorImportPreview> {
  const data = new FormData()
  data.append('file', file)
  data.append('page', String(page))
  data.append('pageSize', String(pageSize))
  return CollectorImportPreviewSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}/points/import-preview`,
      method: 'post',
      data,
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  )
}

export async function commitCollectorPointImport(
  projectId: string,
  connectionId: string,
  importId: string,
): Promise<CollectorPoint[]> {
  const response = await requestData({
    url: `/data/projects/${projectId}/collector/connections/${connectionId}/points/import-commit`,
    method: 'post',
    data: { importId },
  })
  return z.object({ list: z.array(CollectorPointSchema) }).parse(response).list
}

export async function createCollectorTask(
  projectId: string,
  data: {
    agentId: string
    connectionId: string
    operation: string
    input: Record<string, unknown>
    timeoutSeconds?: number
  },
): Promise<CollectorTask> {
  return CollectorTaskSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector-dev/tasks`,
      method: 'post',
      data,
    }),
  )
}

export async function getCollectorTask(projectId: string, taskId: string): Promise<CollectorTask> {
  return CollectorTaskSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/collector-dev/tasks/${taskId}`,
      method: 'get',
    }),
  )
}

export async function cancelCollectorTask(projectId: string, taskId: string): Promise<void> {
  await requestData({
    url: `/data/projects/${projectId}/collector-dev/tasks/${taskId}`,
    method: 'delete',
  })
}
