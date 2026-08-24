import { describe, expect, it } from 'vitest'
import {
  AlarmBatchUpdateSchema,
  AlarmDatapointSummarySchema,
  AlarmExcelPreviewSchema,
  AlarmItemSaveSchema,
  AlarmItemSchema,
} from '../src/api/schemas/alarm.schema'

const condition = {
  id: 'condition-1',
  kind: 'threshold' as const,
  operator: 'gt',
  label: '高',
  severity: 'warning' as const,
  params: { threshold: 80 },
  triggerDelayMs: 0,
  clearDelayMs: 0,
  deadband: 0,
}
const item = {
  id: 'alarm-1',
  projectId: 'project-1',
  datapointId: 'dp-1',
  path: 'line.temperature',
  datapointName: '温度',
  dataType: 'float64',
  groupId: null,
  groupName: null,
  displayName: '越限报警',
  description: null,
  mode: 'point' as const,
  alarmType: 'threshold' as const,
  evaluationMode: 'highest_matching' as const,
  derivedExpression: '',
  inputs: [],
  conditions: [condition],
  notification: { mode: 'inherit' as const, channelIds: [], messageTemplate: '' },
  isEnabled: true,
  revision: 1,
  contract: { schemaVersion: 'alarm.item.v1' },
  createdAt: '2026-08-24T00:00:00Z',
  updatedAt: '2026-08-24T00:00:00Z',
}

describe('alarm item schemas', () => {
  it('parses one datapoint alarm item without targets', () => {
    const parsed = AlarmItemSchema.parse(item)
    expect(parsed.datapointId).toBe('dp-1')
    expect('targets' in parsed).toBe(false)
  })
  it('allows ordinary display name to remain empty for server generation', () => {
    expect(
      AlarmItemSaveSchema.parse({
        datapointId: 'dp-1',
        displayName: '',
        mode: 'point',
        evaluationMode: 'single',
        inputs: [],
        derivedExpression: '',
        conditions: [condition],
        notification: { mode: 'inherit' },
        isEnabled: true,
        revision: 0,
        acknowledgedWarningKeys: [],
      }).displayName,
    ).toBe('')
  })
  it('requires a combined alarm display name', () => {
    expect(() =>
      AlarmItemSaveSchema.parse({
        datapointId: '',
        displayName: '',
        mode: 'derived',
        evaluationMode: 'single',
        inputs: [
          { id: '', datapointId: 'a', inputKey: 'a' },
          { id: '', datapointId: 'b', inputKey: 'b' },
        ],
        derivedExpression: 'a && b',
        conditions: [{ ...condition, kind: 'expression', operator: 'is_true' }],
        notification: { mode: 'inherit' },
        isEnabled: false,
        revision: 0,
        acknowledgedWarningKeys: [],
      }),
    ).toThrow()
  })
  it('validates field-mask batch payloads and datapoint summaries', () => {
    expect(
      AlarmBatchUpdateSchema.parse({
        selection: { ids: ['alarm-1'] },
        fields: ['isEnabled'],
        patch: { isEnabled: false },
        acknowledgedWarningKeys: [],
      }).fields,
    ).toEqual(['isEnabled'])
    expect(
      AlarmDatapointSummarySchema.parse({ datapointId: 'dp-1', items: [item], count: 1 }).items,
    ).toHaveLength(1)
  })
  it('parses Excel preview counts', () => {
    expect(
      AlarmExcelPreviewSchema.parse({
        digest: 'abc',
        createCount: 2,
        updateCount: 1,
        unchangedCount: 0,
        errorCount: 0,
        warningCount: 0,
        issues: [],
        warningKeys: [],
      }).createCount,
    ).toBe(2)
  })
})
