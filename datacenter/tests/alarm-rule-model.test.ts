import { describe, expect, it } from 'vitest'
import {
  alarmItemConditionSummary,
  alarmItemPointSummary,
  buildAlarmItemPayload,
  compatibleAlarmPoints,
  conditionKindsFor,
  createAlarmCondition,
  createAlarmItemDraft,
  sortAlarmLevels,
} from '../src/models/alarm-policy'
import type { AlarmItem } from '../src/api/schemas/alarm.schema'

describe('alarm item model', () => {
  it('creates an enabled point alarm with optional display name', () => {
    const draft = createAlarmItemDraft('point')
    expect(draft.displayName).toBe('')
    expect(draft.isEnabled).toBe(true)
    expect(draft.evaluationMode).toBe('highest_matching')
  })
  it('builds one item payload while the caller keeps the selected datapoint IDs for batch creation', () => {
    const draft = createAlarmItemDraft('point')
    draft.selectedPoints = [
      { datapointId: 'dp-1', path: 'a', name: 'A', dataType: 'float64' },
      { datapointId: 'dp-2', path: 'b', name: 'B', dataType: 'float64' },
    ]
    const payload = buildAlarmItemPayload(draft)
    expect(payload.datapointId).toBe('dp-1')
    expect(draft.selectedPoints.map((point) => point.datapointId)).toEqual(['dp-1', 'dp-2'])
  })
  it('keeps different alarm kinds compatible on one point and rejects mixed datapoint categories', () => {
    expect(createAlarmCondition('rate_of_change').kind).toBe('rate_of_change')
    expect(
      compatibleAlarmPoints([
        { datapointId: 'a', path: 'a', name: 'A', dataType: 'float64' },
        { datapointId: 'b', path: 'b', name: 'B', dataType: 'bool' },
      ]),
    ).toBe(false)
  })
  it('keeps threshold levels out of numeric point other-condition options', () => {
    const point = [{ datapointId: 'dp-1', path: 'a', name: 'A', dataType: 'float64' }]
    expect(conditionKindsFor(point, 'point', 'single')).not.toContain('threshold')
    expect(conditionKindsFor(point, 'point', 'highest_matching')).toContain('threshold')
  })
  it('treats a derived expression as input calculation rather than a result condition', () => {
    const kinds = conditionKindsFor([], 'derived', 'single')
    expect(kinds).not.toContain('expression')
    expect(kinds).toContain('threshold')
    expect(createAlarmCondition('transition').params).toEqual({ from: false, to: true })
  })
  it('sorts threshold levels from high outward and summarizes one item', () => {
    const high = createAlarmCondition('threshold', '高')
    high.params.threshold = 80
    const highHigh = createAlarmCondition('threshold', '高高')
    highHigh.params.threshold = 90
    expect(sortAlarmLevels([highHigh, high]).map((condition) => condition.label)).toEqual([
      '高',
      '高高',
    ])
    const item = makeItem()
    expect(alarmItemPointSummary(item)).toBe('温度')
    expect(alarmItemConditionSummary(item)).toContain('高')
  })
})

function makeItem(): AlarmItem {
  return {
    id: 'alarm-1',
    projectId: 'project',
    datapointId: 'dp-1',
    path: 'line.temperature',
    datapointName: '温度',
    dataType: 'float64',
    groupId: null,
    groupName: null,
    displayName: '越限报警',
    description: null,
    mode: 'point',
    alarmType: 'threshold',
    evaluationMode: 'highest_matching',
    derivedExpression: '',
    inputs: [],
    conditions: [{ ...createAlarmCondition('threshold', '高'), params: { threshold: 80 } }],
    notification: { mode: 'inherit', channelIds: [], messageTemplate: '' },
    isEnabled: true,
    revision: 1,
    contract: {},
    createdAt: '2026-08-24T00:00:00Z',
    updatedAt: '2026-08-24T00:00:00Z',
  }
}
