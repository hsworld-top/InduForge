import { z } from 'zod'
import { TimeFieldSchema } from './common.schema'

const ObjectRecordSchema = z.record(z.string(), z.unknown())

export const AlarmConditionTypeSchema = z.enum([
  'HH',
  'H',
  'L',
  'LL',
  'deviation_high',
  'deviation_low',
  'rate_of_change',
  'bool_equal',
  'bool_transition',
  'string_equal',
  'string_not_equal',
  'string_contains',
  'string_regex',
  'cel',
])

export type AlarmConditionType = z.infer<typeof AlarmConditionTypeSchema>

export const AlarmPolicyModeSchema = z.enum(['per_target', 'derived'])

export type AlarmPolicyMode = z.infer<typeof AlarmPolicyModeSchema>

export const AlarmSeveritySchema = z.enum(['info', 'warning', 'major', 'critical'])

export type AlarmSeverity = z.infer<typeof AlarmSeveritySchema>

export const AlarmTrialStateSchema = z.enum(['triggered', 'not_triggered', 'insufficient_input'])

export type AlarmTrialState = z.infer<typeof AlarmTrialStateSchema>

export const AlarmTargetRefSchema = z.object({
  datapointId: z.string(),
  path: z.string(),
  name: z.string().optional(),
  dataType: z.string(),
})

export type AlarmTargetRef = z.infer<typeof AlarmTargetRefSchema>

export const AlarmInputRefSchema = AlarmTargetRefSchema.extend({
  key: z.string(),
})

export type AlarmInputRef = z.infer<typeof AlarmInputRefSchema>

export const AlarmConditionSchema = z.object({
  id: z.string(),
  type: AlarmConditionTypeSchema,
  name: z.string(),
  isEnabled: z.boolean(),
  severity: AlarmSeveritySchema,
  params: ObjectRecordSchema.default({}),
})

export type AlarmCondition = z.infer<typeof AlarmConditionSchema>

export const AlarmPolicyGroupSchema = z.object({
  id: z.string(),
  projectId: z.string(),
  name: z.string(),
  parentId: z.string().optional().nullable(),
  description: z.string().optional().nullable(),
  isEnabled: z.boolean(),
  sortOrder: z.number(),
  createdAt: TimeFieldSchema,
  updatedAt: TimeFieldSchema,
})

export type AlarmPolicyGroup = z.infer<typeof AlarmPolicyGroupSchema>

export const AlarmPolicyGroupSaveSchema = z
  .object({
    name: z.string().min(1),
    parentId: z.string().optional().nullable(),
    description: z.string().optional().nullable(),
    isEnabled: z.boolean().optional(),
    sortOrder: z.number().optional(),
  })
  .strict()

export type AlarmPolicyGroupSave = z.infer<typeof AlarmPolicyGroupSaveSchema>

export const AlarmPolicyGroupUpdateSchema = AlarmPolicyGroupSaveSchema.partial().strict()

export type AlarmPolicyGroupUpdate = z.infer<typeof AlarmPolicyGroupUpdateSchema>

export const AlarmPolicySchema = z
  .object({
    id: z.string(),
    projectId: z.string(),
    groupId: z.string().optional().nullable(),
    groupName: z.string().optional().nullable(),
    groupEnabled: z.boolean().optional().nullable(),
    name: z.string(),
    description: z.string().optional().nullable(),
    mode: AlarmPolicyModeSchema,
    targets: z.array(AlarmTargetRefSchema),
    inputs: z.array(AlarmInputRefSchema),
    derivedExpression: z.string(),
    conditions: z.array(AlarmConditionSchema),
    suppression: ObjectRecordSchema.default({}),
    messageTemplate: z.string(),
    isEnabled: z.boolean(),
    effectiveEnabled: z.boolean(),
    contract: ObjectRecordSchema.default({}),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export type AlarmPolicy = z.infer<typeof AlarmPolicySchema>

export const AlarmPolicySaveSchema = z
  .object({
    groupId: z.string().nullable().optional(),
    name: z.string().min(1),
    description: z.string().optional().nullable(),
    mode: AlarmPolicyModeSchema,
    targets: z.array(AlarmTargetRefSchema),
    inputs: z.array(AlarmInputRefSchema),
    derivedExpression: z.string(),
    conditions: z.array(AlarmConditionSchema),
    suppression: ObjectRecordSchema.default({}),
    messageTemplate: z.string().default(''),
    isEnabled: z.boolean().default(true),
  })
  .strict()

export type AlarmPolicySave = z.infer<typeof AlarmPolicySaveSchema>

export const AlarmPolicyUpdateSchema = AlarmPolicySaveSchema.partial().strict()

export type AlarmPolicyUpdate = z.infer<typeof AlarmPolicyUpdateSchema>

export const AlarmPolicyTreeSchema = z.object({
  groups: z.array(AlarmPolicyGroupSchema),
  rootPolicies: z.array(AlarmPolicySchema),
  policies: z.array(AlarmPolicySchema).default([]),
  matchedPolicyCount: z.number(),
  totalPolicyCount: z.number(),
})

export type AlarmPolicyTree = z.infer<typeof AlarmPolicyTreeSchema>

export const AlarmPolicyCoverageItemSchema = z.object({
  id: z.string(),
  name: z.string(),
  mode: AlarmPolicyModeSchema,
  targetCount: z.number(),
  isSingleTarget: z.boolean(),
  isEnabled: z.boolean(),
  effectiveEnabled: z.boolean(),
  groupName: z.string().optional(),
})

export type AlarmPolicyCoverageItem = z.infer<typeof AlarmPolicyCoverageItemSchema>

export const AlarmPolicyCoverageSchema = z.object({
  datapointId: z.string().optional(),
  path: z.string().optional(),
  policies: z.array(AlarmPolicyCoverageItemSchema).default([]),
})

export type AlarmPolicyCoverage = z.infer<typeof AlarmPolicyCoverageSchema>

export const AlarmProjectSettingsSchema = z.object({
  projectId: z.string(),
  escalationIntervalSeconds: z.number().int().min(30),
  repeatNotificationIntervalSeconds: z.number().int().min(10),
  createdAt: TimeFieldSchema.optional(),
  updatedAt: TimeFieldSchema.optional(),
})

export type AlarmProjectSettings = z.infer<typeof AlarmProjectSettingsSchema>

export const AlarmProjectSettingsSaveSchema = z
  .object({
    escalationIntervalSeconds: z.number().int().min(30),
    repeatNotificationIntervalSeconds: z.number().int().min(10),
  })
  .strict()

export type AlarmProjectSettingsSave = z.infer<typeof AlarmProjectSettingsSaveSchema>

export const AlarmBulkSelectionSchema = z.discriminatedUnion('mode', [
  z.object({ mode: z.literal('ids'), policyIds: z.array(z.string()) }),
  z.object({
    mode: z.literal('filtered'),
    filters: ObjectRecordSchema,
    excludePolicyIds: z.array(z.string()).default([]),
  }),
])

export type AlarmBulkSelection = z.infer<typeof AlarmBulkSelectionSchema>

export const AlarmPolicyTrialPayloadSchema = z
  .object({
    value: z.unknown().optional(),
    values: ObjectRecordSchema.optional(),
    timestamp: z.string().optional(),
    context: ObjectRecordSchema.default({}),
  })
  .default({ context: {} })

export type AlarmPolicyTrialPayload = z.input<typeof AlarmPolicyTrialPayloadSchema>

export const AlarmPolicyTrialResultSchema = z
  .object({
    triggered: z.boolean(),
    state: AlarmTrialStateSchema,
    message: z.string().optional(),
    triggeredConditions: z.array(AlarmConditionSchema).default([]),
    diagnostics: ObjectRecordSchema.default({}),
    conditionResults: z.array(z.record(z.string(), z.unknown())).default([]),
  })
  .passthrough()

export type AlarmPolicyTrialResult = z.infer<typeof AlarmPolicyTrialResultSchema>

export const AlarmDraftValidationSchema = z
  .object({
    valid: z.boolean(),
    errors: z.array(z.unknown()).default([]),
    target: ObjectRecordSchema.optional(),
    contract: ObjectRecordSchema.optional(),
  })
  .passthrough()

export type AlarmDraftValidation = z.infer<typeof AlarmDraftValidationSchema>

export const AlarmPolicyContractSchema = ObjectRecordSchema

export type AlarmPolicyContract = z.infer<typeof AlarmPolicyContractSchema>
