import { describe, expect, test } from 'vitest'
import { DatapointCustomAttributesSchema } from '../src/api/schemas/datapoint.schema'
import {
  buildDatapointAttributeDefaults,
  createDatapointAttributeRows,
  validateDatapointAttributeRows,
} from '../src/models/datapoint-custom-attributes'

describe('datapoint custom attributes', () => {
  test('schema only accepts string default values', () => {
    expect(
      DatapointCustomAttributesSchema.parse({ attributes: { asset_code: 'PUMP-001' } }),
    ).toEqual({ attributes: { asset_code: 'PUMP-001' } })
    expect(() => DatapointCustomAttributesSchema.parse({ attributes: { priority: 1 } })).toThrow()
  })

  test('loads attributes into stable rows and builds payload', () => {
    const rows = createDatapointAttributeRows({ line_no: '03', asset_code: 'PUMP-001' })
    expect(rows.map((row) => row.key)).toEqual(['asset_code', 'line_no'])
    expect(buildDatapointAttributeDefaults(rows)).toEqual({
      asset_code: 'PUMP-001',
      line_no: '03',
    })
  })

  test('rejects reserved, duplicate, and malformed keys', () => {
    expect(validateDatapointAttributeRows([{ id: 1, key: 'quality', value: 'custom' }])).toBe(
      '属性 Key“quality”已被内置属性占用',
    )
    expect(
      validateDatapointAttributeRows([
        { id: 1, key: 'asset_code', value: 'A' },
        { id: 2, key: ' asset_code ', value: 'B' },
      ]),
    ).toBe('属性 Key“asset_code”重复')
    expect(validateDatapointAttributeRows([{ id: 1, key: 'Asset Code', value: 'A' }])).toContain(
      '格式无效',
    )
  })

  test('allows an empty attribute collection and empty string values', () => {
    expect(validateDatapointAttributeRows([])).toBeNull()
    expect(validateDatapointAttributeRows([{ id: 1, key: 'remark', value: '' }])).toBeNull()
  })
})
