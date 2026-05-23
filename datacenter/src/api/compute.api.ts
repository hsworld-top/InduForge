import request from '@/utils/request'
import { listResponseSchema } from './schemas/common.schema'
import {
  ComputeUnitSchema,
  ComputeUnitDetailSchema,
  ComputeUnitSaveSchema,
  ComputeFolderSaveSchema,
  ComputeFolderSchema,
  ComputeDependencySchema,
  ComputeRunResultSchema,
  ComputeSyntaxCheckResultSchema,
  type ComputeUnit,
  type ComputeUnitDetail,
  type ComputeUnitSave,
  type ComputeFolderSave,
  type ComputeFolder,
  type ComputeDependency,
  type ComputeRunResult,
  type ComputeSyntaxCheckResult,
} from './schemas/compute.schema'

const computeListSchema = listResponseSchema(ComputeUnitSchema)

type ComputeListResp = {
  list: ComputeUnit[]
  pagination?: {
    page?: number
    pageSize?: number
    total?: number
  }
}

const unwrapData = (value: unknown) => {
  if (value && typeof value === 'object' && 'code' in value && 'data' in value) {
    return (value as { data?: unknown }).data
  }
  return value
}

const normalizeComputeListPayload = (value: unknown) => {
  const payload = unwrapData(value)
  if (Array.isArray(payload)) {
    return { list: payload }
  }
  if (!payload || typeof payload !== 'object') {
    return { list: [] }
  }
  const record = payload as Record<string, unknown>
  return {
    ...record,
    list: record.list ?? record.items ?? record.computeUnits ?? record.units ?? [],
  }
}

/** 获取计算单元列表 */
export async function getComputeUnits(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<ComputeListResp> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units`,
    method: 'get',
    params,
  })
  return computeListSchema.parse(normalizeComputeListPayload(res))
}

/** 获取计算单元详情 */
export async function getComputeUnit(projectId: string, id: string): Promise<ComputeUnitDetail> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/${id}`,
    method: 'get',
  })
  return ComputeUnitDetailSchema.parse(unwrapData(res))
}

/** 创建计算单元 */
export async function createComputeUnit(
  projectId: string,
  data: ComputeUnitSave,
): Promise<ComputeUnitDetail> {
  const body = ComputeUnitSaveSchema.parse(data)
  const res = await request({
    url: `/data/projects/${projectId}/compute-units`,
    method: 'post',
    data: body,
  })
  return ComputeUnitDetailSchema.parse(unwrapData(res))
}

/** 更新计算单元 */
export async function updateComputeUnit(
  projectId: string,
  id: string,
  data: Partial<ComputeUnitSave>,
): Promise<ComputeUnitDetail> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/${id}`,
    method: 'put',
    data,
  })
  return ComputeUnitDetailSchema.parse(unwrapData(res))
}

/** 删除计算单元 */
export async function deleteComputeUnit(projectId: string, id: string): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/compute-units/${id}`,
    method: 'delete',
  })
}

/** 更新计算单元启用状态 */
export async function toggleComputeUnit(
  projectId: string,
  id: string,
  enabled: boolean,
): Promise<ComputeUnitDetail> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/${id}/enabled`,
    method: 'patch',
    data: { enabled },
  })
  return ComputeUnitDetailSchema.parse(unwrapData(res))
}

/** 获取计算单元依赖白名单 */
export async function getComputeDependencies(projectId: string): Promise<ComputeDependency[]> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/dependencies`,
    method: 'get',
  })
  const payload = unwrapData(res)
  if (Array.isArray(payload)) {
    return ComputeDependencySchema.array().parse(payload)
  }
  const record = (payload || {}) as Record<string, unknown>
  return ComputeDependencySchema.array().parse(
    record.list ?? record.items ?? record.dependencies ?? [],
  )
}

/** 执行计算单元（data.api.ts 已有，这里提供类型化版本） */
export async function runComputeUnit(
  projectId: string,
  id: string,
  input: Record<string, unknown> = {},
): Promise<ComputeRunResult> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/${id}/run`,
    method: 'post',
    data: { input },
  })
  return ComputeRunResultSchema.parse(unwrapData(res))
}

/** 调试执行计算单元（data.api.ts 已有，这里提供类型化版本） */
export async function debugComputeUnit(
  projectId: string,
  id: string,
  input: Record<string, unknown> = {},
  dryRun = true,
): Promise<ComputeRunResult> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/${id}/debug`,
    method: 'post',
    data: { input, dryRun },
  })
  return ComputeRunResultSchema.parse(unwrapData(res))
}

/** 检查未保存计算脚本语法 */
export async function checkComputeSyntax(
  projectId: string,
  data: { lang?: string; language?: string; code?: string; scriptCode?: string },
): Promise<ComputeSyntaxCheckResult> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/syntax-check`,
    method: 'post',
    data,
  })
  return ComputeSyntaxCheckResultSchema.parse(unwrapData(res))
}

/** 获取计算单元文件夹树 */
export async function getComputeFolders(projectId: string): Promise<ComputeFolder[]> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/folders`,
    method: 'get',
  })
  const payload = unwrapData(res)
  if (Array.isArray(payload)) {
    return ComputeFolderSchema.array().parse(payload)
  }
  const record = (payload || {}) as Record<string, unknown>
  return ComputeFolderSchema.array().parse(record.list ?? record.items ?? record.folders ?? [])
}

/** 创建计算单元文件夹 */
export async function createComputeFolder(
  projectId: string,
  data: ComputeFolderSave,
): Promise<ComputeFolder> {
  const body = ComputeFolderSaveSchema.parse(data)
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/folders`,
    method: 'post',
    data: body,
  })
  return ComputeFolderSchema.parse(unwrapData(res))
}

/**
 * 更新计算单元文件夹。
 * C1 先定义契约，后续拖拽 / 重命名接入。
 */
export async function updateComputeFolder(
  projectId: string,
  folderId: string,
  data: Partial<ComputeFolderSave>,
): Promise<ComputeFolder> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/folders/${folderId}`,
    method: 'put',
    data,
  })
  return ComputeFolderSchema.parse(unwrapData(res))
}

/**
 * 删除计算单元文件夹。
 * C1 先定义契约，后续右键菜单接入。
 */
export async function deleteComputeFolder(projectId: string, folderId: string): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/compute-units/folders/${folderId}`,
    method: 'delete',
  })
}
