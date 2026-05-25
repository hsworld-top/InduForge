export type LayoutOccupancyMode = 'fixed' | 'fill' | 'ratio' | 'auto'
export type LayoutOccupancyAxis = 'row' | 'column'
export type LayoutOccupancyUnit = 'px' | '%'

export interface LayoutOccupancyPatch {
  style: Record<string, unknown>
  flowLayout: {
    grow: number
    shrink: number
    basis: string
  }
}

export interface LayoutOccupancyState {
  visible: boolean
  axis: LayoutOccupancyAxis
  mainStyleKey: 'width' | 'height'
  mode: LayoutOccupancyMode
  fixedValue: string
  fixedUnit: LayoutOccupancyUnit
  ratio: number
}

const SIZE_VALUE_RE = /^(\d+(?:\.\d+)?)(px|%)$/

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object'
}

export function isSupportedLayoutParentType(parentType: unknown): boolean {
  return parentType === 'HorizontalLayout' || parentType === 'VerticalLayout'
}

export function resolveLayoutAxis(parentType: string): LayoutOccupancyAxis {
  return parentType === 'HorizontalLayout' ? 'row' : 'column'
}

export function resolveMainStyleKey(parentType: string): 'width' | 'height' {
  return resolveLayoutAxis(parentType) === 'row' ? 'width' : 'height'
}

function parseSizeValue(value: unknown): { value: string; unit: LayoutOccupancyUnit } | null {
  if (typeof value !== 'string') return null
  const match = value.trim().match(SIZE_VALUE_RE)
  if (!match) return null
  return {
    value: match[1] || '',
    unit: (match[2] || 'px') as LayoutOccupancyUnit,
  }
}

function normalizePositiveNumber(value: unknown, fallback: number): number {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric <= 0) return fallback
  return numeric
}

function normalizeRatioShare(value: unknown): number {
  // 填满剩余对应 1 份；比例模式从 2 份开始，避免和填满剩余写出完全相同的 flowLayout。
  return Math.max(2, normalizePositiveNumber(value, 2))
}

function buildFixedSize(value: unknown, unit: LayoutOccupancyUnit): string {
  const numeric = normalizePositiveNumber(value, 100)
  return `${numeric}${unit}`
}

function withoutMainAxisSize(
  style: Record<string, unknown>,
  mainStyleKey: 'width' | 'height',
): Record<string, unknown> {
  const nextStyle = { ...style }
  delete nextStyle[mainStyleKey]
  return nextStyle
}

export function buildLayoutOccupancyPatch(options: {
  parentType: string
  mode: LayoutOccupancyMode
  currentStyle?: Record<string, unknown> | null
  fixedValue?: number | string
  fixedUnit?: LayoutOccupancyUnit
  ratio?: number | string
}): LayoutOccupancyPatch {
  const mainStyleKey = resolveMainStyleKey(options.parentType)
  const currentStyle = isRecord(options.currentStyle) ? options.currentStyle : {}

  if (options.mode === 'fixed') {
    const unit = options.fixedUnit || 'px'
    const fixedSize = buildFixedSize(options.fixedValue, unit)
    return {
      style: {
        ...currentStyle,
        [mainStyleKey]: fixedSize,
      },
      flowLayout: { grow: 0, shrink: 0, basis: fixedSize },
    }
  }

  if (options.mode === 'ratio') {
    const ratio = normalizeRatioShare(options.ratio)
    return {
      style: withoutMainAxisSize(currentStyle, mainStyleKey),
      flowLayout: { grow: ratio, shrink: 1, basis: '0%' },
    }
  }

  if (options.mode === 'auto') {
    return {
      style: withoutMainAxisSize(currentStyle, mainStyleKey),
      flowLayout: { grow: 0, shrink: 0, basis: 'auto' },
    }
  }

  return {
    style: withoutMainAxisSize(currentStyle, mainStyleKey),
    flowLayout: { grow: 1, shrink: 1, basis: '0%' },
  }
}

export function buildAverageChildOccupancyPatch(options: {
  parentType: string
  currentStyle?: Record<string, unknown> | null
}): LayoutOccupancyPatch {
  const mainStyleKey = resolveMainStyleKey(options.parentType)
  const currentStyle = isRecord(options.currentStyle) ? options.currentStyle : {}
  return {
    style: withoutMainAxisSize(currentStyle, mainStyleKey),
    flowLayout: { grow: 1, shrink: 1, basis: '0%' },
  }
}

export function resolveLayoutOccupancyState(options: {
  node: { style?: unknown; flowLayout?: unknown } | null | undefined
  parentType: string | null | undefined
}): LayoutOccupancyState {
  const parentType = options.parentType || ''
  const axis = resolveLayoutAxis(parentType)
  const mainStyleKey = resolveMainStyleKey(parentType)
  const style = isRecord(options.node?.style) ? options.node.style : {}
  const flowLayout = isRecord(options.node?.flowLayout) ? options.node.flowLayout : {}
  const fixedSize = parseSizeValue(style[mainStyleKey])

  if (!isSupportedLayoutParentType(parentType)) {
    return {
      visible: false,
      axis,
      mainStyleKey,
      mode: 'fill',
      fixedValue: '100',
      fixedUnit: 'px',
      ratio: 1,
    }
  }

  if (fixedSize) {
    return {
      visible: true,
      axis,
      mainStyleKey,
      mode: 'fixed',
      fixedValue: fixedSize.value,
      fixedUnit: fixedSize.unit,
      ratio: 1,
    }
  }

  const grow = normalizePositiveNumber(flowLayout.grow, 1)
  const basis = typeof flowLayout.basis === 'string' ? flowLayout.basis : '0%'
  if (grow > 1 && basis === '0%') {
    return {
      visible: true,
      axis,
      mainStyleKey,
      mode: 'ratio',
      fixedValue: '100',
      fixedUnit: 'px',
      ratio: grow,
    }
  }

  if (Number(flowLayout.grow) === 0 && basis === 'auto') {
    return {
      visible: true,
      axis,
      mainStyleKey,
      mode: 'auto',
      fixedValue: '100',
      fixedUnit: 'px',
      ratio: 1,
    }
  }

  return {
    visible: true,
    axis,
    mainStyleKey,
    mode: 'fill',
    fixedValue: '100',
    fixedUnit: 'px',
    ratio: Math.max(1, grow),
  }
}
