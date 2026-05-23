import request from '@/utils/request'
import {
  ContractCheckResultSchema,
  type ContractCheckResult,
} from './schemas/contract-check.schema'

/**
 * 触发契约检查。
 * 后端接口待实现，404/405 由 useApiError 兜底。
 */
export async function runContractCheck(projectId: string): Promise<ContractCheckResult> {
  const res = await request({
    url: `/data/projects/${projectId}/contract-checks/run`,
    method: 'post',
  })
  return ContractCheckResultSchema.parse(res)
}

/**
 * 获取最近一次契约检查结果。
 * 后端接口待实现，404 由 useApiError 兜底。
 */
export async function getLatestContractCheck(projectId: string): Promise<ContractCheckResult> {
  const res = await request({
    url: `/data/projects/${projectId}/contract-checks/latest`,
    method: 'get',
  })
  return ContractCheckResultSchema.parse(res)
}

/**
 * 获取契约检查历史列表。
 * 后端接口待实现。
 */
export async function getContractChecks(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<{ list: ContractCheckResult[] }> {
  const res = await request({
    url: `/data/projects/${projectId}/contract-checks`,
    method: 'get',
    params,
  })
  const list = ContractCheckResultSchema.array().parse(
    Array.isArray(res) ? res : ((res as unknown as { list?: unknown[] }).list ?? []),
  )
  return { list }
}
