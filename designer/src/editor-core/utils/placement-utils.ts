/**
 * 放置与坐标计算工具
 *
 * 用于统一「拖放落点」相对某容器的坐标计算，避免 DesignCanvas 与 NodeRenderer 等处各自实现导致不一致。
 * 约定：返回坐标为相对 containerElement 的坐标，且已按 zoom 折算（设计态逻辑坐标）。
 *
 * @see docs/designer/placement-and-stacking.md
 */

export interface CanvasPoint {
  x: number;
  y: number;
}

export interface PlacementSize {
  width: number;
  height: number;
}

/**
 * 将事件坐标转换为相对指定容器的画布坐标（已除 zoom）
 */
export function eventToCanvasPosition(
  event: MouseEvent | DragEvent,
  containerElement: HTMLElement,
  zoom = 1,
): CanvasPoint {
  if (!containerElement || typeof event.clientX !== "number") {
    return { x: 0, y: 0 };
  }
  const rect = containerElement.getBoundingClientRect();
  const x = (event.clientX - rect.left) / zoom;
  const y = (event.clientY - rect.top) / zoom;
  return {
    x: Math.max(0, Math.round(x)),
    y: Math.max(0, Math.round(y)),
  };
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
  if (!containerElement) return position;
  const rect = containerElement.getBoundingClientRect();
  const maxX = Math.max(0, Math.round(rect.width / zoom - (size.width || 0)));
  const maxY = Math.max(0, Math.round(rect.height / zoom - (size.height || 0)));
  return {
    x: Math.min(Math.max(0, position.x), maxX),
    y: Math.min(Math.max(0, position.y), maxY),
  };
}
