import { describe, expect, test } from 'vitest'

import {
  computeCapabilities,
  computeTaskModes,
  computeTriggerModes,
} from '../src/components/compute/computeCapabilities'

describe('compute workspace metadata', () => {
  test('明确提供三种计算任务模式，且副作用任务不要求输出数据点', () => {
    expect(computeTaskModes.map((mode) => mode.id)).toEqual([
      'output-datapoint',
      'side-effect',
      'hybrid',
    ])

    const sideEffectMode = computeTaskModes.find((mode) => mode.id === 'side-effect')

    expect(sideEffectMode?.outputPolicy).toBe('none')
    expect(sideEffectMode?.summary).toContain('可以不输出数据点')
    expect(sideEffectMode?.summary).toContain('写库')
    expect(sideEffectMode?.summary).toContain('MQTT/Kafka')
    expect(sideEffectMode?.summary).toContain('HTTP')
  })

  test('触发方式覆盖手动、定时、Bool 点和数据点变化', () => {
    expect(computeTriggerModes.map((trigger) => trigger.id)).toEqual([
      'manual-debug',
      'schedule',
      'bool-datapoint',
      'datapoint-change',
    ])
  })

  test('能力面板覆盖脚本任务需要展示的封装 API', () => {
    const signatures = computeCapabilities.map((capability) => capability.signature)

    expect(signatures).toContain('ctx.datapoint.get(path)')
    expect(signatures).toContain('ctx.sql.query(source, sql, args)')
    expect(signatures).toContain('ctx.sql.execute(source, sql, args)')
    expect(signatures).toContain('ctx.math.avg/sum/clamp/round')
    expect(signatures).toContain('ctx.text.format/regex/trim')
    expect(signatures).toContain('ctx.json.path/parse/stringify')
    expect(signatures).toContain('ctx.http.get/post/put')
    expect(signatures).toContain('ctx.mqtt.publish / ctx.kafka.publish')
  })
})
