import type { V2ModuleId } from '@/router/route-config'

// v2 模块 ID 即 V2ModuleId，这里重导出方便内部复用
export type DatacenterModuleId = V2ModuleId

export interface DatacenterModuleMeta {
  id: DatacenterModuleId
  label: string
  description: string
}

/**
 * 数据中心一级模块定义（v2 模块 ID）。
 * label 使用中文，避免首版壳层依赖尚未补齐的 i18n key。
 */
export const datacenterModules: DatacenterModuleMeta[] = [
  {
    id: 'datapoint',
    label: '数据点',
    description: '统一查看查询、MQTT、计算等来源沉淀的数据点。',
  },
  {
    id: 'access-source',
    label: '接入源',
    description: '维护数据库与 MQTT 连接，并承载接入源二级工作台。',
  },
  {
    id: 'storage-policy',
    label: '存储策略',
    description: '配置统一数据点的历史归档目标、写入模式、保留策略和容量预估。',
  },
  {
    id: 'compute',
    label: '计算单元',
    description: '构建可调度脚本任务，支持输出数据点或执行写库、发布、请求等动作。',
  },
  {
    id: 'alarm',
    label: '报警单元',
    description: '管理报警规则配置与运行态契约预览。',
  },
]

/**
 * 根据模块 ID 查找元数据。
 * 未知 ID 返回 null，便于路由或外部状态做安全降级。
 */
export const getDatacenterModule = (id: string) => {
  return datacenterModules.find((item) => item.id === id) ?? null
}
