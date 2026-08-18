import { describe, expect, test } from 'vitest'
import {
  alarmBindingPreview,
  compatibleAlarmBindings,
  conditionKindsFor,
  createAlarmPolicyDraft,
  ensureBuiltinAlarmNotificationChannel,
  normalizeAlarmConditionsForBindings,
  policyToDraft,
} from '../src/models/alarm-policy'
import type { AlarmPolicy } from '../src/api/schemas/alarm.schema'

describe('alarm policy model', () => {
  test('普通报警默认启用并添加一条未填写阈值条件', () => {
    const draft = createAlarmPolicyDraft('per_target')
    expect(draft.isEnabled).toBe(true)
    expect(draft.conditions).toHaveLength(1)
    expect(draft.conditions[0].kind).toBe('threshold')
    expect(draft.conditions[0].params.threshold).toBeNull()
    expect(draft.notification).toMatchObject({
      mode: 'inherit',
      notifyOnRaise: true,
      notifyOnClear: true,
      repeatIntervalSeconds: null,
      channelIds: ['runtime_inapp'],
    })
  })

  test('组合报警默认停用', () => {
    expect(createAlarmPolicyDraft('derived').isEnabled).toBe(false)
  })

  test('普通报警拒绝混合不兼容点位', () => {
    expect(
      compatibleAlarmBindings([
        { datapointId: '1', path: 'a', name: 'a', dataType: 'number', role: 'target' },
        { datapointId: '2', path: 'b', name: 'b', dataType: 'boolean', role: 'target' },
      ]),
    ).toBe(false)
  })

  test('点位较多时仅保留紧凑预览', () => {
    const bindings = Array.from({ length: 11 }, (_, index) => ({
      datapointId: String(index),
      path: `point.${index}`,
      name: `点位 ${index}`,
      dataType: 'number',
      role: 'target' as const,
    }))

    expect(alarmBindingPreview(bindings.slice(0, 5), 'per_target')).toMatchObject({
      collapsed: false,
      visible: bindings.slice(0, 5),
    })
    expect(alarmBindingPreview(bindings.slice(0, 6), 'per_target')).toMatchObject({
      collapsed: true,
      visible: bindings.slice(0, 3),
    })
    expect(alarmBindingPreview(bindings.slice(0, 10), 'derived').collapsed).toBe(false)
    expect(alarmBindingPreview(bindings, 'derived').collapsed).toBe(true)
  })

  test('数值点只显示兼容条件并保留编辑 payload', () => {
    const bindings = [
      { datapointId: '1', path: 'a', name: 'a', dataType: 'number', role: 'target' as const },
    ]
    expect(conditionKindsFor(bindings, 'per_target')).toContain('rate_of_change')
    const policy = { ...makePolicy(), bindings }
    expect(policyToDraft(policy).bindings).toEqual(bindings)
  })

  test('选择布尔点后将默认阈值条件调整为状态条件', () => {
    const draft = createAlarmPolicyDraft('per_target')
    const bindings = [
      { datapointId: '1', path: 'a', name: 'a', dataType: 'bool', role: 'target' as const },
    ]

    const conditions = normalizeAlarmConditionsForBindings(draft.conditions, bindings, 'per_target')

    expect(conditions[0]).toMatchObject({
      kind: 'state',
      operator: 'eq',
      severity: 'warning',
      params: { expected: true },
    })
  })

  test('通知渠道缺少时补齐内置站内通知且不重复', () => {
    const channels = ensureBuiltinAlarmNotificationChannel('project-1', [])
    expect(channels).toMatchObject([
      { id: 'runtime_inapp', projectId: 'project-1', name: '运行端站内通知' },
    ])
    expect(ensureBuiltinAlarmNotificationChannel('project-1', channels)).toHaveLength(1)
  })
})

function makePolicy(): AlarmPolicy {
  return {
    id: 'policy-1',
    projectId: 'project-1',
    groupId: null,
    groupName: null,
    name: '报警',
    description: null,
    mode: 'per_target',
    derivedExpression: '',
    bindings: [],
    conditions: [],
    notification: { mode: 'inherit', channelIds: [], messageTemplate: '' },
    isEnabled: true,
    revision: 1,
    contract: {},
    createdAt: null,
    updatedAt: null,
  }
}
