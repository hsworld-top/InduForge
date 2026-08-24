import { z } from 'zod'
import { PaginationSchema, TimeFieldSchema } from './common.schema'

const ObjectRecordSchema = z.record(z.string(), z.unknown())

export const AlarmItemModeSchema = z.enum(['point', 'derived'])
export const AlarmEvaluationModeSchema = z.enum(['single', 'highest_matching'])
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

export type AlarmItemMode = z.infer<typeof AlarmItemModeSchema>
export type AlarmEvaluationMode = z.infer<typeof AlarmEvaluationModeSchema>
export type AlarmSeverity = z.infer<typeof AlarmSeveritySchema>
export type AlarmConditionKind = z.infer<typeof AlarmConditionKindSchema>
export type AlarmNotificationMode = z.infer<typeof AlarmNotificationModeSchema>
export type AlarmChannelType = z.infer<typeof AlarmChannelTypeSchema>

export const AlarmItemInputSchema = z.object({
  id: z.string().default(''),
  datapointId: z.string(),
  path: z.string().default(''),
  name: z.string().default(''),
  dataType: z.string().default(''),
  inputKey: z.string().min(1),
})
export type AlarmItemInput = z.infer<typeof AlarmItemInputSchema>

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

export const AlarmGroupSchema = z.object({
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
export type AlarmGroup = z.infer<typeof AlarmGroupSchema>
export const AlarmGroupSaveSchema = z.object({
  name: z.string().trim().min(1),
  parentId: z.string().nullable().optional(),
  description: z.string().nullable().optional(),
  sortOrder: z.number().int().default(0),
})
export type AlarmGroupSave = z.infer<typeof AlarmGroupSaveSchema>

export const AlarmItemSchema = z.object({
  id: z.string(),
  projectId: z.string(),
  datapointId: z.string().default(''),
  path: z.string().default(''),
  datapointName: z.string().default(''),
  dataType: z.string().default(''),
  groupId: z.string().nullable().optional(),
  groupName: z.string().nullable().optional(),
  displayName: z.string(),
  description: z.string().nullable().optional(),
  mode: AlarmItemModeSchema,
  alarmType: AlarmConditionKindSchema,
  evaluationMode: AlarmEvaluationModeSchema,
  derivedExpression: z.string().default(''),
  inputs: z.array(AlarmItemInputSchema).default([]),
  conditions: z.array(AlarmConditionSchema).default([]),
  notification: AlarmNotificationSchema,
  isEnabled: z.boolean(),
  revision: z.number().int().positive(),
  contract: ObjectRecordSchema.default({}),
  createdAt: TimeFieldSchema,
  updatedAt: TimeFieldSchema,
})
export type AlarmItem = z.infer<typeof AlarmItemSchema>

export const AlarmItemSaveSchema = z
  .object({
    itemId: z.string().optional(),
    datapointId: z.string().default(''),
    displayName: z.string().trim().max(100).default(''),
    groupId: z.string().nullable().optional(),
    description: z.string().nullable().optional(),
    mode: AlarmItemModeSchema,
    evaluationMode: AlarmEvaluationModeSchema,
    inputs: z.array(AlarmItemInputSchema).default([]),
    derivedExpression: z.string().default(''),
    conditions: z.array(AlarmConditionSchema).min(1),
    notification: AlarmNotificationSchema,
    isEnabled: z.boolean(),
    revision: z.number().int().nonnegative().default(0),
    acknowledgedWarningKeys: z.array(z.string()).default([]),
  })
  .superRefine((value, context) => {
    if (value.mode === 'point' && !value.datapointId)
      context.addIssue({ code: 'custom', path: ['datapointId'], message: '请选择数据点' })
    if (
      value.mode === 'point' &&
      value.evaluationMode === 'single' &&
      value.conditions.length !== 1
    )
      context.addIssue({
        code: 'custom',
        path: ['conditions'],
        message: '普通报警必须且只能配置一个条件',
      })
    if (value.mode === 'derived') {
      if (!value.displayName)
        context.addIssue({ code: 'custom', path: ['displayName'], message: '组合报警名称不能为空' })
      if (value.inputs.length < 2)
        context.addIssue({
          code: 'custom',
          path: ['inputs'],
          message: '组合报警至少需要两个输入点',
        })
      if (value.conditions.length !== 1)
        context.addIssue({
          code: 'custom',
          path: ['conditions'],
          message: '组合报警只能配置一个结果条件',
        })
    }
  })
export type AlarmItemSave = z.infer<typeof AlarmItemSaveSchema>

export const AlarmItemListSchema = z.object({
  list: z.array(AlarmItemSchema).default([]),
  pagination: PaginationSchema.default({ page: 1, pageSize: 20, total: 0 }),
})
export type AlarmItemList = z.infer<typeof AlarmItemListSchema>
export const AlarmGroupListSchema = z.object({
  list: z.array(AlarmGroupSchema).default([]),
  pagination: PaginationSchema.default({ page: 1, pageSize: 50, total: 0 }),
})

export const AlarmDraftIssueSchema = z.object({
  type: z.enum(['invalid', 'name', 'duplicate', 'overlap']),
  message: z.string(),
  datapointId: z.string().optional().default(''),
  datapointName: z.string().optional().default(''),
  conflictAlarmItemId: z.string().optional().default(''),
  conflictAlarmItemName: z.string().optional().default(''),
  affectedCount: z.number().int().nonnegative().optional().default(0),
  ackKey: z.string().optional().default(''),
})
export type AlarmDraftIssue = z.infer<typeof AlarmDraftIssueSchema>
export const AlarmDraftValidationSchema = z.object({
  valid: z.boolean(),
  errors: z.array(AlarmDraftIssueSchema).default([]),
  warnings: z.array(AlarmDraftIssueSchema).default([]),
})
export type AlarmDraftValidation = z.infer<typeof AlarmDraftValidationSchema>
export const AlarmTrialResultSchema = z.object({
  triggered: z.boolean(),
  state: z.enum(['triggered', 'not_triggered', 'insufficient_input']),
  selectedCondition: AlarmConditionSchema.nullable().optional(),
  steps: z.array(ObjectRecordSchema).default([]),
  message: z.string().optional(),
})
export type AlarmTrialResult = z.infer<typeof AlarmTrialResultSchema>
export const AlarmDatapointSummarySchema = z.object({
  datapointId: z.string(),
  items: z.array(AlarmItemSchema).default([]),
  count: z.number().int().nonnegative(),
})
export type AlarmDatapointSummary = z.infer<typeof AlarmDatapointSummarySchema>

export const AlarmItemFilterSchema = z.object({
  search: z.string().optional(),
  severity: AlarmSeveritySchema.optional(),
  alarmType: AlarmConditionKindSchema.optional(),
  mode: AlarmItemModeSchema.optional(),
  datapointId: z.string().optional(),
  groupId: z.string().nullable().optional(),
  enabled: z.boolean().optional(),
})
export const AlarmItemSelectionSchema = z
  .object({
    ids: z.array(z.string()).default([]),
    filter: AlarmItemFilterSchema.optional(),
  })
  .superRefine((value, context) => {
    if (value.ids.length > 0 === Boolean(value.filter))
      context.addIssue({ code: 'custom', message: '批量范围只能使用 ID 或筛选条件' })
  })
export type AlarmItemSelection = z.infer<typeof AlarmItemSelectionSchema>
export const AlarmBatchResultSchema = z.object({
  affectedCount: z.number().int().nonnegative(),
  itemIds: z.array(z.string()).optional().default([]),
})
export type AlarmBatchResult = z.infer<typeof AlarmBatchResultSchema>
export const AlarmBatchCreateSchema = z.object({
  datapointIds: z.array(z.string()).min(1),
  draft: z.object(AlarmItemSaveSchema.shape).omit({ datapointId: true }),
})
export type AlarmBatchCreate = z.infer<typeof AlarmBatchCreateSchema>
export const AlarmBatchUpdateSchema = z.object({
  selection: AlarmItemSelectionSchema,
  fields: z.array(z.enum(['isEnabled', 'groupId', 'notification', 'conditions'])).min(1),
  patch: z.object({
    isEnabled: z.boolean().optional(),
    groupId: z.string().nullable().optional(),
    notification: AlarmNotificationSchema.optional(),
    conditions: z.array(AlarmConditionSchema).optional(),
  }),
  acknowledgedWarningKeys: z.array(z.string()).default([]),
})
export type AlarmBatchUpdate = z.infer<typeof AlarmBatchUpdateSchema>

export const AlarmExcelIssueSchema = z.object({
  sheet: z.string(),
  row: z.number().int().nonnegative(),
  type: z.enum(['error', 'warning']),
  message: z.string(),
})
export const AlarmExcelPreviewSchema = z.object({
  digest: z.string(),
  createCount: z.number().int().nonnegative(),
  updateCount: z.number().int().nonnegative(),
  unchangedCount: z.number().int().nonnegative(),
  errorCount: z.number().int().nonnegative(),
  warningCount: z.number().int().nonnegative(),
  issues: z.array(AlarmExcelIssueSchema).default([]),
  warningKeys: z.array(z.string()).default([]),
})
export type AlarmExcelPreview = z.infer<typeof AlarmExcelPreviewSchema>

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
export const AlarmItemContractSchema = ObjectRecordSchema
