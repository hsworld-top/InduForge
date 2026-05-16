import request from "@/utils/request";
import { listResponseSchema } from "./schemas/common.schema";
import {
  ComputeUnitSchema,
  ComputeUnitDetailSchema,
  ComputeUnitSaveSchema,
  ComputeFolderSchema,
  ComputeRunResultSchema,
  type ComputeUnit,
  type ComputeUnitDetail,
  type ComputeUnitSave,
  type ComputeFolder,
  type ComputeRunResult,
} from "./schemas/compute.schema";

const computeListSchema = listResponseSchema(ComputeUnitSchema);

type ComputeListResp = {
  list: ComputeUnit[];
  pagination?: {
    page?: number;
    pageSize?: number;
    total?: number;
  };
};

/**
 * 获取计算单元列表。
 * 后端接口待实现，404 由 useApiError 兜底。
 */
export async function getComputeUnits(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<ComputeListResp> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units`,
    method: "get",
    params,
  });
  return computeListSchema.parse(res);
}

/** 获取计算单元详情 */
export async function getComputeUnit(
  projectId: string,
  id: string,
): Promise<ComputeUnitDetail> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/${id}`,
    method: "get",
  });
  return ComputeUnitDetailSchema.parse(res);
}

/** 创建计算单元（data.api.ts 已有，这里提供类型化版本） */
export async function createComputeUnit(
  projectId: string,
  data: ComputeUnitSave,
): Promise<ComputeUnitDetail> {
  const body = ComputeUnitSaveSchema.parse(data);
  const res = await request({
    url: `/data/projects/${projectId}/compute-units`,
    method: "post",
    data: body,
  });
  return ComputeUnitDetailSchema.parse(res);
}

/**
 * 更新计算单元。
 * 后端接口待实现，404 由 useApiError 兜底。
 */
export async function updateComputeUnit(
  projectId: string,
  id: string,
  data: Partial<ComputeUnitSave>,
): Promise<ComputeUnitDetail> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/${id}`,
    method: "put",
    data,
  });
  return ComputeUnitDetailSchema.parse(res);
}

/**
 * 删除计算单元。
 * 后端接口待实现，404/405 由 useApiError 兜底。
 */
export async function deleteComputeUnit(
  projectId: string,
  id: string,
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/compute-units/${id}`,
    method: "delete",
  });
}

/**
 * 启动计算单元。
 * 后端接口待实现。
 */
export async function startComputeUnit(
  projectId: string,
  id: string,
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/compute-units/${id}/start`,
    method: "post",
  });
}

/**
 * 停止计算单元。
 * 后端接口待实现。
 */
export async function stopComputeUnit(
  projectId: string,
  id: string,
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/compute-units/${id}/stop`,
    method: "post",
  });
}

/** 执行计算单元（data.api.ts 已有，这里提供类型化版本） */
export async function runComputeUnit(
  projectId: string,
  id: string,
  input: Record<string, unknown> = {},
): Promise<ComputeRunResult> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/${id}/run`,
    method: "post",
    data: { input },
  });
  return ComputeRunResultSchema.parse(res);
}

/** 调试执行计算单元（data.api.ts 已有，这里提供类型化版本） */
export async function debugComputeUnit(
  projectId: string,
  id: string,
  input: Record<string, unknown> = {},
): Promise<ComputeRunResult> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/${id}/debug`,
    method: "post",
    data: { input },
  });
  return ComputeRunResultSchema.parse(res);
}

/**
 * 获取计算单元文件夹树。
 * 后端接口待实现。
 */
export async function getComputeFolders(
  projectId: string,
): Promise<ComputeFolder[]> {
  const res = await request({
    url: `/data/projects/${projectId}/compute-units/folders`,
    method: "get",
  });
  return ComputeFolderSchema.array().parse(res);
}
