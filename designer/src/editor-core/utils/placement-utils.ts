/**
 * 放置与坐标计算工具
 *
 * 用于统一「拖放落点」相对某容器的坐标计算，避免 DesignCanvas 与 NodeRenderer 等处各自实现导致不一致。
 * 约定：返回坐标为相对 containerElement 的坐标，且已按 zoom 折算（设计态逻辑坐标）。
 *
 * @see docs/designer/placement-and-stacking.md
 */

export interface CanvasPoint {
  x: number
  y: number
}

export interface PlacementSize {
  width: number
  height: number
}

/**
 * 将事件坐标转换为相对指定容器的画布坐标（已除 zoom）
 *
 * 修复 D-12: getBoundingClientRect() 不包含容器内部 scroll 偏移，
 * 需要额外加上 scrollLeft/scrollTop 才能得到正确的逻辑坐标。
 * zoom 除法在函数内只发生一次（输入 zoom → 输出坐标 = 输入坐标 / zoom）。
 */
export function eventToCanvasPosition(
  event: MouseEvent | DragEvent,
  containerElement: HTMLElement,
  zoom = 1,
): CanvasPoint {
  if (!containerElement || typeof event.clientX !== 'number') {
    return { x: 0, y: 0 }
  }
  const rect = containerElement.getBoundingClientRect()
  // D-12 修复：补偿容器内部滚动偏移
  const scrollLeft = containerElement.scrollLeft || 0
  const scrollTop = containerElement.scrollTop || 0
  const x = (event.clientX - rect.left + scrollLeft) / zoom
  const y = (event.clientY - rect.top + scrollTop) / zoom
  return {
    x: Math.max(0, Math.round(x)),
    y: Math.max(0, Math.round(y)),
  }
}

/**
 * 将落点坐标限制在容器范围内（避免超出右/下边界）
 */
export function clampPositionInContainer(
  position: CanvasPoint,
  containerElement: HTMLElement,
  size: PlacementSize,
  zoom = 1,
): CanvasPoint {
  if (!containerElement) return position
  const rect = containerElement.getBoundingClientRect()
  const maxX = Math.max(0, Math.round(rect.width / zoom - (size.width || 0)))
  const maxY = Math.max(0, Math.round(rect.height / zoom - (size.height || 0)))
  return {
    x: Math.min(Math.max(0, position.x), maxX),
    y: Math.min(Math.max(0, position.y), maxY),
  }
}
