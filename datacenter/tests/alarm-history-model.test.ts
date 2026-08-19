import { describe, expect, test } from 'vitest'
import {
  buildAlarmHistoryPayload,
  createAlarmHistoryDraft,
  validateAlarmHistoryDraft,
} from '../src/models/alarm-history'

describe('alarm history settings model', () => {
  test('首次配置使用开启、30 天和保存通知记录的默认值', () => {
    expect(createAlarmHistoryDraft()).toEqual({
      isEnabled: true,
      retentionMode: 'days',
      retentionDays: 30,
      storeNotificationDeliveries: true,
    })
  })

  test('永久保留生成 null payload', () => {
    const draft = createAlarmHistoryDraft()
    draft.retentionMode = 'forever'
    expect(buildAlarmHistoryPayload(draft).retentionDays).toBeNull()
  })

  test('关闭时仍保留期限和通知记录设置', () => {
    const draft = createAlarmHistoryDraft()
    draft.isEnabled = false
    draft.retentionDays = 60
    draft.storeNotificationDeliveries = false
    expect(buildAlarmHistoryPayload(draft)).toEqual({
      isEnabled: false,
      retentionDays: 60,
      storeNotificationDeliveries: false,
    })
  })

  test('按天保留只接受正整数', () => {
    const draft = createAlarmHistoryDraft()
    draft.retentionDays = 0
    expect(validateAlarmHistoryDraft(draft)).toBe('保留天数必须为正整数')
  })
})
