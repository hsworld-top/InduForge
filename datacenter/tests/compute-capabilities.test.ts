import { describe, expect, test } from 'vitest'

import { ComputeCapabilitiesSchema } from '../src/api/schemas/compute.schema'

describe('compute sandbox capabilities', () => {
  test('只接受后端沙箱真实声明的语言、SDK、依赖和限制', () => {
    const result = ComputeCapabilitiesSchema.parse({
      sandboxStatus: 'available',
      languages: [
        { language: 'js', version: 'v22' },
        { language: 'python', version: '3.11' },
      ],
      sdk: ['ctx.datapoint.get', 'ctx.datapoint.meta', 'ctx.sql.query'],
      dependencies: [],
      triggerTypes: ['manual', 'schedule', 'datapoint_change', 'condition'],
      limits: {
        maxExecutionTimeMs: 120000,
        maxInputBytes: 1048576,
        maxOutputBytes: 2097152,
        maxLogBytes: 65536,
      },
    })

    expect(result.sdk).toEqual(['ctx.datapoint.get', 'ctx.datapoint.meta', 'ctx.sql.query'])
    expect(result.dependencies).toEqual([])
    expect(result.triggerTypes).toContain('condition')
    expect(result.sdk).not.toContain('ctx.http.post')
    expect(result.sdk).not.toContain('ctx.mqtt.publish')
  })

  test('沙箱不可用时使用空能力，不伪装成宿主执行器', () => {
    const result = ComputeCapabilitiesSchema.parse({
      sandboxStatus: 'unavailable',
      languages: [],
      sdk: [],
      dependencies: [],
      triggerTypes: [],
      limits: {
        maxExecutionTimeMs: 0,
        maxInputBytes: 0,
        maxOutputBytes: 0,
        maxLogBytes: 0,
      },
    })
    expect(result.sandboxStatus).toBe('unavailable')
    expect(result.sdk).toEqual([])
  })
})
