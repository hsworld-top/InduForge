import request from '@/utils/request'
import {
  CollectorAgentPageSchema,
  CollectorRegistrationCodeSchema,
  type CollectorAgentPage,
  type CollectorRegistrationCode,
} from './schemas/collector-dev.schema'

const unwrapData = (value: unknown) => {
  if (value && typeof value === 'object' && 'code' in value && 'data' in value) {
    return (value as { data?: unknown }).data
  }
  return value
}

export async function createCollectorRegistrationCode(): Promise<CollectorRegistrationCode> {
  const response = await request({
    url: '/data/collector-dev/registration-codes',
    method: 'post',
    data: {},
  })
  return CollectorRegistrationCodeSchema.parse(unwrapData(response))
}

export async function getCollectorAgents(params: {
  page: number
  pageSize: number
}): Promise<CollectorAgentPage> {
  const response = await request({ url: '/data/collector-dev/agents', method: 'get', params })
  return CollectorAgentPageSchema.parse(unwrapData(response))
}

export async function deleteCollectorAgent(agentId: string): Promise<void> {
  await request({ url: `/data/collector-dev/agents/${agentId}`, method: 'delete' })
}
