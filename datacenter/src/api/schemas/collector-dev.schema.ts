import { z } from 'zod'

export const CollectorProtocolCapabilitySchema = z.object({
  protocolType: z.string(),
  capabilityVersion: z.string(),
  operations: z.array(z.string()),
})

export const CollectorAgentSchema = z.object({
  id: z.string(),
  name: z.string(),
  os: z.string(),
  arch: z.string(),
  version: z.string(),
  online: z.boolean(),
  capabilities: z.array(CollectorProtocolCapabilitySchema),
  lastSeenAt: z.string().nullable().optional(),
  createdAt: z.string(),
})

export const CollectorRegistrationCodeSchema = z.object({
  code: z.string(),
  expiresAt: z.string(),
})

export type CollectorAgent = z.infer<typeof CollectorAgentSchema>
export type CollectorRegistrationCode = z.infer<typeof CollectorRegistrationCodeSchema>
