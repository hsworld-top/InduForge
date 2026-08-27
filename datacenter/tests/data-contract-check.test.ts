import { describe, expect, test } from 'vitest'

import {
  ContractCheckResultSchema,
  ContractCheckRunPageSchema,
} from '../src/api/schemas/contract-check.schema'
import {
  dataContractCheckStatusText,
  dataContractCheckSummary,
} from '../src/components/contract/dataContractCheck'

describe('data contract check contract', () => {
  test('使用 data_service 的 list/summary/checkedAt 真实结果', () => {
    const result = ContractCheckResultSchema.parse({
      status: 'failed',
      projectId: 'project-1',
      scope: 'project',
      summary: { passed: 0, warning: 1, pending: 0, failed: 1 },
      list: [
        {
          module: 'datapoint',
          objectType: 'datapoint',
          objectId: 'point-1',
          status: 'failed',
          title: '数据点已失效',
          detail: 'factory.line.speed',
          action: '恢复数据点来源',
        },
      ],
      checkedAt: '2026-08-26T00:00:00Z',
    })

    expect(result.list).toHaveLength(1)
    expect(result.summary.failed).toBe(1)
    expect(result.status).toBe('failed')
  })

  test('历史列表使用 runs 分页形状', () => {
    const page = ContractCheckRunPageSchema.parse({
      list: [
        {
          id: 12,
          projectId: 'project-1',
          scope: 'project',
          status: 'warning',
          summary: { passed: 0, warning: 1, pending: 0, failed: 0 },
          result: {},
          createdAt: '2026-08-26T00:00:00Z',
        },
      ],
      pagination: { page: 1, pageSize: 8, total: 1, totalPages: 1 },
    })
    expect(page.list[0]?.id).toBe(12)
    expect(page.pagination.total).toBe(1)
  })

  test('文案不再将未接入的设计中心或运行态契约计为通过', () => {
    expect(dataContractCheckSummary.title).toContain('发布前')
    expect(dataContractCheckSummary.description).toContain('data_service')
    expect(dataContractCheckSummary.description).toContain('不是运行态事件看板')
    expect(dataContractCheckSummary.extensionNote).toContain('不在本页计为通过')
    expect(dataContractCheckStatusText.failed).toBe('失败')
  })
})
