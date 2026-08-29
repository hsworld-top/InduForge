import type { ContractCheckStatus } from '@/api/schemas/contract-check.schema'

export const getDataContractCheckSummary = (locale: 'zh' | 'en') =>
  locale === 'en'
    ? {
        title: 'Pre-release Data Contract Check',
        description:
          'Runs a project-level data_service dry run to validate development contracts for data points, queries, compute units, and alarms. This is not a runtime event dashboard.',
        extensionNote:
          'Design Center and node runtime contracts outside the data_service dry-run scope are not counted as passed here.',
      }
    : {
        title: '发布前数据契约检查',
        description:
          '执行 data_service 的项目级 dry-run，检查数据点、查询、计算和报警的开发态契约；这里不是运行态事件看板。',
        extensionNote: '未纳入 data_service dry-run 的设计中心和节点运行态契约不在本页计为通过。',
      }

export const getDataContractCheckStatusText = (
  locale: 'zh' | 'en',
): Record<ContractCheckStatus, string> =>
  locale === 'en'
    ? { passed: 'Passed', warning: 'Review', pending: 'Pending', failed: 'Failed' }
    : { passed: '通过', warning: '需确认', pending: '待完成', failed: '失败' }

export const getDataContractCheckModuleText = (locale: 'zh' | 'en'): Record<string, string> =>
  locale === 'en'
    ? { datapoint: 'Data Point', query: 'Query', compute: 'Compute Unit', alarm: 'Alarm Unit' }
    : { datapoint: '数据点', query: '查询', compute: '计算单元', alarm: '报警单元' }

// 保留中文默认导出，兼容现有契约测试；界面使用上面的按语言函数。
export const dataContractCheckSummary = getDataContractCheckSummary('zh')
export const dataContractCheckStatusText = getDataContractCheckStatusText('zh')
export const dataContractCheckModuleText = getDataContractCheckModuleText('zh')
