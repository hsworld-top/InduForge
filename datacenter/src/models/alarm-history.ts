import type { AlarmHistorySettings, AlarmHistorySettingsSave } from '@/api/schemas/alarm.schema'

export type AlarmHistoryRetentionMode = 'days' | 'forever'

export type AlarmHistoryDraft = {
  isEnabled: boolean
  retentionMode: AlarmHistoryRetentionMode
  retentionDays: number
  storeNotificationDeliveries: boolean
}

export function createAlarmHistoryDraft(settings?: AlarmHistorySettings | null): AlarmHistoryDraft {
  return {
    isEnabled: settings?.isEnabled ?? true,
    retentionMode: settings?.retentionDays === null ? 'forever' : 'days',
    retentionDays: settings?.retentionDays ?? 30,
    storeNotificationDeliveries: settings?.storeNotificationDeliveries ?? true,
  }
}

export function validateAlarmHistoryDraft(draft: AlarmHistoryDraft): string {
  if (
    draft.retentionMode === 'days' &&
    (!Number.isSafeInteger(draft.retentionDays) || draft.retentionDays <= 0)
  ) {
    return '保留天数必须为正整数'
  }
  return ''
}

export function buildAlarmHistoryPayload(draft: AlarmHistoryDraft): AlarmHistorySettingsSave {
  return {
    isEnabled: draft.isEnabled,
    retentionDays: draft.retentionMode === 'forever' ? null : draft.retentionDays,
    storeNotificationDeliveries: draft.storeNotificationDeliveries,
  }
}
