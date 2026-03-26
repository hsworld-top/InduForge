/**
 * 拖拽管理器（DragDropManager）
 *
 * 职责：
 * - 查找拖拽目标容器（elementFromPoint + data-node-id）
 * - 计算 Flex 容器内的插入位置（before/after/inside）
 * - 管理当前放置目标与插入指示线
 *
 * 用于：组件从物料面板拖入画布、画布内组件拖拽排序
 */

import type { DocumentModel } from "@/editor-core/document/DocumentModel";
import type { PageNode } from "@/editor-core/document/types";
import { getChildPositioning, isContainerType } from "@/components/descriptors/registry";
import {
  clampPositionInContainer,
  eventToCanvasPosition,
} from "@/editor-core/utils/placement-utils";

export type FlowContainerKind = "flex" | "grid" | "free";

export interface DropTargetResolution {
  parentId: string | null;
  index: number;
  dropPosition: { x: number; y: number } | null;
  containerType: FlowContainerKind | null;
}

export interface FlexInsertLine {
  orientation: "horizontal" | "vertical";
  offset: number;
}

export interface FlexInsertResult {
  index: number;
  position: "before" | "after" | "inside";
  insertLine: FlexInsertLine | null;
}

export interface DropTargetLookup {
  element: HTMLElement | null;
  nodeId: string | null;
  isContainer: boolean;
}

export interface GridCellHint {
  orientation: string;
  row: number;
  col: number;
  highlightRect: { left: number; top: number; width: number; height: number };
}

export type InsertRuleKind = "before_after" | "grid_cell" | "absolute_position";

const GRID_REPEAT_HEAD_RE = /^repeat\s*\(\s*/i;
const GRID_DIGIT_RE = /\d/;
const GRID_WHITESPACE_RE = /\s/;
const GRID_TEMPLATE_SPLIT_RE = /\s+/;

export interface DropDecision {
  containerType: "flex" | "grid" | "free";
  positioning: "absolute" | "flow";
  /** 插入线、网格高亮区或绝对坐标等，随 insertRule 变化 */
  visualHint: Record<string, unknown>;
  insertRule: InsertRuleKind;
}

export class DragDropManager {
  currentDropTarget: HTMLElement | null = null;
  insertIndex = -1;
  insertPosition: "before" | "after" | "inside" = "inside";

  resolveDropTarget(
    event: MouseEvent | DragEvent,
    canvasRoot: HTMLElement,
    doc: DocumentModel,
    currentPage: PageNode,
    zoom = 1,
  ): DropTargetResolution {
    if (!event || !canvasRoot || !doc || !currentPage) {
      return {
        parentId: null,
        index: -1,
        dropPosition: null,
        containerType: null,
      };
    }

    const point = { x: event.clientX, y: event.clientY };
    const element = document.elementFromPoint(point.x, point.y);
    if (!element) {
      const rootId = currentPage.rootNodeId;
      if (rootId) {
        const rootNode = doc.getNode(rootId);
        if (rootNode) {
          const dropPos = eventToCanvasPosition(event, canvasRoot, zoom);
          return {
            parentId: rootId,
            index: (rootNode.children || []).length,
            dropPosition: dropPos,
            containerType: "free",
          };
        }
      }
      return {
        parentId: null,
        index: -1,
        dropPosition: null,
        containerType: null,
      };
    }

    let nodeElement = element.closest("[data-node-id]");
    if (!nodeElement || !canvasRoot.contains(nodeElement)) {
      const rootId = currentPage.rootNodeId;
      if (rootId) {
        const rootNode = doc.getNode(rootId);
        if (rootNode) {
          const dropPos = eventToCanvasPosition(event, canvasRoot, zoom);
          return {
            parentId: rootId,
            index: (rootNode.children || []).length,
            dropPosition: dropPos,
            containerType: "free",
          };
        }
      }
      return {
        parentId: null,
        index: -1,
        dropPosition: null,
        containerType: null,
      };
    }

    let currentNodeId = nodeElement.getAttribute("data-node-id");
    let currentNode = currentNodeId ? doc.getNode(currentNodeId) : null;
    let parentNode = currentNode && currentNodeId ? doc.getParent(currentNodeId) : null;

    while (currentNode) {
      const nodeType = currentNode.type;
      const container = isContainerType(nodeType);
      const childPositioning = getChildPositioning(nodeType);

      if (container) {
        if (childPositioning === "flow") {
          const direction = this.getContainerDirection(nodeElement as HTMLElement);
          const insertInfo = this.calculateFlexInsertPosition(
            nodeElement as HTMLElement,
            event,
            direction,
          );
          return {
            parentId: currentNode.id,
            index: insertInfo.index,
            dropPosition: null,
            containerType: "flex",
          };
        }
        const dropPos = eventToCanvasPosition(event, nodeElement as HTMLElement, zoom);
        const clampedPos = clampPositionInContainer(
          dropPos,
          nodeElement as HTMLElement,
          { width: 0, height: 0 },
          zoom,
        );
        return {
          parentId: currentNode.id,
          index: (currentNode.children || []).length,
          dropPosition: clampedPos,
          containerType: "free",
        };
      }

      if (parentNode) {
        currentNodeId = parentNode.id;
        currentNode = parentNode;
        nodeElement = document.querySelector(`[data-node-id="${currentNodeId}"]`);
        parentNode = doc.getParent(currentNodeId);
      } else {
        break;
      }
    }

    const rootId = currentPage.rootNodeId;
    if (rootId) {
      const rootNode = doc.getNode(rootId);
      if (rootNode) {
        const dropPos = eventToCanvasPosition(event, canvasRoot, zoom);
        return {
          parentId: rootId,
          index: (rootNode.children || []).length,
          dropPosition: dropPos,
          containerType: "free",
        };
      }
    }

    return {
      parentId: null,
      index: -1,
      dropPosition: null,
      containerType: null,
    };
  }

  findDropTarget(event: MouseEvent | DragEvent, canvasRoot: HTMLElement): DropTargetLookup {
    const point = { x: event.clientX, y: event.clientY };

    const element = document.elementFromPoint(point.x, point.y);
    if (!element) {
      return { element: null, nodeId: null, isContainer: false };
    }

    const nodeElement = element.closest("[data-node-id]");
    if (!nodeElement || !canvasRoot.contains(nodeElement)) {
      return { element: null, nodeId: null, isContainer: false };
    }

    const nodeId = nodeElement.getAttribute("data-node-id");
    const nodeType = nodeElement.getAttribute("data-node-type");
    const isContainer = isContainerType(nodeType || "");

    return {
      element: nodeElement as HTMLElement,
      nodeId,
      isContainer,
    };
  }

  calculateFlexInsertPosition(
    containerElement: HTMLElement,
    event: MouseEvent | DragEvent,
    direction = "column",
  ): FlexInsertResult {
    const children = Array.from(containerElement.children).filter((child) =>
      child.hasAttribute("data-node-id"),
    );

    if (children.length === 0) {
      return {
        index: 0,
        position: "inside",
        insertLine: null,
      };
    }

    const point = { x: event.clientX, y: event.clientY };
    const containerRect = containerElement.getBoundingClientRect();
    const isHorizontal = direction === "row" || direction === "row-reverse";

    for (let i = 0; i < children.length; i++) {
      const child = children[i]!;
      const rect = child.getBoundingClientRect();

      if (isHorizontal) {
        const midX = rect.left + rect.width / 2;
        if (point.x < midX) {
          return {
            index: i,
            position: "before",
            insertLine: {
              orientation: "vertical",
              offset: rect.left - containerRect.left,
            },
          };
        }
      } else {
        const midY = rect.top + rect.height / 2;
        if (point.y < midY) {
          return {
            index: i,
            position: "before",
            insertLine: {
              orientation: "horizontal",
              offset: rect.top - containerRect.top,
            },
          };
        }
      }
    }

    const lastChild = children.at(-1)!;
    const lastRect = lastChild.getBoundingClientRect();

    return {
      index: children.length,
      position: "after",
      insertLine: {
        orientation: isHorizontal ? "vertical" : "horizontal",
        offset: isHorizontal
          ? lastRect.right - containerRect.left
          : lastRect.bottom - containerRect.top,
      },
    };
  }

  calculateFreePosition(containerElement: HTMLElement, event: MouseEvent | DragEvent) {
    const rect = containerElement.getBoundingClientRect();
    return {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    };
  }

  getContainerDirection(containerElement: HTMLElement): string {
    const computedStyle = window.getComputedStyle(containerElement);
    return computedStyle.flexDirection || "column";
  }

  clear(): void {
    this.currentDropTarget = null;
    this.insertIndex = -1;
    this.insertPosition = "inside";
  }

  calculateDropDecision(event: DragEvent, targetElement: HTMLElement): DropDecision | null {
    if (!targetElement) return null;

    const nodeType = targetElement.getAttribute("data-node-type");

    switch (nodeType) {
      case "FlexContainer":
      case "HorizontalLayout":
      case "VerticalLayout":
      case "ResponsiveLayout": {
        const direction = this.getContainerDirection(targetElement);
        const insertInfo = this.calculateFlexInsertPosition(targetElement, event, direction);

        return {
          containerType: "flex",
          positioning: "flow",
          visualHint: {
            orientation: insertInfo.insertLine?.orientation || "horizontal",
            offset: insertInfo.insertLine?.offset || 0,
            index: insertInfo.index,
          },
          insertRule: "before_after",
        };
      }

      case "GridContainer":
      case "ColumnLayout1":
      case "ColumnLayout2":
      case "ColumnLayout4": {
        const gridInfo = this.calculateGridCell(targetElement, event);

        return {
          containerType: "grid",
          positioning: "flow",
          visualHint: { ...gridInfo },
          insertRule: "grid_cell",
        };
      }

      case "ElContainer":
      case "ElLayout":
      case "ElLayoutRow":
      case "ElHeader":
      case "ElAside":
      case "ElMain":
      case "ElFooter":
      case "ElCol": {
        const direction = this.getContainerDirection(targetElement);
        const insertInfo = this.calculateFlexInsertPosition(targetElement, event, direction);

        return {
          containerType: "flex",
          positioning: "flow",
          visualHint: {
            orientation: insertInfo.insertLine?.orientation || "horizontal",
            offset: insertInfo.insertLine?.offset || 0,
            index: insertInfo.index,
          },
          insertRule: "before_after",
        };
      }

      case "FreeContainer": {
        const position = this.calculateFreePosition(targetElement, event);

        return {
          containerType: "free",
          positioning: "absolute",
          visualHint: {
            x: position.x,
            y: position.y,
          },
          insertRule: "absolute_position",
        };
      }

      default:
        return null;
    }
  }

  calculateGridCell(containerElement: HTMLElement, event: DragEvent): GridCellHint {
    const gridStyle = window.getComputedStyle(containerElement);
    const rect = containerElement.getBoundingClientRect();

    const colsStr = gridStyle.gridTemplateColumns || "auto";
    const rowsStr = gridStyle.gridTemplateRows || "auto";

    const cols = parseGridTemplateParts(colsStr);
    const rows = parseGridTemplateParts(rowsStr);

    const colCount = cols.length || 3;
    const rowCount = rows.length || 3;

    const colWidth = rect.width / colCount;
    const rowHeight = rect.height / rowCount;

    const relativeX = event.clientX - rect.left;
    const relativeY = event.clientY - rect.top;

    const col = Math.max(0, Math.min(colCount - 1, Math.floor(relativeX / colWidth)));
    const row = Math.max(0, Math.min(rowCount - 1, Math.floor(relativeY / rowHeight)));

    return {
      orientation: "grid",
      row: row + 1,
      col: col + 1,
      highlightRect: {
        left: col * colWidth,
        top: row * rowHeight,
        width: colWidth,
        height: rowHeight,
      },
    };
  }
}

/** gridTemplateColumns/Rows 解析（供落点计算与单测） */
export function parseGridTemplateParts(template: string): string[] {
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

export function createDragDropManager(): DragDropManager {
  return new DragDropManager();
}

export default DragDropManager;
