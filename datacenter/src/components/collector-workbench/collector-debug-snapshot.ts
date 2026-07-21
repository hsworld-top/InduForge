import type {
  CollectorPoint,
  CollectorPointBatchFailure,
  CollectorPointDebugSnapshot,
  CollectorPointReadResult,
} from '@/api/schemas/collector.schema'

export type CollectorDebugQualityTone = 'success' | 'warning' | 'danger' | 'info'

export function formatCollectorDebugValue(value: unknown, valueText: string | null = null): string {
  if (valueText !== null) return valueText
  if (value === null || value === undefined) return 'null'
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value)
  }
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

export function hasCollectorDebugSuccess(snapshot: CollectorPointDebugSnapshot): boolean {
  return snapshot.readAt !== null
}

export function collectorDebugQualityTone(quality: string | null): CollectorDebugQualityTone {
  const normalized = quality?.trim().toLowerCase() || ''
  if (!normalized) return 'info'
  if (normalized === 'good' || normalized.startsWith('good')) return 'success'
  if (normalized.includes('uncertain')) return 'warning'
  return 'danger'
}

export function collectorDebugQualityLabel(quality: string | null): string {
  return quality?.trim() || '—'
}

export function collectorDebugTime(value: string | null): string {
  return value?.trim() || '—'
}

export function collectorDebugFailureText(snapshot: CollectorPointDebugSnapshot): string {
  if (snapshot.lastAttemptStatus !== 'failed') return ''
  return snapshot.lastErrorMessage?.trim() || snapshot.lastErrorCode?.trim() || '最近一次读取失败'
}

export function currentPageCollectorPointIds(points: Array<Pick<CollectorPoint, 'id'>>): string[] {
  return points.map((point) => point.id)
}

export function summarizeCollectorPointRead(
  points: Array<Pick<CollectorPoint, 'id' | 'name'>>,
  result: CollectorPointReadResult,
): { successCount: number; failures: CollectorPointBatchFailure[] } {
  const indexByID = new Map(points.map((point, index) => [point.id, index]))
  const nameByID = new Map(points.map((point) => [point.id, point.name]))
  const failures = result.values
    .filter((item) => !item.succeeded)
    .map((item) => ({
      index: indexByID.get(item.pointId) ?? 0,
      name: nameByID.get(item.pointId) || item.pointId,
      code: item.errorCode || 'COLLECTOR_POINT_READ_FAILED',
      message: item.errorMessage || '变量读取失败',
    }))
  return {
    successCount: result.values.filter((item) => item.succeeded).length,
    failures,
  }
}
