import type { AxiosRequestConfig } from 'axios'
import request from '@/utils/request'
import {
  ContractCheckResultSchema,
  ContractCheckRunPageSchema,
  type ContractCheckResult,
  type ContractCheckRunPage,
} from './schemas/contract-check.schema'

const unwrapData = (value: unknown): unknown => {
  if (value && typeof value === 'object' && 'code' in value && 'data' in value) {
    return (value as { data?: unknown }).data
  }
  return value
}

const requestData = async (config: AxiosRequestConfig): Promise<unknown> =>
  unwrapData(await request(config))

/** 运行一次项目级数据契约检查。 */
export async function runContractCheck(projectId: string): Promise<ContractCheckResult> {
  return ContractCheckResultSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/contract-checks/run`,
      method: 'post',
      data: { scope: 'project' },
    }),
  )
}

/** 获取最近一次已持久化的检查结果；无历史时后端会执行首次检查。 */
export async function getLatestContractCheck(projectId: string): Promise<ContractCheckResult> {
  return ContractCheckResultSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/contract-checks/latest`,
      method: 'get',
    }),
  )
}

/** 分页获取契约检查历史。 */
export async function getContractChecks(
  projectId: string,
  params: { page?: number; pageSize?: number } = {},
): Promise<ContractCheckRunPage> {
  return ContractCheckRunPageSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/contract-checks/runs`,
      method: 'get',
      params,
    }),
  )
}
