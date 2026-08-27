import type { ContractCheckStatus } from '@/api/schemas/contract-check.schema'

export const dataContractCheckSummary = {
  title: '发布前数据契约检查',
  description:
    '执行 data_service 的项目级 dry-run，检查数据点、查询、计算和报警的开发态契约；这里不是运行态事件看板。',
  extensionNote: '未纳入 data_service dry-run 的设计中心和节点运行态契约不在本页计为通过。',
}

export const dataContractCheckStatusText: Record<ContractCheckStatus, string> = {
  passed: '通过',
  warning: '需确认',
  pending: '待完成',
  failed: '失败',
}

export const dataContractCheckModuleText: Record<string, string> = {
  datapoint: '数据点',
  query: '查询',
  compute: '计算单元',
  alarm: '报警单元',
}
