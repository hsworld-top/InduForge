import request from '@/utils/request'
import { z } from 'zod'
import {
  CollectorConnectionPageSchema,
  CollectorConnectionSchema,
  CollectorDriverDetailSchema,
  CollectorDriverPageSchema,
  CollectorImportPreviewSchema,
  CollectorPointGroupSchema,
  CollectorPointPageSchema,
  CollectorPointSchema,
  CollectorTaskSchema,
  type CollectorConnection,
  type CollectorDriverDetail,
  type CollectorDriverSummary,
  type CollectorImportPreview,
  type CollectorPoint,
  type CollectorPointGroup,
  type CollectorTask,
} from './schemas/collector.schema'

export type PageResult<T> = {
  list: T[]
  pagination: { page: number; pageSize: number; total: number; totalPages: number }
}

export async function listCollectorDrivers(
  params: Record<string, unknown>,
): Promise<PageResult<CollectorDriverSummary>> {
  return CollectorDriverPageSchema.parse(
    await request({ url: '/data/collector/drivers', method: 'get', params }),
  )
}

export async function getCollectorDriver(driverId: string): Promise<CollectorDriverDetail> {
  return CollectorDriverDetailSchema.parse(
    await request({ url: `/data/collector/drivers/${driverId}`, method: 'get' }),
  )
}

export async function listCollectorConnections(
  projectId: string,
  params: Record<string, unknown>,
): Promise<PageResult<CollectorConnection>> {
  return CollectorConnectionPageSchema.parse(
    await request({
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
    await request({
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
    await request({
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
    await request({
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
  await request({
    url: `/data/projects/${projectId}/collector/connections/${connectionId}`,
    method: 'delete',
  })
}

export async function listCollectorPointGroups(
  projectId: string,
  connectionId: string,
  parentId?: string | null,
): Promise<CollectorPointGroup[]> {
  const data = await request({
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
    await request({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}/point-groups`,
      method: 'post',
      data,
    }),
  )
}

export async function listCollectorPoints(
  projectId: string,
  connectionId: string,
  params: Record<string, unknown>,
): Promise<PageResult<CollectorPoint>> {
  return CollectorPointPageSchema.parse(
    await request({
      url: `/data/projects/${projectId}/collector/connections/${connectionId}/points`,
      method: 'get',
      params,
    }),
  )
}

async function postPointBatch(
  projectId: string,
  connectionId: string,
  action: string,
  data: unknown,
): Promise<CollectorPoint[]> {
  const response = await request({
    url: `/data/projects/${projectId}/collector/connections/${connectionId}/points/${action}`,
    method: 'post',
    data,
  })
  return z.object({ list: z.array(CollectorPointSchema) }).parse(response).list
}

export const createCollectorPointsBatch = (
  projectId: string,
  connectionId: string,
  points: Record<string, unknown>[],
) => postPointBatch(projectId, connectionId, 'batch', { points })
export const updateCollectorPointsBatch = (
  projectId: string,
  connectionId: string,
  points: Record<string, unknown>[],
) => postPointBatch(projectId, connectionId, 'update-batch', { points })
export async function deleteCollectorPointsBatch(
  projectId: string,
  connectionId: string,
  pointIds: string[],
): Promise<void> {
  await request({
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
  await request({
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
    await request({
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
  const response = await request({
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
    await request({ url: `/data/projects/${projectId}/collector-dev/tasks`, method: 'post', data }),
  )
}

export async function getCollectorTask(projectId: string, taskId: string): Promise<CollectorTask> {
  return CollectorTaskSchema.parse(
    await request({
      url: `/data/projects/${projectId}/collector-dev/tasks/${taskId}`,
      method: 'get',
    }),
  )
}

export async function cancelCollectorTask(projectId: string, taskId: string): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/collector-dev/tasks/${taskId}`,
    method: 'delete',
  })
}
