import { describe, expect, test } from 'vitest'

import { datacenterModules, getDatacenterModule } from '../src/config/datacenterModules'

describe('datacenterModules', () => {
  test('按 v2 设计顺序提供六个数据中心模块', () => {
    expect(datacenterModules.map((item) => item.id)).toEqual([
      'datapoint',
      'access-source',
      'industrial-collector',
      'history-storage',
      'compute',
      'alarm',
    ])
  })

  test('getDatacenterModule 根据 v2 模块 ID 返回元数据', () => {
    expect(getDatacenterModule('datapoint')?.label).toBe('数据点')
    expect(getDatacenterModule('industrial-collector')?.label).toBe('工业采集')
    expect(getDatacenterModule('history-storage')?.label).toBe('历史存储')
    expect(getDatacenterModule('compute')?.label).toBe('计算单元')
    expect(getDatacenterModule('alarm')?.label).toBe('报警单元')
    expect(getDatacenterModule('missing')).toBeNull()
  })
})
