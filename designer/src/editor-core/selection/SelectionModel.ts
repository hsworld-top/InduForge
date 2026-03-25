/**
 * SelectionModel - 选中状态管理
 * 支持混合选择（DOM 节点 + Canvas 图形）
 *
 * 设计原则：
 * - 支持混合选择：同时选中节点和图形
 * - 支持多选：Ctrl+单击切换、Shift+范围选择
 * - 支持绘图工具状态管理
 * - 提供事件通知
 */

import type { DocumentModel } from "../document/DocumentModel";
import type {
  ComponentNode,
  DrawingTool,
  GraphicNode,
  SelectableElement,
  SelectionState,
} from "../document/types.ts";
import { EventEmitter } from "../utils/EventEmitter";

/**
 * 选中状态管理类
 */
export class SelectionModel extends EventEmitter {
  _doc: DocumentModel | null;
  _selectedElements: SelectableElement[];
  _hoveredElement: SelectableElement | null;
  _dropTargetId: string | null;
  _anchorElement: SelectableElement | null;
  _activeTool: DrawingTool | null;

  /**
   * 创建选中状态管理器
   * @param {DocumentModel} [doc] - 文档模型（可选，用于范围选择等操作）
   */
  constructor(doc?: DocumentModel | null) {
    super();

    this._doc = doc ?? null;

    this._selectedElements = [];

    this._hoveredElement = null;

    this._dropTargetId = null;

    this._anchorElement = null;

    this._activeTool = "select";
  }

  // ==================== 状态查询 ====================

  /**
   * 获取选中的元素列表
   * @returns {SelectableElement[]}
   */
  getSelectedElements() {
    return [...this._selectedElements];
  }

  /**
   * 获取选中的节点
   * @returns {ComponentNode[]}
   */
  getSelectedNodes() {
    const doc = this._doc;
    if (!doc) return [];
    return this._selectedElements
      .filter((e) => e.kind === "node")
      .map((e) => doc.getNode(e.id))
      .filter((n) => n !== null);
  }

  /**
   * 获取选中的图形
   * @returns {GraphicNode[]}
   */
  getSelectedGraphics() {
    const doc = this._doc;
    if (!doc) return [];
    return this._selectedElements
      .filter((e) => e.kind === "graphic")
      .map((e) => doc.getGraphic(e.id))
      .filter((g) => g !== null);
  }

  /**
   * 获取选中的节点 ID 列表
   * @returns {string[]}
   */
  getSelectedNodeIds() {
    return this._selectedElements.filter((e) => e.kind === "node").map((e) => e.id);
  }

  /**
   * 获取选中的图形 ID 列表
   * @returns {string[]}
   */
  getSelectedGraphicIds() {
    return this._selectedElements.filter((e) => e.kind === "graphic").map((e) => e.id);
  }

  /**
   * 获取主选中元素（最后选中的）
   * @returns {ComponentNode | GraphicNode | null}
   */
  getPrimarySelection(): ComponentNode | GraphicNode | null {
    if (this._selectedElements.length === 0) return null;
    const primary = this._selectedElements.at(-1)!;
    const doc = this._doc;
    if (!doc) return null;

    if (primary.kind === "node") {
      return doc.getNode(primary.id);
    }
    return doc.getGraphic(primary.id);
  }

  /**
   * 获取主选中元素的元数据
   * @returns {SelectableElement | null}
   */
  getPrimaryElement() {
    if (this._selectedElements.length === 0) return null;
    return this._selectedElements.at(-1);
  }

  /**
   * 获取选择类型
   * @returns {'nodes' | 'graphics' | 'mixed' | 'none'}
   */
  getSelectionType() {
    if (this._selectedElements.length === 0) return "none";

    const hasNodes = this._selectedElements.some((e) => e.kind === "node");
    const hasGraphics = this._selectedElements.some((e) => e.kind === "graphic");

    if (hasNodes && hasGraphics) return "mixed";
    if (hasNodes) return "nodes";
    if (hasGraphics) return "graphics";
    return "none";
  }

  /**
   * 获取选中数量
   * @returns {number}
   */
  getSelectionCount() {
    return this._selectedElements.length;
  }

  /**
   * 检查元素是否被选中
   * @param {string} id - 元素 ID
   * @returns {boolean}
   */
  isSelected(id: string) {
    return this._selectedElements.some((e) => e.id === id);
  }

  /**
   * 检查是否有选中元素
   * @returns {boolean}
   */
  hasSelection() {
    return this._selectedElements.length > 0;
  }

  /**
   * 获取当前 hover 的元素
   * @returns {SelectableElement | null}
   */
  getHoveredElement() {
    return this._hoveredElement;
  }

  /**
   * 获取当前拖拽目标 ID
   * @returns {string | null}
   */
  getDropTargetId() {
    return this._dropTargetId;
  }

  /**
   * 获取当前绘图工具
   * @returns {DrawingTool | null}
   */
  getActiveTool() {
    return this._activeTool;
  }

  // ==================== 选择操作 ====================

  /**
   * 选择元素（清除其他选中）
   * @param {SelectableElement} element - 元素
   */
  select(element: SelectableElement) {
    this._selectedElements = [element];
    this._anchorElement = element;
    this._emitChange();
  }

  /**
   * 通过 ID 选择元素（自动判断类型）
   * @param {string} id - 元素 ID
   */
  selectById(id: string) {
    let kind: SelectableElement["kind"] = "node";
    if (this._doc?.getGraphic(id)) {
      kind = "graphic";
    }
    this.select({ kind, id });
  }

  /**
   * 切换元素选中状态（Ctrl+单击）
   * @param {SelectableElement} element - 元素
   */
  toggleSelect(element: SelectableElement) {
    const index = this._selectedElements.findIndex(
      (e) => e.kind === element.kind && e.id === element.id,
    );

    if (index !== -1) {
      // 已选中，取消选中
      this._selectedElements.splice(index, 1);
    } else {
      // 未选中，添加到选中
      this._selectedElements.push(element);
      this._anchorElement = element;
    }

    this._emitChange();
  }

  /**
   * 添加到选中（不清除已选）
   * @param {SelectableElement} element - 元素
   */
  addToSelection(element: SelectableElement) {
    if (!this.isSelected(element.id)) {
      this._selectedElements.push(element);
      this._anchorElement = element;
      this._emitChange();
    }
  }

  /**
   * 从选中中移除
   * @param {string} id - 元素 ID
   */
  removeFromSelection(id: string) {
    const index = this._selectedElements.findIndex((e) => e.id === id);
    if (index !== -1) {
      this._selectedElements.splice(index, 1);
      this._emitChange();
    }
  }

  /**
   * 范围选择（Shift+单击）
   * @param {SelectableElement} element - 目标元素
   */
  selectRange(element: SelectableElement) {
    // 简化实现：如果没有锚点，直接选择目标
    if (!this._anchorElement) {
      this.select(element);
      return;
    }

    // 如果类型不同，只选择目标
    if (this._anchorElement.kind !== element.kind) {
      this.select(element);
      return;
    }

    // TODO: 实现基于文档顺序的范围选择
    // 目前简化为添加到选中
    if (!this.isSelected(element.id)) {
      this._selectedElements.push(element);
    }

    this._emitChange();
  }

  /**
   * 选择多个节点
   * @param {string[]} nodeIds - 节点 ID 列表
   */
  selectNodes(nodeIds: string[]) {
    this._selectedElements = nodeIds.map((id) => ({
      kind: "node" as const,
      id,
    }));
    this._anchorElement = this._selectedElements.at(-1) ?? null;
    this._emitChange();
  }

  /**
   * 选择多个图形
   * @param {string[]} graphicIds - 图形 ID 列表
   */
  selectGraphics(graphicIds: string[]) {
    this._selectedElements = graphicIds.map((id) => ({
      kind: "graphic" as const,
      id,
    }));
    this._anchorElement = this._selectedElements.at(-1) ?? null;
    this._emitChange();
  }

  /**
   * 选择多个元素
   * @param {SelectableElement[]} elements - 元素列表
   */
  selectMultiple(elements: SelectableElement[]) {
    this._selectedElements = [...elements];
    this._anchorElement = this._selectedElements.at(-1) ?? null;
    this._emitChange();
  }

  /**
   * 全选当前页面所有元素
   * @param {string} pageId - 页面 ID
   */
  selectAll(pageId: string) {
    if (!this._doc) return;

    const page = this._doc.getPage(pageId);
    if (!page) return;

    const elements: SelectableElement[] = [];

    // 获取页面根节点下的所有节点
    const rootNode = this._doc.getNode(page.rootNodeId);
    if (rootNode && rootNode.children) {
      for (const childId of rootNode.children) {
        elements.push({ kind: "node", id: childId });
      }
    }

    // 获取页面的所有图形
    if (page.graphicsIds) {
      for (const graphicId of page.graphicsIds) {
        elements.push({ kind: "graphic", id: graphicId });
      }
    }

    this._selectedElements = elements;
    this._emitChange();
  }

  /**
   * 仅保留节点选中（移除图形选中）
   */
  selectOnlyNodes() {
    this._selectedElements = this._selectedElements.filter((e) => e.kind === "node");
    this._emitChange();
  }

  /**
   * 仅保留图形选中（移除节点选中）
   */
  selectOnlyGraphics() {
    this._selectedElements = this._selectedElements.filter((e) => e.kind === "graphic");
    this._emitChange();
  }

  /**
   * 清除所有选中
   */
  clearSelection() {
    if (this._selectedElements.length === 0) return;

    this._selectedElements = [];
    this._anchorElement = null;
    this._emitChange();
  }

  // ==================== Hover 操作 ====================

  /**
   * 设置 hover 元素
   * @param {SelectableElement | null} element - 元素
   */
  setHover(element: SelectableElement | null) {
    if (this._hoveredElement?.kind === element?.kind && this._hoveredElement?.id === element?.id) {
      return;
    }

    this._hoveredElement = element;
    this.emit("hoverChange", element);
  }

  /**
   * 通过 ID 设置 hover
   * @param {string | null} id - 元素 ID
   */
  setHoverById(id: string) {
    if (!id) {
      this.setHover(null);
      return;
    }

    let kind: SelectableElement["kind"] = "node";
    if (this._doc?.getGraphic(id)) {
      kind = "graphic";
    }
    this.setHover({ kind, id });
  }

  /**
   * 清除 hover
   */
  clearHover() {
    this.setHover(null);
  }

  // ==================== 拖拽目标 ====================

  /**
   * 设置拖拽目标
   * @param {string | null} nodeId - 目标节点 ID
   */
  setDropTarget(nodeId: string | null) {
    if (this._dropTargetId === nodeId) return;

    this._dropTargetId = nodeId;
    this.emit("dropTargetChange", nodeId);
  }

  /**
   * 清除拖拽目标
   */
  clearDropTarget() {
    this.setDropTarget(null);
  }

  // ==================== 绘图工具 ====================

  /**
   * 设置当前绘图工具
   * @param {DrawingTool | null} tool - 工具类型
   */
  setActiveTool(tool: DrawingTool | null) {
    if (this._activeTool === tool) return;

    const oldTool = this._activeTool;
    this._activeTool = tool;

    // 切换到绘图工具时清除选中
    if (tool && !["select", "marquee", "pan"].includes(tool)) {
      this.clearSelection();
    }

    this.emit("toolChange", { tool, oldTool });
  }

  /**
   * 重置为选择工具
   */
  resetToSelectTool() {
    this.setActiveTool("select");
  }

  /**
   * 是否是选择工具
   * @returns {boolean}
   */
  isSelectTool() {
    return this._activeTool === "select" || this._activeTool === "marquee";
  }

  /**
   * 是否是绘图工具
   * @returns {boolean}
   */
  isDrawingTool() {
    const drawingTools = ["line", "rect", "circle", "ellipse", "polygon", "pipe", "text"];
    return this._activeTool !== null && drawingTools.includes(this._activeTool);
  }

  // ==================== 便捷方法 ====================

  /**
   * 获取选中元素的包围盒
   * @returns {{x: number, y: number, width: number, height: number} | null}
   */
  getSelectionBounds() {
    if (!this._doc || this._selectedElements.length === 0) return null;

    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;

    for (const element of this._selectedElements) {
      let bounds: {
        x: number;
        y: number;
        width: number;
        height: number;
      } | null = null;

      if (element.kind === "node") {
        const node = this._doc.getNode(element.id);
        if (node && node.style) {
          const st = node.style;
          bounds = {
            x: Number(st.left) || 0,
            y: Number(st.top) || 0,
            width: Number(st.width) || 100,
            height: Number(st.height) || 100,
          };
        }
      } else {
        const graphic = this._doc.getGraphic(element.id);
        if (graphic && graphic.props) {
          bounds = this._getGraphicBounds(graphic);
        }
      }

      if (bounds) {
        minX = Math.min(minX, bounds.x);
        minY = Math.min(minY, bounds.y);
        maxX = Math.max(maxX, bounds.x + bounds.width);
        maxY = Math.max(maxY, bounds.y + bounds.height);
      }
    }

    if (minX === Infinity) return null;

    return {
      x: minX,
      y: minY,
      width: maxX - minX,
      height: maxY - minY,
    };
  }

  /**
   * 获取图形的包围盒
   * @param {GraphicNode} graphic - 图形节点
   * @returns {{x: number, y: number, width: number, height: number} | null}
   * @private
   */
  _getGraphicBounds(graphic: GraphicNode): {
    x: number;
    y: number;
    width: number;
    height: number;
  } | null {
    const props = graphic.props as Record<string, unknown>;

    switch (graphic.type) {
      case "Canvas.Rect":
        return {
          x: (props.x as number) || 0,
          y: (props.y as number) || 0,
          width: (props.width as number) || 0,
          height: (props.height as number) || 0,
        };

      case "Canvas.Circle": {
        const radius = (props.radius as number) || 0;
        return {
          x: ((props.cx as number) || 0) - radius,
          y: ((props.cy as number) || 0) - radius,
          width: radius * 2,
          height: radius * 2,
        };
      }

      case "Canvas.Ellipse": {
        const rx = (props.rx as number) || 0;
        const ry = (props.ry as number) || 0;
        return {
          x: ((props.cx as number) || 0) - rx,
          y: ((props.cy as number) || 0) - ry,
          width: rx * 2,
          height: ry * 2,
        };
      }

      case "Canvas.Line":
      case "Canvas.Polygon":
      case "Canvas.Pipe":
        if (Array.isArray(props.points) && (props.points as [number, number][]).length > 0) {
          const pts = props.points as [number, number][];
          const xs = pts.map(([x]) => x);
          const ys = pts.map(([, y]) => y);
          const minX = Math.min(...xs);
          const minY = Math.min(...ys);
          const maxX = Math.max(...xs);
          const maxY = Math.max(...ys);
          return {
            x: minX,
            y: minY,
            width: maxX - minX,
            height: maxY - minY,
          };
        }
        break;

      case "Canvas.Text":
        return {
          x: (props.x as number) || 0,
          y: (props.y as number) || 0,
          width: 100, // 估算值
          height: ((props.fontSize as number) || 14) * 1.5,
        };

      case "Canvas.Symbol":
        return {
          x: (props.x as number) || 0,
          y: (props.y as number) || 0,
          width: 50 * ((props.scale as number) || 1),
          height: 50 * ((props.scale as number) || 1),
        };
    }

    return null;
  }

  // ==================== 事件 ====================

  /**
   * 触发选中变更事件
   * @private
   */
  _emitChange() {
    this.emit("change", {
      selectedElements: this.getSelectedElements(),
      selectionType: this.getSelectionType(),
      count: this.getSelectionCount(),
      primary: this.getPrimaryElement(),
    });
  }

  /**
   * 设置文档模型
   * @param {DocumentModel} doc - 文档模型
   */
  setDocument(doc: DocumentModel | null) {
    this._doc = doc;
  }

  /**
   * 获取当前状态快照
   * @returns {SelectionState}
   */
  getState() {
    return {
      selectedElements: [...this._selectedElements],
      hoveredElement: this._hoveredElement,
      dropTargetId: this._dropTargetId,
      anchorElement: this._anchorElement,
      activeTool: this._activeTool,
    };
  }

  /**
   * 恢复状态
   * @param {SelectionState} state - 状态快照
   */
  restoreState(state: SelectionState) {
    this._selectedElements = [...state.selectedElements];
    this._hoveredElement = state.hoveredElement;
    this._dropTargetId = state.dropTargetId;
    this._anchorElement = state.anchorElement;
    this._activeTool = state.activeTool;
    this._emitChange();
  }

  /**
   * 重置状态
   */
  reset() {
    this._selectedElements = [];
    this._hoveredElement = null;
    this._dropTargetId = null;
    this._anchorElement = null;
    this._activeTool = "select";
    this._emitChange();
  }
}

export default SelectionModel;
