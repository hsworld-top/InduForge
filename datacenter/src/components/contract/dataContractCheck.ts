export type DataContractCheckStatus = 'passed' | 'warning' | 'pending'

export interface DataContractCheckItem {
  id: string
  title: string
  scope: string
  status: DataContractCheckStatus
  summary: string
  checkpoint: string
  owner: string
}

export const dataContractCheckSummary = {
  title: '发布前数据契约检查',
  description:
    '面向页面发布和节点运行前的契约一致性检查，用于确认数据中心、设计中心与运行态节点之间的数据点消费边界；这里不是运行态事件看板。',
  extensionNote: '当前为前端本地检查模型，后续可选接入发布流水线 dry-run 或节点契约校验接口。',
}

/**
 * 数据契约检查项是发布前对齐清单，不直接触发运行态查询、订阅或报警事件。
 * 后续接入后端时应保持这些稳定 ID，便于测试和发布流水线复用。
 */
export const dataContractCheckItems: DataContractCheckItem[] = [
  {
    id: 'datapoint-definition',
    title: '数据点定义',
    scope: '数据中心',
    status: 'passed',
    summary: '校验 path、类型、来源、质量策略和读写能力是否完整。',
    checkpoint: '确认数据点定义可被设计中心发现，并能映射为运行态查询/订阅输入。',
    owner: '数据点工作区',
  },
  {
    id: 'access-source-connection',
    title: '接入源连接',
    scope: '数据中心',
    status: 'warning',
    summary: '检查关系库、MQTT 等接入源是否存在连接配置、预览能力和来源追踪。',
    checkpoint: '发布前需要确认接入源连接可用，避免运行态节点拿到失效来源。',
    owner: '接入源工作区',
  },
  {
    id: 'designer-reference',
    title: '设计中心引用',
    scope: '设计中心',
    status: 'passed',
    summary: '检查页面绑定的数据点引用是否仍存在，绑定路径是否发生漂移。',
    checkpoint: '保证设计中心组件绑定的数据点在发布包中可解析、可追溯。',
    owner: '设计中心绑定',
  },
  {
    id: 'runtime-query-subscription',
    title: '运行态查询/订阅契约',
    scope: '运行态节点',
    status: 'passed',
    summary: '核对查询型与订阅型数据点的参数、权限、频率和返回结构。',
    checkpoint: '确认运行态查询/订阅契约完整，节点侧可以按统一接口消费数据点。',
    owner: '节点运行态',
  },
  {
    id: 'compute-side-effect',
    title: '计算单元副作用',
    scope: '数据中心',
    status: 'warning',
    summary: '识别写库、MQTT/Kafka 发布、HTTP 请求等副作用动作及授权状态。',
    checkpoint: '发布前需要明确副作用边界，避免计算任务在运行态产生未授权外部写入。',
    owner: '计算单元',
  },
  {
    id: 'alarm-rule-contract',
    title: '报警规则契约',
    scope: '运行态节点',
    status: 'pending',
    summary: '检查报警目标点、规则表达式、阈值窗口和节点侧执行契约。',
    checkpoint: '数据中心只做规则契约预览，报警实例、事件、确认和消音留在运行态侧。',
    owner: '报警单元',
  },
]

export const dataContractCheckStatusText: Record<DataContractCheckStatus, string> = {
  passed: '可发布',
  warning: '需确认',
  pending: '待接入',
}
