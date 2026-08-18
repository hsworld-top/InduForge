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
    description: '维护数据库、消息和接口类数据接入连接。',
  },
  {
    id: 'industrial-collector',
    label: '工业采集',
    description: '管理工业协议连接、采集点位与开发态调试。',
  },
  {
    id: 'history-storage',
    label: '历史存储',
    description: '按接入源或工业采集连接设置数据点历史保存方式。',
  },
  {
    id: 'compute',
    label: '计算单元',
    description: '构建可调度脚本任务，支持输出数据点或执行写库、发布、请求等动作。',
  },
  {
    id: 'alarm',
    label: '报警单元',
    description: '管理报警策略配置与运行态契约预览。',
  },
]

/**
 * 根据模块 ID 查找元数据。
 * 未知 ID 返回 null，便于路由或外部状态做安全降级。
 */
export const getDatacenterModule = (id: string) => {
  return datacenterModules.find((item) => item.id === id) ?? null
}
