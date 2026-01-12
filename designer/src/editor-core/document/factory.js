/**
 * 组件节点工厂函数
 * 用于创建各种类型的组件节点
 */

import { generateId } from "./types.js";

/**
 * 创建组件节点
 * @param {string} type - 组件类型
 * @param {Object} [options] - 节点选项
 * @param {Object} [options.parentNode] - 父节点（用于推断定位模式）
 * @param {string} [options.label] - 显示标签
 * @param {Record<string, any>} [options.props] - 组件属性
 * @param {Record<string, any>} [options.style] - 样式定义
 * @param {import('./types.js').LayoutItem} [options.layoutItem] - 布局配置（兼容旧版）
 * @returns {import('./types.js').ComponentNode} 组件节点
 */
export function createComponentNode(type, options = {}) {
  const { parentNode, label, props, style, layoutItem } = options;

  // 自动推断定位模式
  const positioning = inferPositioning(type, parentNode);
  
  const node = {
    id: generateId("node_"),
    type,
    label: label || type,
    props: props || {},
    style: style || {},
    layoutItem: layoutItem || null, // 兼容旧版
    positioning, // 新架构：定位模式
    absolutePos: positioning === "absolute" ? createDefaultAbsolutePos() : undefined,
    flowLayout: positioning === "flow" ? createDefaultFlowLayout(parentNode) : undefined,
    bindings: {},
    permissions: {},
    events: {},
    children: [],
    hidden: false,
    locked: false,
  };

  return node;
}

/**
 * 推断节点的定位模式
 * 根据父容器类型自动推断子节点应该使用哪种定位模式
 * @param {string} type - 组件类型
 * @param {Object} [parentNode] - 父节点
 * @returns {'absolute' | 'flow'} 定位模式
 */
export function inferPositioning(type, parentNode) {
  // 页面根节点（没有父节点）：使用绝对定位
  if (!parentNode) {
    return "absolute";
  }

  // 根据父容器类型推断
  switch (parentNode.type) {
    case "FreeContainer":
      // 自由容器：子节点使用绝对定位
      return "absolute";
      
    case "FlexContainer":
    case "ResponsiveLayout":
      // Flex 容器：子节点使用流式布局
      return "flow";
      
    case "GridContainer":
    case "ColumnLayout1":
    case "ColumnLayout2":
    case "ColumnLayout4":
      // Grid 容器：子节点使用流式布局
      return "flow";
      
    default:
      // 默认使用流式布局（更安全）
      return "flow";
  }
}

/**
 * 创建默认绝对定位配置
 * @returns {import('./types.js').AbsolutePosition}
 */
function createDefaultAbsolutePos() {
  return {
    x: 0,
    y: 0,
    w: 200,
    h: 100,
    z: 0,
  };
}

/**
 * 创建默认流式布局配置
 * @param {Object} [parentNode] - 父节点
 * @returns {import('./types.js').FlexLayoutItem | import('./types.js').GridLayoutItem}
 */
function createDefaultFlowLayout(parentNode) {
  if (!parentNode) {
    return {}; // 空配置
  }

  // 如果父容器是 Grid，返回 Grid 布局配置
  if (
    parentNode.type === "GridContainer" ||
    parentNode.type === "ColumnLayout1" ||
    parentNode.type === "ColumnLayout2" ||
    parentNode.type === "ColumnLayout4"
  ) {
    const columns = resolveGridColumns(parentNode.props?.columns);
    const childIndex = parentNode.children?.length || 0;
    const row = Math.floor(childIndex / columns) + 1;
    const col = (childIndex % columns) + 1;
    
    return {
      row,
      col,
      rowSpan: 1,
      colSpan: 1,
    };
  }

  // 默认返回 Flex 布局配置
  return {
    grow: 0,
    shrink: 1,
    basis: "auto",
  };
}

/**
 * 解析 Grid 列数
 * @param {string | number | undefined} columns - 列配置
 * @returns {number} 列数
 */
function resolveGridColumns(columns) {
  if (typeof columns === "number" && Number.isFinite(columns)) {
    return Math.max(1, Math.floor(columns));
  }

  if (typeof columns === "string") {
    // 尝试解析 repeat(N, ...) 格式
    const repeatMatch = columns.match(/repeat\((\d+)/i);
    if (repeatMatch) {
      const count = Number(repeatMatch[1]);
      if (Number.isFinite(count)) {
        return Math.max(1, Math.floor(count));
      }
    }
    
    // 尝试解析空格分隔的列模板
    const tokens = columns.trim().split(/\s+/).filter(Boolean);
    if (tokens.length > 0) {
      return tokens.length;
    }
  }

  // 默认 3 列
  return 3;
}

/**
 * 克隆组件节点
 * @param {import('./types.js').ComponentNode} node - 源节点
 * @param {Object} [overrides] - 覆盖属性
 * @returns {import('./types.js').ComponentNode} 新节点
 */
export function cloneComponentNode(node, overrides = {}) {
  return {
    ...node,
    id: generateId("node_"),
    props: { ...node.props },
    style: { ...node.style },
    bindings: { ...node.bindings },
    permissions: { ...node.permissions },
    events: { ...node.events },
    children: [], // 克隆时不复制子节点
    ...overrides,
  };
}

/**
 * 创建绘图组件节点
 * @param {Object} [options] - 节点选项
 * @param {Object} [options.parentNode] - 父节点
 * @param {string} [options.label] - 显示标签
 * @param {import('./types.js').DiagramProps} [options.props] - 组件属性
 * @returns {{node: import('./types.js').ComponentNode, diagramId: string}} 组件节点和绘图 ID
 */
export function createDiagramNode(options = {}) {
  const { parentNode, label, props } = options;
  const diagramId = generateId("diagram_");

  const node = createComponentNode("Diagram", {
    parentNode,
    label: label || "绘图",
    props: {
      diagramId,
      showGrid: true,
      gridSize: 10,
      background: "#ffffff",
      snapToGrid: true,
      ...props,
    },
  });

  // 绘图组件必须使用绝对定位
  node.positioning = "absolute";
  node.absolutePos = {
    x: 0,
    y: 0,
    w: 400,
    h: 300,
    z: 0,
  };
  node.flowLayout = undefined;

  return { node, diagramId };
}

/**
 * 创建空白绘图数据
 * @param {string} diagramId - 绘图 ID
 * @returns {import('./types.js').DiagramData} 绘图数据
 */
export function createDiagramData(diagramId) {
  return {
    diagramId,
    shapes: [],
    version: 1,
    createdAt: Date.now(),
    updatedAt: Date.now(),
  };
}

/**
 * 创建图元
 * @param {import('./types.js').ShapeType} type - 图元类型
 * @param {Object} data - 图元数据
 * @returns {import('./types.js').Shape} 图元
 */
export function createShape(type, data = {}) {
  const shape = {
    id: generateId("shape_"),
    type,
    x: data.x || 0,
    y: data.y || 0,
    style: {
      fill: "#ffffff",
      stroke: "#000000",
      strokeWidth: 1,
      opacity: 1,
      ...data.style,
    },
    data: {},
    zIndex: 0,
    locked: false,
    hidden: false,
  };

  // 根据类型设置特定数据
  switch (type) {
    case "line":
      shape.data = {
        x1: data.x1 || 0,
        y1: data.y1 || 0,
        x2: data.x2 || 100,
        y2: data.y2 || 100,
      };
      break;
    case "rect":
      shape.data = {
        x: data.x || 0,
        y: data.y || 0,
        width: data.width || 100,
        height: data.height || 100,
      };
      break;
    case "circle":
      shape.data = {
        cx: data.cx || 50,
        cy: data.cy || 50,
        radius: data.radius || 50,
      };
      break;
    case "text":
      shape.data = {
        x: data.x || 0,
        y: data.y || 0,
        text: data.text || "Text",
      };
      shape.style.fontSize = data.fontSize || 14;
      shape.style.fontFamily = data.fontFamily || "Arial";
      break;
    case "image":
      shape.data = {
        x: data.x || 0,
        y: data.y || 0,
        width: data.width || 100,
        height: data.height || 100,
        src: data.src || "",
      };
      break;
    case "path":
      shape.data = {
        d: data.d || "",
      };
      break;
  }

  return shape;
}
