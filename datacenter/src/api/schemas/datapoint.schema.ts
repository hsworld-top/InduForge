import { z } from 'zod'
import { IdSchema, TimeFieldSchema, RuntimeGrantSchema } from './common.schema'

// 数据点状态枚举
export const DatapointStatusSchema = z.enum(['active', 'inactive', 'invalid', 'error', 'unknown'])

export type DatapointStatus = z.infer<typeof DatapointStatusSchema>

// 单个数据点
export const DatapointSchema = z
  .object({
    id: IdSchema,
    path: z.string(),
    name: z.string(),
    sourceType: z.string().optional(),
    dataType: z.string().optional(),
    unit: z.string().optional().nullable(),
    description: z.string().optional().nullable(),
    status: DatapointStatusSchema.optional(),
    sourceError: z.string().optional().nullable(),
    invalidReason: z.string().optional().nullable(),
    runtimeGrant: RuntimeGrantSchema.optional(),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export type Datapoint = z.infer<typeof DatapointSchema>

// 数据点更新参数
export const DatapointUpdateSchema = z
  .object({
    name: z.string().optional(),
    description: z.string().optional().nullable(),
    unit: z.string().optional().nullable(),
    dataType: z.string().optional(),
  })
  .passthrough()

export type DatapointUpdate = z.infer<typeof DatapointUpdateSchema>
