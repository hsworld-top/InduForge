import type { Datapoint } from '@/api/schemas/datapoint.schema'
import type {
  AlarmCondition,
  AlarmConditionKind,
  AlarmItem,
  AlarmEvaluationMode,
  AlarmItemInput,
  AlarmItemMode,
  AlarmItemSave,
  AlarmNotificationChannel,
  AlarmSeverity,
  AlarmSeverityDefinition,
} from '@/api/schemas/alarm.schema'

export type AlarmPointCategory = 'number' | 'boolean' | 'text' | 'structured'
export interface AlarmPointSelection {
  datapointId: string
  path: string
  name: string
  dataType: string
  inputKey?: string
}
export interface AlarmItemDraft extends AlarmItemSave {
  selectedPoints: AlarmPointSelection[]
}

export function builtinAlarmNotificationChannel(projectId: string): AlarmNotificationChannel {
  return {
    id: 'runtime_inapp',
    projectId,
    name: '运行端站内通知',
    channelType: 'runtime_inapp',
    config: { recipientScope: 'project_runtime_users' },
    secretStatus: {},
    isEnabled: true,
    lastTestStatus: 'not_tested',
  }
}
export function ensureBuiltinAlarmNotificationChannel(
  projectId: string,
  channels: AlarmNotificationChannel[],
) {
  return channels.some((channel) => channel.id === 'runtime_inapp')
    ? channels
    : [builtinAlarmNotificationChannel(projectId), ...channels]
}

export const alarmSeverityLabels: Record<string, string> = {
  info: '提示',
  warning: '警告',
  major: '重要',
  critical: '紧急',
}
export const defaultAlarmSeverityDefinitions: AlarmSeverityDefinition[] = [
  {
    key: 'info',
    displayName: '提示',
    color: '#3b82f6',
    sortOrder: 10,
    isBuiltin: true,
  },
  {
    key: 'warning',
    displayName: '警告',
    color: '#f59e0b',
    sortOrder: 20,
    isBuiltin: true,
  },
  {
    key: 'major',
    displayName: '重要',
    color: '#f97316',
    sortOrder: 30,
    isBuiltin: true,
  },
  {
    key: 'critical',
    displayName: '紧急',
    color: '#ef4444',
    sortOrder: 40,
    isBuiltin: true,
  },
]
export function alarmSeverityLabel(
  severity: AlarmSeverity,
  definitions: AlarmSeverityDefinition[] = [],
) {
  return (
    definitions.find((definition) => definition.key === severity)?.displayName ||
    alarmSeverityLabels[severity] ||
    severity
  )
}
export const alarmConditionLabels: Record<AlarmConditionKind, string> = {
  threshold: '越限报警',
  range: '区间报警',
  state: '状态报警',
  transition: '状态变化报警',
  text_match: '文本报警',
  rate_of_change: '变化率报警',
  deviation: '偏差报警',
  offline: '离线报警',
  quality: '质量报警',
  stale: '数据陈旧报警',
}

export function alarmPointCategory(dataType?: string): AlarmPointCategory {
  const value = String(dataType || '').toLowerCase()
  if (/^(u?int\d*|float\d*|double|decimal|number|numeric)$/.test(value)) return 'number'
  if (value === 'bool' || value === 'boolean') return 'boolean'
  if (
    ['object', 'array', 'json', 'bytes', 'byte[]', 'datetime', 'date', 'timestamp'].includes(value)
  )
    return 'structured'
  return 'text'
}
export function compatibleAlarmPoints(points: AlarmPointSelection[]) {
  if (points.length < 2) return true
  const first = alarmPointCategory(points[0]?.dataType)
  return points.every((point) => alarmPointCategory(point.dataType) === first)
}
export function conditionKindsFor(
  points: AlarmPointSelection[],
  mode: AlarmItemMode,
  evaluationMode?: AlarmEvaluationMode,
): AlarmConditionKind[] {
  // 组合表达式是输入值的计算方式，不是结果条件本身；结果再用普通条件判断。
  if (mode === 'derived') return ['threshold', 'range', 'state', 'text_match']
  if (!points.length || alarmPointCategory(points[0]?.dataType) === 'number') {
    const kinds: AlarmConditionKind[] = [
      'threshold',
      'range',
      'rate_of_change',
      'deviation',
      'quality',
      'stale',
      'offline',
    ]
    // 数值点的越限由 highest_matching 统一管理，避免“其他类型”重复创建单阈值越限。
    return evaluationMode === 'single' ? kinds.filter((kind) => kind !== 'threshold') : kinds
  }
  if (alarmPointCategory(points[0]?.dataType) === 'boolean')
    return ['state', 'transition', 'quality', 'stale', 'offline']
  if (alarmPointCategory(points[0]?.dataType) === 'structured')
    return ['quality', 'stale', 'offline']
  return ['state', 'transition', 'text_match', 'quality', 'stale', 'offline']
}

export function createAlarmCondition(
  kind: AlarmConditionKind = 'threshold',
  label = '',
): AlarmCondition {
  const operators: Record<AlarmConditionKind, string> = {
    threshold: 'gt',
    range: 'outside',
    state: 'eq',
    transition: 'changed',
    text_match: 'contains',
    rate_of_change: 'gt',
    deviation: 'gt',
    offline: 'is_offline',
    quality: 'in',
    stale: 'age_gte',
  }
  const params: Partial<Record<AlarmConditionKind, Record<string, unknown>>> = {
    threshold: { threshold: null },
    range: { lower: null, upper: null },
    state: { expected: true },
    transition: { from: false, to: true },
    text_match: { expected: '' },
    rate_of_change: { direction: 'absolute', limit: null, windowMs: 60000 },
    deviation: { baseline: null, limit: null },
    quality: { qualities: ['bad'] },
    stale: { maxAgeMs: 60000 },
  }
  return {
    id: crypto.randomUUID(),
    kind,
    operator: operators[kind],
    label: label || alarmConditionLabels[kind],
    severity: 'warning',
    params: params[kind] || {},
    triggerDelayMs: 0,
    clearDelayMs: 0,
    deadband: 0,
  }
}

export function createAlarmItemDraft(mode: AlarmItemMode): AlarmItemDraft {
  return {
    datapointId: '',
    displayName: '',
    groupId: null,
    description: null,
    mode,
    evaluationMode: mode === 'point' ? 'highest_matching' : 'single',
    inputs: [],
    derivedExpression: '',
    conditions: [
      createAlarmCondition(
        mode === 'derived' ? 'state' : 'threshold',
        mode === 'derived' ? '结果条件' : '高',
      ),
    ],
    notification: {
      mode: 'inherit',
      notifyOnRaise: true,
      notifyOnClear: true,
      repeatIntervalSeconds: null,
      channelIds: ['runtime_inapp'],
      messageTemplate: '',
    },
    isEnabled: mode === 'point',
    revision: 0,
    acknowledgedWarningKeys: [],
    selectedPoints: [],
  }
}

export function alarmItemToDraft(item: AlarmItem): AlarmItemDraft {
  const selectedPoints: AlarmPointSelection[] =
    item.mode === 'point'
      ? [
          {
            datapointId: item.datapointId,
            path: item.path,
            name: item.datapointName,
            dataType: item.dataType,
          },
        ]
      : item.inputs.map((input) => ({
          datapointId: input.datapointId,
          path: input.path,
          name: input.name,
          dataType: input.dataType,
          inputKey: input.inputKey,
        }))
  return {
    itemId: item.id,
    datapointId: item.datapointId,
    displayName: item.displayName,
    presetSlot: item.presetSlot || null,
    groupId: item.groupId || null,
    description: item.description || null,
    mode: item.mode,
    evaluationMode: item.evaluationMode,
    inputs: item.inputs.map((entry) => ({ ...entry })),
    derivedExpression: item.derivedExpression,
    conditions: item.conditions.map((condition) => ({
      ...condition,
      params: { ...condition.params },
    })),
    notification: { ...item.notification, channelIds: [...item.notification.channelIds] },
    isEnabled: item.isEnabled,
    revision: item.revision,
    acknowledgedWarningKeys: [],
    selectedPoints,
  }
}

export function datapointToAlarmPoint(
  datapoint: Datapoint,
  mode: AlarmItemMode,
  index: number,
): AlarmPointSelection {
  const name = String(datapoint.name || datapoint.path || `变量${index + 1}`)
  const baseKey = name.replace(/[^A-Za-z0-9_]/g, '_').replace(/^[^A-Za-z_]+/, '')
  return {
    datapointId: String(datapoint.id),
    path: datapoint.path,
    name,
    dataType: datapoint.dataType || '',
    inputKey: mode === 'derived' ? baseKey || `v${index + 1}` : undefined,
  }
}

export function buildAlarmItemPayload(draft: AlarmItemDraft): AlarmItemSave {
  const inputs: AlarmItemInput[] =
    draft.mode === 'derived'
      ? draft.selectedPoints.map((point, index) => ({
          id: '',
          datapointId: point.datapointId,
          path: point.path,
          name: point.name,
          dataType: point.dataType,
          inputKey: point.inputKey?.trim() || `v${index + 1}`,
        }))
      : []
  return {
    itemId: draft.itemId,
    datapointId:
      draft.mode === 'point' ? draft.selectedPoints[0]?.datapointId || draft.datapointId : '',
    displayName: draft.displayName.trim(),
    groupId: draft.groupId,
    description: draft.description,
    mode: draft.mode,
    evaluationMode: draft.evaluationMode,
    inputs,
    derivedExpression: draft.derivedExpression.trim(),
    conditions: sortAlarmLevels(draft.conditions),
    notification: draft.notification,
    isEnabled: draft.isEnabled,
    revision: draft.revision,
    acknowledgedWarningKeys: draft.acknowledgedWarningKeys,
  }
}

export function sortAlarmLevels(conditions: AlarmCondition[]) {
  return [...conditions].sort((left, right) => {
    const leftHigh = ['gt', 'gte'].includes(left.operator)
    const rightHigh = ['gt', 'gte'].includes(right.operator)
    if (leftHigh !== rightHigh) return leftHigh ? -1 : 1
    const a = Number(left.params.threshold ?? 0)
    const b = Number(right.params.threshold ?? 0)
    return leftHigh ? a - b : b - a
  })
}
export function alarmItemConditionSummary(item: AlarmItem) {
  if (item.evaluationMode === 'highest_matching')
    return `越限 · ${item.conditions.map((condition) => condition.label).join('/')}`
  return item.conditions[0] ? alarmConditionLabels[item.conditions[0].kind] : '未配置'
}
export function alarmItemPointSummary(item: AlarmItem) {
  if (item.mode === 'point') return item.datapointName || item.path || '数据点'
  const first = item.inputs[0]
  if (!first) return '组合报警'
  return item.inputs.length === 1
    ? first.name || first.path
    : `组合报警 · ${first.name || first.path} 等 ${item.inputs.length} 点`
}
export function alarmItemHighestSeverity(
  item: AlarmItem,
  definitions: AlarmSeverityDefinition[] = [],
): AlarmSeverity {
  const order = definitions.length
    ? definitions.map((definition) => definition.key)
    : ['info', 'warning', 'major', 'critical']
  return item.conditions.reduce<AlarmSeverity>(
    (highest, condition) =>
      order.indexOf(condition.severity) > order.indexOf(highest) ? condition.severity : highest,
    'info',
  )
}
