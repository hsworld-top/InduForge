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
  features: z.array(z.string()),
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
  code: z.string(),
  configurationState: z.enum(['ready', 'incomplete']),
  isEnabled: z.boolean().default(true),
  defaultAcquisition: JsonObjectSchema.nullish().transform((value) => value ?? {}),
  lastTestStatus: z.string().nullable().optional(),
  lastTestedAt: z.string().nullable().optional(),
  displayOrder: z.number().int(),
  protocolFamily: z.string(),
  driverId: z.string(),
  driverVersion: z.string(),
  schemaVersion: z.number().int().positive(),
  config: JsonObjectSchema.nullish().transform((value) => value ?? {}),
  metadata: JsonObjectSchema.nullish().transform((value) => value ?? {}),
  secretStatus: z
    .record(z.string(), z.boolean())
    .nullish()
    .transform((value) => value ?? {}),
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

export const CollectorPointDebugSnapshotSchema = z.object({
  value: z.unknown().nullable(),
  valueText: z.string().nullable(),
  dataType: z.string().nullable(),
  quality: z.string().nullable(),
  sourceTimestamp: z.string().nullable(),
  serverTimestamp: z.string().nullable(),
  readAt: z.string().nullable(),
  lastAttemptStatus: z.enum(['succeeded', 'failed']),
  lastAttemptAt: z.string(),
  lastErrorCode: z.string().nullable(),
  lastErrorMessage: z.string().nullable(),
})

export const CollectorPointReadValueSchema = z.object({
  pointId: z.string(),
  succeeded: z.boolean(),
  value: z.unknown().nullable(),
  dataType: z.string().nullable(),
  quality: z.string().nullable(),
  sourceTimestamp: z.string().nullable(),
  serverTimestamp: z.string().nullable(),
  errorCode: z.string().nullable(),
  errorMessage: z.string().nullable(),
})

export const CollectorPointReadResultSchema = z.object({
  values: z.array(CollectorPointReadValueSchema),
  diagnostics: z.array(z.unknown()).default([]),
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
  acquisitionMode: z.enum(['inherit', 'override']).default('inherit'),
  acquisitionOverrides: JsonObjectSchema.default({}),
  enabled: z.boolean(),
  sortOrder: z.number().int(),
  metadata: JsonObjectSchema,
  latestDebugSnapshot: CollectorPointDebugSnapshotSchema.nullable(),
})

export const CollectorAddressNormalizationSchema = z.object({
  address: JsonObjectSchema,
  addressText: z.string(),
  allowedDataTypes: z.array(z.string()),
  errors: z.array(z.object({ field: z.string(), message: z.string() })).default([]),
})

export const CollectorConnectionDiagnosticSchema = z.object({
  agent: z.object({
    id: z.string().optional(),
    name: z.string().optional(),
    version: z.string().optional(),
    lastSeenAt: z.string().nullable().optional(),
    ready: z.boolean(),
    reason: z.string().default(''),
  }),
  lastTest: z.object({
    status: z.string().default('not_tested'),
    testedAt: z.string().nullable().optional(),
    durationMs: z.number().nonnegative().default(0),
    errorCode: z.string().default(''),
    message: z.string().default(''),
  }),
  pointCount: z.number().int().nonnegative(),
  attemptedPointCount: z.number().int().nonnegative(),
  succeededPointCount: z.number().int().nonnegative(),
  failedPointCount: z.number().int().nonnegative(),
  recentReadSuccessRate: z.number().min(0).max(1).nullable(),
  failedPoints: z
    .array(
      z.object({
        pointId: z.string(),
        name: z.string(),
        addressText: z.string(),
        errorCode: z.string().default(''),
        errorMessage: z.string(),
        attemptedAt: z.string(),
      }),
    )
    .default([]),
})

export const CollectorPointPageSchema = z.object({
  list: z.array(CollectorPointSchema),
  pagination: CollectorPaginationSchema,
})

export const CollectorPointBatchFailureSchema = z.object({
  index: z.number().int().nonnegative(),
  name: z.string(),
  code: z.string(),
  message: z.string(),
})

export const CollectorPointBatchResultSchema = z.object({
  list: z.array(CollectorPointSchema),
  failed: z.array(CollectorPointBatchFailureSchema),
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
export type CollectorPointDebugSnapshot = z.infer<typeof CollectorPointDebugSnapshotSchema>
export type CollectorPointReadResult = z.infer<typeof CollectorPointReadResultSchema>
export type CollectorPointReadValue = z.infer<typeof CollectorPointReadValueSchema>
export type CollectorPointBatchFailure = z.infer<typeof CollectorPointBatchFailureSchema>
export type CollectorPointBatchResult = z.infer<typeof CollectorPointBatchResultSchema>
export type CollectorImportPreview = z.infer<typeof CollectorImportPreviewSchema>
export type CollectorTask = z.infer<typeof CollectorTaskSchema>
export type CollectorAddressNormalization = z.infer<typeof CollectorAddressNormalizationSchema>
export type CollectorConnectionDiagnostic = z.infer<typeof CollectorConnectionDiagnosticSchema>
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

export type CollectorUISchema = Record<string, unknown> & {
  addressHelper?: 'siemens' | 'modbus' | 'mitsubishi' | 'omron' | 'allen-bradley'
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
