import type {
  HistoryStorageBehavior,
  HistoryStorageConfiguration,
  HistoryStorageSourceItem,
  HistoryStorageTargetOption,
  HistoryStorageWriteMode,
} from '@/api/schemas/history-storage.schema'
import type {
  HistoryStorageConfigurationPayload,
  HistoryStorageSavePayload,
} from '@/api/history-storage.api'

export const historyStorageModeLabels: Record<HistoryStorageWriteMode, string> = {
  on_change: '变化时保存',
  interval_latest: '按间隔保存',
  periodic_snapshot: '按周期保存',
  every_sample: '保存每次采样',
}

export type HistoryStorageDraftTarget = {
  connectionId: string
  retentionDays: number | null
}

export type HistoryStorageDraft = {
  behavior: HistoryStorageBehavior
  writeMode: HistoryStorageWriteMode
  intervalMs: number
  deadband: number
  maxSilenceMs: number | null
  offlineBehavior: 'store_stale' | 'skip'
  targets: HistoryStorageDraftTarget[]
}

export function createHistoryStorageDraft(
  behavior: HistoryStorageBehavior,
  configuration: HistoryStorageConfiguration | null | undefined,
  targets: HistoryStorageTargetOption[],
): HistoryStorageDraft {
  const uniqueBuiltin = targets.filter((target) => target.type === 'builtin.timeseries')
  const defaultTargetId = uniqueBuiltin.length === 1 ? uniqueBuiltin[0].id : ''
  const configuredTargets = configuration?.targets.map((target) => ({
    connectionId: target.connectionId,
    retentionDays: target.retentionDays,
  }))
  return {
    behavior,
    writeMode: configuration?.writeMode || 'on_change',
    intervalMs: configuration?.intervalMs || 60_000,
    deadband: configuration?.deadband ?? 0,
    maxSilenceMs:
      configuration?.maxSilenceMs === undefined ? 3_600_000 : configuration.maxSilenceMs,
    offlineBehavior: configuration?.offlineBehavior || 'store_stale',
    targets: configuredTargets?.length
      ? configuredTargets
      : [{ connectionId: defaultTargetId, retentionDays: 30 }],
  }
}

export function buildHistoryStoragePayload(draft: HistoryStorageDraft): HistoryStorageSavePayload {
  if (draft.behavior !== 'custom') return { behavior: draft.behavior }
  const configuration: HistoryStorageConfigurationPayload = {
    writeMode: draft.writeMode,
    intervalMs:
      draft.writeMode === 'interval_latest' || draft.writeMode === 'periodic_snapshot'
        ? draft.intervalMs
        : null,
    deadband: draft.writeMode === 'on_change' ? draft.deadband : null,
    maxSilenceMs: draft.writeMode === 'on_change' ? draft.maxSilenceMs : null,
    offlineBehavior:
      draft.writeMode === 'periodic_snapshot' ? draft.offlineBehavior : 'store_stale',
    targets: draft.targets.map((target, index) => ({
      connectionId: target.connectionId,
      isPrimary: index === 0,
      sortOrder: index,
      retentionDays: target.retentionDays,
    })),
  }
  return { behavior: 'custom', configuration }
}

export function validateHistoryStorageDraft(draft: HistoryStorageDraft): string {
  if (draft.behavior !== 'custom') return ''
  if (draft.targets.length === 0 || !draft.targets[0]?.connectionId) return '请选择主存储目标'
  if (draft.targets.some((target) => !target.connectionId)) return '请选择所有附加目标'
  if (new Set(draft.targets.map((target) => target.connectionId)).size !== draft.targets.length)
    return '存储目标不能重复'
  if (
    (draft.writeMode === 'interval_latest' || draft.writeMode === 'periodic_snapshot') &&
    (!Number.isSafeInteger(draft.intervalMs) || draft.intervalMs <= 0)
  )
    return '保存间隔必须大于 0'
  if (draft.writeMode === 'on_change' && draft.deadband < 0) return '死区不能小于 0'
  if (
    draft.writeMode === 'on_change' &&
    draft.maxSilenceMs !== null &&
    (!Number.isSafeInteger(draft.maxSilenceMs) || draft.maxSilenceMs <= 0)
  )
    return '最长静默必须大于 0'
  if (
    draft.targets.some(
      (target) =>
        target.retentionDays !== null &&
        (!Number.isSafeInteger(target.retentionDays) || target.retentionDays <= 0),
    )
  )
    return '保留天数必须为正整数'
  return ''
}

export function historyStorageSummary(item: HistoryStorageSourceItem): string {
  if (item.historyState !== 'enabled' || !item.writeMode) return '未保存'
  const parts = [historyStorageModeLabels[item.writeMode]]
  if (item.primaryTargetName) parts.push(item.primaryTargetName)
  parts.push(item.retentionDays === null ? '永久' : `${item.retentionDays || 30}天`)
  if (item.targetCount > 1) parts.push(`${item.targetCount}个目标`)
  return parts.join(' · ')
}
