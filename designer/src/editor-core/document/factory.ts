/**
 * 组件节点工厂函数
 * 用于创建各种类型的组件节点
 */

import type {
  AbsolutePosition,
  ComponentNode,
  DiagramData,
  FlexLayoutItem,
  GridLayoutItem,
  LayoutItem,
} from "./types.ts";
import { generateId } from "./types.ts";

const GRID_COLUMN_REPEAT_REGEX = /repeat\((\d+)/i;
const GRID_COLUMN_SPLIT_REGEX = /\s+/;

export interface DescriptorRegistryModule {
  getChildPositioning?: (containerType: string) => "absolute" | "flow" | null | undefined;
}

let _descriptorRegistry: DescriptorRegistryModule | null = null;

function getDescriptorRegistry(): DescriptorRegistryModule | null {
  return _descriptorRegistry;
}

export function initDescriptorRegistry(registry: DescriptorRegistryModule | null) {
  _descriptorRegistry = registry;
}

export interface CreateComponentNodeOptions {
  parentNode?: Pick<ComponentNode, "type" | "children" | "props"> | null;
  label?: string;
  props?: Record<string, unknown>;
  style?: Record<string, unknown>;
  layoutItem?: LayoutItem | null;
}

/**
 * 创建组件节点
 */
export function createComponentNode(
  type: string,
  options: CreateComponentNodeOptions = {},
): ComponentNode {
  const { parentNode, label, props, style, layoutItem } = options;

  // 自动推断定位模式
  const positioning = inferPositioning(type, parentNode);

  const node: ComponentNode = {
    id: generateId("node_"),
    type,
    label: label || type,
    props: (props || {}) as ComponentNode["props"],
    style: (style || {}) as ComponentNode["style"],
    layoutItem: layoutItem || null,
    positioning,
    bindings: {},
    permissions: {},
    events: {},
    children: [],
    hidden: false,
    locked: false,
  };
  if (positioning === "absolute") {
    node.absolutePos = createDefaultAbsolutePos();
  }
  if (positioning === "flow") {
    node.flowLayout = createDefaultFlowLayout(parentNode ?? undefined) as
      | FlexLayoutItem
      | GridLayoutItem;
  }

  return node;
}

/**
 * 推断节点的定位模式
 * 根据父容器类型自动推断子节点应该使用哪种定位模式
 * @param {string} type - 组件类型
 * @param {object} [parentNode] - 父节点
 * @returns {'absolute' | 'flow'} 定位模式
 */
export function inferPositioning(
  type: string,
  parentNode?: Pick<ComponentNode, "type"> | null,
): "absolute" | "flow" {
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
    case "HorizontalLayout":
    case "VerticalLayout":
    case "ResponsiveLayout":
    case "ElContainer":
    case "ElLayout":
    case "ElLayoutRow":
    case "ElHeader":
    case "ElAside":
    case "ElMain":
    case "ElFooter":
    case "ElCol":
      // Flex 容器：子节点使用流式布局
      return "flow";

    case "GridContainer":
    case "ColumnLayout1":
    case "ColumnLayout2":
    case "ColumnLayout4":
      // Grid 容器：子节点使用流式布局
      return "flow";

    default: {
      // 优先从 descriptor 读取父容器的子项定位策略
      const registry = getDescriptorRegistry();
      if (registry?.getChildPositioning) {
        const fromDescriptor = registry.getChildPositioning(parentNode.type);
        if (fromDescriptor) return fromDescriptor;
      }
      // 默认使用流式布局（更安全）
      return "flow";
    }
  }
}

/**
 * 创建默认绝对定位配置
 * @returns {import('./types.ts').AbsolutePosition}
 */
function createDefaultAbsolutePos(): AbsolutePosition {
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
 * @param {object} [parentNode] - 父节点
 * @returns {import('./types.ts').FlexLayoutItem | import('./types.ts').GridLayoutItem}
 */
function createDefaultFlowLayout(
  parentNode?: Pick<ComponentNode, "type" | "children" | "props"> | null,
): FlexLayoutItem | GridLayoutItem | Record<string, never> {
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
    const columns = resolveGridColumns(parentNode.props?.columns as string | number | undefined);
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
function resolveGridColumns(columns: string | number | undefined): number {
  if (typeof columns === "number" && Number.isFinite(columns)) {
    return Math.max(1, Math.floor(columns));
  }

  if (typeof columns === "string") {
    // 尝试解析 repeat(N, ...) 格式
    const repeatMatch = columns.match(GRID_COLUMN_REPEAT_REGEX);
    if (repeatMatch) {
      const count = Number(repeatMatch[1]);
      if (Number.isFinite(count)) {
        return Math.max(1, Math.floor(count));
      }
    }

    // 尝试解析空格分隔的列模板
    const tokens = columns.trim().split(GRID_COLUMN_SPLIT_REGEX).filter(Boolean);
    if (tokens.length > 0) {
      return tokens.length;
    }
  }

  // 默认 3 列
  return 3;
}

/**
 * 克隆组件节点
 * @param {import('./types.ts').ComponentNode} node - 源节点
 * @param {object} [overrides] - 覆盖属性
 * @returns {import('./types.ts').ComponentNode} 新节点
 */
export function cloneComponentNode(
  node: ComponentNode,
  overrides: Partial<ComponentNode> = {},
): ComponentNode {
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
 * @param {object} [options] - 节点选项
 * @param {object} [options.parentNode] - 父节点
 * @param {string} [options.label] - 显示标签
 * @param {import('./types.ts').DiagramProps} [options.props] - 组件属性
 * @returns {{node: import('./types.ts').ComponentNode, diagramId: string}} 组件节点和绘图 ID
 */
export interface CreateDiagramNodeOptions {
  parentNode?: CreateComponentNodeOptions["parentNode"];
  label?: string;
  props?: Record<string, unknown>;
}

export function createDiagramNode(options: CreateDiagramNodeOptions = {}) {
  const { parentNode, label, props } = options;
  const diagramId = generateId("diagram_");

  const diagramOpts: CreateComponentNodeOptions = {
    label: label || "绘图",
    props: {
      diagramId,
      showGrid: true,
      gridSize: 10,
      background: "#ffffff",
      snapToGrid: true,
      ...props,
    },
  };
  if (parentNode !== undefined) {
    diagramOpts.parentNode = parentNode;
  }
  const node = createComponentNode("Diagram", diagramOpts);

  // 绘图组件必须使用绝对定位
  node.positioning = "absolute";
  node.absolutePos = {
    x: 0,
    y: 0,
    w: 400,
    h: 300,
    z: 0,
  };
  delete node.flowLayout;

  return { node, diagramId };
}

/**
 * 创建空白绘图数据
 * @param {string} diagramId - 绘图 ID
 * @returns {import('./types.ts').DiagramData} 绘图数据
 */
export function createDiagramData(diagramId: string): DiagramData {
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
 * @param {import('./types.ts').ShapeType} type - 图元类型
 * @param {object} data - 图元数据
 * @returns {import('./types.ts').Shape} 图元
 */
export function createShape(
  type: string,
  data: Record<string, unknown> = {},
): Record<string, unknown> {
  const style: Record<string, unknown> = {
    fill: "#ffffff",
    stroke: "#000000",
    strokeWidth: 1,
    opacity: 1,
    ...(typeof data.style === "object" && data.style !== null
      ? (data.style as Record<string, unknown>)
      : {}),
  };

  const shape: Record<string, unknown> = {
    id: generateId("shape_"),
    type,
    x: (data.x as number) ?? 0,
    y: (data.y as number) ?? 0,
    style,
    data: {},
    zIndex: 0,
    locked: false,
    hidden: false,
  };

  const num = (v: unknown, d: number) => (typeof v === "number" && Number.isFinite(v) ? v : d);
  const str = (v: unknown, d: string) => (typeof v === "string" ? v : d);

  switch (type) {
    case "line":
      shape.data = {
        x1: num(data.x1, 0),
        y1: num(data.y1, 0),
        x2: num(data.x2, 100),
        y2: num(data.y2, 100),
      };
      break;
    case "rect":
      shape.data = {
        x: num(data.x, 0),
        y: num(data.y, 0),
        width: num(data.width, 100),
        height: num(data.height, 100),
      };
      break;
    case "circle":
      shape.data = {
        cx: num(data.cx, 50),
        cy: num(data.cy, 50),
        radius: num(data.radius, 50),
      };
      break;
    case "text":
      shape.data = {
        x: num(data.x, 0),
        y: num(data.y, 0),
        text: str(data.text, "Text"),
      };
      style.fontSize = num(data.fontSize, 14);
      style.fontFamily = str(data.fontFamily, "Arial");
      break;
    case "image":
      shape.data = {
        x: num(data.x, 0),
        y: num(data.y, 0),
        width: num(data.width, 100),
        height: num(data.height, 100),
        src: str(data.src, ""),
      };
      break;
    case "path":
      shape.data = {
        d: str(data.d, ""),
      };
      break;
  }

  return shape;
}
