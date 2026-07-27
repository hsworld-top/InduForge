import { z } from 'zod'

export const CollectorProtocolCapabilitySchema = z.object({
  driverId: z.string(),
  driverVersion: z.string(),
  schemaVersions: z.array(z.number().int().positive()),
  operations: z.array(z.string()),
  resources: z
    .object({
      serialPorts: z.array(z.string()),
    })
    .optional(),
})

export const CollectorAgentSchema = z.object({
  id: z.string(),
  name: z.string(),
  os: z.string(),
  arch: z.string(),
  version: z.string(),
  ipAddress: z.string(),
  status: z.enum(['online', 'offline', 'invalid']),
  capabilities: z.array(CollectorProtocolCapabilitySchema),
  lastSeenAt: z.string().nullable().optional(),
  createdAt: z.string(),
})

export const CollectorRegistrationCodeSchema = z.object({
  code: z.string(),
  expiresAt: z.string(),
})

export const CollectorAgentPageSchema = z.object({
  list: z.array(CollectorAgentSchema),
  pagination: z.object({
    page: z.number().int().positive(),
    pageSize: z.number().int().positive(),
    total: z.number().int().nonnegative(),
    totalPages: z.number().int().nonnegative(),
  }),
})

export type CollectorAgent = z.infer<typeof CollectorAgentSchema>
export type CollectorAgentPage = z.infer<typeof CollectorAgentPageSchema>
export type CollectorRegistrationCode = z.infer<typeof CollectorRegistrationCodeSchema>
