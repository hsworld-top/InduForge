import request from '@/utils/request'
import { listResponseSchema } from './schemas/common.schema'
import {
  AccessSourceSummarySchema,
  AccessSourceDetailSchema,
  AccessSourceCreateSchema,
  type AccessSourceSummary,
  type AccessSourceDetail,
  type AccessSourceCreate,
} from './schemas/access-source.schema'

const accessSourceListSchema = listResponseSchema(AccessSourceSummarySchema)

type AccessSourceListResp = {
  list: AccessSourceSummary[]
  pagination?: {
    page?: number
    pageSize?: number
    total?: number
  }
}

/** 获取接入源列表 */
export async function getAccessSources(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<AccessSourceListResp> {
  const res = await request({
    url: `/data/projects/${projectId}/connections`,
    method: 'get',
    params,
  })
  return accessSourceListSchema.parse(res)
}

/** 获取接入源详情 */
export async function getAccessSource(
  projectId: string,
  sourceId: string,
): Promise<AccessSourceDetail> {
  const res = await request({
    url: `/data/projects/${projectId}/connections/${sourceId}`,
    method: 'get',
  })
  return AccessSourceDetailSchema.parse(res)
}

/** 创建接入源 */
export async function createAccessSource(
  projectId: string,
  data: AccessSourceCreate,
): Promise<AccessSourceDetail> {
  const body = AccessSourceCreateSchema.parse(data)
  const res = await request({
    url: `/data/projects/${projectId}/connections`,
    method: 'post',
    data: body,
  })
  return AccessSourceDetailSchema.parse(res)
}

/** 更新接入源 */
export async function updateAccessSource(
  projectId: string,
  sourceId: string,
  data: Partial<AccessSourceCreate>,
): Promise<AccessSourceDetail> {
  const res = await request({
    url: `/data/projects/${projectId}/connections/${sourceId}`,
    method: 'put',
    data,
  })
  return AccessSourceDetailSchema.parse(res)
}

/** 删除接入源 */
export async function deleteAccessSource(projectId: string, sourceId: string): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/connections/${sourceId}`,
    method: 'delete',
  })
}

/** 更新接入源卡片展示顺序 */
export async function updateAccessSourceOrder(
  projectId: string,
  connectionIds: string[],
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/connections/order`,
    method: 'patch',
    data: { connectionIds },
  })
}

/** 测试接入源连通性 */
export async function testAccessSource(
  projectId: string,
  data: Record<string, unknown>,
): Promise<{ success: boolean; message?: string }> {
  const res = await request({
    url: `/data/projects/${projectId}/connections/test`,
    method: 'post',
    data,
  })
  return res as unknown as { success: boolean; message?: string }
}

/** 更新接入源连接状态 */
export async function updateAccessSourceStatus(
  projectId: string,
  sourceId: string,
  status: string,
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/connections/${sourceId}/status`,
    method: 'patch',
    data: { status },
  })
}
