import { z } from 'zod'
import { IdSchema, TimeFieldSchema } from './common.schema'

// 接入源类型枚举
export const AccessSourceTypeSchema = z.enum([
  'mqtt',
  'kafka',
  'http',
  'websocket',
  'redis',
  'opcua',
  'opcda',
  's7',
  'modbus',
  'tdengine',
  'database',
])

export type AccessSourceType = z.infer<typeof AccessSourceTypeSchema>

// 接入源状态
export const AccessSourceStatusSchema = z.enum(['connected', 'disconnected', 'error', 'unknown'])

export type AccessSourceStatus = z.infer<typeof AccessSourceStatusSchema>

// 接入源摘要（列表项）
export const AccessSourceSummarySchema = z
  .object({
    id: IdSchema,
    name: z.string(),
    type: AccessSourceTypeSchema.or(z.string()).optional(),
    status: AccessSourceStatusSchema.or(z.string()).optional(),
    description: z.string().optional().nullable(),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export type AccessSourceSummary = z.infer<typeof AccessSourceSummarySchema>

// 接入源详情（含配置，各协议差异大，用 passthrough）
export const AccessSourceDetailSchema = AccessSourceSummarySchema.extend({
  config: z.record(z.string(), z.unknown()).optional(),
})

export type AccessSourceDetail = z.infer<typeof AccessSourceDetailSchema>

// 创建接入源参数
export const AccessSourceCreateSchema = z
  .object({
    name: z.string(),
    type: z.string(),
    config: z.record(z.string(), z.unknown()).optional(),
    description: z.string().optional().nullable(),
  })
  .passthrough()

export type AccessSourceCreate = z.infer<typeof AccessSourceCreateSchema>
