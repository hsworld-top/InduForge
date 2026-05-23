import { describe, expect, test } from 'vitest'

import {
  dataContractCheckItems,
  dataContractCheckSummary,
} from '../src/components/contract/dataContractCheck'

describe('data contract check metadata', () => {
  test('发布前契约检查覆盖数据中心到设计中心和运行态节点的关键边界', () => {
    expect(dataContractCheckItems.map((item) => item.id)).toEqual([
      'datapoint-definition',
      'access-source-connection',
      'designer-reference',
      'runtime-query-subscription',
      'compute-side-effect',
      'alarm-rule-contract',
    ])
  })

  test('契约检查文案明确面向发布或运行前一致性，不表达为运行态事件看板', () => {
    expect(dataContractCheckSummary.title).toContain('发布前')
    expect(dataContractCheckSummary.description).toContain('契约一致性检查')
    expect(dataContractCheckSummary.description).toContain('不是运行态事件看板')

    const allCopy = dataContractCheckItems
      .map((item) => `${item.title} ${item.summary} ${item.checkpoint}`)
      .join(' ')

    expect(allCopy).toContain('数据点')
    expect(allCopy).toContain('接入源')
    expect(allCopy).toContain('设计中心')
    expect(allCopy).toContain('查询/订阅')
    expect(allCopy).toContain('副作用')
    expect(allCopy).toContain('报警规则')
  })
})
