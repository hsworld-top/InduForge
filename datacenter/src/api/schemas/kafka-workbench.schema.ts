import { z } from 'zod'
import { IdSchema, TimeFieldSchema, listResponseSchema } from './common.schema'

export const KafkaTopicGroupSchema = z
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

export const KafkaTopicMappingSchema = z
  .object({
    id: IdSchema,
    projectId: IdSchema.optional(),
    connectionId: IdSchema,
    groupId: IdSchema.nullable().optional(),
    name: z.string(),
    topic: z.string(),
    description: z.string().default(''),
    partitionMode: z.enum(['all', 'single']).default('all'),
    partition: z.number().nullable().optional(),
    startPosition: z.enum(['latest', 'earliest', 'offset']).default('latest'),
    decode: z.enum(['json', 'string', 'binary']).default('json'),
    sampleLimit: z.number().default(100),
    timeoutMs: z.number().default(5000),
    sortOrder: z.number().default(0),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export const KafkaFieldSchema = z
  .object({
    id: IdSchema,
    projectId: IdSchema.optional(),
    connectionId: IdSchema,
    topicMappingId: IdSchema,
    name: z.string(),
    valuePath: z.string(),
    keyPath: z.string().default(''),
    dataType: z.string(),
    enabled: z.boolean().default(true),
    description: z.string().default(''),
    sortOrder: z.number().default(0),
    sourceType: z.string().optional(),
    dataPointId: IdSchema.optional().or(z.literal('')),
    dataPointPath: z.string().optional(),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export const KafkaPreviewSampleSchema = z
  .object({
    topic: z.string().optional(),
    partition: z.number().optional(),
    offset: z.number().optional(),
    key: z.string().optional(),
    headers: z.record(z.string(), z.string()).optional(),
    value: z.unknown(),
    rawPayload: z.string().optional(),
    timestamp: z.string().optional(),
  })
  .passthrough()

export const KafkaSchemaFieldSchema = z.object({
  path: z.string(),
  type: z.string(),
})

const KafkaPreviewSchemaPayload = z.union([
  z.array(KafkaSchemaFieldSchema),
  z.record(z.string(), z.unknown()).transform((value) =>
    Object.entries(value).map(([path, type]) => ({
      path,
      type: typeof type === 'string' ? type : 'unknown',
    })),
  ),
])

export const KafkaPreviewSchema = z
  .object({
    protocol: z.string().optional(),
    connectionId: IdSchema.optional(),
    status: z.string(),
    topic: z.string().optional(),
    samples: z.array(KafkaPreviewSampleSchema).default([]),
    schema: KafkaPreviewSchemaPayload.default([]),
    diagnostics: z.record(z.string(), z.unknown()).default({}),
    durationMs: z.number().optional(),
    truncated: z.boolean().optional(),
  })
  .passthrough()

export const KafkaTopicGroupListSchema = listResponseSchema(KafkaTopicGroupSchema)
export const KafkaTopicMappingListSchema = listResponseSchema(KafkaTopicMappingSchema)
export const KafkaFieldListSchema = listResponseSchema(KafkaFieldSchema)

export type KafkaTopicGroup = z.infer<typeof KafkaTopicGroupSchema>
export type KafkaTopicMapping = z.infer<typeof KafkaTopicMappingSchema>
export type KafkaField = z.infer<typeof KafkaFieldSchema>
export type KafkaSchemaField = z.infer<typeof KafkaSchemaFieldSchema>
export type KafkaPreviewSample = z.infer<typeof KafkaPreviewSampleSchema>
export type KafkaPreview = z.infer<typeof KafkaPreviewSchema>
