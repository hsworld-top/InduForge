/**
 * editor-store 纯函数辅助：标签生成、图层目标解析、布局项构造。
 * 仅依赖最小文档/选中形状，避免把实现细节继续堆回主 store。
 */

import type { ComponentNode, LayoutItem } from "@/editor-core/document/types";

interface EditorDocLike {
  getNode: (id: string) => ComponentNode | null;
}

interface SelectionLike {
  getPrimaryElement?: () => { kind?: string; id?: string } | null | undefined;
}

/**
 * 根据根节点遍历收集现有标签
 * @param {EditorDocLike | null | undefined} doc - 文档对象
 * @param {string | null | undefined} rootNodeId - 根节点 ID
 * @returns {Set<string>}
 */
function collectNodeLabels(
  doc: EditorDocLike | null | undefined,
  rootNodeId: string | null | undefined,
): Set<string> {
  const labels = new Set<string>();
  if (!doc || !rootNodeId) return labels;

  const stack = [rootNodeId];
  while (stack.length > 0) {
    const currentId = stack.pop();
    if (!currentId) continue;
    const node = doc.getNode(currentId);
    if (!node) continue;
    if (node.label) labels.add(node.label);
    if (Array.isArray(node.children)) {
      stack.push(...node.children);
    }
  }

  return labels;
}

/**
 * 检查节点标签是否唯一
 * @param {EditorDocLike | null | undefined} doc - 文档对象
 * @param {string | null | undefined} rootNodeId - 根节点 ID
 * @param {string} name - 待检查名称
 * @param {string} [excludeId] - 排除节点 ID
 * @returns {boolean}
 */
export function isNodeLabelUnique(
  doc: EditorDocLike | null | undefined,
  rootNodeId: string | null | undefined,
  name: string,
  excludeId?: string,
): boolean {
  if (!doc || !rootNodeId) return true;
  const stack = [rootNodeId];
  while (stack.length > 0) {
    const currentId = stack.pop();
    if (!currentId) continue;
    const node = doc.getNode(currentId);
    if (!node) continue;
    if (node.label === name && node.id !== excludeId) {
      return false;
    }
    if (Array.isArray(node.children)) {
      stack.push(...node.children);
    }
  }
  return true;
}

/**
 * 为新节点生成唯一标签
 * @param {EditorDocLike | null | undefined} doc - 文档对象
 * @param {string | null | undefined} rootNodeId - 根节点 ID
 * @param {string} baseLabel - 基础标签
 * @returns {string}
 */
export function buildUniqueNodeLabel(
  doc: EditorDocLike | null | undefined,
  rootNodeId: string | null | undefined,
  baseLabel: string,
): string {
  const normalized = baseLabel || "容器";
  const labels = collectNodeLabels(doc, rootNodeId);
  if (!labels.has(normalized)) return normalized;

  let index = 1;
  let nextLabel = `${normalized}${index}`;
  while (labels.has(nextLabel)) {
    index += 1;
    nextLabel = `${normalized}${index}`;
  }
  return nextLabel;
}

/**
 * 解析图层操作目标节点
 * @param {SelectionLike | null | undefined} selection - 选中模型
 * @param {string} [nodeId] - 显式节点 ID
 * @returns {string}
 */
export function resolveLayerTargetFromSelection(
  selection: SelectionLike | null | undefined,
  nodeId?: string,
): string {
  if (nodeId) return nodeId;
  const primary = selection?.getPrimaryElement?.();
  return primary?.kind === "node" ? String(primary.id || "") : "";
}

/**
 * 构建绝对布局项
 * @param {{ x: number; y: number; width: number; height: number }} dropInfo - 拖拽位置信息
 * @returns {LayoutItem}
 */
export function buildFreeLayoutItem(dropInfo: {
  x: number;
  y: number;
  width: number;
  height: number;
}): LayoutItem {
  return {
    free: {
      mode: "abs",
      abs: {
        x: Math.max(0, Math.round(dropInfo.x)),
        y: Math.max(0, Math.round(dropInfo.y)),
        w: dropInfo.width,
        h: dropInfo.height,
        z: 1,
      },
    },
  };
}

/**
 * 构建 Flex 布局项
 * @returns {LayoutItem}
 */
export function buildFlexLayoutItem(): LayoutItem {
  return {
    flex: {
      grow: 0,
      shrink: 0,
      basis: "auto",
    },
  };
}

/**
 * 构建 Grid 布局项
 * @param {ComponentNode} parentNode - 父节点
 * @param {(value: unknown) => number} resolveGridCount - Grid 列数解析器
 * @returns {LayoutItem}
 */
export function buildGridLayoutItem(
  parentNode: ComponentNode,
  resolveGridCount: (value: unknown) => number,
): LayoutItem {
  const columns = resolveGridCount(parentNode.props?.columns);
  const colCount = Math.max(1, columns);
  const index = parentNode.children?.length ?? 0;
  const row = Math.floor(index / colCount) + 1;
  const col = (index % colCount) + 1;

  return {
    grid: {
      row,
      col,
      rowSpan: 1,
      colSpan: 1,
    },
  };
}

/**
 * 解析默认尺寸
 * @param {string} type - 组件类型
 * @param {{ defaultSize?: { width?: number; height?: number } } | null} manifest - 组件清单
 * @returns {{ width: number; height: number }}
 */
export function resolveDefaultSize(
  type: string,
  manifest?: { defaultSize?: { width?: number; height?: number } } | null,
): { width: number; height: number } {
  if (manifest?.defaultSize) {
    return {
      width: manifest.defaultSize.width || 120,
      height: manifest.defaultSize.height || 32,
    };
  }

  const sizeMap: Record<string, { width: number; height: number }> = {
    FlexContainer: { width: 360, height: 200 },
    HorizontalLayout: { width: 400, height: 160 },
    VerticalLayout: { width: 240, height: 240 },
    FreeContainer: { width: 360, height: 200 },
    GridContainer: { width: 360, height: 200 },
    ElContainer: { width: 360, height: 240 },
    ElLayout: { width: 360, height: 200 },
    Text: { width: 120, height: 32 },
    Button: { width: 120, height: 36 },
  };

  return sizeMap[type] || { width: 160, height: 80 };
}

/**
 * 解析 Grid 列数
 * @param {unknown} value - 列配置
 * @returns {number}
 */
export function resolveGridCount(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.max(1, Math.floor(value));
  }

  if (typeof value === "string") {
    const repeatMatch = value.match(/repeat\((\d+)/i);
    if (repeatMatch) {
      const count = Number(repeatMatch[1]);
      if (Number.isFinite(count)) return Math.max(1, Math.floor(count));
    }
    const tokens = value.trim().split(/\s+/).filter(Boolean);
    if (tokens.length > 0) return tokens.length;
  }

  return 1;
}

/**
 * 构建布局配置
 * @param {ComponentNode | null} parentNode - 父节点
 * @param {{ x: number; y: number; width: number; height: number }} dropInfo - 拖拽位置信息
 * @returns {LayoutItem}
 */
export function buildLayoutItem(
  parentNode: ComponentNode | null,
  dropInfo: { x: number; y: number; width: number; height: number },
): LayoutItem {
  if (!parentNode) {
    return buildFreeLayoutItem(dropInfo);
  }

  if (
    parentNode.type === "GridContainer" ||
    parentNode.type === "ColumnLayout1" ||
    parentNode.type === "ColumnLayout2" ||
    parentNode.type === "ColumnLayout4"
  ) {
    return buildGridLayoutItem(parentNode, resolveGridCount);
  }

  if (
    parentNode.type === "FlexContainer" ||
    parentNode.type === "HorizontalLayout" ||
    parentNode.type === "VerticalLayout" ||
    parentNode.type === "ResponsiveLayout" ||
    parentNode.type === "ElContainer" ||
    parentNode.type === "ElLayout" ||
    parentNode.type === "ElLayoutRow" ||
    parentNode.type === "ElHeader" ||
    parentNode.type === "ElAside" ||
    parentNode.type === "ElMain" ||
    parentNode.type === "ElFooter" ||
    parentNode.type === "ElCol"
  ) {
    return buildFlexLayoutItem();
  }

  return buildFreeLayoutItem(dropInfo);
}
