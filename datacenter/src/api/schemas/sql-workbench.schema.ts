import { z } from 'zod'
import { CanonicalDataPointTypeSchema, SourceOutputSchema } from './source-output.schema'
import { IdSchema, TimeFieldSchema, listResponseSchema } from './common.schema'

export const SqlResultLimitsSchema = z
  .object({
    maxRows: z.number().int().positive().default(500),
    maxBytes: z
      .number()
      .int()
      .positive()
      .default(5 * 1024 * 1024),
    timeoutSeconds: z.number().int().positive().default(30),
  })
  .default({ maxRows: 500, maxBytes: 5 * 1024 * 1024, timeoutSeconds: 30 })

export const SqlWorkbenchResultSchema = z
  .object({
    columns: z.array(z.string()).default([]),
    columnTypes: z.preprocess(
      (value) => value ?? {},
      z.record(z.string(), CanonicalDataPointTypeSchema),
    ),
    rows: z.array(z.array(z.unknown())).default([]),
    rowCount: z.number().int().nonnegative().default(0),
    executionTime: z.number().nonnegative().default(0),
    truncated: z.boolean().default(false),
    truncatedBy: z.enum(['rows', 'bytes']).optional().or(z.literal('')),
    limits: SqlResultLimitsSchema,
  })
  .passthrough()

export type SqlWorkbenchResult = z.infer<typeof SqlWorkbenchResultSchema>

export const SavedQuerySchema = z
  .object({
    id: IdSchema,
    projectId: IdSchema,
    connectionId: IdSchema,
    name: z.string(),
    description: z.string().nullable().optional(),
    groupId: IdSchema.nullable().optional(),
    queryType: z.string(),
    config: z.record(z.string(), z.unknown()).default({}),
    isEnabled: z.boolean().default(true),
    outputs: z.array(SourceOutputSchema).default([]),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export const SavedQueryListSchema = listResponseSchema(SavedQuerySchema)
export type SavedQuery = z.infer<typeof SavedQuerySchema>
