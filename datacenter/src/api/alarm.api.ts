import request from '@/utils/request'
import {
  AlarmBatchCreateSchema,
  AlarmBatchResultSchema,
  AlarmBatchUpdateSchema,
  AlarmDatapointSummarySchema,
  AlarmDraftValidationSchema,
  AlarmExcelPreviewSchema,
  AlarmGroupListSchema,
  AlarmGroupSaveSchema,
  AlarmGroupSchema,
  AlarmHistorySettingsSaveSchema,
  AlarmHistorySettingsSchema,
  AlarmItemContractSchema,
  AlarmItemListSchema,
  AlarmItemSaveSchema,
  AlarmItemSchema,
  AlarmItemSelectionSchema,
  AlarmNotificationChannelSaveSchema,
  AlarmNotificationChannelSchema,
  AlarmProjectSettingsSaveSchema,
  AlarmProjectSettingsSchema,
  AlarmTrialResultSchema,
  type AlarmBatchCreate,
  type AlarmBatchResult,
  type AlarmBatchUpdate,
  type AlarmDatapointSummary,
  type AlarmExcelPreview,
  type AlarmGroup,
  type AlarmGroupSave,
  type AlarmHistorySettings,
  type AlarmHistorySettingsSave,
  type AlarmItem,
  type AlarmItemList,
  type AlarmItemSave,
  type AlarmItemSelection,
  type AlarmNotificationChannel,
  type AlarmNotificationChannelSave,
  type AlarmProjectSettings,
  type AlarmProjectSettingsSave,
  type AlarmTrialResult,
} from './schemas/alarm.schema'

const unwrap = (value: unknown) =>
  value && typeof value === 'object' && 'data' in value ? (value as { data: unknown }).data : value

export async function listAlarmItems(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<AlarmItemList> {
  return AlarmItemListSchema.parse(
    unwrap(
      await request({ url: `/data/projects/${projectId}/alarm-items`, method: 'get', params }),
    ),
  )
}
export async function getAlarmItem(projectId: string, id: string): Promise<AlarmItem> {
  return AlarmItemSchema.parse(
    unwrap(await request({ url: `/data/projects/${projectId}/alarm-items/${id}`, method: 'get' })),
  )
}
export async function createAlarmItem(
  projectId: string,
  payload: AlarmItemSave,
): Promise<AlarmItem> {
  return AlarmItemSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items`,
        method: 'post',
        data: AlarmItemSaveSchema.parse(payload),
      }),
    ),
  )
}
export async function updateAlarmItem(
  projectId: string,
  id: string,
  payload: AlarmItemSave,
): Promise<AlarmItem> {
  return AlarmItemSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/${id}`,
        method: 'put',
        data: AlarmItemSaveSchema.parse(payload),
      }),
    ),
  )
}
export async function setAlarmItemEnabled(
  projectId: string,
  id: string,
  isEnabled: boolean,
): Promise<AlarmItem> {
  return AlarmItemSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/${id}/enabled`,
        method: 'patch',
        data: { isEnabled },
      }),
    ),
  )
}
export async function deleteAlarmItem(projectId: string, id: string): Promise<void> {
  await request({ url: `/data/projects/${projectId}/alarm-items/${id}`, method: 'delete' })
}
export async function validateAlarmItem(projectId: string, payload: AlarmItemSave) {
  return AlarmDraftValidationSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/validate-draft`,
        method: 'post',
        data: AlarmItemSaveSchema.parse(payload),
      }),
    ),
  )
}
export async function testAlarmItem(
  projectId: string,
  draft: AlarmItemSave,
  values: unknown[],
  context: Record<string, unknown> = {},
): Promise<AlarmTrialResult> {
  return AlarmTrialResultSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/test-draft`,
        method: 'post',
        data: { draft: AlarmItemSaveSchema.parse(draft), values, context },
      }),
    ),
  )
}
export async function getAlarmItemContract(projectId: string, id: string) {
  return AlarmItemContractSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/${id}/contract`,
        method: 'get',
      }),
    ),
  )
}
export async function batchCreateAlarmItems(
  projectId: string,
  payload: AlarmBatchCreate,
): Promise<AlarmBatchResult> {
  return AlarmBatchResultSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/batch-create`,
        method: 'post',
        data: AlarmBatchCreateSchema.parse(payload),
      }),
    ),
  )
}
export async function validateBatchCreateAlarmItems(projectId: string, payload: AlarmBatchCreate) {
  return AlarmDraftValidationSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/batch-create/validate`,
        method: 'post',
        data: AlarmBatchCreateSchema.parse(payload),
      }),
    ),
  )
}
export async function batchUpdateAlarmItems(
  projectId: string,
  payload: AlarmBatchUpdate,
): Promise<AlarmBatchResult> {
  return AlarmBatchResultSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/batch`,
        method: 'patch',
        data: AlarmBatchUpdateSchema.parse(payload),
      }),
    ),
  )
}
export async function batchDeleteAlarmItems(
  projectId: string,
  selection: AlarmItemSelection,
): Promise<AlarmBatchResult> {
  return AlarmBatchResultSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/batch-delete`,
        method: 'post',
        data: { selection: AlarmItemSelectionSchema.parse(selection) },
      }),
    ),
  )
}
export async function exportAlarmItems(
  projectId: string,
  selection: AlarmItemSelection,
): Promise<Blob> {
  return (await request({
    url: `/data/projects/${projectId}/alarm-items/export`,
    method: 'post',
    data: { selection: AlarmItemSelectionSchema.parse(selection) },
    responseType: 'blob',
  })) as Blob
}
export async function downloadAlarmImportTemplate(projectId: string): Promise<Blob> {
  return (await request({
    url: `/data/projects/${projectId}/alarm-items/import-template`,
    method: 'get',
    responseType: 'blob',
  })) as Blob
}
export async function previewAlarmImport(
  projectId: string,
  file: File,
): Promise<AlarmExcelPreview> {
  const data = new FormData()
  data.append('file', file)
  return AlarmExcelPreviewSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/import/preview`,
        method: 'post',
        data,
        headers: { 'Content-Type': 'multipart/form-data' },
      }),
    ),
  )
}
export async function downloadAlarmImportErrors(projectId: string, file: File): Promise<Blob> {
  const data = new FormData()
  data.append('file', file)
  return (await request({
    url: `/data/projects/${projectId}/alarm-items/import/error-workbook`,
    method: 'post',
    data,
    headers: { 'Content-Type': 'multipart/form-data' },
    responseType: 'blob',
  })) as Blob
}
export async function applyAlarmImport(
  projectId: string,
  file: File,
  digest: string,
  acknowledgedWarningKeys: string[],
): Promise<AlarmBatchResult> {
  const data = new FormData()
  data.append('file', file)
  data.append('digest', digest)
  data.append('acknowledgedWarningKeys', JSON.stringify(acknowledgedWarningKeys))
  return AlarmBatchResultSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-items/import/apply`,
        method: 'post',
        data,
        headers: { 'Content-Type': 'multipart/form-data' },
      }),
    ),
  )
}

export async function listAlarmGroups(projectId: string, params: Record<string, unknown> = {}) {
  return AlarmGroupListSchema.parse(
    unwrap(
      await request({ url: `/data/projects/${projectId}/alarm-groups`, method: 'get', params }),
    ),
  )
}
export async function listAlarmGroupTree(projectId: string): Promise<AlarmGroup[]> {
  return AlarmGroupSchema.array().parse(
    unwrap(await request({ url: `/data/projects/${projectId}/alarm-groups/tree`, method: 'get' })),
  )
}
export async function createAlarmGroup(projectId: string, payload: AlarmGroupSave) {
  return AlarmGroupSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-groups`,
        method: 'post',
        data: AlarmGroupSaveSchema.parse(payload),
      }),
    ),
  )
}
export async function updateAlarmGroup(projectId: string, id: string, payload: AlarmGroupSave) {
  return AlarmGroupSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-groups/${id}`,
        method: 'put',
        data: AlarmGroupSaveSchema.parse(payload),
      }),
    ),
  )
}
export async function deleteAlarmGroup(projectId: string, id: string): Promise<void> {
  await request({ url: `/data/projects/${projectId}/alarm-groups/${id}`, method: 'delete' })
}

export async function getAlarmSettings(projectId: string): Promise<AlarmProjectSettings> {
  return AlarmProjectSettingsSchema.parse(
    unwrap(await request({ url: `/data/projects/${projectId}/alarm-settings`, method: 'get' })),
  )
}
export async function saveAlarmSettings(
  projectId: string,
  payload: AlarmProjectSettingsSave,
): Promise<AlarmProjectSettings> {
  return AlarmProjectSettingsSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-settings`,
        method: 'put',
        data: AlarmProjectSettingsSaveSchema.parse(payload),
      }),
    ),
  )
}
export async function getAlarmHistorySettings(projectId: string): Promise<AlarmHistorySettings> {
  return AlarmHistorySettingsSchema.parse(
    unwrap(
      await request({ url: `/data/projects/${projectId}/alarm-history-settings`, method: 'get' }),
    ),
  )
}
export async function saveAlarmHistorySettings(
  projectId: string,
  payload: AlarmHistorySettingsSave,
): Promise<AlarmHistorySettings> {
  return AlarmHistorySettingsSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-history-settings`,
        method: 'put',
        data: AlarmHistorySettingsSaveSchema.parse(payload),
      }),
    ),
  )
}
export async function listAlarmChannels(projectId: string): Promise<AlarmNotificationChannel[]> {
  return AlarmNotificationChannelSchema.array().parse(
    unwrap(await request({ url: `/data/projects/${projectId}/alarm-channels`, method: 'get' })),
  )
}
export async function createAlarmChannel(projectId: string, payload: AlarmNotificationChannelSave) {
  return AlarmNotificationChannelSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-channels`,
        method: 'post',
        data: AlarmNotificationChannelSaveSchema.parse(payload),
      }),
    ),
  )
}
export async function updateAlarmChannel(
  projectId: string,
  id: string,
  payload: AlarmNotificationChannelSave,
) {
  return AlarmNotificationChannelSchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/alarm-channels/${id}`,
        method: 'put',
        data: AlarmNotificationChannelSaveSchema.parse(payload),
      }),
    ),
  )
}
export async function deleteAlarmChannel(projectId: string, id: string): Promise<void> {
  await request({ url: `/data/projects/${projectId}/alarm-channels/${id}`, method: 'delete' })
}
export async function getDatapointAlarmSummary(
  projectId: string,
  datapointId: string,
): Promise<AlarmDatapointSummary> {
  return AlarmDatapointSummarySchema.parse(
    unwrap(
      await request({
        url: `/data/projects/${projectId}/datapoints/${datapointId}/alarms`,
        method: 'get',
      }),
    ),
  )
}
