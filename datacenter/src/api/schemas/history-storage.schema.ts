import { z } from 'zod'

export const HistoryStorageScopeTypeSchema = z.enum(['access_source', 'collector_connection'])
export const HistoryStorageWriteModeSchema = z.enum([
  'every_sample',
  'interval_latest',
  'on_change',
  'periodic_snapshot',
])
export const HistoryStorageBehaviorSchema = z.enum(['inherit', 'off', 'custom'])

export const HistoryStorageScopeSchema = z.object({
  type: HistoryStorageScopeTypeSchema,
  id: z.string(),
  name: z.string().optional(),
  sourceType: z.string().optional(),
})

export const HistoryStorageTargetSchema = z.object({
  connectionId: z.string(),
  connectionName: z.string(),
  connectionType: z.enum(['builtin.timeseries', 'tdengine']),
  connectionStatus: z.string(),
  isPrimary: z.boolean(),
  sortOrder: z.number().int().nonnegative(),
  retentionDays: z.number().int().positive().nullable(),
})

export const HistoryStorageConfigurationSchema = z.object({
  writeMode: HistoryStorageWriteModeSchema,
  intervalMs: z.number().int().positive().nullable().optional(),
  deadband: z.number().nonnegative().nullable().optional(),
  maxSilenceMs: z.number().int().positive().nullable().optional(),
  offlineBehavior: z.enum(['store_stale', 'skip']),
  targets: z.array(HistoryStorageTargetSchema),
})

export const HistoryStorageSourceItemSchema = z.object({
  scope: HistoryStorageScopeSchema,
  datapointCount: z.number().int().nonnegative(),
  pointOverrideCount: z.number().int().nonnegative(),
  historyState: z.enum(['enabled', 'disabled']),
  writeMode: HistoryStorageWriteModeSchema.nullable().optional(),
  targetCount: z.number().int().nonnegative(),
  primaryTargetName: z.string().nullable().optional(),
  primaryTargetType: z.string().nullable().optional(),
  retentionDays: z.number().int().positive().nullable().optional(),
})

export const HistoryStoragePaginationSchema = z.object({
  page: z.number().int().positive(),
  pageSize: z.number().int().positive(),
  total: z.number().int().nonnegative(),
  totalPages: z.number().int().nonnegative(),
})

export const HistoryStorageSourceListSchema = z.object({
  list: z.array(HistoryStorageSourceItemSchema),
  pagination: HistoryStoragePaginationSchema,
})

export const HistoryStorageSourceDetailSchema = z.object({
  scope: HistoryStorageScopeSchema,
  behavior: z.enum(['off', 'custom']),
  configuration: HistoryStorageConfigurationSchema.nullable().optional(),
  pointOverrideCount: z.number().int().nonnegative(),
})

export const HistoryStorageDatapointDetailSchema = z.object({
  datapointId: z.string(),
  datapointName: z.string(),
  datapointPath: z.string(),
  dataType: z.string(),
  behavior: HistoryStorageBehaviorSchema,
  effectiveEnabled: z.boolean(),
  source: HistoryStorageScopeSchema.nullable().optional(),
  configuration: HistoryStorageConfigurationSchema.nullable().optional(),
})

export const HistoryStorageTargetOptionSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.enum(['builtin.timeseries', 'tdengine']),
  status: z.string(),
  createdAt: z.string().optional(),
  updatedAt: z.string().optional(),
})

export const HistoryStorageTargetsSchema = z.object({
  targets: z.array(HistoryStorageTargetOptionSchema),
})

export const HistoryStorageBatchResultSchema = z.object({
  updatedCount: z.number().int().nonnegative(),
})

export type HistoryStorageScopeType = z.infer<typeof HistoryStorageScopeTypeSchema>
export type HistoryStorageWriteMode = z.infer<typeof HistoryStorageWriteModeSchema>
export type HistoryStorageBehavior = z.infer<typeof HistoryStorageBehaviorSchema>
export type HistoryStorageConfiguration = z.infer<typeof HistoryStorageConfigurationSchema>
export type HistoryStorageSourceItem = z.infer<typeof HistoryStorageSourceItemSchema>
export type HistoryStorageSourceDetail = z.infer<typeof HistoryStorageSourceDetailSchema>
export type HistoryStorageDatapointDetail = z.infer<typeof HistoryStorageDatapointDetailSchema>
export type HistoryStorageTargetOption = z.infer<typeof HistoryStorageTargetOptionSchema>
