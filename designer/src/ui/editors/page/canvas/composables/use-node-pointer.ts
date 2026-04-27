/**
 * 节点指针拖拽 Composable
 *
 * 从 NodeRenderer 抽取的 handlePointerDown 逻辑（含 activeDragHandlers / cleanupDragHandlers）。
 *
 * @module ui/Canvas/composables/use-node-pointer
 */

import type { NodePatchLike, PointerDragHandlers, UseNodePointerDeps } from "./types";
import type { ComponentNode } from "@/editor-core/document/types";
import { onBeforeUnmount } from "vue";
import {
  getDefaultSize,
  getDescriptor,
  isContainerType,
  isFlexContainer,
  isRegionType,
} from "@/editor-core/descriptors/registry";
import { MoveNodeCommand, UpdateNodeCommand } from "@/editor-core/commands/nodeCommands";
import { createSelectableElement } from "@/editor-core/document/types";
import {
  buildFlowResetStyle,
  clampElContainerPropsBySize,
  resolveAbsoluteLayout,
  resolveElContainerMain,
  resolveElContainerMinSize,
} from "@/editor-core/utils/layout-utils";

/** 与 use-node-drop 保持一致的插入线边缘阈值（px） */
const rowInsertEdgeThreshold = 8;
const colInsertEdgeThreshold = 8;
/** 元素对齐吸附的画布坐标阈值 */
const alignmentSnapThreshold = 6;

interface AbsoluteLayoutLike {
  h: number;
  w: number;
  x: number;
  y: number;
  z: number;
}

interface AlignmentSnapTarget extends AbsoluteLayoutLike {
  id: string;
}

interface AlignmentSnapOptions {
  doc: any;
  movingNode: ComponentNode;
  dragNodeIds: Set<string>;
  nextRect: AbsoluteLayoutLike;
  zoomValue: number;
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

function resolveNodeAbsoluteLayout(
  targetNode: ComponentNode | null | undefined,
): AbsoluteLayoutLike | null {
  if (!targetNode) return null;
  if (targetNode.positioning === "flow") return null;
  const abs = targetNode.absolutePos;
  const legacyAbs = targetNode.layoutItem?.free?.abs;
  if (
    abs &&
    (targetNode.positioning === "absolute" || isFiniteNumber(abs.x) || isFiniteNumber(abs.y))
  ) {
    return {
      x: isFiniteNumber(abs.x) ? abs.x : 0,
      y: isFiniteNumber(abs.y) ? abs.y : 0,
      w: isFiniteNumber(abs.w) ? abs.w : 100,
      h: isFiniteNumber(abs.h) ? abs.h : 100,
      z: isFiniteNumber(abs.z) ? abs.z : 1,
    };
  }
  if (legacyAbs) {
    return {
      x: isFiniteNumber(legacyAbs.x) ? legacyAbs.x : 0,
      y: isFiniteNumber(legacyAbs.y) ? legacyAbs.y : 0,
      w: isFiniteNumber(legacyAbs.w) ? legacyAbs.w : 100,
      h: isFiniteNumber(legacyAbs.h) ? legacyAbs.h : 100,
      z: isFiniteNumber(legacyAbs.z) ? legacyAbs.z : 1,
    };
  }
  return null;
}

function escapeNodeIdForSelector(nodeId: string): string {
  if (typeof CSS !== "undefined" && typeof CSS.escape === "function") {
    return CSS.escape(nodeId);
  }
  return nodeId.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

function queryNodeElement(nodeId: string): HTMLElement | null {
  return document.querySelector<HTMLElement>(`[data-node-id="${escapeNodeIdForSelector(nodeId)}"]`);
}

function resolveNodeRenderedSize(
  nodeId: string,
  layout: AbsoluteLayoutLike,
  zoomValue: number,
): { h: number; w: number } {
  const rect = queryNodeElement(nodeId)?.getBoundingClientRect?.();
  if (!rect || rect.width <= 0 || rect.height <= 0) {
    return { w: layout.w, h: layout.h };
  }
  return {
    w: Math.round(rect.width / zoomValue),
    h: Math.round(rect.height / zoomValue),
  };
}

function collectAlignmentSnapTargets(options: AlignmentSnapOptions): AlignmentSnapTarget[] {
  const { doc, movingNode, dragNodeIds, zoomValue } = options;
  if (!doc) return [];
  const movingParentId = doc.getParent?.(movingNode.id)?.id || null;
  const nodesById = doc.nodesById;
  const rawNodes: unknown[] =
    nodesById instanceof Map ? Array.from(nodesById.values()) : Object.values(nodesById || {});
  const targets: AlignmentSnapTarget[] = [];
  for (const item of rawNodes) {
    const targetNode = item as ComponentNode | null | undefined;
    if (!targetNode?.id) continue;
    if (targetNode.hidden) continue;
    if (dragNodeIds.has(targetNode.id)) continue;
    const targetParentId = doc.getParent?.(targetNode.id)?.id || null;
    if (targetParentId !== movingParentId) continue;
    const layout = resolveNodeAbsoluteLayout(targetNode);
    if (!layout) continue;
    const renderedSize = resolveNodeRenderedSize(targetNode.id, layout, zoomValue);
    targets.push({
      id: targetNode.id,
      x: layout.x,
      y: layout.y,
      w: renderedSize.w,
      h: renderedSize.h,
      z: layout.z,
    });
  }
  return targets;
}

function resolveNearestLineDelta(
  sourceLines: number[],
  targetLines: number[],
  threshold = alignmentSnapThreshold,
): number | null {
  let bestDistance = Number.POSITIVE_INFINITY;
  let bestDelta = 0;
  for (const sourceLine of sourceLines) {
    for (const targetLine of targetLines) {
      const delta = targetLine - sourceLine;
      const distance = Math.abs(delta);
      if (distance <= threshold && distance < bestDistance) {
        bestDistance = distance;
        bestDelta = delta;
      }
    }
  }
  return bestDistance <= threshold ? bestDelta : null;
}

function resolveAlignmentSnap(options: AlignmentSnapOptions): AbsoluteLayoutLike {
  const { nextRect } = options;
  const targets = collectAlignmentSnapTargets(options);
  if (!targets.length) return nextRect;
  const xLines = [nextRect.x, nextRect.x + nextRect.w / 2, nextRect.x + nextRect.w];
  const yLines = [nextRect.y, nextRect.y + nextRect.h / 2, nextRect.y + nextRect.h];
  const targetXLines = targets.flatMap((item) => [item.x, item.x + item.w / 2, item.x + item.w]);
  const targetYLines = targets.flatMap((item) => [item.y, item.y + item.h / 2, item.y + item.h]);
  const snapDeltaX = resolveNearestLineDelta(xLines, targetXLines);
  const snapDeltaY = resolveNearestLineDelta(yLines, targetYLines);
  return {
    ...nextRect,
    x: snapDeltaX === null ? nextRect.x : nextRect.x + snapDeltaX,
    y: snapDeltaY === null ? nextRect.y : nextRect.y + snapDeltaY,
  };
}

/**
 * 创建节点指针按下 / 拖拽移动逻辑
 * @param {UseNodePointerDeps} deps - 依赖项
 * @returns {{ handlePointerDown: Function }}
 */
export function useNodePointer(deps: UseNodePointerDeps): {
  handlePointerDown: (event: PointerEvent) => void;
} {
  const {
    node,
    doc,
    nodeRef,
    readonly,
    isMovable,
    isContainer,
    selection,
    canvasZoom,
    enableSnap,
    history,
    editorStore,
    currentPage,
    startDrag,
    endDrag,
    updateDropTarget,
    clearDropTarget,
    dragDropManager,
    canAcceptChild,
    showInsertLine,
    insertLineStyle,
    genericInsertLineBox,
    rowInsertInfo,
    layoutInsertInfo,
    activeTabName,
    tabsList,
    activeCollapseName,
    collapseItems,
    resolveFlexDirection,
  } = deps;

  const resolveActiveTabKey = (): string => {
    const raw =
      activeTabName.value || tabsList.value?.[0]?.name || tabsList.value?.[0]?.label || "";
    return String(raw || "").trim();
  };

  const resolveActiveCollapseKey = (): string => {
    const raw =
      activeCollapseName.value ||
      collapseItems.value?.[0]?.name ||
      collapseItems.value?.[0]?.title ||
      collapseItems.value?.[0]?.label ||
      "";
    return String(raw || "").trim();
  };

  /**
   * 根据目标容器同步/清理 Tabs/Collapse 归属 key，避免拖拽后残留状态
   * @param {Record<string, unknown> | undefined} sourceProps - 源属性
   * @param {string | null | undefined} targetType - 目标容器类型
   * @returns {Record<string, unknown>}
   */
  const resolveScopedSlotProps = (
    sourceProps: Record<string, unknown> | undefined,
    targetType: string | null | undefined,
    scopedKey?: string | null,
  ): Record<string, unknown> => {
    const nextProps = { ...(sourceProps || {}) };
    if (targetType === "Tabs") {
      const tabKey = String(scopedKey || resolveActiveTabKey() || "").trim();
      if (tabKey) {
        nextProps.tabKey = tabKey;
      }
      if ("collapseKey" in nextProps) {
        delete nextProps.collapseKey;
      }
      return nextProps;
    }
    if (targetType === "Collapse") {
      const collapseKey = String(scopedKey || resolveActiveCollapseKey() || "").trim();
      if (collapseKey) {
        nextProps.collapseKey = collapseKey;
      }
      if ("tabKey" in nextProps) {
        delete nextProps.tabKey;
      }
      return nextProps;
    }
    if ("tabKey" in nextProps) {
      delete nextProps.tabKey;
    }
    if ("collapseKey" in nextProps) {
      delete nextProps.collapseKey;
    }
    return nextProps;
  };

  const resolveScopedSlotKeyFromPoint = (
    targetNode: ComponentNode,
    pointEvent?: MouseEvent | PointerEvent | null,
  ): string => {
    const isTabs = targetNode.type === "Tabs";
    const attrName = isTabs ? "data-tab-key" : "data-collapse-key";
    const fallbackKey = isTabs ? resolveActiveTabKey() : resolveActiveCollapseKey();
    if (pointEvent && typeof document.elementsFromPoint === "function") {
      const hits = document.elementsFromPoint(pointEvent.clientX, pointEvent.clientY);
      const nodeSelector = `[data-node-id="${escapeNodeIdForSelector(targetNode.id)}"][${attrName}]`;
      for (const hit of hits) {
        const scopedElement = hit.closest?.(nodeSelector);
        if (scopedElement) {
          const key = String(scopedElement.getAttribute(attrName) || "").trim();
          if (key) return key;
        }
      }
    }
    return String(fallbackKey || "").trim();
  };

  /**
   * 解析 Tabs/Collapse 的活动内容区宿主元素与子节点集合
   * @param {import('@/editor-core').ComponentNode | null | undefined} targetNode - 目标容器
   * @returns {{ hostElement: Element | null, childIds: string[] } | null}
   */
  const resolveScopedSlotHostMeta = (
    targetNode: ComponentNode | null | undefined,
    pointEvent?: MouseEvent | PointerEvent | null,
  ) => {
    if (!targetNode || (targetNode.type !== "Tabs" && targetNode.type !== "Collapse")) {
      return null;
    }
    const isTabs = targetNode.type === "Tabs";
    const key = resolveScopedSlotKeyFromPoint(targetNode, pointEvent);
    const attrName = isTabs ? "data-tab-key" : "data-collapse-key";
    const keyProp = isTabs ? "tabKey" : "collapseKey";
    const escapedKey =
      typeof CSS !== "undefined" && typeof CSS.escape === "function" ? CSS.escape(key) : key;
    const selector = `[data-node-id="${targetNode.id}"][${attrName}="${escapedKey}"]`;
    const hostElement = document.querySelector(selector);
    const childIds = (targetNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      if (!childNode) return false;
      const slotKey = childNode.props?.[keyProp];
      if (!slotKey) return Boolean(key);
      return String(slotKey) === key;
    });
    return { childIds, hostElement, key, keyProp };
  };

  /**
   * 将 Tabs/Collapse 活动区相对插入索引映射为容器绝对索引
   * @param {import('@/editor-core').ComponentNode | null | undefined} targetNode - 目标容器
   * @param {string[]} scopedChildIds - 活动区子节点列表
   * @param {number} scopedInsertIndex - 活动区相对索引
   * @returns {number}
   */
  const mapScopedInsertIndex = (
    targetNode: ComponentNode | null | undefined,
    scopedChildIds: string[],
    scopedInsertIndex: number,
  ): number => {
    const allChildIds = targetNode?.children || [];
    if (!scopedChildIds.length) return allChildIds.length;
    const scopedIndices = scopedChildIds
      .map((childId) => allChildIds.indexOf(childId))
      .filter((value) => value >= 0)
      .sort((a, b) => a - b);
    if (!scopedIndices.length) return allChildIds.length;
    if (scopedInsertIndex <= 0) return scopedIndices[0] ?? allChildIds.length;
    if (scopedInsertIndex >= scopedIndices.length) {
      const lastIndex = scopedIndices[scopedIndices.length - 1];
      return (lastIndex ?? allChildIds.length - 1) + 1;
    }
    return scopedIndices[scopedInsertIndex] ?? allChildIds.length;
  };

  let activeDragHandlers: PointerDragHandlers | null = null;
  const createUpdateCommand = (nodeId: string, patch: NodePatchLike) =>
    new UpdateNodeCommand(nodeId, patch as Partial<ComponentNode>);

  /**
   * 清理拖拽事件监听
   * @returns {void}
   * @throws {Error} 无
   */
  const cleanupDragHandlers = () => {
    if (!activeDragHandlers) return;
    const {
      move,
      up,
      userSelect,
      pointerTarget,
      pointerId,
      usePointer,
      pointerEvents,
      pointerElement,
    } = activeDragHandlers;
    if (usePointer) {
      document.removeEventListener("pointermove", move);
      document.removeEventListener("pointerup", up);
      document.removeEventListener("pointercancel", up);
    } else {
      document.removeEventListener("mousemove", move);
      document.removeEventListener("mouseup", up);
    }
    if (pointerTarget?.releasePointerCapture && pointerId !== undefined) {
      try {
        pointerTarget.releasePointerCapture(pointerId);
      } catch {
        // 忽略释放失败
      }
    }
    if (pointerElement && pointerEvents !== undefined) {
      pointerElement.style.pointerEvents = pointerEvents;
    }
    document.body.style.userSelect = userSelect ?? "";
    activeDragHandlers = null;
  };

  onBeforeUnmount(() => {
    cleanupDragHandlers();
  });

  /**
   * 处理节点指针按下事件
   * @param {PointerEvent} event - 指针事件
   */
  const handlePointerDown = (event: PointerEvent) => {
    if (readonly.value) return;
    if (activeDragHandlers) return;
    if (!node.value || !isMovable.value) return;
    if (event.pointerType === "mouse" && event.button !== 0) return;
    const targetElement = event.target instanceof Element ? event.target : null;
    if (targetElement?.closest(".resize-handle")) return;
    const targetNodeEl = targetElement?.closest("[data-node-id]");
    const targetNodeId = targetNodeEl?.getAttribute?.("data-node-id");
    if (node.value.type === "Tabs" && targetNodeId && targetNodeId !== node.value.id) {
      return;
    }
    if (targetNodeId && targetNodeId !== node.value.id) {
      if (event.altKey) return;
      if (!isContainer.value) return;
      // descriptor 架构流式容器：让子节点自行处理选中和拖拽
      const desc = getDescriptor(node.value.type);
      if (desc?.childPositioning === "flow") return;
    }

    event.preventDefault();
    event.stopPropagation();

    if (selection.value) {
      // 捕获阶段避免破坏 Ctrl/Meta/Shift 多选逻辑，交由 click 阶段统一处理
      if (event.ctrlKey || event.metaKey || event.shiftKey) {
        return;
      }
      const element = createSelectableElement("node", node.value.id);
      // 如果节点已在选中集合中，不覆盖多选，保持多选状态以便后续拖拽
      if (!selection.value.isSelected(node.value.id)) {
        selection.value.select(element);
      }
    }

    const resetFlowStyle = () => {
      if (!node.value) return;
      const parentNode = doc.value?.getParent?.(node.value.id);
      const isFlow =
        parentNode?.type &&
        parentNode.type !== "FreeContainer" &&
        node.value.positioning !== "absolute";
      if (!isFlow) return;
      const patch = { ...(node.value.style || {}) };
      delete patch.width;
      delete patch.height;
      if (Object.keys(patch).length === 0) {
        return;
      }
      if (history.value?.execute) {
        history.value.execute(new UpdateNodeCommand(node.value.id, { style: patch }));
        return;
      }
      if (doc.value?._updateNode) {
        doc.value._updateNode(node.value.id, { style: patch });
      }
    };

    const originParent = doc.value?.getParent?.(node.value.id);
    const isRegionNode =
      node.value.type === "ElHeader" ||
      node.value.type === "ElAside" ||
      node.value.type === "ElMain" ||
      node.value.type === "ElFooter";
    const isRegionParent =
      originParent?.type === "ElHeader" ||
      originParent?.type === "ElAside" ||
      originParent?.type === "ElMain" ||
      originParent?.type === "ElFooter";
    const originContainer = isRegionParent
      ? doc.value?.getParent?.(originParent?.id)
      : isRegionNode
        ? originParent
        : null;
    const allowRegionMoveOut = isRegionNode;
    const rootNodeId = currentPage.value?.rootNodeId || "";
    const isInsideElContainer = (targetNode: ComponentNode | null | undefined) => {
      if (!targetNode || !doc.value) return false;
      if (targetNode.type === "ElContainer") return true;
      let current = doc.value.getParent?.(targetNode.id);
      while (current) {
        if (current.type === "ElContainer") return true;
        current = doc.value.getParent?.(current.id);
      }
      return false;
    };
    /**
     * 获取最近的 ElContainer 祖先
     * @param {string} nodeId - 节点 ID
     * @returns {import('@/editor-core').ComponentNode | null}
     */
    const resolveAncestorContainer = (nodeId: string) => {
      if (!nodeId || !doc.value) return null;
      let current = doc.value.getParent?.(nodeId);
      while (current) {
        if (current.type === "ElContainer") return current;
        current = doc.value.getParent?.(current.id);
      }
      return null;
    };
    /**
     * Alt 拖拽时，将布局内部落点提升到布局外层容器，避免落入 ElLayout/Row/Col 内部。
     * @param {ComponentNode | null | undefined} targetNode - 命中的目标节点
     * @returns {ComponentNode | null}
     */
    const resolveOuterDropContainer = (
      targetNode: ComponentNode | null | undefined,
    ): ComponentNode | null => {
      if (!targetNode || !doc.value) return null;
      /**
       * 判断 Alt 拖拽时是否应继续向上提升容器层级。
       * - 兼容历史 ElLayout/Row/Col
       * - 新增 descriptor 流式容器（如 VerticalLayout/HorizontalLayout）
       */
      const shouldElevateByAlt = (currentNode: ComponentNode | null) => {
        if (!currentNode) return false;
        if (
          currentNode.type === "ElCol" ||
          currentNode.type === "ElLayoutRow" ||
          currentNode.type === "ElLayout"
        ) {
          return true;
        }
        const descriptor = getDescriptor(currentNode.type);
        return Boolean(descriptor?.isContainer && descriptor.childPositioning === "flow");
      };
      let current: ComponentNode | null = targetNode;
      while (current && shouldElevateByAlt(current)) {
        const parent = doc.value.getParent?.(current.id);
        current = parent || null;
      }
      return current;
    };
    const dragContainer = resolveAncestorContainer(node.value?.id);
    const restrictToContainer = Boolean(dragContainer) && node.value?.type !== "ElContainer";

    const zoomValue = Number(canvasZoom?.value) || 1;
    const baseLayout = resolveAbsoluteLayout(node.value, nodeRef.value ?? null);
    const selectedNodeIds =
      selection.value
        ?.getSelectedElements?.()
        ?.filter((el) => el.kind === "node")
        ?.map((el) => el.id) || [];
    const dragNodeIds =
      selection.value?.isSelected?.(node.value.id) && selectedNodeIds.length
        ? selectedNodeIds
        : [node.value.id];
    const isMultiDrag = dragNodeIds.length > 1;
    /**
     * 解析节点基础绝对布局（不依赖 DOM，适用于批量拖拽）
     * @param {import('@/editor-core').ComponentNode | null | undefined} targetNode - 目标节点
     * @returns {{ x: number, y: number, w: number, h: number, z: number } | null}
     */
    const resolveBaseLayoutFromNode = (
      targetNode: ComponentNode | null | undefined,
    ): AbsoluteLayoutLike | null => {
      if (!targetNode) return null;
      if (targetNode.positioning === "flow") return null;
      const abs = targetNode.absolutePos;
      const legacyAbs = targetNode.layoutItem?.free?.abs;
      if (
        abs &&
        (targetNode.positioning === "absolute" || Number.isFinite(abs.x) || Number.isFinite(abs.y))
      ) {
        return {
          x: Number.isFinite(abs.x) ? abs.x : 0,
          y: Number.isFinite(abs.y) ? abs.y : 0,
          w: Number.isFinite(abs.w) ? abs.w : 100,
          h: Number.isFinite(abs.h) ? abs.h : 100,
          z: typeof abs.z === "number" && Number.isFinite(abs.z) ? abs.z : 1,
        };
      }
      if (legacyAbs) {
        return {
          x: Number.isFinite(legacyAbs.x) ? legacyAbs.x : 0,
          y: Number.isFinite(legacyAbs.y) ? legacyAbs.y : 0,
          w: Number.isFinite(legacyAbs.w) ? legacyAbs.w : 100,
          h: Number.isFinite(legacyAbs.h) ? legacyAbs.h : 100,
          z: typeof legacyAbs.z === "number" && Number.isFinite(legacyAbs.z) ? legacyAbs.z : 1,
        };
      }
      return {
        x:
          typeof targetNode.style?.left === "number" && Number.isFinite(targetNode.style.left)
            ? targetNode.style.left
            : 0,
        y:
          typeof targetNode.style?.top === "number" && Number.isFinite(targetNode.style.top)
            ? targetNode.style.top
            : 0,
        w:
          typeof targetNode.style?.width === "number" && Number.isFinite(targetNode.style.width)
            ? targetNode.style.width
            : 100,
        h:
          typeof targetNode.style?.height === "number" && Number.isFinite(targetNode.style.height)
            ? targetNode.style.height
            : 100,
        z:
          typeof targetNode.style?.zIndex === "number" && Number.isFinite(targetNode.style.zIndex)
            ? targetNode.style.zIndex
            : 1,
      };
    };
    const baseLayoutsById = new Map<string, AbsoluteLayoutLike>();
    for (const dragNodeId of dragNodeIds) {
      const dragNode = doc.value?.getNode?.(dragNodeId);
      const base = resolveBaseLayoutFromNode(dragNode);
      if (base) {
        baseLayoutsById.set(dragNodeId, base);
      }
    }
    const startClientX = event.clientX;
    const startClientY = event.clientY;
    let hasMoved = false;
    let startedDragFromMove = false;

    // 判断当前节点是否为 descriptor 架构流式容器的直接子项
    const isFlowChildInDescContainer = (() => {
      if (!node.value || node.value.positioning !== "flow") return false;
      if (!originParent) return false;
      const parentDesc = getDescriptor(originParent.type);
      return parentDesc?.childPositioning === "flow";
    })();
    // 流式子项拖拽阈值：超过自身宽/高一半才真正移出容器
    let flowDragThresholdW = 0;
    let flowDragThresholdH = 0;
    let flowDragExceeded = false;
    if (isFlowChildInDescContainer) {
      // nodeRef 可能在 pointerdown 时尚未就绪，降级通过 data-node-id 查询 DOM
      const rect =
        nodeRef.value?.getBoundingClientRect?.() ??
        document.querySelector(`[data-node-id="${node.value?.id}"]`)?.getBoundingClientRect?.();
      if (rect && rect.width > 0 && rect.height > 0) {
        flowDragThresholdW = rect.width / 2;
        flowDragThresholdH = rect.height / 2;
      } else {
        // 最终降级：默认 20px
        flowDragThresholdW = 20;
        flowDragThresholdH = 20;
      }
    }

    const resolveDropRegion = (
      upEvent: MouseEvent | PointerEvent | null | undefined,
      preferOuterDropByAlt = false,
    ) => {
      if (!upEvent) return null;
      const hitList = document.elementsFromPoint(upEvent.clientX, upEvent.clientY);
      const childType = node.value?.type;
      let containerNode: ComponentNode | null = null;
      /**
       * 判断是否为区域容器类型
       * @param {string} type - 组件类型
       * @returns {boolean}
       */
      for (const hit of hitList) {
        const nodeElement = hit.closest?.("[data-node-id]");
        const nodeId = nodeElement?.getAttribute?.("data-node-id");
        if (!nodeId) continue;
        if (isSelfOrDescendant(nodeId)) continue;
        const hitNode = doc.value?.getNode?.(nodeId);
        if (!hitNode) continue;
        const targetNode = preferOuterDropByAlt ? resolveOuterDropContainer(hitNode) : hitNode;
        if (!targetNode) continue;
        if (isSelfOrDescendant(targetNode.id)) continue;
        if (restrictToContainer && !isInsideElContainer(targetNode)) {
          continue;
        }
        if (isRegionType(targetNode.type)) {
          if (childType && !canAcceptChild(targetNode, childType)) continue;
          return targetNode;
        }
        if (targetNode.type === "ElContainer") {
          if (!containerNode) {
            containerNode = resolveElContainerMain(doc.value, targetNode) || targetNode;
          }
          continue;
        }
        if (isContainerType(targetNode.type)) {
          if (childType && !canAcceptChild(targetNode, childType)) continue;
          return targetNode;
        }
      }
      if (containerNode && (!childType || canAcceptChild(containerNode, childType))) {
        return containerNode;
      }
      return null;
    };
    function isSelfOrDescendant(targetId: string | null | undefined) {
      if (!targetId || !node.value?.id || !doc.value) return false;
      if (targetId === node.value.id) return true;
      let current = doc.value.getParent?.(targetId);
      while (current) {
        if (current.id === node.value.id) return true;
        current = doc.value.getParent?.(current.id);
      }
      return false;
    }

    cleanupDragHandlers();
    const originalUserSelect = document.body.style.userSelect;
    document.body.style.userSelect = "none";

    if (history.value && !history.value.isInTransaction?.()) {
      history.value.beginTransaction?.();
    }

    const usePointer = event.type === "pointerdown";
    const pointerTarget = event.target instanceof window.Element ? event.target : nodeRef.value;
    if (usePointer && pointerTarget?.setPointerCapture && event.pointerId !== undefined) {
      try {
        pointerTarget.setPointerCapture?.(event.pointerId);
      } catch {
        // 忽略捕获失败
      }
    }

    const resolveRowInsertFromPoint = (pointEvent: MouseEvent | PointerEvent) => {
      const hitList = document.elementsFromPoint(pointEvent.clientX, pointEvent.clientY);
      for (const hit of hitList) {
        const nodeElement = hit.closest?.("[data-node-id][data-node-type]");
        if (!nodeElement) continue;
        const nodeType = nodeElement.getAttribute("data-node-type");
        if (nodeType !== "ElCol") continue;
        const nodeId = nodeElement.getAttribute("data-node-id");
        if (isSelfOrDescendant(nodeId)) continue;
        const colNode = nodeId ? doc.value?.getNode?.(nodeId) : null;
        if (!colNode) continue;
        const parentNode = doc.value?.getParent?.(colNode.id);
        if (parentNode?.type !== "ElLayoutRow") continue;
        if (isSelfOrDescendant(parentNode.id)) continue;
        const colRect = nodeElement.getBoundingClientRect?.();
        const edgeThreshold = colInsertEdgeThreshold;
        const nearEdge = colRect
          ? pointEvent.clientX - colRect.left <= edgeThreshold ||
            colRect.right - pointEvent.clientX <= edgeThreshold
          : false;
        const rowSelector = `[data-node-id="${parentNode.id}"]`;
        const rowElement =
          document.querySelector(rowSelector) || nodeElement.closest?.(rowSelector);
        if (!rowElement) continue;
        return { rowElement, parentNode, nearEdge };
      }
      return null;
    };

    const resolveLayoutInsertFromPoint = (pointEvent: MouseEvent | PointerEvent) => {
      const hitList = document.elementsFromPoint(pointEvent.clientX, pointEvent.clientY);
      for (const hit of hitList) {
        const layoutElement = hit.closest?.('[data-node-type="ElLayout"][data-node-id]');
        if (!layoutElement) continue;
        const layoutId = layoutElement.getAttribute("data-node-id");
        if (isSelfOrDescendant(layoutId)) continue;
        const layoutNode = layoutId ? doc.value?.getNode?.(layoutId) : null;
        if (!layoutNode || layoutNode.type !== "ElLayout") continue;
        const rowIds = (layoutNode.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElLayoutRow";
        });
        if (rowIds.length === 0) return null;
        const pointY = pointEvent.clientY;
        const threshold = rowInsertEdgeThreshold;
        for (let i = 0; i < rowIds.length; i += 1) {
          const rowId = rowIds[i];
          const rowElement = layoutElement.querySelector(`[data-node-id="${rowId}"]`);
          if (!rowElement) continue;
          const rect = rowElement.getBoundingClientRect?.();
          if (!rect) continue;
          if (Math.abs(pointY - rect.top) <= threshold) {
            return { layoutNode, layoutElement, index: i, lineY: rect.top };
          }
          if (Math.abs(pointY - rect.bottom) <= threshold) {
            return {
              layoutNode,
              layoutElement,
              index: i + 1,
              lineY: rect.bottom,
            };
          }
        }
      }
      return null;
    };

    const updateRowInsertLineFromPoint = (pointEvent: MouseEvent | PointerEvent) => {
      if (!node.value || node.value.type === "ElCol") return false;
      const resolvedRow = resolveRowInsertFromPoint(pointEvent);
      if (!resolvedRow || !resolvedRow.nearEdge) {
        showInsertLine.value = false;
        insertLineStyle.value = null;
        genericInsertLineBox.value = null;
        rowInsertInfo.value = null;
        return false;
      }
      const { rowElement, parentNode } = resolvedRow;
      const rowRect = rowElement.getBoundingClientRect?.();
      if (!rowRect) return false;
      const direction = resolveFlexDirection("ElLayoutRow", rowElement);
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        rowElement,
        pointEvent,
        direction,
      );
      const insertLine = insertInfo.insertLine || {
        orientation: "vertical",
        offset: rowRect.right - rowRect.left,
      };
      const adjustedLine = {
        ...insertLine,
        offset: Math.max(0, insertLine.offset),
      };
      showInsertLine.value = true;
      insertLineStyle.value = adjustedLine;
      genericInsertLineBox.value = null;
      rowInsertInfo.value = {
        rowId: parentNode.id,
        index: insertInfo.index,
        lineBox: {
          left: rowRect.left,
          top: rowRect.top,
          width: rowRect.width,
          height: rowRect.height,
        },
      };
      return true;
    };

    const updateLayoutInsertLineFromPoint = (pointEvent: MouseEvent | PointerEvent) => {
      if (!node.value || node.value.type === "ElLayoutRow") return false;
      const resolvedLayout = resolveLayoutInsertFromPoint(pointEvent);
      if (!resolvedLayout) {
        showInsertLine.value = false;
        insertLineStyle.value = null;
        genericInsertLineBox.value = null;
        layoutInsertInfo.value = null;
        return false;
      }
      const { layoutElement, layoutNode, index, lineY } = resolvedLayout;
      const layoutRect = layoutElement.getBoundingClientRect?.();
      if (!layoutRect) return false;
      showInsertLine.value = true;
      insertLineStyle.value = {
        orientation: "horizontal",
        offset: Math.max(0, lineY - layoutRect.top),
      };
      genericInsertLineBox.value = null;
      rowInsertInfo.value = null;
      layoutInsertInfo.value = {
        layoutId: layoutNode.id,
        index,
        lineBox: {
          left: layoutRect.left,
          top: layoutRect.top,
          width: layoutRect.width,
          height: layoutRect.height,
        },
      };
      return true;
    };

    const updateGenericFlexInsertLineFromPoint = (
      pointEvent: MouseEvent | PointerEvent,
      dropRegion: ComponentNode | null,
    ) => {
      if (!dropRegion?.type) {
        showInsertLine.value = false;
        insertLineStyle.value = null;
        genericInsertLineBox.value = null;
        return false;
      }
      const scopedMeta = resolveScopedSlotHostMeta(dropRegion, pointEvent);
      if (scopedMeta) {
        if (!scopedMeta.childIds.length || !scopedMeta.hostElement) {
          showInsertLine.value = false;
          insertLineStyle.value = null;
          genericInsertLineBox.value = null;
          return false;
        }
        const insertInfo = dragDropManager.calculateFlexInsertPosition(
          scopedMeta.hostElement,
          pointEvent,
          "column",
        );
        if (!insertInfo.insertLine) {
          showInsertLine.value = false;
          insertLineStyle.value = null;
          genericInsertLineBox.value = null;
          return false;
        }
        showInsertLine.value = true;
        insertLineStyle.value = insertInfo.insertLine;
        const contentRect = scopedMeta.hostElement.getBoundingClientRect?.();
        genericInsertLineBox.value = contentRect
          ? {
              left: contentRect.left,
              top: contentRect.top,
              width: contentRect.width,
              height: contentRect.height,
            }
          : null;
        rowInsertInfo.value = null;
        layoutInsertInfo.value = null;
        return true;
      }
      if (!isFlexContainer(dropRegion.type)) {
        showInsertLine.value = false;
        insertLineStyle.value = null;
        genericInsertLineBox.value = null;
        return false;
      }
      const outerElement = document.querySelector(`[data-node-id="${dropRegion.id}"]`);
      const contentElement =
        outerElement?.querySelector?.("[data-node-id]")?.parentElement || outerElement;
      if (!contentElement) {
        showInsertLine.value = false;
        insertLineStyle.value = null;
        genericInsertLineBox.value = null;
        return false;
      }
      const direction = resolveFlexDirection(dropRegion.type, contentElement);
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        contentElement,
        pointEvent,
        direction,
      );
      if (!insertInfo.insertLine) {
        showInsertLine.value = false;
        insertLineStyle.value = null;
        genericInsertLineBox.value = null;
        return false;
      }
      showInsertLine.value = true;
      insertLineStyle.value = insertInfo.insertLine;
      const contentRect = contentElement.getBoundingClientRect?.();
      genericInsertLineBox.value = contentRect
        ? {
            left: contentRect.left,
            top: contentRect.top,
            width: contentRect.width,
            height: contentRect.height,
          }
        : null;
      rowInsertInfo.value = null;
      layoutInsertInfo.value = null;
      return true;
    };

    const move = (moveEvent: MouseEvent | PointerEvent) => {
      if (!node.value) return;
      const preferOuterDropByAlt = Boolean(moveEvent.altKey);
      const deltaX = (moveEvent.clientX - startClientX) / zoomValue;
      const deltaY = (moveEvent.clientY - startClientY) / zoomValue;
      // 流式容器子项：未超过自身宽/高一半时保持不动
      if (isFlowChildInDescContainer && !flowDragExceeded) {
        if (Math.abs(deltaX) <= flowDragThresholdW && Math.abs(deltaY) <= flowDragThresholdH) {
          return;
        }
        flowDragExceeded = true;
      }
      if (Math.abs(deltaX) > 1 || Math.abs(deltaY) > 1) {
        hasMoved = true;
        if (!startedDragFromMove) {
          startDrag(node.value.type);
          startedDragFromMove = true;
          resetFlowStyle();
        }
        if (!isMultiDrag) {
          const dropRegion = resolveDropRegion(moveEvent, preferOuterDropByAlt);
          if (dropRegion?.id && !isSelfOrDescendant(dropRegion.id)) {
            const rect = document
              .querySelector(`[data-node-id="${dropRegion.id}"]`)
              ?.getBoundingClientRect?.();
            updateDropTarget({
              containerId: dropRegion.id,
              insertIndex: (dropRegion.children || []).length,
              position: rect
                ? {
                    x: rect.left,
                    y: rect.top,
                    width: rect.width,
                    height: rect.height,
                  }
                : { x: 0, y: 0, width: 0, height: 0 },
              layoutType: "flex",
              direction: "column",
            });
          } else {
            clearDropTarget();
          }
          if (preferOuterDropByAlt) {
            showInsertLine.value = false;
            insertLineStyle.value = null;
            genericInsertLineBox.value = null;
            rowInsertInfo.value = null;
            layoutInsertInfo.value = null;
          } else if (
            !updateLayoutInsertLineFromPoint(moveEvent) &&
            !updateRowInsertLineFromPoint(moveEvent)
          ) {
            updateGenericFlexInsertLineFromPoint(moveEvent, dropRegion);
          }
        }
        if (restrictToContainer && dragContainer) {
          const containerEl = document.querySelector(`[data-node-id="${dragContainer.id}"]`);
          const containerRect = containerEl?.getBoundingClientRect?.();
          const isInsideContainer = containerRect
            ? moveEvent.clientX >= containerRect.left &&
              moveEvent.clientX <= containerRect.right &&
              moveEvent.clientY >= containerRect.top &&
              moveEvent.clientY <= containerRect.bottom
            : true;
          if (isInsideContainer) {
            return;
          }
        }
      }
      const executePatch = (nodeId: string, patch: NodePatchLike) => {
        if (history.value?.isInTransaction?.()) {
          history.value.executeInTransaction?.(createUpdateCommand(nodeId, patch));
          return;
        }
        if (history.value?.execute) {
          history.value.execute(createUpdateCommand(nodeId, patch));
          return;
        }
        if (doc.value?._updateNode) {
          doc.value._updateNode(nodeId, patch);
        }
      };

      const primaryBase = (baseLayoutsById.get(node.value.id) || baseLayout) as AbsoluteLayoutLike;
      if (!primaryBase) return;
      const primaryNextAbs = {
        x: Math.round(primaryBase.x + deltaX),
        y: Math.round(primaryBase.y + deltaY),
        w: primaryBase.w,
        h: primaryBase.h,
        z: primaryBase.z,
      };
      if (node.value.type === "ElLayout") {
        const layoutMin = getDefaultSize("ElLayout");
        if (layoutMin?.width) {
          primaryNextAbs.w = Math.max(primaryNextAbs.w, layoutMin.width);
        }
      }
      if (
        hasMoved &&
        enableSnap?.value !== false &&
        !preferOuterDropByAlt &&
        !isFlowChildInDescContainer
      ) {
        const snappedAbs = resolveAlignmentSnap({
          doc: doc.value,
          movingNode: node.value,
          dragNodeIds: new Set(dragNodeIds),
          nextRect: primaryNextAbs,
          zoomValue,
        });
        primaryNextAbs.x = Math.round(snappedAbs.x);
        primaryNextAbs.y = Math.round(snappedAbs.y);
      }
      const movementDeltaX = primaryNextAbs.x - primaryBase.x;
      const movementDeltaY = primaryNextAbs.y - primaryBase.y;
      if (nodeRef.value) {
        nodeRef.value.style.position = "absolute";
        nodeRef.value.style.left = `${primaryNextAbs.x}px`;
        nodeRef.value.style.top = `${primaryNextAbs.y}px`;
        nodeRef.value.style.width = `${primaryNextAbs.w}px`;
        nodeRef.value.style.height = `${primaryNextAbs.h}px`;
        nodeRef.value.style.zIndex = `${primaryNextAbs.z}`;
      }

      for (const [dragNodeId, dragBase] of baseLayoutsById.entries()) {
        const dragNode = doc.value?.getNode?.(dragNodeId);
        if (!dragNode) continue;
        if (!dragBase) continue;
        const nextAbs = {
          x: Math.round(dragBase.x + movementDeltaX),
          y: Math.round(dragBase.y + movementDeltaY),
          w: dragBase.w,
          h: dragBase.h,
          z: dragBase.z,
        };
        const nextLayoutItem = {
          ...(dragNode.layoutItem || {}),
          free: {
            mode: "abs" as const,
            abs: { ...nextAbs },
          },
        };
        const regionClampPatch =
          dragNode.type === "ElContainer"
            ? clampElContainerPropsBySize(dragNode, nextAbs.w, nextAbs.h)
            : null;
        const patch: NodePatchLike = {
          positioning: "absolute",
          absolutePos: nextAbs,
          layoutItem: nextLayoutItem,
          ...(regionClampPatch
            ? { props: { ...(dragNode.props || {}), ...regionClampPatch } }
            : {}),
        };
        executePatch(dragNodeId, patch);
      }

      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new window.CustomEvent("designer:node-transform", {
            detail: { x: primaryNextAbs.x, y: primaryNextAbs.y },
          }),
        );
      }
    };

    const up = (upEvent: MouseEvent | PointerEvent) => {
      let transactionMode: "commit" | "rollback" | "none" = "commit";
      let transactionLabel = "移动组件";
      let shouldReselectDraggedNode = false;
      const draggedNodeId = node.value?.id || null;
      const preferOuterDropByAlt = Boolean(upEvent.altKey);
      try {
        const currentNode = node.value;
        if (!currentNode) {
          transactionMode = "rollback";
          return;
        }
        // 流式容器子项：未超出阈值时回滚事务，保持原位
        if (isFlowChildInDescContainer && !flowDragExceeded) {
          transactionMode = "rollback";
          return;
        }
        if (startedDragFromMove) {
          endDrag();
          clearDropTarget();
        }
        const rowInsertSnapshot = preferOuterDropByAlt ? null : rowInsertInfo.value;
        const layoutInsertSnapshot = preferOuterDropByAlt ? null : layoutInsertInfo.value;
        showInsertLine.value = false;
        insertLineStyle.value = null;
        genericInsertLineBox.value = null;
        rowInsertInfo.value = null;
        layoutInsertInfo.value = null;
        if (hasMoved && !isRegionNode && upEvent && !isMultiDrag) {
          shouldReselectDraggedNode = true;
        }
        // 流式容器子项超出阈值：移出原容器，放入新容器或画布根
        if (isFlowChildInDescContainer && flowDragExceeded && originParent) {
          if (startedDragFromMove) {
            endDrag();
            clearDropTarget();
          }
          showInsertLine.value = false;
          insertLineStyle.value = null;
          genericInsertLineBox.value = null;

          const dropRegion = resolveDropRegion(upEvent, preferOuterDropByAlt);
          const scopedMeta = resolveScopedSlotHostMeta(dropRegion, upEvent);
          const scopedKey = scopedMeta?.key || null;
          const currentScopedKey = scopedMeta?.keyProp
            ? String(currentNode.props?.[scopedMeta.keyProp] || "")
            : "";
          const isSameScopedContainerMove = Boolean(
            dropRegion &&
            dropRegion.id === originParent.id &&
            scopedKey &&
            currentScopedKey !== scopedKey,
          );
          const hasValidDrop =
            dropRegion &&
            (dropRegion.id !== originParent.id || isSameScopedContainerMove) &&
            canAcceptChild(dropRegion, node.value.type) &&
            !isSelfOrDescendant(dropRegion.id);

          if (hasValidDrop) {
            // 拖入其他容器
            const targetId = dropRegion.id;
            let insertIdx = (dropRegion.children || []).length;
            if (scopedMeta?.hostElement && scopedMeta.childIds.length > 0) {
              const scopedInsert = dragDropManager.calculateFlexInsertPosition(
                scopedMeta.hostElement,
                upEvent,
                "column",
              );
              const scopedIndex =
                typeof scopedInsert.index === "number"
                  ? scopedInsert.index
                  : scopedMeta.childIds.length;
              insertIdx = mapScopedInsertIndex(dropRegion, scopedMeta.childIds, scopedIndex);
            }
            const moveCmd = new MoveNodeCommand(currentNode.id, targetId, insertIdx);
            const parentDesc = getDescriptor(dropRegion.type);
            const isFlowTarget = parentDesc?.childPositioning === "flow";
            // 从流式容器拖出后落入绝对定位容器时，baseLayout 是原父坐标系，不能直接用；用鼠标释放位置相对目标容器计算落点
            let absPosForTarget = baseLayout;
            if (!isFlowTarget && isFlowChildInDescContainer) {
              const targetEl = document.querySelector(`[data-node-id="${targetId}"]`);
              const targetRect = targetEl?.getBoundingClientRect?.();
              const nextX = targetRect ? (upEvent.clientX - targetRect.left) / zoomValue : 0;
              const nextY = targetRect ? (upEvent.clientY - targetRect.top) / zoomValue : 0;
              const nodeElRect = document
                .querySelector(`[data-node-id="${node.value.id}"]`)
                ?.getBoundingClientRect?.();
              const nodeW = nodeElRect ? nodeElRect.width / zoomValue : (baseLayout.w ?? 100);
              const nodeH = nodeElRect ? nodeElRect.height / zoomValue : (baseLayout.h ?? 40);
              absPosForTarget = {
                x: Math.round(nextX - nodeW / 2),
                y: Math.round(nextY - nodeH / 2),
                w: Math.round(nodeW),
                h: Math.round(nodeH),
                z: baseLayout.z ?? 1,
              };
            }
            const updatePatch: NodePatchLike = isFlowTarget
              ? {
                  positioning: "flow",
                  absolutePos: undefined,
                  flowLayout: parentDesc.childFlowLayout
                    ? { ...parentDesc.childFlowLayout }
                    : undefined,
                  layoutItem: undefined,
                  props: resolveScopedSlotProps(
                    currentNode.props || {},
                    dropRegion.type,
                    scopedKey,
                  ),
                  style: {
                    ...(buildFlowResetStyle(currentNode.style) || {}),
                    ...(parentDesc.childStyle ? parentDesc.childStyle(dropRegion.type) : {}),
                  },
                }
              : {
                  positioning: "absolute",
                  absolutePos: { ...absPosForTarget },
                  layoutItem: {
                    ...(currentNode.layoutItem || {}),
                    free: { mode: "abs" as const, abs: { ...absPosForTarget } },
                  },
                  props: resolveScopedSlotProps(
                    currentNode.props || {},
                    dropRegion.type,
                    scopedKey,
                  ),
                };
            const updateCmd = createUpdateCommand(currentNode.id, updatePatch);
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction?.(moveCmd);
              history.value.executeInTransaction?.(updateCmd);
            } else if (history.value?.execute) {
              history.value.execute(moveCmd);
              history.value.execute(updateCmd);
            }
          } else {
            // 无合适容器：移到画布根，绝对定位，节点中心对准鼠标释放位置
            // 优先用根节点 DOM 的 rect；找不到时用 .design-canvas 作为坐标系，避免 nextX/nextY 为 0 导致节点跑到左上角
            const rootEl =
              document.querySelector(`[data-node-id="${rootNodeId}"]`) ??
              document.querySelector(".design-canvas");
            const rootRect = rootEl?.getBoundingClientRect?.();
            const nextX = rootRect ? (upEvent.clientX - rootRect.left) / zoomValue : 0;
            const nextY = rootRect ? (upEvent.clientY - rootRect.top) / zoomValue : 0;
            // 从 DOM 读取节点实际宽高，使节点中心对准鼠标释放点
            const nodeElRect = document
              .querySelector(`[data-node-id="${currentNode.id}"]`)
              ?.getBoundingClientRect?.();
            const nodeW = nodeElRect ? nodeElRect.width / zoomValue : (baseLayout.w ?? 100);
            const nodeH = nodeElRect ? nodeElRect.height / zoomValue : (baseLayout.h ?? 40);
            const nextAbs = {
              x: Math.round(nextX - nodeW / 2),
              y: Math.round(nextY - nodeH / 2),
              w: Math.round(nodeW),
              h: Math.round(nodeH),
              z: baseLayout.z ?? 1,
            };
            const rootNode = doc.value?.getNode?.(rootNodeId);
            const insertIdx = rootNode?.children?.length ?? 0;
            const moveCmd = new MoveNodeCommand(currentNode.id, rootNodeId, insertIdx);
            const updateCmd = createUpdateCommand(currentNode.id, {
              positioning: "absolute",
              absolutePos: nextAbs,
              layoutItem: {
                ...(currentNode.layoutItem || {}),
                free: { mode: "abs" as const, abs: { ...nextAbs } },
              },
              props: resolveScopedSlotProps(currentNode.props || {}, "FreeContainer"),
              style: buildFlowResetStyle(currentNode.style),
            });
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction?.(moveCmd);
              history.value.executeInTransaction?.(updateCmd);
            } else if (history.value?.execute) {
              history.value.execute(moveCmd);
              history.value.execute(updateCmd);
            }
          }
          transactionLabel = "移出布局容器";
          return;
        }
        if (
          layoutInsertSnapshot?.layoutId &&
          !isSelfOrDescendant(layoutInsertSnapshot.layoutId) &&
          node.value?.type !== "ElLayoutRow" &&
          node.value?.type !== "ElCol"
        ) {
          const layoutNode = doc.value?.getNode?.(layoutInsertSnapshot.layoutId);
          if (layoutNode) {
            const rowNode = editorStore.insertNode(
              "ElLayoutRow",
              layoutNode.id,
              layoutInsertSnapshot.index,
              { autoSelectInserted: false },
            );
            if (rowNode) {
              const latestLayout = doc.value?.getNode?.(layoutNode.id);
              const rowCount = (latestLayout?.children || []).filter((childId) => {
                const childNode = doc.value?.getNode?.(childId);
                return childNode?.type === "ElLayoutRow";
              }).length;
              editorStore.updateNode(layoutNode.id, {
                props: {
                  ...(latestLayout?.props || layoutNode.props || {}),
                  rows: Math.max(1, rowCount),
                },
              });
              editorStore.updateNode(rowNode.id, {
                props: { ...(rowNode.props || {}), columns: 1 },
              });
              const latestRow = doc.value?.getNode?.(rowNode.id);
              const colIds = (latestRow?.children || []).filter((childId) => {
                const childNode = doc.value?.getNode?.(childId);
                return childNode?.type === "ElCol";
              });
              let colId = colIds[0];
              if (!colId) {
                const colNode = editorStore.insertNode("ElCol", rowNode.id, 0, {
                  autoSelectInserted: false,
                });
                if (!colNode) return;
                colId = colNode.id;
              }
              const moveCommand = new MoveNodeCommand(
                currentNode.id,
                colId,
                (latestRow?.children || []).length,
              );
              const updatePatch: NodePatchLike = {
                positioning: "flow",
                absolutePos: undefined,
                flowLayout: undefined,
                layoutItem: undefined,
                style: {
                  ...(buildFlowResetStyle(currentNode.style) || {}),
                  width: "100%",
                  height: isContainer.value ? "100%" : "auto",
                },
              };
              const updateCommand = createUpdateCommand(currentNode.id, updatePatch);
              if (history.value?.isInTransaction?.()) {
                history.value.executeInTransaction?.(moveCommand);
                history.value.executeInTransaction?.(updateCommand);
              } else if (history.value?.execute) {
                history.value.execute(moveCommand);
                history.value.execute(updateCommand);
              } else if (doc.value?._moveNode && doc.value?._updateNode) {
                doc.value._moveNode(currentNode.id, colId, (latestRow?.children || []).length);
                doc.value._updateNode(currentNode.id, updatePatch);
              }
            }
          }
          return;
        }
        if (
          rowInsertSnapshot?.rowId &&
          !isSelfOrDescendant(rowInsertSnapshot.rowId) &&
          node.value?.type !== "ElCol"
        ) {
          const rowNode = doc.value?.getNode?.(rowInsertSnapshot.rowId);
          if (rowNode) {
            const colNode = editorStore.insertNode("ElCol", rowNode.id, rowInsertSnapshot.index, {
              autoSelectInserted: false,
            });
            if (colNode) {
              const latestRow = doc.value?.getNode?.(rowNode.id);
              const colCount = (latestRow?.children || []).filter((childId) => {
                const childNode = doc.value?.getNode?.(childId);
                return childNode?.type === "ElCol";
              }).length;
              editorStore.updateNode(rowNode.id, {
                props: {
                  ...(latestRow?.props || rowNode.props || {}),
                  columns: colCount,
                },
              });
              const moveCommand = new MoveNodeCommand(
                currentNode.id,
                colNode.id,
                (colNode.children || []).length,
              );
              const shouldResetSize =
                currentNode.positioning === "absolute" || currentNode.layoutItem?.free;
              const updatePatch: NodePatchLike = {
                positioning: "flow",
                absolutePos: undefined,
                flowLayout: undefined,
                layoutItem: undefined,
              };
              // 放入 ElCol 时默认铺满容器
              updatePatch.style = {
                ...(buildFlowResetStyle(currentNode.style) || {}),
                width: "100%",
                height: isContainer.value ? "100%" : "auto",
              };
              if (shouldResetSize) {
                updatePatch.style = {
                  ...(buildFlowResetStyle(currentNode.style) || {}),
                  width: "100%",
                  height: isContainer.value ? "100%" : "auto",
                };
              }
              const updateCommand = createUpdateCommand(currentNode.id, updatePatch);
              if (history.value?.isInTransaction?.()) {
                history.value.executeInTransaction?.(moveCommand);
                history.value.executeInTransaction?.(updateCommand);
              } else if (history.value?.execute) {
                history.value.execute(moveCommand);
                history.value.execute(updateCommand);
              } else if (doc.value?._moveNode && doc.value?._updateNode) {
                doc.value._moveNode(currentNode.id, colNode.id, (colNode.children || []).length);
                doc.value._updateNode(currentNode.id, updatePatch);
              }
            }
          }
          return;
        }
        const dropRegion = resolveDropRegion(upEvent, preferOuterDropByAlt);
        if (restrictToContainer && dropRegion && isInsideElContainer(dropRegion)) {
          clearDropTarget();
          return;
        }
        const scopedMeta = resolveScopedSlotHostMeta(dropRegion, upEvent);
        const scopedKey = scopedMeta?.key || null;
        const currentScopedKey = scopedMeta?.keyProp
          ? String(currentNode.props?.[scopedMeta.keyProp] || "")
          : "";
        const isSameScopedContainerMove = Boolean(
          dropRegion &&
          dropRegion.id === originParent?.id &&
          scopedKey &&
          currentScopedKey !== scopedKey,
        );
        if (
          dropRegion &&
          dropRegion.id &&
          (dropRegion.id !== originParent?.id || isSameScopedContainerMove) &&
          canAcceptChild(dropRegion, currentNode.type) &&
          !isSelfOrDescendant(dropRegion.id)
        ) {
          let insertIndex = (dropRegion.children || []).length;
          if (scopedMeta?.hostElement) {
            if (scopedMeta.childIds.length > 0) {
              const scopedInsert = dragDropManager.calculateFlexInsertPosition(
                scopedMeta.hostElement,
                upEvent,
                "column",
              );
              const scopedIndex =
                typeof scopedInsert.index === "number"
                  ? scopedInsert.index
                  : scopedMeta.childIds.length;
              insertIndex = mapScopedInsertIndex(dropRegion, scopedMeta.childIds, scopedIndex);
            } else {
              insertIndex = (dropRegion.children || []).length;
            }
          } else if (dropRegion.type && isFlexContainer(dropRegion.type)) {
            const outerElement = document.querySelector(`[data-node-id="${dropRegion.id}"]`);
            const contentElement =
              outerElement?.querySelector?.("[data-node-id]")?.parentElement || outerElement;
            if (contentElement) {
              const direction = resolveFlexDirection(dropRegion.type, contentElement);
              const insertInfo = dragDropManager.calculateFlexInsertPosition(
                contentElement,
                upEvent,
                direction,
              );
              insertIndex = insertInfo.index;
            }
          }
          const moveCommand = new MoveNodeCommand(currentNode.id, dropRegion.id, insertIndex);
          const shouldResetSize =
            currentNode.positioning === "absolute" || currentNode.layoutItem?.free;
          if (dropRegion?.type === "FreeContainer") {
            const containerEl = document.querySelector(`[data-node-id="${dropRegion.id}"]`);
            const containerRect = containerEl?.getBoundingClientRect?.();
            const nextX = containerRect ? (upEvent.clientX - containerRect.left) / zoomValue : 0;
            const nextY = containerRect ? (upEvent.clientY - containerRect.top) / zoomValue : 0;
            const nextAbs = {
              x: Math.round(nextX),
              y: Math.round(nextY),
              w: baseLayout.w,
              h: baseLayout.h,
              z: baseLayout.z,
            };
            const nextLayoutItem = {
              ...(currentNode.layoutItem || {}),
              free: { mode: "abs" as const, abs: { ...nextAbs } },
            };
            const nextProps = resolveScopedSlotProps(currentNode.props || {}, "FreeContainer");
            const updateCommand = createUpdateCommand(currentNode.id, {
              positioning: "absolute",
              absolutePos: nextAbs,
              layoutItem: nextLayoutItem,
              props: nextProps,
            });
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction?.(moveCommand);
              history.value.executeInTransaction?.(updateCommand);
            } else if (history.value?.execute) {
              history.value.execute(moveCommand);
              history.value.execute(updateCommand);
            } else if (doc.value?._moveNode && doc.value?._updateNode) {
              doc.value._moveNode(currentNode.id, dropRegion.id, insertIndex);
              doc.value._updateNode(currentNode.id, {
                positioning: "absolute",
                absolutePos: nextAbs,
                layoutItem: nextLayoutItem,
                props: nextProps,
              });
            }
            return;
          }
          const updatePatch: NodePatchLike = {
            positioning: "flow",
            absolutePos: undefined,
            flowLayout: undefined,
            layoutItem: undefined,
          };
          if (dropRegion?.type === "ElCol") {
            updatePatch.style = {
              ...(buildFlowResetStyle(currentNode.style) || {}),
              width: "100%",
              height: isContainer.value ? "100%" : "auto",
            };
          } else if (dropRegion?.type === "Tabs") {
            updatePatch.style = {
              ...(buildFlowResetStyle(currentNode.style) || {}),
              width: "100%",
              height: isContainer.value ? "100%" : "auto",
            };
            updatePatch.props = resolveScopedSlotProps(currentNode.props || {}, "Tabs", scopedKey);
          } else if (dropRegion?.type === "Collapse") {
            updatePatch.style = {
              ...(buildFlowResetStyle(currentNode.style) || {}),
              width: "100%",
              height: isContainer.value ? "100%" : "auto",
            };
            updatePatch.props = resolveScopedSlotProps(
              currentNode.props || {},
              "Collapse",
              scopedKey,
            );
          } else if (shouldResetSize) {
            updatePatch.style = buildFlowResetStyle(currentNode.style);
          }
          if (!updatePatch.props) {
            updatePatch.props = resolveScopedSlotProps(
              currentNode.props || {},
              dropRegion?.type,
              scopedKey,
            );
          }
          const updateCommand = createUpdateCommand(currentNode.id, updatePatch);
          if (history.value?.isInTransaction?.()) {
            history.value.executeInTransaction?.(moveCommand);
            history.value.executeInTransaction?.(updateCommand);
          } else if (history.value?.execute) {
            history.value.execute(moveCommand);
            history.value.execute(updateCommand);
          } else if (doc.value?._moveNode && doc.value?._updateNode) {
            doc.value._moveNode(currentNode.id, dropRegion.id, insertIndex);
            doc.value._updateNode(currentNode.id, updatePatch);
          }
        }
        if (restrictToContainer && dragContainer && rootNodeId) {
          const containerEl = document.querySelector(`[data-node-id="${dragContainer.id}"]`);
          const containerRect = containerEl?.getBoundingClientRect?.();
          const isInsideContainer = containerRect
            ? upEvent.clientX >= containerRect.left &&
              upEvent.clientX <= containerRect.right &&
              upEvent.clientY >= containerRect.top &&
              upEvent.clientY <= containerRect.bottom
            : false;
          if (!isInsideContainer) {
            const rootEl = document.querySelector(`[data-node-id="${rootNodeId}"]`);
            const rootRect = rootEl?.getBoundingClientRect?.();
            const nextX = rootRect ? (upEvent.clientX - rootRect.left) / zoomValue : 0;
            const nextY = rootRect ? (upEvent.clientY - rootRect.top) / zoomValue : 0;
            const nextAbs = {
              x: Math.round(nextX),
              y: Math.round(nextY),
              w: baseLayout.w,
              h: baseLayout.h,
              z: baseLayout.z,
            };
            if (node.value.type === "ElLayout") {
              const layoutMin = getDefaultSize("ElLayout");
              if (layoutMin?.width) {
                nextAbs.w = Math.max(nextAbs.w, layoutMin.width);
              }
            }
            const nextLayoutItem = {
              ...(currentNode.layoutItem || {}),
              free: {
                mode: "abs" as const,
                abs: { ...nextAbs },
              },
            };
            const rootNode = doc.value?.getNode?.(rootNodeId);
            const insertIndex = rootNode?.children?.length ?? 0;
            const moveCommand = new MoveNodeCommand(currentNode.id, rootNodeId, insertIndex);
            const updateCommand = createUpdateCommand(currentNode.id, {
              positioning: "absolute",
              absolutePos: nextAbs,
              layoutItem: nextLayoutItem,
              props: resolveScopedSlotProps(currentNode.props || {}, null),
            });
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction?.(moveCommand);
              history.value.executeInTransaction?.(updateCommand);
            } else if (history.value?.execute) {
              history.value.execute(moveCommand);
              history.value.execute(updateCommand);
            } else if (doc.value?._moveNode && doc.value?._updateNode) {
              doc.value._moveNode(currentNode.id, rootNodeId, insertIndex);
              doc.value._updateNode(currentNode.id, {
                positioning: "absolute",
                absolutePos: nextAbs,
                layoutItem: nextLayoutItem,
                props: resolveScopedSlotProps(currentNode.props || {}, null),
              });
            }
          }
        }
        if (
          hasMoved &&
          (isRegionParent || allowRegionMoveOut) &&
          originContainer?.type === "ElContainer" &&
          rootNodeId &&
          upEvent &&
          !restrictToContainer
        ) {
          const containerEl = document.querySelector(`[data-node-id="${originContainer.id}"]`);
          const containerRect = containerEl?.getBoundingClientRect?.();
          const isInsideContainer = containerRect
            ? upEvent.clientX >= containerRect.left &&
              upEvent.clientX <= containerRect.right &&
              upEvent.clientY >= containerRect.top &&
              upEvent.clientY <= containerRect.bottom
            : (() => {
                const hit = document.elementFromPoint(upEvent.clientX, upEvent.clientY);
                return Boolean(containerEl && hit && containerEl.contains(hit));
              })();
          if (!isInsideContainer) {
            const rootEl = document.querySelector(`[data-node-id="${rootNodeId}"]`);
            const rootRect = rootEl?.getBoundingClientRect?.();
            const nextX = rootRect ? (upEvent.clientX - rootRect.left) / zoomValue : 0;
            const nextY = rootRect ? (upEvent.clientY - rootRect.top) / zoomValue : 0;
            const nextAbs = {
              x: Math.round(nextX),
              y: Math.round(nextY),
              w: baseLayout.w,
              h: baseLayout.h,
              z: baseLayout.z,
            };
            if (currentNode.type === "ElLayout") {
              const layoutMin = getDefaultSize("ElLayout");
              if (layoutMin?.width) {
                nextAbs.w = Math.max(nextAbs.w, layoutMin.width);
              }
            }
            if (currentNode.type === "ElContainer") {
              const minSize = resolveElContainerMinSize(currentNode, doc.value ?? null);
              if (minSize) {
                if (Number.isFinite(minSize.width) && nextAbs.w < minSize.width) {
                  nextAbs.w = minSize.width;
                }
                if (Number.isFinite(minSize.height) && nextAbs.h < minSize.height) {
                  nextAbs.h = minSize.height;
                }
              }
            }
            const nextLayoutItem = {
              ...(currentNode.layoutItem || {}),
              free: {
                mode: "abs" as const,
                abs: { ...nextAbs },
              },
            };
            const rootNode = doc.value?.getNode?.(rootNodeId);
            const insertIndex = rootNode?.children?.length ?? 0;
            const moveCommand = new MoveNodeCommand(currentNode.id, rootNodeId, insertIndex);
            const updateCommand = createUpdateCommand(currentNode.id, {
              positioning: "absolute",
              absolutePos: nextAbs,
              layoutItem: nextLayoutItem,
              props: resolveScopedSlotProps(currentNode.props || {}, null),
            });
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction?.(moveCommand);
              history.value.executeInTransaction?.(updateCommand);
            } else if (history.value?.execute) {
              history.value.execute(moveCommand);
              history.value.execute(updateCommand);
            } else if (doc.value?._moveNode && doc.value?._updateNode) {
              doc.value._moveNode(currentNode.id, rootNodeId, insertIndex);
              doc.value._updateNode(currentNode.id, {
                positioning: "absolute",
                absolutePos: nextAbs,
                layoutItem: nextLayoutItem,
                props: resolveScopedSlotProps(currentNode.props || {}, null),
              });
            }
          }
        }
      } finally {
        endDrag();
        clearDropTarget();
        showInsertLine.value = false;
        insertLineStyle.value = null;
        genericInsertLineBox.value = null;
        rowInsertInfo.value = null;
        layoutInsertInfo.value = null;
        cleanupDragHandlers();
        if (history.value?.isInTransaction?.()) {
          if (transactionMode === "rollback") {
            history.value.rollbackTransaction?.();
          } else if (transactionMode === "commit") {
            history.value.commitTransaction?.(transactionLabel);
          }
        }
        if (shouldReselectDraggedNode && draggedNodeId && selection.value && !readonly.value) {
          selection.value.select(createSelectableElement("node", draggedNodeId));
        }
        if (typeof window !== "undefined") {
          window.dispatchEvent(new window.CustomEvent("designer:node-transform-end"));
        }
      }
    };

    activeDragHandlers = {
      move,
      up,
      userSelect: originalUserSelect,
      pointerTarget,
      pointerId: event.pointerId,
      usePointer,
    };

    if (usePointer) {
      document.addEventListener("pointermove", move);
      document.addEventListener("pointerup", up);
      document.addEventListener("pointercancel", up);
    } else {
      document.addEventListener("mousemove", move);
      document.addEventListener("mouseup", up);
    }
  };

  return { handlePointerDown };
}
