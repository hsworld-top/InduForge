/**
 * 布局工具函数
 *
 * 从 NodeRenderer 抽取的布局/尺寸计算辅助函数，供 composable 和组件复用。
 *
 * @module editor-core/utils/layout-utils
 */

import type { DocumentModel } from '@/editor-core/document/DocumentModel'
import type { ComponentNode } from '@/editor-core/document/types'

type DocLike = Pick<DocumentModel, 'getNode'>

interface AbsPosLike {
  x?: number
  y?: number
  w?: number
  h?: number
  z?: number
}

const NUMERIC_TEXT_RE = /^-?\d+(?:\.\d+)?$/

export function resolveAbsoluteLayout(
  currentNode: ComponentNode,
  nodeElement: HTMLElement | null,
): { x: number; y: number; w: number; h: number; z: number } {
  const fallbackAbs = (currentNode.layoutItem?.free?.abs || {}) as AbsPosLike
  const useAbsolute =
    currentNode.positioning === 'absolute' || currentNode.layoutItem?.free?.mode === 'abs'
  const absolutePos: AbsPosLike = useAbsolute
    ? ((currentNode.absolutePos || fallbackAbs) as AbsPosLike)
    : {}
  const rect = nodeElement?.getBoundingClientRect?.()
  const width = Number.isFinite(absolutePos.w) ? (absolutePos.w as number) : (rect?.width ?? 120)
  const height = Number.isFinite(absolutePos.h) ? (absolutePos.h as number) : (rect?.height ?? 40)

  let x = Number.isFinite(absolutePos.x) ? (absolutePos.x as number) : 0
  let y = Number.isFinite(absolutePos.y) ? (absolutePos.y as number) : 0

  if (!Number.isFinite(absolutePos.x) || !Number.isFinite(absolutePos.y)) {
    const parentElement = nodeElement?.parentElement?.closest?.('[data-node-id]')
    const parentRect = parentElement?.getBoundingClientRect?.()
    if (rect && parentRect) {
      x = rect.left - parentRect.left
      y = rect.top - parentRect.top
    }
  }

  return {
    x: Math.round(x),
    y: Math.round(y),
    w: Math.max(1, Math.round(width)),
    h: Math.max(1, Math.round(height)),
    z: Number.isFinite(absolutePos.z) ? (absolutePos.z as number) : 1,
  }
}

export function buildFlowResetStyle(
  currentStyle: Record<string, unknown> | undefined,
): Record<string, unknown> {
  const nextStyle = { ...(currentStyle || {}) }
  delete nextStyle.width
  delete nextStyle.height
  return nextStyle
}

export function parseSizeToNumber(value: string | number | undefined | null): number | undefined {
  if (value === null || value === undefined) return undefined
  if (typeof value === 'number' && Number.isFinite(value)) return value
  const text = String(value).trim()
  if (!text || text === 'auto') return undefined
  if (text.endsWith('px')) {
    const num = Number.parseFloat(text.slice(0, -2))
    return Number.isFinite(num) ? num : undefined
  }
  if (NUMERIC_TEXT_RE.test(text)) {
    const num = Number.parseFloat(text)
    return Number.isFinite(num) ? num : undefined
  }
  return undefined
}

export function resolveElLayoutMinHeight(layoutNode: ComponentNode | null): number {
  if (!layoutNode || layoutNode.type !== 'ElLayout') return 0
  return 0
}

export function resolveElContainerMinSize(
  containerNode: ComponentNode | null,
  doc: DocLike | null | undefined,
): { width: number; height: number } | null {
  if (!containerNode || containerNode.type !== 'ElContainer') return null
  const children = containerNode.children || []
  let hasHeader = false
  let hasFooter = false
  let hasAside = false
  let hasMain = false
  for (const childId of children) {
    const childNode = doc?.getNode?.(childId)
    if (!childNode) continue
    if (childNode.type === 'ElHeader') hasHeader = true
    if (childNode.type === 'ElFooter') hasFooter = true
    if (childNode.type === 'ElAside') hasAside = true
    if (childNode.type === 'ElMain') hasMain = true
  }

  const props = (containerNode.props || {}) as Record<string, unknown>
  if (typeof props.showHeader === 'boolean') hasHeader = props.showHeader
  if (typeof props.showFooter === 'boolean') hasFooter = props.showFooter
  if (typeof props.showAside === 'boolean') hasAside = props.showAside
  if (typeof props.showMain === 'boolean') hasMain = props.showMain

  const headerHeight = parseSizeToNumber(props.headerHeight as string | number) ?? 60
  const footerHeight = parseSizeToNumber(props.footerHeight as string | number) ?? 60
  const asideWidth = parseSizeToNumber(props.asideWidth as string | number) ?? 200
  const minBodySize = 40

  const hasBody = hasAside || hasMain
  let minWidth = 0
  if (hasAside && hasMain) {
    minWidth = asideWidth + minBodySize
  } else if (hasAside) {
    minWidth = asideWidth
  } else if (hasMain) {
    minWidth = minBodySize
  }

  let minHeight = 0
  if (hasHeader) minHeight += headerHeight
  if (hasFooter) minHeight += footerHeight
  if (hasBody) minHeight += minBodySize

  if (minWidth <= 0 && minHeight <= 0) return null
  return { width: minWidth, height: minHeight }
}

export function buildContainerSectionSizePatch(
  containerNode: ComponentNode | null,
  baseWidth: number,
  nextWidth: number,
  baseHeight: number,
  nextHeight: number,
  baseSectionSizes: {
    headerHeight?: number
    footerHeight?: number
    asideWidth?: number
  },
): Record<string, string> | null {
  if (!containerNode || containerNode.type !== 'ElContainer') return null
  const scaleX =
    Number.isFinite(baseWidth) && baseWidth > 0 && Number.isFinite(nextWidth)
      ? nextWidth / baseWidth
      : 1
  const scaleY =
    Number.isFinite(baseHeight) && baseHeight > 0 && Number.isFinite(nextHeight)
      ? nextHeight / baseHeight
      : 1
  const hasScaleX = Number.isFinite(scaleX) && Math.abs(scaleX - 1) >= 0.001
  const hasScaleY = Number.isFinite(scaleY) && Math.abs(scaleY - 1) >= 0.001
  if (!hasScaleX && !hasScaleY) return null
  const props = (containerNode.props || {}) as Record<string, unknown>
  const minSectionSize = 40
  const patch: Record<string, string> = {}

  if (hasScaleY && props.showHeader !== false) {
    const headerHeight = baseSectionSizes?.headerHeight
    if (Number.isFinite(headerHeight)) {
      patch.headerHeight = `${Math.max(
        minSectionSize,
        Math.round((headerHeight as number) * scaleY),
      )}px`
    }
  }
  if (hasScaleY && props.showFooter !== false) {
    const footerHeight = baseSectionSizes?.footerHeight
    if (Number.isFinite(footerHeight)) {
      patch.footerHeight = `${Math.max(
        minSectionSize,
        Math.round((footerHeight as number) * scaleY),
      )}px`
    }
  }
  if (hasScaleX && props.showAside !== false) {
    const asideWidth = baseSectionSizes?.asideWidth
    if (Number.isFinite(asideWidth)) {
      patch.asideWidth = `${Math.max(
        minSectionSize,
        Math.round((asideWidth as number) * scaleX),
      )}px`
    }
  }

  return Object.keys(patch).length > 0 ? patch : null
}

export function clampElContainerPropsBySize(
  containerNode: ComponentNode | null,
  width: number,
  height: number,
): Record<string, string> | null {
  if (!containerNode || containerNode.type !== 'ElContainer') return null
  if (!Number.isFinite(width) || !Number.isFinite(height)) return null
  if (width <= 0 || height <= 0) return null
  const props = (containerNode.props || {}) as Record<string, unknown>
  const hasHeader = props.showHeader !== false
  const hasFooter = props.showFooter !== false
  const hasAside = props.showAside !== false
  const hasMain = props.showMain !== false
  const hasBody = hasAside || hasMain
  const minBodySize = 40
  const minSectionSize = 40
  const headerHeight = parseSizeToNumber(props.headerHeight as string | number) ?? 60
  const footerHeight = parseSizeToNumber(props.footerHeight as string | number) ?? 60
  let asideWidth = parseSizeToNumber(props.asideWidth as string | number) ?? 200
  const patch: Record<string, string> = {}

  if (hasAside) {
    const maxAside = Math.max(0, width - (hasMain ? minBodySize : 0))
    if (Number.isFinite(maxAside)) {
      asideWidth = Math.min(asideWidth, maxAside)
      asideWidth = Math.max(minSectionSize, asideWidth)
      const nextAside = `${Math.round(asideWidth)}px`
      if (nextAside !== props.asideWidth) {
        patch.asideWidth = nextAside
      }
    }
  }

  if (hasHeader || hasFooter) {
    const available = Math.max(0, height - (hasBody ? minBodySize : 0))
    let nextHeader = hasHeader ? headerHeight : 0
    let nextFooter = hasFooter ? footerHeight : 0
    const total = nextHeader + nextFooter
    if (total > available && total > 0) {
      const scale = available / total
      nextHeader = Math.max(minSectionSize, Math.round(nextHeader * scale))
      nextFooter = Math.max(minSectionSize, Math.round(nextFooter * scale))
    } else {
      if (hasHeader) nextHeader = Math.min(nextHeader, available)
      if (hasFooter) nextFooter = Math.min(nextFooter, available)
    }
    if (hasHeader) {
      const nextHeaderText = `${nextHeader}px`
      if (nextHeaderText !== props.headerHeight) {
        patch.headerHeight = nextHeaderText
      }
    }
    if (hasFooter) {
      const nextFooterText = `${nextFooter}px`
      if (nextFooterText !== props.footerHeight) {
        patch.footerHeight = nextFooterText
      }
    }
  }

  return Object.keys(patch).length > 0 ? patch : null
}

export function resolveElContainerMain(
  doc: DocLike | null | undefined,
  container: ComponentNode | null,
): ComponentNode | null {
  if (!container || container.type !== 'ElContainer') return null
  const mainChildId = (container.children || []).find((childId) => {
    const childNode = doc?.getNode?.(childId)
    return childNode?.type === 'ElMain'
  })
  return mainChildId ? doc?.getNode?.(mainChildId) || null : null
}
