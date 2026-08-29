import { datacenterLocale } from '@/i18n/runtime'

export type DatapointSourceModule = 'access-source' | 'industrial-collector' | 'compute'

export interface DatapointSourceLike {
  sourceType?: string
  sourceId?: string | null
  accessSourceId?: string | null
  accessSourceName?: string | null
  sourceConfig?: Record<string, unknown>
}

export interface DatapointSourceNavigation {
  module: DatapointSourceModule
  objectId: string
  label: string
  actionLabel: string
}

const DIRECT_CONNECTION_SOURCE_TYPES = new Set([
  'http.request',
  'websocket.session',
  'realtime.key',
  'kafka.field',
  'kafka.raw',
])

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function resolveConnectionId(datapoint: DatapointSourceLike): string {
  const config = datapoint.sourceConfig || {}
  const configuredId =
    stringValue(datapoint.accessSourceId) ||
    stringValue(config.connectionId) ||
    stringValue(config.sourceConnectionId)
  if (configuredId) return configuredId

  return DIRECT_CONNECTION_SOURCE_TYPES.has(stringValue(datapoint.sourceType))
    ? stringValue(datapoint.sourceId)
    : ''
}

/** 将不同数据点来源统一解析为数据中心可导航对象。 */
export function resolveDatapointSourceNavigation(
  datapoint: DatapointSourceLike | null | undefined,
): DatapointSourceNavigation | null {
  if (!datapoint) return null

  const sourceType = stringValue(datapoint.sourceType)
  const config = datapoint.sourceConfig || {}
  if (sourceType === 'calc.output') {
    const computeUnitId = stringValue(datapoint.sourceId) || stringValue(config.computeUnitId)
    return computeUnitId
      ? {
          module: 'compute',
          objectId: computeUnitId,
          label: datacenterLocale.value === 'en' ? 'Compute Unit' : '计算单元',
          actionLabel: datacenterLocale.value === 'en' ? 'Open Compute Unit' : '打开计算单元',
        }
      : null
  }

  const connectionId = resolveConnectionId(datapoint)
  if (!connectionId) return null
  if (sourceType === 'collector.point') {
    return {
      module: 'industrial-collector',
      objectId: connectionId,
      label: datacenterLocale.value === 'en' ? 'Industrial Collector' : '工业采集连接',
      actionLabel: datacenterLocale.value === 'en' ? 'Open Industrial Collector' : '打开工业采集连接',
    }
  }

  return {
    module: 'access-source',
    objectId: connectionId,
    label: datacenterLocale.value === 'en' ? 'Access Source' : '接入源',
    actionLabel: datacenterLocale.value === 'en' ? 'Open Access Source' : '打开接入源',
  }
}

/** 列表未返回具体来源名称时仍给出有意义的来源类型。 */
export function resolveDatapointSourceDisplayName(datapoint: DatapointSourceLike): string {
  const accessSourceName = stringValue(datapoint.accessSourceName)
  if (accessSourceName) return accessSourceName
  switch (stringValue(datapoint.sourceType)) {
    case 'calc.output':
      return datacenterLocale.value === 'en' ? 'Compute Unit' : '计算单元'
    case 'collector.point':
      return datacenterLocale.value === 'en' ? 'Industrial Collector' : '工业采集连接'
    case 'static.var':
      return datacenterLocale.value === 'en' ? 'Static Variable' : '静态变量'
    default:
      return '-'
  }
}
