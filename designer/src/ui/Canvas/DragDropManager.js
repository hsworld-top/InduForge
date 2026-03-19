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

import { isContainerType as getIsContainerType, getChildPositioning } from "@/components/registry.js";
import { eventToCanvasPosition, clampPositionInContainer } from "@/editor-core/utils/placementUtils.js";

export class DragDropManager {
  constructor() {
    /** @type {HTMLElement | null} */
    this.currentDropTarget = null;
    /** @type {number} */
    this.insertIndex = -1;
    /** @type {'before' | 'after' | 'inside'} */
    this.insertPosition = "inside";
  }

  /**
   * 统一放置解析：根据鼠标位置确定放置目标、插入索引、落点坐标
   * @param {MouseEvent | DragEvent} event - 拖拽事件
   * @param {HTMLElement} canvasRoot - 画布根元素
   * @param {import('@/editor-core').DocumentModel} doc - 文档模型
   * @param {import('@/editor-core').PageNode} currentPage - 当前页面
   * @param {number} [zoom=1] - 画布缩放比例
   * @returns {{ parentId: string | null, index: number, dropPosition: {x: number, y: number} | null, containerType: 'flex' | 'grid' | 'free' | null }}
   */
  resolveDropTarget(event, canvasRoot, doc, currentPage, zoom = 1) {
    if (!event || !canvasRoot || !doc || !currentPage) {
      return { parentId: null, index: -1, dropPosition: null, containerType: null };
    }

    const point = { x: event.clientX, y: event.clientY };
    const element = document.elementFromPoint(point.x, point.y);
    if (!element) {
      // 未命中任何元素，放置到页面根
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
      return { parentId: null, index: -1, dropPosition: null, containerType: null };
    }

    // 向上查找最近的节点元素
    let nodeElement = element.closest("[data-node-id]");
    if (!nodeElement || !canvasRoot.contains(nodeElement)) {
      // 未命中节点，放置到页面根
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
      return { parentId: null, index: -1, dropPosition: null, containerType: null };
    }

    // 向上遍历直到找到可接受的容器
    let currentNodeId = nodeElement.getAttribute("data-node-id");
    let currentNode = doc.getNode(currentNodeId);
    let parentNode = currentNode ? doc.getParent(currentNodeId) : null;

    while (currentNode) {
      const nodeType = currentNode.type;
      const isContainer = getIsContainerType(nodeType);
      const childPositioning = getChildPositioning(nodeType);

      if (isContainer) {
        // 找到容器，根据容器类型决定 flow/absolute
        if (childPositioning === "flow") {
          // 流式布局：计算插入索引
          const direction = this.getContainerDirection(nodeElement);
          const insertInfo = this.calculateFlexInsertPosition(nodeElement, event, direction);
          return {
            parentId: currentNode.id,
            index: insertInfo.index,
            dropPosition: null,
            containerType: direction === "row" || direction === "row-reverse" ? "flex" : "flex",
          };
        } else {
          // 绝对定位：计算落点坐标
          const dropPos = eventToCanvasPosition(event, nodeElement, zoom);
          const clampedPos = clampPositionInContainer(dropPos, nodeElement, { width: 0, height: 0 }, zoom);
          return {
            parentId: currentNode.id,
            index: (currentNode.children || []).length,
            dropPosition: clampedPos,
            containerType: "free",
          };
        }
      }

      // 继续向上查找
      if (parentNode) {
        currentNodeId = parentNode.id;
        currentNode = parentNode;
        nodeElement = document.querySelector(`[data-node-id="${currentNodeId}"]`);
        parentNode = doc.getParent(currentNodeId);
      } else {
        break;
      }
    }

    // 未找到容器，放置到页面根
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

    return { parentId: null, index: -1, dropPosition: null, containerType: null };
  }

  /**
   * 查找拖拽目标容器节点
   * @param {MouseEvent | DragEvent} event - 拖拽事件
   * @param {HTMLElement} canvasRoot - 画布根元素
   * @returns {{ element: HTMLElement | null, nodeId: string | null, isContainer: boolean }}
   */
  findDropTarget(event, canvasRoot) {
    const point = { x: event.clientX, y: event.clientY };

    // 获取鼠标位置下的元素
    const element = document.elementFromPoint(point.x, point.y);
    if (!element) {
      return { element: null, nodeId: null, isContainer: false };
    }

    // 向上查找最近的节点元素
    const nodeElement = element.closest("[data-node-id]");
    if (!nodeElement || !canvasRoot.contains(nodeElement)) {
      return { element: null, nodeId: null, isContainer: false };
    }

    const nodeId = nodeElement.getAttribute("data-node-id");
    const nodeType = nodeElement.getAttribute("data-node-type");
    const isContainer = getIsContainerType(nodeType);

    return {
      element: nodeElement,
      nodeId,
      isContainer,
    };
  }

  /**
   * 计算 Flex 容器内的插入位置
   * @param {HTMLElement} containerElement - 容器元素
   * @param {MouseEvent | DragEvent} event - 拖拽事件
   * @param {string} direction - Flex 方向 (row | column)
   * @returns {{ index: number, position: 'before' | 'after', insertLine: { orientation: 'horizontal' | 'vertical', offset: number } }}
   */
  calculateFlexInsertPosition(containerElement, event, direction = "column") {
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

    // 遍历子元素找到最近的插入位置
    for (let i = 0; i < children.length; i++) {
      const child = children[i];
      const rect = child.getBoundingClientRect();

      if (isHorizontal) {
        // 水平布局:比较 X 坐标
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
        // 垂直布局:比较 Y 坐标
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

    // 如果没有匹配,插入到末尾
    const lastChild = children[children.length - 1];
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

  /**
   * 计算自由容器内的绝对位置
   * @param {HTMLElement} containerElement - 容器元素
   * @param {MouseEvent | DragEvent} event - 拖拽事件
   * @returns {{ x: number, y: number }}
   */
  calculateFreePosition(containerElement, event) {
    const rect = containerElement.getBoundingClientRect();
    return {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    };
  }

  /**
   * 获取容器的布局方向
   * @param {HTMLElement} containerElement - 容器元素
   * @returns {string} 布局方向
   */
  getContainerDirection(containerElement) {
    const computedStyle = window.getComputedStyle(containerElement);
    return computedStyle.flexDirection || "column";
  }

  /**
   * 清除当前拖拽状态
   */
  clear() {
    this.currentDropTarget = null;
    this.insertIndex = -1;
    this.insertPosition = "inside";
  }

  /**
   * 识别目标容器类型并返回布局决策
   * 这是核心的布局感知方法，根据容器类型返回不同的插入策略和视觉提示
   * @param {DragEvent} event - 拖拽事件
   * @param {HTMLElement} targetElement - 目标元素
   * @returns {DropDecision | null} 布局决策
   *
   * @typedef {Object} DropDecision
   * @property {'flex' | 'grid' | 'free'} containerType - 容器类型
   * @property {'absolute' | 'flow'} positioning - 定位模式
   * @property {Object} visualHint - 视觉提示信息
   * @property {'before_after' | 'grid_cell' | 'absolute_position'} insertRule - 插入规则
   */
  calculateDropDecision(event, targetElement) {
    if (!targetElement) return null;

    const nodeType = targetElement.getAttribute("data-node-type");

    switch (nodeType) {
      case "FlexContainer":
      case "HorizontalLayout":
      case "VerticalLayout":
      case "ResponsiveLayout": {
        // Flex 容器：子节点使用流式布局
        const direction = this.getContainerDirection(targetElement);
        const insertInfo = this.calculateFlexInsertPosition(
          targetElement,
          event,
          direction,
        );

        return {
          containerType: "flex",
          positioning: "flow",
          visualHint: {
            orientation: insertInfo.insertLine?.orientation || "horizontal",
            offset: insertInfo.insertLine?.offset || 0,
            index: insertInfo.index,
          },
          insertRule: "before_after", // 显示主轴方向提示
        };
      }

      case "GridContainer":
      case "ColumnLayout1":
      case "ColumnLayout2":
      case "ColumnLayout4": {
        // Grid 容器：子节点使用流式布局
        const gridInfo = this.calculateGridCell(targetElement, event);

        return {
          containerType: "grid",
          positioning: "flow",
          visualHint: gridInfo,
          insertRule: "grid_cell", // 显示单元格高亮
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
        const insertInfo = this.calculateFlexInsertPosition(
          targetElement,
          event,
          direction,
        );

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
        // 自由容器：子节点使用绝对定位
        const position = this.calculateFreePosition(targetElement, event);

        return {
          containerType: "free",
          positioning: "absolute",
          visualHint: {
            x: position.x,
            y: position.y,
          },
          insertRule: "absolute_position", // 只允许绝对定位
        };
      }

      default:
        return null;
    }
  }

  /**
   * 计算 Grid 单元格位置
   * @param {HTMLElement} containerElement - Grid 容器元素
   * @param {DragEvent} event - 拖拽事件
   * @returns {{orientation: string, row: number, col: number, highlightRect: {left: number, top: number, width: number, height: number}}}
   */
  calculateGridCell(containerElement, event) {
    const gridStyle = window.getComputedStyle(containerElement);
    const rect = containerElement.getBoundingClientRect();

    // 解析 grid-template-columns
    const colsStr = gridStyle.gridTemplateColumns || "auto";
    const rowsStr = gridStyle.gridTemplateRows || "auto";

    const cols = this._parseGridTemplate(colsStr);
    const rows = this._parseGridTemplate(rowsStr);

    const colCount = cols.length || 3; // 默认 3 列
    const rowCount = rows.length || 3; // 默认 3 行

    const colWidth = rect.width / colCount;
    const rowHeight = rect.height / rowCount;

    const relativeX = event.clientX - rect.left;
    const relativeY = event.clientY - rect.top;

    const col = Math.max(
      0,
      Math.min(colCount - 1, Math.floor(relativeX / colWidth)),
    );
    const row = Math.max(
      0,
      Math.min(rowCount - 1, Math.floor(relativeY / rowHeight)),
    );

    return {
      orientation: "grid",
      row: row + 1, // Grid 行列从 1 开始
      col: col + 1,
      highlightRect: {
        left: col * colWidth,
        top: row * rowHeight,
        width: colWidth,
        height: rowHeight,
      },
    };
  }

  /**
   * 解析 Grid 模板（简化版）
   * @param {string} template - Grid 模板字符串
   * @returns {string[]} 模板数组
   * @private
   */
  _parseGridTemplate(template) {
    if (!template || template === "none") return [];

    // 处理 repeat() 函数
    const repeatMatch = template.match(/repeat\((\d+),\s*([^)]+)\)/i);
    if (repeatMatch) {
      const count = parseInt(repeatMatch[1], 10);
      const value = repeatMatch[2].trim();
      return Array(count).fill(value);
    }

    // 按空格分割
    return template.split(/\s+/).filter(Boolean);
  }
}

/**
 * 创建拖拽管理器实例
 * @returns {DragDropManager}
 */
export function createDragDropManager() {
  return new DragDropManager();
}

export default DragDropManager;
