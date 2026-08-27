import { z } from 'zod'
import { IdSchema, TimeFieldSchema, listResponseSchema } from './common.schema'
import { SourceOutputSchema } from './source-output.schema'

const KeyValueRowSchema = z
  .object({
    enabled: z.boolean().default(true),
    key: z.string().default(''),
    value: z.string().default(''),
    description: z.string().default(''),
  })
  .passthrough()

export const HttpRequestGroupSchema = z
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

export const HttpRequestSchema = z
  .object({
    id: IdSchema,
    projectId: IdSchema.optional(),
    connectionId: IdSchema,
    groupId: IdSchema.nullable().optional(),
    name: z.string(),
    method: z.enum(['GET', 'POST', 'PUT', 'PATCH', 'DELETE']).default('GET'),
    url: z.string().default(''),
    params: z.array(KeyValueRowSchema).default([]),
    headers: z.array(KeyValueRowSchema).default([]),
    auth: z.record(z.string(), z.unknown()).default({ type: 'none' }),
    bodyType: z.enum(['none', 'json', 'raw', 'form-data', 'x-www-form-urlencoded']).default('none'),
    body: z.record(z.string(), z.unknown()).default({}),
    settings: z.record(z.string(), z.unknown()).default({}),
    enabled: z.boolean().default(true),
    sortOrder: z.number().default(0),
    sourceType: z.string().optional(),
    dataPointId: IdSchema.optional().or(z.literal('')),
    dataPointPath: z.string().optional(),
    outputs: z.array(SourceOutputSchema).min(1),
    lastResponse: z.unknown().optional().nullable(),
    quality: z.enum(['good', 'bad', 'unknown']).default('unknown'),
    lastSentAt: TimeFieldSchema,
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export const HttpSendResponseSchema = z
  .object({
    status: z.number(),
    statusText: z.string(),
    headers: z.record(z.string(), z.string()).default({}),
    body: z.unknown(),
    rawBody: z.string().default(''),
    durationMs: z.number().default(0),
    sizeBytes: z.number().default(0),
    receivedAt: TimeFieldSchema,
  })
  .passthrough()

export const HttpRequestGroupListSchema = listResponseSchema(HttpRequestGroupSchema)
export const HttpRequestListSchema = listResponseSchema(HttpRequestSchema)

export type HttpRequestGroup = z.infer<typeof HttpRequestGroupSchema>
export type HttpRequest = z.infer<typeof HttpRequestSchema>
export type HttpSendResponse = z.infer<typeof HttpSendResponseSchema>
export type HttpKeyValueRow = z.infer<typeof KeyValueRowSchema>
