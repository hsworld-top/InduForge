import request from '@/utils/request'
import {
  CollectorAgentSchema,
  CollectorRegistrationCodeSchema,
  type CollectorAgent,
  type CollectorRegistrationCode,
} from './schemas/collector-dev.schema'

export async function createCollectorRegistrationCode(): Promise<CollectorRegistrationCode> {
  const response = await request({
    url: '/data/collector-dev/registration-codes',
    method: 'post',
    data: {},
  })
  return CollectorRegistrationCodeSchema.parse(response)
}

export async function getCollectorAgents(): Promise<CollectorAgent[]> {
  const response = await request({ url: '/data/collector-dev/agents', method: 'get' })
  return CollectorAgentSchema.array().parse(response)
}

export async function deleteCollectorAgent(agentId: string): Promise<void> {
  await request({ url: `/data/collector-dev/agents/${agentId}`, method: 'delete' })
}
