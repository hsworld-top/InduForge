import { z } from 'zod'
import { IdSchema, TimeFieldSchema } from './common.schema'

export const ConnectionTestCapabilitySchema = z.object({
  status: z.enum(['supported', 'unsupported']),
  reason: z.string().optional(),
})

export const ConnectionLastTestSchema = z.object({
  status: z.enum(['not_tested', 'succeeded', 'failed']),
  testedAt: TimeFieldSchema.optional(),
  durationMs: z.number().int().nonnegative().optional(),
  message: z.string().optional(),
})

export const ConnectionSchema = z
  .object({
    id: IdSchema,
    projectId: IdSchema,
    tenantId: z.string(),
    name: z.string(),
    type: z.string(),
    enabled: z.boolean(),
    configurationState: z.enum(['ready', 'incomplete']),
    testCapability: ConnectionTestCapabilitySchema,
    lastTest: ConnectionLastTestSchema,
    config: z.record(z.string(), z.unknown()).default({}),
    relationalConfig: z
      .object({
        dbType: z.string().optional(),
        host: z.string().optional(),
        port: z.union([z.number(), z.string()]).optional(),
        database: z.string().optional(),
        username: z.string().optional(),
      })
      .passthrough()
      .optional(),
    mqttConfig: z
      .object({
        protocol: z.string().optional(),
        brokerUrl: z.string().optional(),
        host: z.string().optional(),
        port: z.union([z.number(), z.string()]).optional(),
        topic: z.string().optional(),
        defaultTopic: z.string().optional(),
      })
      .passthrough()
      .optional(),
    secretStatus: z
      .record(z.string(), z.boolean())
      .nullish()
      .transform((value) => value ?? {}),
    displayOrder: z.number().int(),
    variableCount: z.number().int().nonnegative(),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export const ConnectionPaginationSchema = z.object({
  page: z.number().int().positive(),
  pageSize: z.number().int().positive(),
  total: z.number().int().nonnegative(),
  totalPages: z.number().int().nonnegative(),
})

export const ConnectionPageSchema = z.object({
  list: z.array(ConnectionSchema),
  pagination: ConnectionPaginationSchema,
})

export const ConnectionTestResultSchema = z.object({
  connected: z.boolean(),
  dbType: z.string().default(''),
  type: z.string().optional(),
  detail: z.string().optional(),
  message: z.string(),
})

export type Connection = z.infer<typeof ConnectionSchema>
export type ConnectionPage = z.infer<typeof ConnectionPageSchema>
export type ConnectionTestResult = z.infer<typeof ConnectionTestResultSchema>
