export type AlarmRuleTypeId = "high" | "low" | "deviation" | "rate" | "expression";

export type AlarmSeverityId = "critical" | "major" | "minor" | "notice";

export type AlarmRuleStatus = "enabled" | "draft" | "invalid";

export type AlarmRuleType = {
  id: AlarmRuleTypeId;
  label: string;
  badge: string;
  summary: string;
  expressionHint: string;
};

export type AlarmSeverity = {
  id: AlarmSeverityId;
  label: string;
  level: number;
  tone: "danger" | "warning" | "info" | "neutral";
};

export type AlarmRuleSample = {
  id: string;
  name: string;
  group: string;
  status: AlarmRuleStatus;
  targetPath: string;
  qualityCondition: string;
  onlineCondition: string;
  ruleType: AlarmRuleTypeId;
  trigger: string;
  recovery: string;
  severity: AlarmSeverityId;
  suppression: string;
  messageTemplate: string;
  runtimeNodeContract: string;
  contractStatus: "valid" | "pending" | "broken";
};

export type AlarmContractField = {
  key: string;
  label: string;
  required: boolean;
  summary: string;
};

// 本文件只维护报警规则构建工作台的前端元数据，不声明数据中心具备运行态事件处理能力。
export const alarmRuleTypes: AlarmRuleType[] = [
  {
    id: "high",
    label: "高限阈值",
    badge: "HH/H",
    summary: "目标数据点高于阈值并满足持续时间后，由运行态节点判定触发。",
    expressionHint: 'point.value > threshold && point.quality == "good"',
  },
  {
    id: "low",
    label: "低限阈值",
    badge: "L/LL",
    summary: "目标数据点低于阈值后进入报警判定，适合压力、液位等下限保护。",
    expressionHint: 'point.value < threshold && point.online == true',
  },
  {
    id: "deviation",
    label: "偏差规则",
    badge: "DEV",
    summary: "比较目标值与基准值、设定值或另一数据点之间的偏差。",
    expressionHint: "abs(point.value - reference.value) > deadband",
  },
  {
    id: "rate",
    label: "变化率",
    badge: "ROC",
    summary: "按时间窗口检测变化速度，适合温升、压降和能耗突变。",
    expressionHint: "rate(point.value, window) > limit",
  },
  {
    id: "expression",
    label: "表达式",
    badge: "CEL",
    summary: "面向复杂条件组合，契约中只保存可由节点侧解释执行的表达式。",
    expressionHint: 'point.value > 80 && inputs.pumpRunning == true',
  },
];

export const alarmSeverities: AlarmSeverity[] = [
  { id: "critical", label: "1 紧急", level: 1, tone: "danger" },
  { id: "major", label: "2 重要", level: 2, tone: "warning" },
  { id: "minor", label: "3 一般", level: 3, tone: "info" },
  { id: "notice", label: "4 提示", level: 4, tone: "neutral" },
];

export const alarmRuleSamples: AlarmRuleSample[] = [
  {
    id: "line-a-temperature-high",
    name: "一号产线高温",
    group: "温度规则",
    status: "enabled",
    targetPath: "mqtt.EMQX.LineA.temperature",
    qualityCondition: "quality == good",
    onlineCondition: "online == true",
    ruleType: "high",
    trigger: "> 80 ℃，持续 10s",
    recovery: "< 75 ℃，持续 30s",
    severity: "major",
    suppression: "10 分钟内同目标合并",
    messageTemplate: "{{target.name}} 当前 {{value}}℃，超过 {{threshold}}℃",
    runtimeNodeContract: "node follows page-runtime deployment",
    contractStatus: "valid",
  },
  {
    id: "line-a-pressure-low",
    name: "一号产线低压",
    group: "压力规则",
    status: "enabled",
    targetPath: "mqtt.EMQX.LineA.pressure",
    qualityCondition: "quality == good",
    onlineCondition: "online == true",
    ruleType: "low",
    trigger: "< 0.35 MPa，持续 15s",
    recovery: "> 0.42 MPa，持续 45s",
    severity: "critical",
    suppression: "恢复前不重复通知",
    messageTemplate: "{{target.name}} 压力低于安全下限：{{value}}MPa",
    runtimeNodeContract: "node-agent alarm.threshold.low.v1",
    contractStatus: "valid",
  },
  {
    id: "power-deviation",
    name: "功率偏差",
    group: "能耗规则",
    status: "draft",
    targetPath: "calc.LineA.power",
    qualityCondition: "quality in [good, stale]",
    onlineCondition: "online == true",
    ruleType: "deviation",
    trigger: "abs(value - baseline) > 12%，持续 60s",
    recovery: "abs(value - baseline) < 6%，持续 120s",
    severity: "minor",
    suppression: "按班次最多 1 次提醒",
    messageTemplate: "{{target.name}} 与基线偏差 {{delta}}%",
    runtimeNodeContract: "node-agent alarm.deviation.v1",
    contractStatus: "pending",
  },
  {
    id: "legacy-temperature",
    name: "旧温度规则",
    group: "待修复",
    status: "invalid",
    targetPath: "invalid.LineA.temp",
    qualityCondition: "quality == good",
    onlineCondition: "online == true",
    ruleType: "high",
    trigger: "> 90 ℃，持续 5s",
    recovery: "< 84 ℃，持续 30s",
    severity: "notice",
    suppression: "无",
    messageTemplate: "旧规则目标点失效，需要重新绑定",
    runtimeNodeContract: "missing target datapoint",
    contractStatus: "broken",
  },
];

export const alarmContractFields: AlarmContractField[] = [
  {
    key: "targetPath",
    label: "目标数据点",
    required: true,
    summary: "运行态节点按 path 订阅或读取数据点值、质量和在线状态。",
  },
  {
    key: "qualityCondition",
    label: "质量/在线条件",
    required: true,
    summary: "触发前先判断 quality 与 online，避免离线或坏点误触发。",
  },
  {
    key: "ruleType",
    label: "规则类型与表达式",
    required: true,
    summary: "阈值、偏差、变化率或表达式必须能被节点侧规则引擎解释。",
  },
  {
    key: "recoveryPolicy",
    label: "恢复策略",
    required: true,
    summary: "恢复阈值、持续时间和回差用于节点侧关闭报警状态。",
  },
  {
    key: "suppressionPolicy",
    label: "抑制策略",
    required: true,
    summary: "只定义重复触发合并策略，不提供数据中心运行态事件处置操作。",
  },
  {
    key: "messageTemplate",
    label: "消息模板",
    required: true,
    summary: "模板随契约发布，节点侧在产生事件时填充变量。",
  },
];

export const alarmRuntimeBoundaryNotes = [
  "数据中心只负责构建规则、校验目标数据点和预览节点侧执行契约。",
  "规则执行、报警事件产生、事件确认等运行态流程由节点侧负责。",
  "本工作台不提供任何运行态事件管理模块。",
];
