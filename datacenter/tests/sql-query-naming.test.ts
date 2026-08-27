import { describe, expect, test } from 'vitest'
import {
  generatedQueryCopyName,
  nextGeneratedQueryName,
} from '@/components/database/sqlQueryNaming'

describe('SQL 查询自动命名', () => {
  test('关闭临时查询后会复用最小可用序号', () => {
    expect(nextGeneratedQueryName([])).toBe('查询_1')
    expect(nextGeneratedQueryName(['查询_2'])).toBe('查询_1')
    expect(nextGeneratedQueryName(['查询_1', '查询_3'])).toBe('查询_2')
  })

  test('旧空格格式也占用相同序号', () => {
    expect(nextGeneratedQueryName(['查询 1', '查询_2'])).toBe('查询_3')
  })

  test('副本名称使用下划线', () => {
    expect(generatedQueryCopyName('查询_1')).toBe('查询_1_副本')
  })
})
