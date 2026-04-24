export type DatacenterModuleId =
  | "datapoints"
  | "access-sources"
  | "compute-units"
  | "alarm-units";

export interface DatacenterModuleMeta {
  id: DatacenterModuleId;
  label: string;
  description: string;
}

/**
 * 数据中心一级模块定义。
 * 这里使用中文 label，避免首版壳层依赖尚未补齐的 i18n key。
 */
export const datacenterModules: DatacenterModuleMeta[] = [
  {
    id: "datapoints",
    label: "数据点",
    description: "统一查看查询、MQTT、计算等来源沉淀的数据点。",
  },
  {
    id: "access-sources",
    label: "接入源",
    description: "维护数据库与 MQTT 连接，并继续承载旧查询工作台。",
  },
  {
    id: "compute-units",
    label: "计算单元",
    description: "构建可调度脚本任务，支持输出数据点或执行写库、发布、请求等动作。",
  },
  {
    id: "alarm-units",
    label: "报警单元",
    description: "管理报警规则配置与运行态契约预览，首版先保留规则构建入口。",
  },
];

/**
 * 根据模块 ID 查找元数据。
 * 未知 ID 返回 null，便于路由或外部状态做安全降级。
 */
export const getDatacenterModule = (id: string) => {
  return datacenterModules.find((item) => item.id === id) ?? null;
};
