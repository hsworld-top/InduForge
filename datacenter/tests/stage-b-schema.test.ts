import { describe, expect, it } from 'vitest'
import { SourceDeleteImpactSchema } from '@/api/schemas/source-delete-impact.schema'
import { SqlWorkbenchResultSchema } from '@/api/schemas/sql-workbench.schema'

describe('Stage B API schemas', () => {
  it('preserves SQL truncation metadata', () => {
    const result = SqlWorkbenchResultSchema.parse({
      columns: ['value'],
      rows: [[1]],
      rowCount: 1,
      executionTime: 12,
      truncated: true,
      truncatedBy: 'rows',
      limits: { maxRows: 500, maxBytes: 5 * 1024 * 1024, timeoutSeconds: 30 },
    })
    expect(result.truncated).toBe(true)
    expect(result.limits.maxRows).toBe(500)
  })

  it('normalizes empty column metadata for non-query SQL', () => {
    const result = SqlWorkbenchResultSchema.parse({
      columns: [],
      columnTypes: null,
      rows: [],
      rowCount: 1,
      executionTime: 3,
      truncated: false,
      limits: { maxRows: 500, maxBytes: 5 * 1024 * 1024, timeoutSeconds: 30 },
    })
    expect(result.columnTypes).toEqual({})
  })

  it('parses source delete blockers and generated-point action', () => {
    const impact = SourceDeleteImpactSchema.parse({
      scopeType: 'connection',
      scopeId: 'source-1',
      name: 'MQTT',
      canDelete: false,
      generatedDatapoints: { count: 2, action: 'mark_invalid' },
      ownedResources: [{ type: 'mqtt_subscription', label: 'MQTT 订阅', count: 1 }],
      blockingUsages: [
        {
          type: 'alarm',
          label: '报警项',
          count: 1,
          examples: [{ id: 'alarm-1', name: '高温报警', datapointPath: 'line1.temp' }],
        },
      ],
    })
    expect(impact.canDelete).toBe(false)
    expect(impact.blockingUsages[0]?.examples[0]?.name).toBe('高温报警')
  })
})
