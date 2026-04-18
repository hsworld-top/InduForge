/**
 * 放置解析器（placementResolver）
 *
 * 职责：
 * - 单一入口处理容器命中检测
 * - Strategy 模式处理 flex/free/grid 布局类型
 * - 实现 D-01~D-06 放置规则
 *
 * @module editor-core/utils/placement-resolver
 */

import type { ComponentNode } from "@/editor-core/document/types";
import type { CanvasDocLike } from "@/ui/editors/page/canvas/composables/types";
import {
  getChildPositioning,
  isContainerType,
} from "@/editor-core/descriptors/registry";
import {
  clampPositionInContainer,
  eventToCanvasPosition,
  type CanvasPoint,
} from "@/editor-core/utils/placement-utils";
import {
  DragDropManager,
  type DropTargetResolution,
  type FlowContainerKind,
  type GridCellHint,
} from "@/ui/editors/page/canvas/interaction/DragDropManager";
import type { FlexInsertLine } from "@/ui/editors/page/canvas/interaction/DragDropManager";

// DragDropManager 单例，用于调用实例方法
const dragDropManager = new DragDropManager();

// ---------------------------------------------------------------------------
// Strategy 接口
// ---------------------------------------------------------------------------

/**
 * 放置策略接口
 *
 * 不同容器类型（flex/free/grid）实现此接口，提供各自的：
 * - canAcceptChild: 判断是否能接受子组件
 * - resolveInsertIndex: 计算插入位置
 * - resolveDropPosition: 计算绝对坐标（仅 free 容器需要）
 */
export interface PlacementStrategy {
  /** 策略处理的容器类型 */
  readonly containerKind: FlowContainerKind;

  /**
   * 判断目标容器是否能接受指定类型的子组件
   * @param parentNode 父节点
   * @param childType 子组件类型
   * @returns 是否可以接受
   */
  canAcceptChild(parentNode: ComponentNode, childType: string): boolean;

  /**
   * 计算在容器中的插入索引
   * @param event 拖拽事件
   * @param containerNode 容器节点
   * @param containerElement 容器 DOM 元素
   * @param zoom 当前缩放比例
   * @returns 插入索引
   */
  resolveInsertIndex(
    event: DragEvent | MouseEvent,
    containerNode: ComponentNode,
    containerElement: HTMLElement,
    zoom: number,
  ): number;

  /**
   * 计算放置的绝对坐标（仅 free 容器需要）
   * @param event 拖拽事件
   * @param containerElement 容器 DOM 元素
   * @param zoom 当前缩放比例
   * @returns 画布坐标，或 null 表示使用 index 插入
   */
  resolveDropPosition?(
    event: DragEvent | MouseEvent,
    containerElement: HTMLElement,
    zoom: number,
  ): CanvasPoint | null;
}

// ---------------------------------------------------------------------------
// 解析结果
// ---------------------------------------------------------------------------

export interface PlacementResolution {
  parentId: string | null;
  index: number;
  dropPosition: CanvasPoint | null;
  containerType: FlowContainerKind | null;
  /** 插入线提示（flex 容器） */
  insertLine?: FlexInsertLine | null;
  /** 网格高亮提示（grid 容器） */
  gridHint?: GridCellHint | null;
}

// ---------------------------------------------------------------------------
// 容器命中检测选项
// ---------------------------------------------------------------------------

export interface ResolvePlacementOptions {
  /** 优先根级容器（主要目标） */
  preferRootLevel?: boolean;
  /** 根级不明确时使用深度优先 fallback */
  depthFirstFallback?: boolean;
}

// ---------------------------------------------------------------------------
// 工具函数
// ---------------------------------------------------------------------------

/**
 * 获取节点的视觉顶部 Y 坐标（相对于页面）
 * 优先使用 props 中的 x/y（绝对定位），否则使用自然文档流位置 0
 */
function getNodeVisualTop(node: ComponentNode): CanvasPoint {
  const props = node.props as Record<string, unknown> | undefined;
  if (props && typeof props.y === "number") {
    return {
      x: typeof props.x === "number" ? props.x : 0,
      y: props.y as number,
    };
  }
  return { x: 0, y: 0 };
}

/**
 * 获取容器的直接子节点数组
 */
function getContainerChildren(containerNode: ComponentNode): ComponentNode[] {
  const childIds = containerNode.children || [];
  return childIds as unknown as ComponentNode[];
}

/**
 * 检查插入索引是否有效（在 siblings 长度范围内）
 */
function isValidInsertIndex(index: number, siblings: ComponentNode[]): boolean {
  return index >= 0 && index <= siblings.length;
}

// ---------------------------------------------------------------------------
// Y-first + nearest neighbor + append fallback（独立函数，供策略使用）
// ---------------------------------------------------------------------------

/**
 * Y-first + nearest neighbor insertIndex 计算
 *
 * D-04: 按 Y 升序排序，Y 相同时按 X 辅助排序
 * D-05: 索引无效时 fallback 到 append（siblings.length）
 * D-06: 以 dropPos 为基准找到最近邻节点并精确调整索引
 *
 * @param dropPos 落点画布坐标
 * @param siblings 兄弟节点数组
 * @param containerElement 容器 DOM 元素（用于 getBoundingClientRect 计算）
 * @param zoom 当前缩放比例
 * @param originalIndex 原始建议索引（来自容器特定策略计算）
 * @returns 最终插入索引
 */
export function calculateInsertIndexWithFallback(
  dropPos: CanvasPoint,
  siblings: ComponentNode[],
  containerElement: HTMLElement,
  zoom: number,
  originalIndex: number,
): number {
  if (siblings.length === 0) return 0;

  // D-04: Y-first 排序
  const sorted = [...siblings].sort((a, b) => {
    const posA = getNodeVisualTop(a);
    const posB = getNodeVisualTop(b);
    return posA.y - posB.y || posA.x - posB.x;
  });

  // D-06: 最近邻精调 — 以 dropPos 为基准找 Y+X 距离最小的节点
  let nearestIdx = 0;
  let nearestDistance = Infinity;
  const rect = containerElement.getBoundingClientRect();

  for (let i = 0; i < sorted.length; i++) {
    const sibling = sorted[i]!;
    const siblingPos = getNodeVisualTop(sibling);
    // 将节点逻辑坐标转换为视口坐标进行比较
    const siblingScreenX = rect.left + siblingPos.x * zoom;
    const siblingScreenY = rect.top + siblingPos.y * zoom;
    const dropScreenX = rect.left + dropPos.x * zoom;
    const dropScreenY = rect.top + dropPos.y * zoom;

    const distance = Math.abs(siblingScreenX - dropScreenX) + Math.abs(siblingScreenY - dropScreenY);
    if (distance < nearestDistance) {
      nearestDistance = distance;
      nearestIdx = i;
    }
  }

  // D-05: append 兜底 — 验证 nearestIdx 是否指向有效节点
  // 如果 originalIndex 已经指向有效范围但 nearest 计算异常，以 originalIndex 为主
  if (isValidInsertIndex(originalIndex, siblings)) {
    const distanceFromOriginal = Math.abs(nearestIdx - originalIndex);
    if (distanceFromOriginal <= 1) {
      return originalIndex;
    }
  }

  // nearestIdx 指向的节点可能已被删除，fallback 到 append
  if (!isValidInsertIndex(nearestIdx, siblings)) {
    return siblings.length; // append
  }

  return nearestIdx;
}

// ---------------------------------------------------------------------------
// 抽象策略基类 — 实现公共逻辑
// ---------------------------------------------------------------------------

abstract class ContainerStrategy implements PlacementStrategy {
  abstract readonly containerKind: FlowContainerKind;

  abstract canAcceptChild(parentNode: ComponentNode, childType: string): boolean;

  abstract resolveInsertIndex(
    event: DragEvent | MouseEvent,
    containerNode: ComponentNode,
    containerElement: HTMLElement,
    zoom: number,
  ): number;

  resolveDropPosition?(
    event: DragEvent | MouseEvent,
    containerElement: HTMLElement,
    zoom: number,
  ): CanvasPoint | null {
    return null; // 默认实现：flow 容器不需要绝对坐标
  }
}

// ---------------------------------------------------------------------------
// Flex 容器策略
// ---------------------------------------------------------------------------

class FlexContainerStrategy extends ContainerStrategy {
  readonly containerKind = "flex" as const;

  canAcceptChild(parentNode: ComponentNode, childType: string): boolean {
    // 使用 descriptor registry 判断
    const { canAcceptChildByDescriptor } = require("@/editor-core/descriptors/registry");
    const currentChildCount = (parentNode.children || []).length;
    return canAcceptChildByDescriptor(parentNode.type, childType, currentChildCount);
  }

  resolveInsertIndex(
    event: DragEvent | MouseEvent,
    containerNode: ComponentNode,
    containerElement: HTMLElement,
    zoom: number,
  ): number {
    const direction = this.getContainerDirection(containerElement);
    const flexResult = dragDropManager.calculateFlexInsertPosition(containerElement, event, direction);
    return flexResult.index;
  }

  /**
   * 获取容器 flex 方向
   */
  private getContainerDirection(containerElement: HTMLElement): string {
    const computedStyle = window.getComputedStyle(containerElement);
    return computedStyle.flexDirection || "column";
  }
}

// ---------------------------------------------------------------------------
// Free 容器策略（绝对定位）
// ---------------------------------------------------------------------------

class FreeContainerStrategy extends ContainerStrategy {
  readonly containerKind = "free" as const;

  canAcceptChild(_parentNode: ComponentNode, _childType: string): boolean {
    // free 容器理论上接受任何子组件（绝对定位无布局约束）
    return true;
  }

  resolveInsertIndex(
    _event: DragEvent | MouseEvent,
    containerNode: ComponentNode,
    _containerElement: HTMLElement,
    _zoom: number,
  ): number {
    // free 容器始终 append 到末尾
    return (containerNode.children || []).length;
  }

  resolveDropPosition(
    event: DragEvent | MouseEvent,
    containerElement: HTMLElement,
    zoom: number,
  ): CanvasPoint {
    const rawPos = dragDropManager.calculateFreePosition(containerElement, event);
    return clampPositionInContainer(
      { x: rawPos.x, y: rawPos.y },
      containerElement,
      { width: 0, height: 0 },
      zoom,
    );
  }
}

// ---------------------------------------------------------------------------
// Grid 容器策略
// ---------------------------------------------------------------------------

class GridContainerStrategy extends ContainerStrategy {
  readonly containerKind = "grid" as const;

  canAcceptChild(parentNode: ComponentNode, childType: string): boolean {
    const { canAcceptChildByDescriptor } = require("@/editor-core/descriptors/registry");
    const currentChildCount = (parentNode.children || []).length;
    return canAcceptChildByDescriptor(parentNode.type, childType, currentChildCount);
  }

  resolveInsertIndex(
    event: DragEvent | MouseEvent,
    containerNode: ComponentNode,
    containerElement: HTMLElement,
    _zoom: number,
  ): number {
    // Grid 容器按 cell 计算 index = row * colCount + col
    const gridHint = dragDropManager.calculateGridCell(containerElement, event as DragEvent);
    const colsStr = window.getComputedStyle(containerElement).gridTemplateColumns || "auto";
    const cols = parseGridTemplateParts(colsStr);
    const colCount = cols.length || 3;
    const row = Math.max(1, gridHint.row || 1);
    const col = Math.max(1, gridHint.col || 1);
    return (row - 1) * colCount + (col - 1);
  }
}

// ---------------------------------------------------------------------------
// Strategy 实例缓存
// ---------------------------------------------------------------------------

const strategies: Record<FlowContainerKind, ContainerStrategy> = {
  flex: new FlexContainerStrategy(),
  free: new FreeContainerStrategy(),
  grid: new GridContainerStrategy(),
};

function getStrategy(kind: FlowContainerKind): PlacementStrategy {
  return strategies[kind] ?? strategies.free;
}

// ---------------------------------------------------------------------------
// gridTemplateColumns 解析（从 DragDropManager 复制，保持一致）
// ---------------------------------------------------------------------------

const GRID_REPEAT_HEAD_RE = /^repeat\s*\(\s*/i;
const GRID_DIGIT_RE = /\d/;
const GRID_WHITESPACE_RE = /\s/;
const GRID_TEMPLATE_SPLIT_RE = /\s+/;

function parseGridTemplateParts(template: string): string[] {
  if (!template || template === "none") return [];

  const trimmed = template.trim();
  const repeatHead = GRID_REPEAT_HEAD_RE.exec(trimmed);
  if (repeatHead?.index === 0) {
    let i = repeatHead[0].length;
    let countStr = "";
    while (i < trimmed.length && GRID_DIGIT_RE.test(trimmed[i]!)) {
      countStr += trimmed[i]!;
      i++;
    }
    while (i < trimmed.length && GRID_WHITESPACE_RE.test(trimmed[i]!)) {
      i++;
    }
    if (trimmed[i] !== ",") {
      return template.split(GRID_TEMPLATE_SPLIT_RE).filter(Boolean);
    }
    i++;
    while (i < trimmed.length && GRID_WHITESPACE_RE.test(trimmed[i]!)) {
      i++;
    }
    const valueStart = i;
    let depth = 0;
    for (; i < trimmed.length; i++) {
      const c = trimmed[i]!;
      if (c === "(") {
        depth++;
      } else if (c === ")") {
        if (depth === 0) {
          break;
        }
        depth--;
      }
    }
    const value = trimmed.slice(valueStart, i).trim();
    const count = Number.parseInt(countStr, 10);
    if (Number.isFinite(count) && count > 0 && value) {
      return Array.from<string>({ length: count } as ArrayLike<string>).fill(value);
    }
  }

  return template.split(GRID_TEMPLATE_SPLIT_RE).filter(Boolean);
}

// ---------------------------------------------------------------------------
// 容器命中检测
// ---------------------------------------------------------------------------

/**
 * 查找目标容器
 *
 * D-01: 根级容器为主要目标（优先返回根级容器的直接子级）
 * D-02: 深度优先为辅助规则，当根级容器逻辑不明确时降级使用
 *
 * @param event 鼠标/拖拽事件
 * @param canvasRoot 画布根元素
 * @param doc 文档模型
 * @param currentPage 当前页面
 * @param options 命中检测选项
 * @returns 命中的容器节点和 DOM 元素
 */
export function findTargetContainer(
  event: MouseEvent | DragEvent,
  canvasRoot: HTMLElement,
  doc: CanvasDocLike,
  currentPage: { rootNodeId: string | null },
  options: ResolvePlacementOptions = {},
): { node: ComponentNode | null; element: HTMLElement | null } {
  const point = { x: event.clientX, y: event.clientY };
  const element = document.elementFromPoint(point.x, point.y);

  if (!element || !canvasRoot.contains(element)) {
    // 鼠标不在画布内 → fallback 到根容器
    return findRootContainer(doc, currentPage, canvasRoot);
  }

  const nodeElement = element.closest("[data-node-id]") as HTMLElement | null;
  if (!nodeElement || !canvasRoot.contains(nodeElement)) {
    return findRootContainer(doc, currentPage, canvasRoot);
  }

  const nodeId = nodeElement.getAttribute("data-node-id");
  if (!nodeId) {
    return findRootContainer(doc, currentPage, canvasRoot);
  }

  // 从当前元素向上遍历，寻找最近的容器
  let currentNodeId: string | null = nodeId;
  let currentNode = doc.getNode(currentNodeId) as ComponentNode | null;
  let nodeElement_cur: HTMLElement | null = nodeElement;

  while (currentNode) {
    const nodeType = currentNode.type;
    const container = isContainerType(nodeType);

    if (container) {
      // D-01: 找到容器，检查是否为根级容器的直接子级
      if (options.preferRootLevel !== false) {
        const rootId = currentPage.rootNodeId;
        if (rootId) {
          const rootNode = doc.getNode(rootId) as ComponentNode | null;
          if (rootNode) {
            const rootChildren = rootNode.children || [];
            // 如果当前容器是根容器的直接子级，优先使用
            if ((rootChildren as string[]).includes(currentNodeId)) {
              return { node: currentNode, element: nodeElement_cur };
            }
          }
        }
      }

      // 否则返回当前找到的容器（深度优先 fallback）
      return { node: currentNode, element: nodeElement_cur };
    }

    // 向上继续查找父节点
    const parentId = doc.getParent(currentNodeId) as ComponentNode | null;
    if (parentId) {
      currentNodeId = parentId.id;
      currentNode = doc.getNode(currentNodeId) as ComponentNode | null;
      nodeElement_cur = document.querySelector(`[data-node-id="${CSS.escape(currentNodeId)}"]`) as HTMLElement | null;
    } else {
      break;
    }
  }

  // 遍历不到容器 → fallback 到根容器
  return findRootContainer(doc, currentPage, canvasRoot);
}

/**
 * 获取根容器作为 fallback
 */
function findRootContainer(
  doc: CanvasDocLike,
  currentPage: { rootNodeId: string | null },
  canvasRoot: HTMLElement,
): { node: ComponentNode | null; element: HTMLElement | null } {
  const rootId = currentPage.rootNodeId;
  if (!rootId) {
    return { node: null, element: null };
  }
  const rootNode = doc.getNode(rootId) as ComponentNode | null;
  if (!rootNode) {
    return { node: null, element: null };
  }
  const rootElement = document.querySelector(`[data-node-id="${CSS.escape(rootId)}"]`) as HTMLElement | null;
  return { node: rootNode, element: rootElement || canvasRoot };
}

// ---------------------------------------------------------------------------
// 单一入口函数
// ---------------------------------------------------------------------------

/**
 * 放置解析 — 单一入口
 *
 * 整合容器命中检测、策略选择、insertIndex 计算，返回完整的放置解析结果。
 *
 * D-01: 根级容器为主要放置目标
 * D-02: 深度优先为 fallback
 * D-03: Strategy 模式按容器类型分发
 * D-04: Y-first + X 辅助排序
 * D-05: append 兜底
 * D-06: 最近邻精调
 *
 * @param event 拖拽事件
 * @param canvasRoot 画布根元素
 * @param doc 文档模型
 * @param currentPage 当前页面
 * @param zoom 当前缩放比例
 * @param options 解析选项
 * @returns 放置解析结果
 */
export function resolvePlacement(
  event: DragEvent | MouseEvent,
  canvasRoot: HTMLElement,
  doc: CanvasDocLike,
  currentPage: { rootNodeId: string | null },
  zoom: number,
  options: ResolvePlacementOptions = {},
): PlacementResolution {
  if (!event || !canvasRoot || !doc || !currentPage) {
    return {
      parentId: null,
      index: -1,
      dropPosition: null,
      containerType: null,
      insertLine: null,
      gridHint: null,
    };
  }

  // 1. 容器命中检测（根级优先 + 深度优先 fallback）
  const { node: containerNode, element: containerElement } = findTargetContainer(
    event,
    canvasRoot,
    doc,
    currentPage,
    options,
  );

  if (!containerNode || !containerElement) {
    return {
      parentId: null,
      index: -1,
      dropPosition: null,
      containerType: null,
      insertLine: null,
      gridHint: null,
    };
  }

  // 2. 确定容器类型
  const nodeType = containerNode.type;
  const childPositioning = getChildPositioning(nodeType);
  const isFlow = childPositioning === "flow";

  let containerKind: FlowContainerKind = "free";
  if (isFlow) {
    // 判断是 flex 还是 grid
    if (nodeType === "GridContainer" || nodeType.includes("ColumnLayout")) {
      containerKind = "grid";
    } else {
      containerKind = "flex";
    }
  } else {
    containerKind = "free";
  }

  // 3. 选择策略
  const strategy = getStrategy(containerKind);

  // 4. 计算 insertIndex（Y-first + nearest neighbor + append fallback）
  const dropPos = eventToCanvasPosition(event, containerElement, zoom);
  const siblings = (containerNode.children || []).map((id: string) => doc.getNode(id) as ComponentNode).filter(Boolean);

  // 策略计算原始索引
  const originalIndex = strategy.resolveInsertIndex(event, containerNode, containerElement, zoom);

  // Y-first + nearest neighbor + append fallback
  const finalIndex = calculateInsertIndexWithFallback(
    dropPos,
    siblings,
    containerElement,
    zoom,
    originalIndex,
  );

  // 5. 计算 dropPosition（仅 free 容器需要）
  const dropPosition = strategy.resolveDropPosition
    ? strategy.resolveDropPosition(event, containerElement, zoom)
    : null;

  return {
    parentId: containerNode.id,
    index: finalIndex,
    dropPosition,
    containerType: containerKind,
    insertLine: containerKind === "flex" ? null : null, // TODO: flex 需要返回 insertLine
    gridHint: containerKind === "grid" ? null : null, // TODO: grid 需要返回 gridHint
  };
}

// ---------------------------------------------------------------------------
// 导出（供外部调用）
// ---------------------------------------------------------------------------

export {
  FlexContainerStrategy,
  FreeContainerStrategy,
  GridContainerStrategy,
};
