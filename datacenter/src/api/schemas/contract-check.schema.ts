import { z } from 'zod'
import { IdSchema } from './common.schema'

export const ContractCheckStatusSchema = z.enum(['passed', 'warning', 'pending', 'failed'])
export type ContractCheckStatus = z.infer<typeof ContractCheckStatusSchema>

export const ContractCheckSummarySchema = z
  .object({
    passed: z.number().int().nonnegative().default(0),
    warning: z.number().int().nonnegative().default(0),
    pending: z.number().int().nonnegative().default(0),
    failed: z.number().int().nonnegative().default(0),
  })
  .passthrough()

export const ContractCheckItemSchema = z
  .object({
    module: z.string(),
    objectType: z.string(),
    objectId: IdSchema.optional(),
    status: ContractCheckStatusSchema,
    title: z.string(),
    detail: z.string().optional(),
    action: z.string().optional(),
  })
  .passthrough()

export const ContractCheckResultSchema = z
  .object({
    status: ContractCheckStatusSchema,
    projectId: IdSchema,
    scope: z.string(),
    summary: ContractCheckSummarySchema,
    list: z.array(ContractCheckItemSchema).default([]),
    checkedAt: z.string(),
  })
  .passthrough()

export const ContractCheckRunSchema = z
  .object({
    id: IdSchema,
    projectId: IdSchema,
    scope: z.string(),
    objectType: z.string().optional().nullable(),
    objectId: IdSchema.optional().nullable(),
    status: ContractCheckStatusSchema,
    summary: ContractCheckSummarySchema,
    result: z.unknown().optional(),
    createdBy: z.string().optional().nullable(),
    createdAt: z.string(),
  })
  .passthrough()

export const ContractCheckRunPageSchema = z.object({
  list: z.array(ContractCheckRunSchema).default([]),
  pagination: z
    .object({
      page: z.number().int().positive(),
      pageSize: z.number().int().positive(),
      total: z.number().int().nonnegative(),
      totalPages: z.number().int().nonnegative(),
    })
    .passthrough(),
})

export type ContractCheckItem = z.infer<typeof ContractCheckItemSchema>
export type ContractCheckResult = z.infer<typeof ContractCheckResultSchema>
export type ContractCheckRun = z.infer<typeof ContractCheckRunSchema>
export type ContractCheckRunPage = z.infer<typeof ContractCheckRunPageSchema>
