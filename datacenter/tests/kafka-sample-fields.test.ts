import { describe, expect, test } from 'vitest'
import {
  inferKafkaSampleFields,
  normalizeKafkaSampleEditorText,
} from '@/components/kafka/kafkaSampleFields'

describe('kafkaSampleFields', () => {
  test('从 Kafka sample.value 对象展平字段', () => {
    const fields = inferKafkaSampleFields(
      [
        {
          topic: 'device.telemetry',
          value: {
            deviceId: 'A001',
            temperature: 26.5,
            metrics: { pressure: 0.72 },
          },
        },
      ],
      [],
    )

    expect(fields.map((field) => field.path)).toEqual([
      'deviceId',
      'temperature',
      'metrics.pressure',
    ])
    expect(fields.find((field) => field.path === 'temperature')?.dataType).toBe('number')
  })

  test('支持 value 为 JSON 字符串并标记已存在字段', () => {
    const fields = inferKafkaSampleFields(
      [{ value: '{"deviceId":"A001","running":true}' }],
      ['deviceId'],
    )

    expect(fields.find((field) => field.path === 'deviceId')?.exists).toBe(true)
    expect(fields.find((field) => field.path === 'running')?.dataType).toBe('boolean')
  })

  test('数组按普通索引路径展开，不生成数组拆分规则', () => {
    const fields = inferKafkaSampleFields(
      [{ value: { items: [{ name: 'temp', value: 26.5 }] } }],
      [],
    )

    expect(fields.map((field) => field.path)).toContain('items.0.name')
    expect(fields.map((field) => field.path)).toContain('items.0.value')
  })

  test('编辑器文本格式化为 JSON 对象字符串', () => {
    const text = normalizeKafkaSampleEditorText({ value: { status: 'ok' } })
    expect(text).toContain('"status": "ok"')
  })
})
