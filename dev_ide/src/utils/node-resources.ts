import type { OpsNode } from '@/api/ops.api'

export function diskPercent(value: number) {
  // 容量与服务端阈值保持一致，避免舍入后跨过关注/严重边界。
  return Math.trunc(Math.min(100, Math.max(0, value)) * 100) / 100
}

export function nodeResourcePercent(node: OpsNode, key: 'cpu' | 'memory' | 'disk') {
  const resource = node.resourceSummary?.[key]
  if (!resource || typeof resource !== 'object') return null
  const raw = (resource as Record<string, unknown>).usedPercent
  if (typeof raw !== 'number' || !Number.isFinite(raw) || raw < 0 || raw > 100) return null
  return key === 'disk' ? diskPercent(raw) : Math.round(raw)
}
