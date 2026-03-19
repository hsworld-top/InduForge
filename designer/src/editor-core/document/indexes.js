/**
 * 索引管理工具
 * 提供独立的索引操作函数，用于 DocumentModel
 */

/**
 * @typedef {import('./types.js').ComponentNode} ComponentNode
 * @typedef {import('./types.js').GraphicNode} GraphicNode
 * @typedef {import('./types.js').Binding} Binding
 */

/**
 * 从绑定中提取数据点路径
 * @param {Record<string, Binding>} bindings - 绑定配置
 * @returns {string[]} 数据点路径列表
 */
export function extractBindingPaths(bindings) {
  if (!bindings) return [];

  const paths = [];
  for (const binding of Object.values(bindings)) {
    if (binding && binding.kind === "datapoint" && binding.path) {
      paths.push(binding.path);
    }
  }
  return paths;
}

/**
 * 构建父子关系索引
 * @param {Record<string, ComponentNode>} nodesById - 节点映射
 * @returns {Map<string, string>} nodeId → parentId
 */
export function buildParentIndex(nodesById) {
  const index = new Map();
  for (const node of Object.values(nodesById)) {
    if (node.children) {
      for (const childId of node.children) {
        index.set(childId, node.id);
      }
    }
  }
  return index;
}

/**
 * 构建类型索引
 * @param {Record<string, ComponentNode>} nodesById - 节点映射
 * @returns {Map<string, Set<string>>} type → nodeIds
 */
export function buildTypeIndex(nodesById) {
  const index = new Map();
  for (const node of Object.values(nodesById)) {
    if (!index.has(node.type)) {
      index.set(node.type, new Set());
    }
    index.get(node.type).add(node.id);
  }
  return index;
}

/**
 * 构建绑定索引
 * @param {Record<string, ComponentNode>} nodesById - 节点映射
 * @param {Record<string, GraphicNode>} graphicsById - 图形映射
 * @returns {Map<string, Set<string>>} datapointPath → elementIds
 */
export function buildBindingIndex(nodesById, graphicsById) {
  const index = new Map();

  // 处理节点
  for (const node of Object.values(nodesById)) {
    const paths = extractBindingPaths(node.bindings);
    for (const path of paths) {
      if (!index.has(path)) {
        index.set(path, new Set());
      }
      index.get(path).add(node.id);
    }
  }

  // 处理图形
  for (const graphic of Object.values(graphicsById)) {
    const paths = extractBindingPaths(graphic.bindings);
    for (const path of paths) {
      if (!index.has(path)) {
        index.set(path, new Set());
      }
      index.get(path).add(graphic.id);
    }
  }

  return index;
}

/**
 * 构建图形页面索引
 * @param {Record<string, import('./types.js').PageNode>} pagesById - 页面映射
 * @returns {Map<string, string>} graphicId → pageId
 */
export function buildGraphicPageIndex(pagesById) {
  const index = new Map();
  for (const page of Object.values(pagesById)) {
    if (page.graphicsIds) {
      for (const graphicId of page.graphicsIds) {
        index.set(graphicId, page.id);
      }
    }
  }
  return index;
}

/**
 * 查找节点的所有后代 ID
 * @param {string} nodeId - 节点 ID
 * @param {Record<string, ComponentNode>} nodesById - 节点映射
 * @returns {string[]} 后代节点 ID 列表
 */
export function findDescendantIds(nodeId, nodesById) {
  const descendants = [];
  const node = nodesById[nodeId];
  if (!node) return descendants;

  const stack = [...(node.children || [])];
  const visited = new Set();

  while (stack.length > 0) {
    const childId = stack.pop();
    if (visited.has(childId)) continue;
    visited.add(childId);

    descendants.push(childId);

    const child = nodesById[childId];
    if (child && child.children) {
      stack.push(...child.children);
    }
  }

  return descendants;
}

/**
 * 查找节点的所有祖先 ID
 * @param {string} nodeId - 节点 ID
 * @param {Map<string, string>} parentIndex - 父索引
 * @returns {string[]} 祖先节点 ID 列表
 */
export function findAncestorIds(nodeId, parentIndex) {
  const ancestors = [];
  let currentId = parentIndex.get(nodeId);
  const visited = new Set();

  while (currentId && !visited.has(currentId)) {
    visited.add(currentId);
    ancestors.push(currentId);
    currentId = parentIndex.get(currentId);
  }

  return ancestors;
}

/**
 * 查找节点所属的页面 ID
 * @param {string} nodeId - 节点 ID
 * @param {Map<string, string>} parentIndex - 父索引
 * @param {Record<string, import('./types.js').PageNode>} pagesById - 页面映射
 * @returns {string | null} 页面 ID
 */
export function findNodePageId(nodeId, parentIndex, pagesById) {
  // 向上查找根节点
  let currentId = nodeId;
  const visited = new Set();

  while (currentId && !visited.has(currentId)) {
    visited.add(currentId);
    const parentId = parentIndex.get(currentId);
    if (!parentId) {
      // 当前节点是根节点，查找它属于哪个页面
      for (const page of Object.values(pagesById)) {
        if (page.rootNodeId === currentId) {
          return page.id;
        }
      }
      return null;
    }
    currentId = parentId;
  }
  return null;
}

/**
 * 验证节点树的完整性
 * @param {Record<string, ComponentNode>} nodesById - 节点映射
 * @param {Record<string, import('./types.js').PageNode>} pagesById - 页面映射
 * @returns {{valid: boolean, errors: string[]}}
 */
export function validateNodeTree(nodesById, pagesById) {
  const errors = [];
  const referencedIds = new Set();

  // 收集所有被引用的节点 ID
  for (const page of Object.values(pagesById)) {
    if (page.rootNodeId) {
      referencedIds.add(page.rootNodeId);
    }
  }

  for (const node of Object.values(nodesById)) {
    if (node.children) {
      for (const childId of node.children) {
        referencedIds.add(childId);
        // 检查子节点是否存在
        if (!nodesById[childId]) {
          errors.push(`节点 ${node.id} 引用了不存在的子节点 ${childId}`);
        }
      }
    }
  }

  // 检查是否有孤儿节点（未被任何页面或父节点引用）
  for (const nodeId of Object.keys(nodesById)) {
    if (!referencedIds.has(nodeId)) {
      // 检查是否是某个页面的根节点
      let isRoot = false;
      for (const page of Object.values(pagesById)) {
        if (page.rootNodeId === nodeId) {
          isRoot = true;
          break;
        }
      }
      if (!isRoot) {
        errors.push(`节点 ${nodeId} 是孤儿节点（未被引用）`);
      }
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

export default {
  extractBindingPaths,
  buildParentIndex,
  buildTypeIndex,
  buildBindingIndex,
  buildGraphicPageIndex,
  findDescendantIds,
  findAncestorIds,
  findNodePageId,
  validateNodeTree,
};
