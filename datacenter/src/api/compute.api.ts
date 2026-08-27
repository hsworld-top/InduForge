import request from '@/utils/request'
import { listResponseSchema } from './schemas/common.schema'
import {
  ComputeUnitSchema,
  ComputeUnitDetailSchema,
  ComputeUnitSaveSchema,
  ComputeFolderSaveSchema,
  ComputeFolderSchema,
  ComputeFolderPageSchema,
  ComputeDependencySchema,
  ComputeRunResultSchema,
  ComputeSyntaxCheckResultSchema,
  ComputeCapabilitiesSchema,
  type ComputeUnit,
  type ComputeUnitDetail,
  type ComputeUnitSave,
  type ComputeFolderSave,
  type ComputeFolder,
  type ComputeFolderPage,
  type ComputeDependency,
  type ComputeRunResult,
  type ComputeSyntaxCheckResult,
  type ComputeCapabilities,
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

/** 下载并安装工程级计算依赖。 */
export async function installComputeDependency(
  projectId: string,
  data: { language: 'js' | 'python'; packageName: string; version?: string },
): Promise<ComputeDependency> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/dependencies`,
    method: 'post',
    data,
    timeout: 125000,
  })
  return ComputeDependencySchema.parse(unwrapData(res))
}

/** 从 npm tgz 或 Python wheel 离线导入工程级依赖。 */
export async function importComputeDependency(
  projectId: string,
  language: 'js' | 'python',
  file: File,
): Promise<ComputeDependency> {
  const data = new FormData()
  data.append('language', language)
  data.append('file', file)
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/dependencies/import`,
    method: 'post',
    data,
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 125000,
  })
  return ComputeDependencySchema.parse(unwrapData(res))
}

/** 卸载未被计算单元引用的工程级依赖。 */
export async function uninstallComputeDependency(
  projectId: string,
  dependencyId: string,
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/compute-units/dependencies/${dependencyId}`,
    method: 'delete',
    timeout: 125000,
  })
}

/** 执行计算单元 */
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

/** 调试执行计算单元 */
export async function debugComputeUnit(
  projectId: string,
  id: string,
  input: Record<string, unknown> = {},
  dryRun = true,
  signal?: AbortSignal,
): Promise<ComputeRunResult> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/${id}/debug`,
    method: 'post',
    data: { input, dryRun },
    signal,
  })
  return ComputeRunResultSchema.parse(unwrapData(res))
}

/** 检查未保存计算脚本语法 */
export async function checkComputeSyntax(
  projectId: string,
  data: { lang?: string; language?: string; code?: string; scriptCode?: string },
  signal?: AbortSignal,
): Promise<ComputeSyntaxCheckResult> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/syntax-check`,
    method: 'post',
    data,
    signal,
  })
  return ComputeSyntaxCheckResultSchema.parse(unwrapData(res))
}

/** 获取独立计算沙箱声明的真实能力。 */
export async function getComputeCapabilities(projectId: string): Promise<ComputeCapabilities> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/capabilities`,
    method: 'get',
  })
  return ComputeCapabilitiesSchema.parse(unwrapData(res))
}

/** 预览未保存的周期、每日或每周触发配置，不会创建节点调度任务。 */
export async function previewComputeSchedule(
  projectId: string,
  data: { triggerType: string; triggerConfig: Record<string, unknown> },
): Promise<{
  triggerType: string
  triggerConfig: Record<string, unknown>
  summary: string
  nextRuns: string[]
  errors: Array<{ field: string; message: string }>
}> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/schedule-preview`,
    method: 'post',
    data,
  })
  const payload = unwrapData(res) as Record<string, unknown>
  return {
    triggerType: String(payload.triggerType || ''),
    triggerConfig: (payload.triggerConfig || {}) as Record<string, unknown>,
    summary: String(payload.summary || ''),
    nextRuns: Array.isArray(payload.nextRuns) ? payload.nextRuns.map(String) : [],
    errors: Array.isArray(payload.errors)
      ? payload.errors.map((item) => {
          const error = item as Record<string, unknown>
          return { field: String(error.field || ''), message: String(error.message || '') }
        })
      : [],
  }
}

/** 按父目录分页读取直接子目录；search 会在全目录树中按完整路径检索。 */
export async function getComputeFolders(
  projectId: string,
  params: { parentId?: string; search?: string; page?: number; pageSize?: number } = {},
): Promise<ComputeFolderPage> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/folders`,
    method: 'get',
    params,
  })
  return ComputeFolderPageSchema.parse(unwrapData(res))
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
