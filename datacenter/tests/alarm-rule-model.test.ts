import { describe, expect, test } from 'vitest'

import type { AlarmPolicy } from '../src/api/schemas/alarm.schema'
import {
  createDefaultAlarmPolicyDraft,
  draftToAlarmPolicySavePayload,
  makeFilteredSelection,
  togglePolicyInSelection,
  toAlarmPolicyDraft,
} from '../src/components/alarm/alarmPolicyModel'

describe('alarm policy model', () => {
  test('创建逐点判断草稿', () => {
    const draft = createDefaultAlarmPolicyDraft()

    expect(draft.mode).toBe('per_target')
    expect(draft.targets).toEqual([])
    expect(draft.conditions).toEqual([])
  })

  test('序列化多个条件', () => {
    const payload = draftToAlarmPolicySavePayload({
      ...createDefaultAlarmPolicyDraft(),
      name: '温度策略',
      conditions: [
        {
          id: 'c-h',
          type: 'H',
          name: '高限',
          isEnabled: true,
          severity: 'major',
          params: { limit: 80 },
        },
        {
          id: 'c-l',
          type: 'L',
          name: '低限',
          isEnabled: true,
          severity: 'warning',
          params: { limit: 20 },
        },
      ],
    })

    expect(payload.conditions).toHaveLength(2)
  })

  test('未分组策略保留 groupId null', () => {
    const draft = toAlarmPolicyDraft(makePolicy({ groupId: null }))

    expect(draft.groupId).toBeNull()
  })

  test('筛选后全选支持排除单个策略', () => {
    const selection = makeFilteredSelection({ search: '温度', enabled: true })
    const next = togglePolicyInSelection(selection, 'policy-2', false)

    expect(next.mode).toBe('filtered')
    if (next.mode === 'filtered') {
      expect(next.excludePolicyIds).toContain('policy-2')
    }
  })
})

function makePolicy(patch: Partial<AlarmPolicy> = {}): AlarmPolicy {
  return {
    id: 'policy-1',
    projectId: 'project-1',
    groupId: null,
    name: '温度策略',
    mode: 'per_target',
    targets: [],
    inputs: [],
    derivedExpression: '',
    conditions: [],
    suppression: {},
    messageTemplate: '',
    isEnabled: true,
    effectiveEnabled: true,
    contract: {},
    createdAt: '2026-05-20T00:00:00Z',
    updatedAt: '2026-05-20T00:00:00Z',
    ...patch,
  }
}
