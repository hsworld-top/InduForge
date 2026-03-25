/**
 * 索引管理工具：独立索引函数，供 DocumentModel 使用
 */

import type { Binding, ComponentNode, GraphicNode, PageNode } from "./types.ts";

export function extractBindingPaths(bindings: Record<string, Binding> | undefined): string[] {
  if (!bindings) return [];

  const paths: string[] = [];
  for (const binding of Object.values(bindings)) {
    if (binding && binding.kind === "datapoint" && binding.path) {
      paths.push(binding.path);
    }
  }
  return paths;
}

export function buildParentIndex(nodesById: Record<string, ComponentNode>): Map<string, string> {
  const index = new Map<string, string>();
  for (const node of Object.values(nodesById)) {
    if (node.children) {
      for (const childId of node.children) {
        index.set(childId, node.id);
      }
    }
  }
  return index;
}

export function buildTypeIndex(nodesById: Record<string, ComponentNode>): Map<string, Set<string>> {
  const index = new Map<string, Set<string>>();
  for (const node of Object.values(nodesById)) {
    if (!index.has(node.type)) {
      index.set(node.type, new Set());
    }
    index.get(node.type)!.add(node.id);
  }
  return index;
}

export function buildBindingIndex(
  nodesById: Record<string, ComponentNode>,
  graphicsById: Record<string, GraphicNode>,
): Map<string, Set<string>> {
  const index = new Map<string, Set<string>>();

  for (const node of Object.values(nodesById)) {
    const paths = extractBindingPaths(node.bindings);
    for (const path of paths) {
      if (!index.has(path)) {
        index.set(path, new Set());
      }
      index.get(path)!.add(node.id);
    }
  }

  for (const graphic of Object.values(graphicsById)) {
    const paths = extractBindingPaths(graphic.bindings);
    for (const path of paths) {
      if (!index.has(path)) {
        index.set(path, new Set());
      }
      index.get(path)!.add(graphic.id);
    }
  }

  return index;
}

export function buildGraphicPageIndex(pagesById: Record<string, PageNode>): Map<string, string> {
  const index = new Map<string, string>();
  for (const page of Object.values(pagesById)) {
    if (page.graphicsIds) {
      for (const graphicId of page.graphicsIds) {
        index.set(graphicId, page.id);
      }
    }
  }
  return index;
}

export function findDescendantIds(
  nodeId: string,
  nodesById: Record<string, ComponentNode>,
): string[] {
  const descendants: string[] = [];
  const node = nodesById[nodeId];
  if (!node) return descendants;

  const stack = [...(node.children || [])];
  const visited = new Set<string>();

  while (stack.length > 0) {
    const childId = stack.pop()!;
    if (visited.has(childId)) continue;
    visited.add(childId);

    descendants.push(childId);

    const child = nodesById[childId];
    if (child?.children) {
      stack.push(...child.children);
    }
  }

  return descendants;
}

export function findAncestorIds(nodeId: string, parentIndex: Map<string, string>): string[] {
  const ancestors: string[] = [];
  let currentId = parentIndex.get(nodeId);
  const visited = new Set<string>();

  while (currentId && !visited.has(currentId)) {
    visited.add(currentId);
    ancestors.push(currentId);
    currentId = parentIndex.get(currentId);
  }

  return ancestors;
}

export function findNodePageId(
  nodeId: string,
  parentIndex: Map<string, string>,
  pagesById: Record<string, PageNode>,
): string | null {
  let currentId: string | undefined = nodeId;
  const visited = new Set<string>();

  while (currentId && !visited.has(currentId)) {
    visited.add(currentId);
    const parentId = parentIndex.get(currentId);
    if (!parentId) {
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

export function validateNodeTree(
  nodesById: Record<string, ComponentNode>,
  pagesById: Record<string, PageNode>,
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];
  const referencedIds = new Set<string>();

  for (const page of Object.values(pagesById)) {
    if (page.rootNodeId) {
      referencedIds.add(page.rootNodeId);
    }
  }

  for (const node of Object.values(nodesById)) {
    if (node.children) {
      for (const childId of node.children) {
        referencedIds.add(childId);
        if (!nodesById[childId]) {
          errors.push(`节点 ${node.id} 引用了不存在的子节点 ${childId}`);
        }
      }
    }
  }

  for (const nodeId of Object.keys(nodesById)) {
    if (!referencedIds.has(nodeId)) {
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
