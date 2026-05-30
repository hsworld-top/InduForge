import { z } from 'zod'
import { IdSchema, TimeFieldSchema, listResponseSchema } from './common.schema'

const KeyValueRowSchema = z
  .object({
    enabled: z.boolean().default(true),
    key: z.string().default(''),
    value: z.string().default(''),
    description: z.string().default(''),
  })
  .passthrough()

const ProtocolRowSchema = z
  .object({
    enabled: z.boolean().default(true),
    value: z.string().default(''),
    description: z.string().default(''),
  })
  .passthrough()

const SubscribeMessageRowSchema = z
  .object({
    enabled: z.boolean().default(true),
    name: z.string().default(''),
    payload: z.string().default(''),
    description: z.string().default(''),
  })
  .passthrough()

export const WebSocketSessionGroupSchema = z
  .object({
    id: IdSchema,
    projectId: IdSchema.optional(),
    connectionId: IdSchema,
    parentId: IdSchema.nullable().optional(),
    name: z.string(),
    sortOrder: z.number().default(0),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export const WebSocketSessionSchema = z
  .object({
    id: IdSchema,
    projectId: IdSchema.optional(),
    connectionId: IdSchema,
    groupId: IdSchema.nullable().optional(),
    name: z.string(),
    url: z.string().default(''),
    headers: z.array(KeyValueRowSchema).default([]),
    auth: z.record(z.string(), z.unknown()).default({ type: 'none' }),
    protocols: z.array(ProtocolRowSchema).default([]),
    messages: z.array(SubscribeMessageRowSchema).default([]),
    settings: z.record(z.string(), z.unknown()).default({}),
    enabled: z.boolean().default(true),
    sortOrder: z.number().default(0),
    sourceType: z.literal('websocket.session').or(z.string()).optional(),
    dataPointId: IdSchema.optional().or(z.literal('')),
    dataPointPath: z.string().optional(),
    lastMessage: z.unknown().optional().nullable(),
    lastDiagnostic: z.string().default(''),
    quality: z.enum(['good', 'bad', 'unknown']).default('unknown'),
    lastMessageAt: TimeFieldSchema,
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export const WebSocketPreviewMessageSchema = z
  .object({
    direction: z.enum(['in', 'out']).or(z.string()),
    type: z.string(),
    payload: z.unknown(),
    rawPayload: z.string().default(''),
    sizeBytes: z.number().default(0),
    timestamp: TimeFieldSchema,
  })
  .passthrough()

export const WebSocketPreviewResponseSchema = z
  .object({
    status: z.enum(['ok', 'error']).or(z.string()),
    messages: z.array(WebSocketPreviewMessageSchema).default([]),
    diagnostics: z.record(z.string(), z.unknown()).default({}),
    durationMs: z.number().default(0),
    truncated: z.boolean().default(false),
  })
  .passthrough()

export const WebSocketSessionGroupListSchema = listResponseSchema(WebSocketSessionGroupSchema)
export const WebSocketSessionListSchema = listResponseSchema(WebSocketSessionSchema)

export type WebSocketSessionGroup = z.infer<typeof WebSocketSessionGroupSchema>
export type WebSocketSession = z.infer<typeof WebSocketSessionSchema>
export type WebSocketPreviewMessage = z.infer<typeof WebSocketPreviewMessageSchema>
export type WebSocketPreviewResponse = z.infer<typeof WebSocketPreviewResponseSchema>
export type WebSocketKeyValueRow = z.infer<typeof KeyValueRowSchema>
export type WebSocketProtocolRow = z.infer<typeof ProtocolRowSchema>
export type WebSocketSubscribeMessageRow = z.infer<typeof SubscribeMessageRowSchema>
