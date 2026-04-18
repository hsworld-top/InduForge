export type MarqueeStartSource = "insideCanvas" | "outsideCanvas";

export interface MarqueeModifiers {
  ctrl: boolean;
  meta: boolean;
  shift: boolean;
}

export interface OutsideMarqueeStartDetail {
  clientX: number;
  clientY: number;
  modifiers: MarqueeModifiers;
}

/**
 * 工作台灰区触发画布框选的内部事件名。
 */
export const CANVAS_OUTSIDE_MARQUEE_START_EVENT = "designer:canvas-outside-marquee-start";

/**
 * 判断画布内 pointerdown 是否应该启动 marquee。
 * 规则：
 * - 命中根节点：允许
 * - 命中非节点空白区：允许
 * - 命中任意非根节点：不允许（交给节点拖拽/缩放处理）
 * @param {{ hasNodeElement: boolean; isRootNode: boolean }} params - 命中上下文
 * @returns {boolean}
 */
export function shouldStartMarqueeFromCanvasPointerDown(params: {
  hasNodeElement: boolean;
  isRootNode: boolean;
}): boolean {
  const { hasNodeElement, isRootNode } = params;
  if (!hasNodeElement) return true;
  return isRootNode;
}

/**
 * 判断是否应该在框选结束时清空选中。
 * 仅当“页面外起手且未移动（单击）”时清空。
 * @param {{ startSource: MarqueeStartSource; moved: boolean }} params - 框选结束上下文
 * @returns {boolean}
 */
export function shouldClearSelectionOnMarqueeUp(params: {
  startSource: MarqueeStartSource;
  moved: boolean;
}): boolean {
  const { startSource, moved } = params;
  return startSource === "outsideCanvas" && !moved;
}
