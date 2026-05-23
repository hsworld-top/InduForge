export type ComputeTaskModeId = 'output-datapoint' | 'side-effect' | 'hybrid'

export type ComputeTriggerModeId =
  | 'manual-debug'
  | 'schedule'
  | 'bool-datapoint'
  | 'datapoint-change'

export type ComputeCapabilityCategory =
  | 'datapoint'
  | 'sql'
  | 'math'
  | 'text'
  | 'json'
  | 'http'
  | 'messaging'

export type ComputeTaskMode = {
  id: ComputeTaskModeId
  label: string
  badge: string
  summary: string
  outputPolicy: 'required' | 'none' | 'optional'
}

export type ComputeTriggerMode = {
  id: ComputeTriggerModeId
  label: string
  badge: string
  summary: string
}

export type ComputeCapability = {
  category: ComputeCapabilityCategory
  title: string
  signature: string
  summary: string
}

// 这些元数据只描述前端工作台文案与脚本 SDK 能力入口，不新增任何后端必需调用。
export const computeTaskModes: ComputeTaskMode[] = [
  {
    id: 'output-datapoint',
    label: '输出数据点',
    badge: 'calc.*',
    summary: '脚本结果生成或更新计算输出数据点，供设计器、运行态页面和其他任务订阅使用。',
    outputPolicy: 'required',
  },
  {
    id: 'side-effect',
    label: '无输出副作用任务',
    badge: 'task',
    summary:
      '计算可以不输出数据点，可以在脚本内写库、发布 MQTT/Kafka、发 HTTP 请求或完成同步动作。',
    outputPolicy: 'none',
  },
  {
    id: 'hybrid',
    label: '混合任务',
    badge: 'mixed',
    summary: '同一次执行既可输出 calc.* 数据点，也可写目标库、发布消息或调用外部接口。',
    outputPolicy: 'optional',
  },
]

export const computeTriggerModes: ComputeTriggerMode[] = [
  {
    id: 'manual-debug',
    label: '手动/调试',
    badge: 'dev',
    summary: '开发态手动运行，用于校验输入、输出、副作用和错误分支。',
  },
  {
    id: 'schedule',
    label: '定时执行',
    badge: 'cron',
    summary: '按固定周期或计划时间触发，适合日报、同步和批处理任务。',
  },
  {
    id: 'bool-datapoint',
    label: 'Bool 数据点触发',
    badge: 'edge',
    summary: '监听布尔点位，建议按 false -> true 边沿触发并设置最小间隔。',
  },
  {
    id: 'datapoint-change',
    label: '数据点变化触发',
    badge: 'change',
    summary: '监听一个或多个数据点值变化，适合事件驱动计算或联动动作。',
  },
]

export const computeCapabilities: ComputeCapability[] = [
  {
    category: 'datapoint',
    title: '获取数据点当前值',
    signature: 'ctx.datapoint.get(path)',
    summary: '读取 value、quality、time，用于计算、判断和触发前校验。',
  },
  {
    category: 'sql',
    title: '关系库 SQL 查询',
    signature: 'ctx.sql.query(source, sql, args)',
    summary: '面向各接入源关系库执行参数化查询，返回 rows 与执行信息。',
  },
  {
    category: 'sql',
    title: '关系库 SQL 写入',
    signature: 'ctx.sql.execute(source, sql, args)',
    summary: '写入目标关系库，必须使用参数化参数并受目标接入源授权约束。',
  },
  {
    category: 'math',
    title: '数学方法',
    signature: 'ctx.math.avg/sum/clamp/round',
    summary: '提供均值、求和、限幅、四舍五入等常用计算辅助方法。',
  },
  {
    category: 'text',
    title: '字符串方法',
    signature: 'ctx.text.format/regex/trim',
    summary: '处理标签、模板文本、正则提取和字符串清洗。',
  },
  {
    category: 'json',
    title: 'JSON 方法',
    signature: 'ctx.json.path/parse/stringify',
    summary: '读取嵌套字段、解析载荷和生成结构化消息体。',
  },
  {
    category: 'http',
    title: 'HTTP 请求',
    signature: 'ctx.http.get/post/put',
    summary: '调用外部接口，受域名白名单、超时和密钥引用策略约束。',
  },
  {
    category: 'messaging',
    title: 'MQTT/Kafka 发布',
    signature: 'ctx.mqtt.publish / ctx.kafka.publish',
    summary: '向指定接入源发布消息，Topic、序列化格式和目标源需要授权。',
  },
]

export const computeSafetyConstraints = [
  '副作用动作需要显式授权目标接入源、Topic 或 HTTP 域名。',
  'SQL 示例坚持参数化查询/写入，不在脚本文案中鼓励拼接 SQL。',
  '无输出任务允许没有 calc.* 数据点，调试结果应明确展示副作用执行情况。',
  '运行态执行仍由节点侧负责，前端只提供配置、调试入口和契约提示。',
]

export const computeDebugHints = [
  '先用手动/调试验证输入 JSON、脚本返回值和异常分支。',
  'Bool 数据点触发建议配置边沿触发和最小间隔，避免抖动重复执行。',
  'HTTP/MQTT/Kafka/写库任务建议准备幂等键，便于失败重试。',
]
