import type { Datapoint } from '@/api/schemas/datapoint.schema'
import type {
  AlarmBinding,
  AlarmCondition,
  AlarmConditionKind,
  AlarmNotificationChannel,
  AlarmPolicy,
  AlarmPolicyMode,
  AlarmPolicySave,
  AlarmSeverity,
} from '@/api/schemas/alarm.schema'

export function builtinAlarmNotificationChannel(projectId: string): AlarmNotificationChannel {
  return {
    id: 'runtime_inapp',
    projectId,
    name: '运行端站内通知',
    channelType: 'runtime_inapp',
    config: { recipientScope: 'project_runtime_users' },
    secretStatus: {},
    isEnabled: true,
  }
}

export function ensureBuiltinAlarmNotificationChannel(
  projectId: string,
  channels: AlarmNotificationChannel[],
): AlarmNotificationChannel[] {
  if (channels.some((channel) => channel.id === 'runtime_inapp')) return channels
  return [builtinAlarmNotificationChannel(projectId), ...channels]
}

export const alarmSeverityLabels: Record<AlarmSeverity, string> = {
  info: '提示',
  warning: '警告',
  major: '重要',
  critical: '紧急',
}

export const alarmConditionLabels: Record<AlarmConditionKind, string> = {
  threshold: '阈值',
  range: '区间',
  state: '状态',
  transition: '状态变化',
  text_match: '文本匹配',
  rate_of_change: '变化率',
  deviation: '偏差',
  offline: '离线',
  expression: '表达式',
}

export type AlarmPointCategory = 'number' | 'boolean' | 'text' | 'structured'

export function alarmPointCategory(dataType?: string): AlarmPointCategory {
  const value = String(dataType || '').toLowerCase()
  if (/^(u?int\d*|float\d*|double|decimal|number|numeric)$/.test(value)) return 'number'
  if (value === 'bool' || value === 'boolean') return 'boolean'
  if (value === 'object' || value === 'array' || value === 'json') return 'structured'
  return 'text'
}

export function compatibleAlarmBindings(bindings: AlarmBinding[]): boolean {
  if (bindings.length < 2) return true
  const first = alarmPointCategory(bindings[0]?.dataType)
  return bindings.every((binding) => alarmPointCategory(binding.dataType) === first)
}

export function alarmBindingPreview(bindings: AlarmBinding[], mode: AlarmPolicyMode) {
  const threshold = mode === 'derived' ? 10 : 5
  const collapsed = bindings.length > threshold
  return {
    collapsed,
    visible: collapsed ? bindings.slice(0, 3) : bindings,
  }
}

export function conditionKindsFor(bindings: AlarmBinding[], mode: AlarmPolicyMode) {
  if (mode === 'derived')
    return ['expression', 'threshold', 'range', 'state'] as AlarmConditionKind[]
  if (!bindings.length) {
    return ['threshold', 'range', 'rate_of_change', 'deviation', 'offline'] as AlarmConditionKind[]
  }
  const category = alarmPointCategory(bindings[0]?.dataType)
  if (category === 'number') {
    return ['threshold', 'range', 'rate_of_change', 'deviation', 'offline'] as AlarmConditionKind[]
  }
  if (category === 'boolean') return ['state', 'transition', 'offline'] as AlarmConditionKind[]
  return ['state', 'text_match', 'offline'] as AlarmConditionKind[]
}

export function normalizeAlarmConditionsForBindings(
  conditions: AlarmCondition[],
  bindings: AlarmBinding[],
  mode: AlarmPolicyMode,
): AlarmCondition[] {
  const allowedKinds = conditionKindsFor(bindings, mode)
  const fallbackKind = allowedKinds[0]
  return conditions.map((condition) =>
    allowedKinds.includes(condition.kind)
      ? condition
      : {
          ...createAlarmCondition(fallbackKind),
          severity: condition.severity,
          triggerDelayMs: condition.triggerDelayMs,
          clearDelayMs: condition.clearDelayMs,
          deadband: condition.deadband,
        },
  )
}

export function createAlarmCondition(kind: AlarmConditionKind = 'threshold'): AlarmCondition {
  const operators: Record<AlarmConditionKind, string> = {
    threshold: 'gt',
    range: 'outside',
    state: 'eq',
    transition: 'changed',
    text_match: 'contains',
    rate_of_change: 'gt',
    deviation: 'gt',
    offline: 'is_offline',
    expression: 'is_true',
  }
  const params: Partial<Record<AlarmConditionKind, Record<string, unknown>>> = {
    threshold: { threshold: null },
    range: { lower: null, upper: null },
    state: { expected: true },
    text_match: { expected: '' },
    rate_of_change: { limit: null, windowMs: 60000 },
    deviation: { baseline: null, limit: null },
    expression: { expression: '' },
  }
  return {
    id: crypto.randomUUID(),
    kind,
    operator: operators[kind],
    label: alarmConditionLabels[kind],
    severity: 'warning',
    params: params[kind] || {},
    triggerDelayMs: 0,
    clearDelayMs: 0,
    deadband: 0,
  }
}

export function createAlarmPolicyDraft(mode: AlarmPolicyMode): AlarmPolicySave {
  return {
    groupId: null,
    name: '',
    description: null,
    mode,
    bindings: [],
    derivedExpression: '',
    conditions: [createAlarmCondition(mode === 'derived' ? 'state' : 'threshold')],
    notification: {
      mode: 'inherit',
      notifyOnRaise: true,
      notifyOnClear: true,
      repeatIntervalSeconds: null,
      channelIds: ['runtime_inapp'],
      messageTemplate: '',
    },
    isEnabled: mode === 'per_target',
  }
}

export function policyToDraft(policy: AlarmPolicy): AlarmPolicySave {
  return {
    groupId: policy.groupId || null,
    name: policy.name,
    description: policy.description || null,
    mode: policy.mode,
    bindings: policy.bindings.map((binding) => ({ ...binding })),
    derivedExpression: policy.derivedExpression,
    conditions: policy.conditions.map((condition) => ({
      ...condition,
      params: { ...condition.params },
    })),
    notification: { ...policy.notification, channelIds: [...policy.notification.channelIds] },
    isEnabled: policy.isEnabled,
  }
}

export function datapointToAlarmBinding(
  datapoint: Datapoint,
  mode: AlarmPolicyMode,
  index: number,
): AlarmBinding {
  const name = String(datapoint.name || datapoint.path || `变量${index + 1}`)
  const baseKey = name.replace(/[^A-Za-z0-9_]/g, '_').replace(/^[^A-Za-z_]+/, '')
  return {
    datapointId: String(datapoint.id),
    path: datapoint.path,
    name,
    dataType: datapoint.dataType || '',
    role: mode === 'derived' ? 'input' : 'target',
    inputKey: mode === 'derived' ? baseKey || `v${index + 1}` : null,
  }
}

export function alarmPolicyConditionSummary(policy: AlarmPolicy): string {
  const labels = policy.conditions.slice(0, 2).map((item) => alarmConditionLabels[item.kind])
  if (policy.conditions.length > 2) labels.push(`+${policy.conditions.length - 2}`)
  return labels.join('、') || '未配置'
}

export function alarmPolicyPointSummary(policy: AlarmPolicy): string {
  const first = policy.bindings[0]
  if (!first) return '未选择数据点'
  if (policy.bindings.length === 1) return first.name || first.path
  return `${first.name || first.path} 等 ${policy.bindings.length} 点`
}

export function alarmPolicyHighestSeverity(policy: AlarmPolicy): AlarmSeverity {
  const order: AlarmSeverity[] = ['info', 'warning', 'major', 'critical']
  return policy.conditions.reduce<AlarmSeverity>(
    (highest, condition) =>
      order.indexOf(condition.severity) > order.indexOf(highest) ? condition.severity : highest,
    'info',
  )
}
