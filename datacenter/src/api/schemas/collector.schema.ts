import { z } from 'zod'

const JsonObjectSchema = z.record(z.string(), z.unknown())

export const CollectorPaginationSchema = z.object({
  page: z.number().int().positive(),
  pageSize: z.number().int().positive(),
  total: z.number().int().nonnegative(),
  totalPages: z.number().int().nonnegative(),
})

export const CollectorDriverSummarySchema = z.object({
  protocolFamily: z.string(),
  driverId: z.string(),
  driverVersion: z.string(),
  schemaVersion: z.number().int().positive(),
  displayName: z.string(),
  category: z.string(),
  transports: z.array(z.string()),
  operations: z.array(z.string()),
  dataTypes: z.array(z.string()),
  acquisitionModes: z.array(z.string()),
  platforms: z.record(z.string(), z.array(z.string())),
})

export const CollectorDriverDetailSchema = CollectorDriverSummarySchema.extend({
  connectionSchema: JsonObjectSchema,
  addressSchema: JsonObjectSchema,
  uiSchema: JsonObjectSchema,
})

export const CollectorDriverPageSchema = z.object({
  list: z.array(CollectorDriverSummarySchema),
  pagination: CollectorPaginationSchema,
})

export const CollectorConnectionSchema = z.object({
  id: z.string(),
  projectId: z.string(),
  name: z.string(),
  status: z.string(),
  lastTestStatus: z.string().nullable().optional(),
  lastTestedAt: z.string().nullable().optional(),
  displayOrder: z.number().int(),
  protocolFamily: z.string(),
  driverId: z.string(),
  driverVersion: z.string(),
  schemaVersion: z.number().int().positive(),
  config: JsonObjectSchema,
  metadata: JsonObjectSchema,
  secretStatus: z.record(z.string(), z.boolean()),
  createdAt: z.string(),
  updatedAt: z.string(),
})

export const CollectorConnectionPageSchema = z.object({
  list: z.array(CollectorConnectionSchema),
  pagination: CollectorPaginationSchema,
})

export const CollectorPointGroupSchema = z.object({
  id: z.string(),
  parentId: z.string().nullable(),
  name: z.string(),
  sortOrder: z.number().int(),
  metadata: JsonObjectSchema,
})

export const CollectorPointSchema = z.object({
  id: z.string(),
  groupId: z.string().nullable(),
  code: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  address: JsonObjectSchema,
  addressText: z.string(),
  addressSchemaVersion: z.number().int().positive(),
  dataType: z.string(),
  elementCount: z.number().int().positive(),
  readOptions: JsonObjectSchema,
  acquisition: JsonObjectSchema,
  enabled: z.boolean(),
  sortOrder: z.number().int(),
  metadata: JsonObjectSchema,
})

export const CollectorPointPageSchema = z.object({
  list: z.array(CollectorPointSchema),
  pagination: CollectorPaginationSchema,
})

export const CollectorImportErrorSchema = z.object({
  row: z.number().int().positive(),
  field: z.string(),
  code: z.string(),
  message: z.string(),
})

export const CollectorImportPreviewSchema = z.object({
  importId: z.string(),
  totalRows: z.number().int().nonnegative(),
  validRows: z.number().int().nonnegative(),
  candidates: z.array(CollectorPointSchema),
  errors: z.array(CollectorImportErrorSchema),
  pagination: CollectorPaginationSchema,
  expiresAt: z.string(),
})

export const CollectorTaskSchema = z.object({
  taskId: z.string(),
  projectId: z.string(),
  connectionId: z.string(),
  agentId: z.string(),
  operation: z.string(),
  status: z.string(),
  request: z.unknown().nullable().optional(),
  result: z.unknown().nullable().optional(),
  errorCode: z.string().nullable(),
  errorMessage: z.string().nullable(),
  deadlineAt: z.string(),
  claimedAt: z.string().nullable(),
  finishedAt: z.string().nullable(),
  createdAt: z.string(),
})

export const CollectorProtocolCapabilitySchema = z.object({
  driverId: z.string(),
  driverVersion: z.string(),
  schemaVersions: z.array(z.number().int().positive()),
  operations: z.array(z.string()),
})

export type CollectorDriverSummary = z.infer<typeof CollectorDriverSummarySchema>
export type CollectorDriverDetail = z.infer<typeof CollectorDriverDetailSchema>
export type CollectorConnection = z.infer<typeof CollectorConnectionSchema>
export type CollectorPointGroup = z.infer<typeof CollectorPointGroupSchema>
export type CollectorPoint = z.infer<typeof CollectorPointSchema>
export type CollectorImportPreview = z.infer<typeof CollectorImportPreviewSchema>
export type CollectorTask = z.infer<typeof CollectorTaskSchema>
export type CollectorProtocolCapability = z.infer<typeof CollectorProtocolCapabilitySchema>

export type CollectorJsonSchemaProperty = {
  type?: 'string' | 'number' | 'integer' | 'boolean' | 'object'
  title?: string
  description?: string
  enum?: unknown[]
  default?: unknown
  minimum?: number
  maximum?: number
  properties?: Record<string, CollectorJsonSchemaProperty>
  required?: string[]
  'x-induforge-secret'?: boolean
  'x-induforge-advanced'?: boolean
  'x-induforge-unit'?: string
  'x-induforge-enum-labels'?: Record<string, string>
}

export type CollectorJsonSchema = CollectorJsonSchemaProperty & {
  properties?: Record<string, CollectorJsonSchemaProperty>
  required?: string[]
}

export function splitCollectorFormValues(
  schema: CollectorJsonSchema,
  values: Record<string, unknown>,
) {
  const config: Record<string, unknown> = {}
  const secrets: Record<string, string> = {}
  for (const [name, value] of Object.entries(values)) {
    if (schema.properties?.[name]?.['x-induforge-secret']) {
      if (value !== undefined && value !== null && value !== '') secrets[name] = String(value)
      continue
    }
    config[name] = value
  }
  return { config, secrets }
}
