import { z } from 'zod'
import { PaginationSchema, TimeFieldSchema } from './common.schema'

const ObjectRecordSchema = z.record(z.string(), z.unknown())

export const AlarmPolicyModeSchema = z.enum(['per_target', 'derived'])
export const AlarmSeveritySchema = z.enum(['info', 'warning', 'major', 'critical'])
export const AlarmConditionKindSchema = z.enum([
  'threshold',
  'range',
  'state',
  'transition',
  'text_match',
  'rate_of_change',
  'deviation',
  'offline',
  'expression',
])
export const AlarmNotificationModeSchema = z.enum(['inherit', 'off', 'custom'])
export const AlarmChannelTypeSchema = z.enum(['runtime_inapp', 'webhook', 'dingtalk', 'wecom'])

export type AlarmPolicyMode = z.infer<typeof AlarmPolicyModeSchema>
export type AlarmSeverity = z.infer<typeof AlarmSeveritySchema>
export type AlarmConditionKind = z.infer<typeof AlarmConditionKindSchema>
export type AlarmNotificationMode = z.infer<typeof AlarmNotificationModeSchema>
export type AlarmChannelType = z.infer<typeof AlarmChannelTypeSchema>

export const AlarmBindingSchema = z.object({
  datapointId: z.string(),
  path: z.string().default(''),
  name: z.string().default(''),
  dataType: z.string().default(''),
  role: z.enum(['target', 'input']),
  inputKey: z.string().nullable().optional(),
})
export type AlarmBinding = z.infer<typeof AlarmBindingSchema>

export const AlarmConditionSchema = z.object({
  id: z.string().default(''),
  kind: AlarmConditionKindSchema,
  operator: z.string(),
  label: z.string().default(''),
  severity: AlarmSeveritySchema,
  params: ObjectRecordSchema.default({}),
  triggerDelayMs: z.number().int().nonnegative().default(0),
  clearDelayMs: z.number().int().nonnegative().default(0),
  deadband: z.number().nonnegative().default(0),
})
export type AlarmCondition = z.infer<typeof AlarmConditionSchema>

export const AlarmNotificationSchema = z.object({
  mode: AlarmNotificationModeSchema.default('inherit'),
  notifyOnRaise: z.boolean().nullable().optional(),
  notifyOnClear: z.boolean().nullable().optional(),
  repeatIntervalSeconds: z.number().int().positive().nullable().optional(),
  channelIds: z.array(z.string()).default([]),
  messageTemplate: z.string().default(''),
})
export type AlarmNotification = z.infer<typeof AlarmNotificationSchema>

export const AlarmPolicyGroupSchema = z.object({
  id: z.string(),
  projectId: z.string(),
  name: z.string(),
  parentId: z.string().nullable().optional(),
  description: z.string().nullable().optional(),
  sortOrder: z.number().int().default(0),
  fullPath: z.string().default(''),
  hasChildren: z.boolean().default(false),
  createdAt: TimeFieldSchema,
  updatedAt: TimeFieldSchema,
})
export type AlarmPolicyGroup = z.infer<typeof AlarmPolicyGroupSchema>

export const AlarmPolicyGroupSaveSchema = z.object({
  name: z.string().min(1),
  parentId: z.string().nullable().optional(),
  description: z.string().nullable().optional(),
  sortOrder: z.number().int().default(0),
})
export type AlarmPolicyGroupSave = z.infer<typeof AlarmPolicyGroupSaveSchema>

export const AlarmPolicySchema = z.object({
  id: z.string(),
  projectId: z.string(),
  groupId: z.string().nullable().optional(),
  groupName: z.string().nullable().optional(),
  name: z.string(),
  description: z.string().nullable().optional(),
  mode: AlarmPolicyModeSchema,
  derivedExpression: z.string().default(''),
  bindings: z.array(AlarmBindingSchema).default([]),
  conditions: z.array(AlarmConditionSchema).default([]),
  notification: AlarmNotificationSchema,
  isEnabled: z.boolean(),
  revision: z.number().int().positive(),
  contract: ObjectRecordSchema.default({}),
  createdAt: TimeFieldSchema,
  updatedAt: TimeFieldSchema,
})
export type AlarmPolicy = z.infer<typeof AlarmPolicySchema>

export const AlarmPolicySaveSchema = z.object({
  groupId: z.string().nullable().optional(),
  name: z.string().min(1),
  description: z.string().nullable().optional(),
  mode: AlarmPolicyModeSchema,
  bindings: z.array(AlarmBindingSchema),
  derivedExpression: z.string().default(''),
  conditions: z.array(AlarmConditionSchema).min(1),
  notification: AlarmNotificationSchema,
  isEnabled: z.boolean(),
})
export type AlarmPolicySave = z.infer<typeof AlarmPolicySaveSchema>

export const AlarmPolicyListSchema = z.object({
  list: z.array(AlarmPolicySchema).default([]),
  pagination: PaginationSchema.default({ page: 1, pageSize: 20, total: 0 }),
})
export type AlarmPolicyList = z.infer<typeof AlarmPolicyListSchema>

export const AlarmPolicyGroupListSchema = z.object({
  list: z.array(AlarmPolicyGroupSchema).default([]),
  pagination: PaginationSchema.default({ page: 1, pageSize: 50, total: 0 }),
})

export const AlarmProjectSettingsSchema = z.object({
  projectId: z.string(),
  notifyOnRaise: z.boolean(),
  notifyOnClear: z.boolean(),
  repeatIntervalSeconds: z.number().int().positive().nullable().optional(),
  defaultMessageTemplate: z.string(),
  defaultChannelIds: z.array(z.string()).default([]),
  createdAt: TimeFieldSchema,
  updatedAt: TimeFieldSchema,
})
export type AlarmProjectSettings = z.infer<typeof AlarmProjectSettingsSchema>

export const AlarmProjectSettingsSaveSchema = AlarmProjectSettingsSchema.pick({
  notifyOnRaise: true,
  notifyOnClear: true,
  repeatIntervalSeconds: true,
  defaultMessageTemplate: true,
  defaultChannelIds: true,
})
export type AlarmProjectSettingsSave = z.infer<typeof AlarmProjectSettingsSaveSchema>

export const AlarmHistorySettingsSchema = z.object({
  projectId: z.string(),
  isEnabled: z.boolean(),
  retentionDays: z.number().int().positive().nullable(),
  storeNotificationDeliveries: z.boolean(),
  createdAt: TimeFieldSchema,
  updatedAt: TimeFieldSchema,
})
export type AlarmHistorySettings = z.infer<typeof AlarmHistorySettingsSchema>

export const AlarmHistorySettingsSaveSchema = AlarmHistorySettingsSchema.pick({
  isEnabled: true,
  retentionDays: true,
  storeNotificationDeliveries: true,
})
export type AlarmHistorySettingsSave = z.infer<typeof AlarmHistorySettingsSaveSchema>

export const AlarmNotificationChannelSchema = z.object({
  id: z.string(),
  projectId: z.string(),
  name: z.string(),
  channelType: AlarmChannelTypeSchema,
  config: ObjectRecordSchema.default({}),
  secretStatus: ObjectRecordSchema.default({}),
  isEnabled: z.boolean(),
  createdAt: TimeFieldSchema.optional(),
  updatedAt: TimeFieldSchema.optional(),
})
export type AlarmNotificationChannel = z.infer<typeof AlarmNotificationChannelSchema>

export const AlarmNotificationChannelSaveSchema = z.object({
  name: z.string().min(1),
  channelType: z.enum(['webhook', 'dingtalk', 'wecom']),
  config: ObjectRecordSchema,
  secrets: z.record(z.string(), z.string()).default({}),
  deleteSecretKeys: z.array(z.string()).default([]),
  isEnabled: z.boolean().default(true),
})
export type AlarmNotificationChannelSave = z.infer<typeof AlarmNotificationChannelSaveSchema>

export const AlarmDatapointSummarySchema = z.object({
  datapointId: z.string(),
  policies: z.array(AlarmPolicySchema).default([]),
  count: z.number().int().nonnegative(),
})
export type AlarmDatapointSummary = z.infer<typeof AlarmDatapointSummarySchema>

export const AlarmTrialResultSchema = z.object({
  triggered: z.boolean(),
  state: z.enum(['triggered', 'not_triggered', 'insufficient_input']),
  triggeredConditions: z.array(AlarmConditionSchema).default([]),
  conditionResults: z.array(ObjectRecordSchema).default([]),
  message: z.string().optional(),
})
export type AlarmTrialResult = z.infer<typeof AlarmTrialResultSchema>

export const AlarmDraftValidationSchema = z.object({
  valid: z.boolean(),
  errors: z.array(z.string()).default([]),
})

export const AlarmPolicyContractSchema = ObjectRecordSchema
