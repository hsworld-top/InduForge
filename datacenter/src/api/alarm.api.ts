import request from '@/utils/request'
import { listResponseSchema } from './schemas/common.schema'
import {
  AlarmBulkSelectionSchema,
  AlarmDraftValidationSchema,
  AlarmPolicyContractSchema,
  AlarmPolicyCoverageSchema,
  AlarmPolicyGroupSaveSchema,
  AlarmPolicyGroupSchema,
  AlarmPolicyGroupUpdateSchema,
  AlarmProjectSettingsSaveSchema,
  AlarmProjectSettingsSchema,
  AlarmPolicySaveSchema,
  AlarmPolicySchema,
  AlarmPolicyTreeSchema,
  AlarmPolicyTrialPayloadSchema,
  AlarmPolicyTrialResultSchema,
  AlarmPolicyUpdateSchema,
  type AlarmBulkSelection,
  type AlarmDraftValidation,
  type AlarmPolicy,
  type AlarmPolicyContract,
  type AlarmPolicyCoverage,
  type AlarmPolicyGroup,
  type AlarmPolicyGroupSave,
  type AlarmPolicyGroupUpdate,
  type AlarmProjectSettings,
  type AlarmProjectSettingsSave,
  type AlarmPolicySave,
  type AlarmPolicyTree,
  type AlarmPolicyTrialPayload,
  type AlarmPolicyTrialResult,
  type AlarmPolicyUpdate,
} from './schemas/alarm.schema'

const alarmPolicyListSchema = listResponseSchema(AlarmPolicySchema)
const alarmPolicyGroupListSchema = AlarmPolicyGroupSchema.array()

type AlarmPolicyListResp = {
  list: AlarmPolicy[]
  pagination?: {
    page?: number
    pageSize?: number
    total?: number
  }
}

const unwrapData = (value: unknown) => {
  if (value && typeof value === 'object' && 'code' in value && 'data' in value) {
    return (value as { data?: unknown }).data
  }
  return value
}

export async function getAlarmPolicyGroups(projectId: string): Promise<AlarmPolicyGroup[]> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policy-groups`,
    method: 'get',
  })
  return alarmPolicyGroupListSchema.parse(unwrapData(res))
}

export async function getAlarmProjectSettings(projectId: string): Promise<AlarmProjectSettings> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-settings`,
    method: 'get',
  })
  return AlarmProjectSettingsSchema.parse(unwrapData(res))
}

export async function updateAlarmProjectSettings(
  projectId: string,
  data: AlarmProjectSettingsSave,
): Promise<AlarmProjectSettings> {
  const body = AlarmProjectSettingsSaveSchema.parse(data)
  const res = await request({
    url: `/data/projects/${projectId}/alarm-settings`,
    method: 'put',
    data: body,
  })
  return AlarmProjectSettingsSchema.parse(unwrapData(res))
}

export async function createAlarmPolicyGroup(
  projectId: string,
  data: AlarmPolicyGroupSave,
): Promise<AlarmPolicyGroup> {
  const body = AlarmPolicyGroupSaveSchema.parse(data)
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policy-groups`,
    method: 'post',
    data: body,
  })
  return AlarmPolicyGroupSchema.parse(unwrapData(res))
}

export async function updateAlarmPolicyGroup(
  projectId: string,
  groupId: string,
  data: AlarmPolicyGroupUpdate,
): Promise<AlarmPolicyGroup> {
  const body = AlarmPolicyGroupUpdateSchema.parse(data)
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policy-groups/${groupId}`,
    method: 'put',
    data: body,
  })
  return AlarmPolicyGroupSchema.parse(unwrapData(res))
}

export async function toggleAlarmPolicyGroup(
  projectId: string,
  groupId: string,
  isEnabled: boolean,
): Promise<AlarmPolicyGroup> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policy-groups/${groupId}/enabled`,
    method: 'patch',
    data: { isEnabled },
  })
  return AlarmPolicyGroupSchema.parse(unwrapData(res))
}

export async function deleteAlarmPolicyGroup(projectId: string, groupId: string): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/alarm-policy-groups/${groupId}`,
    method: 'delete',
  })
}

export async function getAlarmPolicyTree(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<AlarmPolicyTree> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies/tree`,
    method: 'get',
    params,
  })
  return AlarmPolicyTreeSchema.parse(unwrapData(res))
}

export async function getAlarmPolicies(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<AlarmPolicyListResp> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies`,
    method: 'get',
    params,
  })
  return alarmPolicyListSchema.parse(unwrapData(res))
}

export async function getAlarmPolicy(projectId: string, policyId: string): Promise<AlarmPolicy> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}`,
    method: 'get',
  })
  return AlarmPolicySchema.parse(unwrapData(res))
}

export async function getAlarmPolicyCoverage(
  projectId: string,
  params: {
    datapointId?: string
    path?: string
    excludePolicyId?: string
  },
): Promise<AlarmPolicyCoverage> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies/coverage`,
    method: 'get',
    params,
  })
  return AlarmPolicyCoverageSchema.parse(unwrapData(res))
}

export async function createAlarmPolicy(
  projectId: string,
  data: AlarmPolicySave,
): Promise<AlarmPolicy> {
  const body = AlarmPolicySaveSchema.parse(data)
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies`,
    method: 'post',
    data: body,
  })
  return AlarmPolicySchema.parse(unwrapData(res))
}

export async function updateAlarmPolicy(
  projectId: string,
  policyId: string,
  data: AlarmPolicyUpdate,
): Promise<AlarmPolicy> {
  const body = AlarmPolicyUpdateSchema.parse(data)
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}`,
    method: 'put',
    data: body,
  })
  return AlarmPolicySchema.parse(unwrapData(res))
}

export async function toggleAlarmPolicy(
  projectId: string,
  policyId: string,
  isEnabled: boolean,
): Promise<AlarmPolicy> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}/enabled`,
    method: 'patch',
    data: { isEnabled },
  })
  return AlarmPolicySchema.parse(unwrapData(res))
}

export async function deleteAlarmPolicy(projectId: string, policyId: string): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}`,
    method: 'delete',
  })
}

export async function testAlarmPolicy(
  projectId: string,
  policyId: string,
  payload: AlarmPolicyTrialPayload = { context: {} },
): Promise<AlarmPolicyTrialResult> {
  const body = AlarmPolicyTrialPayloadSchema.parse(payload)
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}/test`,
    method: 'post',
    data: body,
  })
  return AlarmPolicyTrialResultSchema.parse(unwrapData(res))
}

export async function getAlarmPolicyContract(
  projectId: string,
  policyId: string,
): Promise<AlarmPolicyContract> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}/contract`,
    method: 'get',
  })
  return AlarmPolicyContractSchema.parse(unwrapData(res))
}

export async function validateAlarmPolicyDraft(
  projectId: string,
  data: AlarmPolicySave,
): Promise<AlarmDraftValidation> {
  const body = AlarmPolicySaveSchema.parse(data)
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies/validate-draft`,
    method: 'post',
    data: body,
  })
  return AlarmDraftValidationSchema.parse(unwrapData(res))
}

export async function batchEnableAlarmPolicies(
  projectId: string,
  selection: AlarmBulkSelection,
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/alarm-policies/batch-enable`,
    method: 'post',
    data: { selection: AlarmBulkSelectionSchema.parse(selection) },
  })
}

export async function batchDisableAlarmPolicies(
  projectId: string,
  selection: AlarmBulkSelection,
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/alarm-policies/batch-disable`,
    method: 'post',
    data: { selection: AlarmBulkSelectionSchema.parse(selection) },
  })
}

export async function batchMoveAlarmPolicies(
  projectId: string,
  selection: AlarmBulkSelection,
  groupId: string | null,
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/alarm-policies/batch-move`,
    method: 'post',
    data: { selection: AlarmBulkSelectionSchema.parse(selection), groupId },
  })
}

export async function batchApplyAlarmConditions(
  projectId: string,
  selection: AlarmBulkSelection,
  conditions: AlarmPolicy['conditions'],
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/alarm-policies/batch-apply-conditions`,
    method: 'post',
    data: { selection: AlarmBulkSelectionSchema.parse(selection), conditions },
  })
}
