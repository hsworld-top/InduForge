import request from '@/utils/request'
import {
  AlarmDatapointSummarySchema,
  AlarmDraftValidationSchema,
  AlarmHistorySettingsSaveSchema,
  AlarmHistorySettingsSchema,
  AlarmNotificationChannelSaveSchema,
  AlarmNotificationChannelSchema,
  AlarmPolicyContractSchema,
  AlarmPolicyGroupListSchema,
  AlarmPolicyGroupSaveSchema,
  AlarmPolicyGroupSchema,
  AlarmPolicyListSchema,
  AlarmPolicySaveSchema,
  AlarmPolicySchema,
  AlarmProjectSettingsSaveSchema,
  AlarmProjectSettingsSchema,
  AlarmTrialResultSchema,
  type AlarmDatapointSummary,
  type AlarmHistorySettings,
  type AlarmHistorySettingsSave,
  type AlarmNotificationChannel,
  type AlarmNotificationChannelSave,
  type AlarmPolicy,
  type AlarmPolicyGroup,
  type AlarmPolicyGroupSave,
  type AlarmPolicyList,
  type AlarmPolicySave,
  type AlarmProjectSettings,
  type AlarmProjectSettingsSave,
  type AlarmTrialResult,
} from './schemas/alarm.schema'

const unwrap = (value: unknown) =>
  value && typeof value === 'object' && 'data' in value ? (value as { data: unknown }).data : value

export async function listAlarmPolicies(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<AlarmPolicyList> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policies`,
    method: 'get',
    params,
  })
  return AlarmPolicyListSchema.parse(unwrap(response))
}

export async function getAlarmPolicy(projectId: string, policyId: string): Promise<AlarmPolicy> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}`,
    method: 'get',
  })
  return AlarmPolicySchema.parse(unwrap(response))
}

export async function createAlarmPolicy(
  projectId: string,
  payload: AlarmPolicySave,
): Promise<AlarmPolicy> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policies`,
    method: 'post',
    data: AlarmPolicySaveSchema.parse(payload),
  })
  return AlarmPolicySchema.parse(unwrap(response))
}

export async function updateAlarmPolicy(
  projectId: string,
  policyId: string,
  payload: AlarmPolicySave,
): Promise<AlarmPolicy> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}`,
    method: 'put',
    data: AlarmPolicySaveSchema.parse(payload),
  })
  return AlarmPolicySchema.parse(unwrap(response))
}

export async function setAlarmPolicyEnabled(
  projectId: string,
  policyId: string,
  isEnabled: boolean,
): Promise<AlarmPolicy> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}/enabled`,
    method: 'patch',
    data: { isEnabled },
  })
  return AlarmPolicySchema.parse(unwrap(response))
}

export async function deleteAlarmPolicy(projectId: string, policyId: string): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}`,
    method: 'delete',
  })
}

export async function validateAlarmPolicy(projectId: string, payload: AlarmPolicySave) {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policies/validate-draft`,
    method: 'post',
    data: AlarmPolicySaveSchema.parse(payload),
  })
  return AlarmDraftValidationSchema.parse(unwrap(response))
}

export async function testAlarmPolicy(
  projectId: string,
  policyId: string,
  value: unknown,
  context: Record<string, unknown> = {},
): Promise<AlarmTrialResult> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}/test`,
    method: 'post',
    data: { value, context },
  })
  return AlarmTrialResultSchema.parse(unwrap(response))
}

export async function getAlarmPolicyContract(projectId: string, policyId: string) {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policies/${policyId}/contract`,
    method: 'get',
  })
  return AlarmPolicyContractSchema.parse(unwrap(response))
}

export async function listAlarmGroups(projectId: string, params: Record<string, unknown> = {}) {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policy-groups`,
    method: 'get',
    params,
  })
  return AlarmPolicyGroupListSchema.parse(unwrap(response))
}

export async function listAlarmGroupTree(projectId: string): Promise<AlarmPolicyGroup[]> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policy-groups/tree`,
    method: 'get',
  })
  return AlarmPolicyGroupSchema.array().parse(unwrap(response))
}

export async function createAlarmGroup(projectId: string, payload: AlarmPolicyGroupSave) {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policy-groups`,
    method: 'post',
    data: AlarmPolicyGroupSaveSchema.parse(payload),
  })
  return AlarmPolicyGroupSchema.parse(unwrap(response))
}

export async function updateAlarmGroup(
  projectId: string,
  groupId: string,
  payload: AlarmPolicyGroupSave,
) {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-policy-groups/${groupId}`,
    method: 'put',
    data: AlarmPolicyGroupSaveSchema.parse(payload),
  })
  return AlarmPolicyGroupSchema.parse(unwrap(response))
}

export async function deleteAlarmGroup(projectId: string, groupId: string): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/alarm-policy-groups/${groupId}`,
    method: 'delete',
  })
}

export async function getAlarmSettings(projectId: string): Promise<AlarmProjectSettings> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-settings`,
    method: 'get',
  })
  return AlarmProjectSettingsSchema.parse(unwrap(response))
}

export async function saveAlarmSettings(
  projectId: string,
  payload: AlarmProjectSettingsSave,
): Promise<AlarmProjectSettings> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-settings`,
    method: 'put',
    data: AlarmProjectSettingsSaveSchema.parse(payload),
  })
  return AlarmProjectSettingsSchema.parse(unwrap(response))
}

export async function getAlarmHistorySettings(projectId: string): Promise<AlarmHistorySettings> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-history-settings`,
    method: 'get',
  })
  return AlarmHistorySettingsSchema.parse(unwrap(response))
}

export async function saveAlarmHistorySettings(
  projectId: string,
  payload: AlarmHistorySettingsSave,
): Promise<AlarmHistorySettings> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-history-settings`,
    method: 'put',
    data: AlarmHistorySettingsSaveSchema.parse(payload),
  })
  return AlarmHistorySettingsSchema.parse(unwrap(response))
}

export async function listAlarmChannels(projectId: string): Promise<AlarmNotificationChannel[]> {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-channels`,
    method: 'get',
  })
  return AlarmNotificationChannelSchema.array().parse(unwrap(response))
}

export async function createAlarmChannel(projectId: string, payload: AlarmNotificationChannelSave) {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-channels`,
    method: 'post',
    data: AlarmNotificationChannelSaveSchema.parse(payload),
  })
  return AlarmNotificationChannelSchema.parse(unwrap(response))
}

export async function updateAlarmChannel(
  projectId: string,
  channelId: string,
  payload: AlarmNotificationChannelSave,
) {
  const response = await request({
    url: `/data/projects/${projectId}/alarm-channels/${channelId}`,
    method: 'put',
    data: AlarmNotificationChannelSaveSchema.parse(payload),
  })
  return AlarmNotificationChannelSchema.parse(unwrap(response))
}

export async function deleteAlarmChannel(projectId: string, channelId: string): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/alarm-channels/${channelId}`,
    method: 'delete',
  })
}

export async function getDatapointAlarmSummary(
  projectId: string,
  datapointId: string,
): Promise<AlarmDatapointSummary> {
  const response = await request({
    url: `/data/projects/${projectId}/datapoints/${datapointId}/alarm-summary`,
    method: 'get',
  })
  return AlarmDatapointSummarySchema.parse(unwrap(response))
}
