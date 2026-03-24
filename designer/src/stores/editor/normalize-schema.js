/**
 * 文档 Schema 规范化：工程/页面结构、节点映射、布局修复
 *
 * @module stores/editor/normalize-schema
 */

import {
  createComponentNode,
  createEmptySchema,
  createPageNode,
  buildPagePathFromName,
  Serializer,
} from "@/editor-core";

/**
 * 规范化页面列表
 * @param {*} payload - 原始响应
 * @returns {Array}
 */
const normalizePageList = (payload) => {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.items)) return payload.items;
  if (Array.isArray(payload?.list)) return payload.list;
  return [];
};

/**
 * 规范化页面 Schema
 * @param {*} payload - 原始响应
 * @returns {Object}
 */
const normalizePageSchema = (payload) => {
  if (!payload || typeof payload !== "object") return payload;
  if (payload.schema) return payload.schema;
  return payload;
};

/**
 * 判断是否为工程级 Schema
 * @param {*} payload - 原始响应
 * @returns {boolean}
 */
const isProjectSchemaPayload = (payload) => {
  return Boolean(
    payload &&
    typeof payload === "object" &&
    payload.pagesById &&
    payload.nodesById,
  );
};

/**
 * 规范化页面节点映射，并修复键值不一致问题
 * @param {Object | Array} nodesById - 节点映射或数组
 * @returns {{ nodesById: Object, idMap: Map<string, string> }}
 */
const normalizeNodesById = (nodesById) => {
  const rawMap = Array.isArray(nodesById)
    ? nodesById.reduce((map, node) => {
        if (node && node.id) {
          map[node.id] = node;
        }
        return map;
      }, {})
    : nodesById && typeof nodesById === "object"
      ? nodesById
      : {};

  const normalized = {};
  const idMap = new Map();

  for (const [key, node] of Object.entries(rawMap)) {
    if (!node || !node.id) continue;
    normalized[node.id] = node;
    if (key !== node.id) {
      idMap.set(key, node.id);
    }
  }

  if (idMap.size > 0) {
    for (const node of Object.values(normalized)) {
      if (!Array.isArray(node.children)) continue;
      node.children = node.children
        .map((childId) => idMap.get(childId) || childId)
        .filter(Boolean);
    }
  }

  return { nodesById: normalized, idMap };
};

/**
 * 解析根节点 ID
 * @param {Object} page - 页面数据
 * @param {Object} nodesById - 节点映射
 * @returns {string}
 */
const resolveRootNodeId = (page, nodesById) => {
  if (page?.rootNodeId && nodesById?.[page.rootNodeId]) {
    return page.rootNodeId;
  }

  const nodeIds = Object.keys(nodesById || {});
  if (nodeIds.length === 0) return "";

  const childSet = new Set();
  for (const node of Object.values(nodesById)) {
    if (Array.isArray(node?.children)) {
      node.children.forEach((id) => childSet.add(id));
    }
  }

  const candidates = nodeIds.filter((id) => !childSet.has(id));
  return candidates[0] || nodeIds[0];
};

/**
 * 补齐工程级 Schema 结构
 * @param {Object} schema - 工程 Schema
 * @returns {Object}
 */
const ensureProjectSchemaStructure = (schema) => {
  if (!schema || typeof schema !== "object") return schema;
  if (!schema.pagesById || typeof schema.pagesById !== "object") return schema;

  const { nodesById, idMap } = normalizeNodesById(schema.nodesById);
  schema.nodesById = nodesById;

  if (!schema.graphicsById || typeof schema.graphicsById !== "object") {
    schema.graphicsById = {};
  }
  if (!schema.entry || typeof schema.entry !== "object") {
    schema.entry = {};
  }

  for (const page of Object.values(schema.pagesById)) {
    if (!page) continue;
    if (page.rootNodeId && idMap.has(page.rootNodeId)) {
      page.rootNodeId = idMap.get(page.rootNodeId);
    }
    const resolvedRootId = resolveRootNodeId(page, nodesById);
    if (resolvedRootId) {
      page.rootNodeId = resolvedRootId;
    }
  }

  return schema;
};

/**
 * 确保页面数据包含 ID，并修复根节点指向
 * @param {Object} payload - 页面数据
 * @param {string} fallbackPageId - 兜底页面 ID
 * @returns {Object}
 */
const ensurePagePayloadId = (payload, fallbackPageId) => {
  if (!payload || typeof payload !== "object") return payload;
  if (!fallbackPageId) return payload;

  const page =
    payload.page && typeof payload.page === "object" ? payload.page : {};
  page.id = fallbackPageId;

  const { nodesById, idMap } = normalizeNodesById(payload.nodesById);
  if (page.rootNodeId && idMap.has(page.rootNodeId)) {
    page.rootNodeId = idMap.get(page.rootNodeId);
  }

  const resolvedRootId = resolveRootNodeId(page, nodesById);
  if (resolvedRootId) {
    page.rootNodeId = resolvedRootId;
  }

  payload.page = page;
  payload.nodesById = nodesById;
  return payload;
};

/**
 * 将响应数据统一解析为工程级 Schema
 * @param {*} payload - 原始响应
 * @param {string} projectId - 工程 ID
 * @param {string} fallbackPageId - 页面兜底 ID
 * @returns {import('@/editor-core').ProjectSchema}
 */
const resolveProjectSchema = (payload, projectId, fallbackPageId) => {
  const normalized = normalizePageSchema(payload);
  if (isProjectSchemaPayload(normalized)) {
    return ensureProjectSchemaStructure(normalized);
  }
  if (normalized && typeof normalized === "object" && normalized.pagesById) {
    return ensureProjectSchemaStructure(normalized);
  }
  if (normalized && typeof normalized === "object") {
    const ensured = ensurePagePayloadId(normalized, fallbackPageId);
    return buildSchemaFromPagePayload(ensured, projectId);
  }
  throw new Error("无效的页面载荷：无法解析为工程 Schema");
};

/**
 * 允许的页面根容器类型（非法即失败，不再静默创建/改写根节点）
 * @param {Object} schema - 工程 Schema
 */
const KNOWN_LAYOUT_TYPES = new Set([
  "FreeContainer",
  "ElLayout",
  "ElLayoutRow",
  "ElCol",
  "VerticalLayout",
  "HorizontalLayout",
]);

const LEGACY_LAYOUT_TYPES = new Set([
  "Container",
  "Row",
  "Col",
  "Elayout",
  "EILayout",
  "ElayoutRow",
  "EILayoutRow",
  "Elcol",
  "EICol",
]);

const ensurePageRootNodes = (schema) => {
  if (!schema || typeof schema !== "object") return;
  if (!schema.pagesById || !schema.nodesById) return;

  for (const page of Object.values(schema.pagesById)) {
    if (!page) continue;
    const rootId = page.rootNodeId;
    if (!rootId) {
      throw new Error("页面缺少 rootNodeId");
    }
    const rootNode = schema.nodesById[rootId];
    if (!rootNode) {
      throw new Error(`页面根节点不存在: ${rootId}`);
    }
    if (typeof rootNode.type !== "string" || !rootNode.type) {
      throw new Error("页面根节点类型非法");
    }
    if (!KNOWN_LAYOUT_TYPES.has(rootNode.type)) {
      throw new Error(`页面根类型不在允许列表: ${rootNode.type}`);
    }
  }
};

/**
 * 校验 ElLayout / ElLayoutRow 结构；禁止遗留类型与静默补节点
 * @param {import('@/editor-core').ProjectSchema} schema
 */
const assertLayoutStructureStrict = (schema) => {
  const nodesById = schema.nodesById;
  if (!nodesById) return;

  for (const node of Object.values(nodesById)) {
    if (!node?.type) continue;
    if (LEGACY_LAYOUT_TYPES.has(node.type)) {
      throw new Error(`不支持的遗留布局类型: ${node.type}`);
    }
    if (node.type === "ElLayout") {
      const children = Array.isArray(node.children) ? node.children : [];
      for (const cid of children) {
        const child = nodesById[cid];
        if (!child || child.type !== "ElLayoutRow") {
          throw new Error(`ElLayout ${node.id} 只能包含 ElLayoutRow 子节点`);
        }
      }
      const rows = node.props?.rows;
      if (rows != null && Number(rows) !== children.length) {
        throw new Error(`ElLayout ${node.id} 的 props.rows 与子行数量不一致`);
      }
    }
    if (node.type === "ElLayoutRow") {
      const children = Array.isArray(node.children) ? node.children : [];
      for (const cid of children) {
        const child = nodesById[cid];
        if (!child || child.type !== "ElCol") {
          throw new Error(`ElLayoutRow ${node.id} 只能包含 ElCol 子节点`);
        }
      }
      const columns = node.props?.columns;
      if (columns != null && Number(columns) !== children.length) {
        throw new Error(
          `ElLayoutRow ${node.id} 的 props.columns 与 ElCol 子节点数量不一致`,
        );
      }
    }
  }
};

/**
 * 布局校验（不再自动修补 ElCol/行/列或改写遗留类型）
 * @param {import('@/editor-core').ProjectSchema} schema - 工程 Schema
 * @returns {import('@/editor-core').ProjectSchema}
 */
const normalizeLayoutSchema = (schema) => {
  if (!schema || typeof schema !== "object") return schema;
  if (!schema.nodesById) return schema;

  ensurePageRootNodes(schema);
  assertLayoutStructureStrict(schema);
  return schema;
};

/**
 * 更新页面 Schema 中的名称与路径
 * @param {Object} schema - 页面 Schema
 * @param {string} pageId - 页面 ID
 * @param {string} name - 页面名称
 * @param {string} path - 页面路径
 * @returns {Object}
 */
const applyPageNamePath = (schema, pageId, name, path) => {
  if (!schema || typeof schema !== "object") return schema;
  if (schema.page) {
    return {
      ...schema,
      page: {
        ...schema.page,
        name,
        path,
      },
    };
  }
  if (schema.pagesById && schema.pagesById[pageId]) {
    return {
      ...schema,
      pagesById: {
        ...schema.pagesById,
        [pageId]: {
          ...schema.pagesById[pageId],
          name,
          path,
        },
      },
    };
  }
  return schema;
};

/**
 * 构建最小可编辑的工程 Schema
 * @param {string} projectId - 工程 ID
 * @returns {import('@/editor-core').ProjectSchema}
 */
const createBaseSchema = (projectId) => {
  const schema = createEmptySchema({
    projectId,
    name: projectId ? `工程 ${projectId}` : "新工程",
  });

  const page = createPageNode({
    name: "首页",
    path: "/",
  });

  const rootNode = createComponentNode("FreeContainer", {
    id: page.rootNodeId,
    label: "画布",
    props: {},
    style: {
      width: "100%",
      height: "100%",
    },
  });

  schema.pagesById[page.id] = page;
  schema.nodesById[rootNode.id] = rootNode;
  schema.entry.homePageId = page.id;

  return schema;
};

/**
 * 从页面级数据构建工程 Schema
 * @param {Object} payload - 页面级数据
 * @param {string} projectId - 工程 ID
 * @returns {import('@/editor-core').ProjectSchema}
 */
const buildSchemaFromPagePayload = (payload, projectId) => {
  const schema = createEmptySchema({
    projectId,
    name: projectId ? `工程 ${projectId}` : "新工程",
  });

  const page = createPageNode(payload.page || payload);
  if (payload?.vars && typeof payload.vars === "object") {
    const nextPages =
      payload.vars.pages && typeof payload.vars.pages === "object"
        ? payload.vars.pages
        : {};
    schema.vars = {
      ...schema.vars,
      ...payload.vars,
      pages: {
        ...(schema.vars?.pages || {}),
        ...nextPages,
      },
    };
  }
  if (payload?.entry && typeof payload.entry === "object") {
    schema.entry = { ...schema.entry, ...payload.entry };
  }
  const derivedPath = buildPagePathFromName(page.name || "");
  if (!page.path || page.path === `/${page.id}`) {
    page.path = derivedPath;
  }
  const { nodesById, idMap } = normalizeNodesById(payload.nodesById);
  const graphicsById = payload.graphicsById || {};

  if (page.rootNodeId && idMap.has(page.rootNodeId)) {
    page.rootNodeId = idMap.get(page.rootNodeId);
  }

  const rootId = resolveRootNodeId(page, nodesById);
  if (!rootId) {
    throw new Error("页面载荷缺少可解析的根节点");
  }
  if (!nodesById[rootId]) {
    throw new Error(`根节点 ${rootId} 在 nodesById 中不存在`);
  }

  page.rootNodeId = rootId;

  schema.pagesById[page.id] = page;
  schema.nodesById = nodesById;
  schema.graphicsById = graphicsById;
  schema.entry.homePageId = page.id;

  return schema;
};

/**
 * 构建页面 Schema Payload
 * @param {{ id: string, name: string, path: string }} page - 页面信息
 * @returns {Object}
 */
const createPageSchemaPayload = (page) => {
  const schema = createEmptySchema({
    projectId: page.projectId || "",
    name: page.projectName || "工程",
  });
  const pageNode = createPageNode({
    id: page.id,
    name: page.name,
    path: page.path,
  });
  const rootNode = createComponentNode("FreeContainer", {
    id: pageNode.rootNodeId,
    label: "画布",
    props: {},
    style: {
      width: "100%",
      height: "100%",
    },
  });

  schema.pagesById[pageNode.id] = pageNode;
  schema.nodesById[rootNode.id] = rootNode;
  schema.entry = { ...schema.entry };

  const tempSerializer = new Serializer();
  const tempDoc = tempSerializer.importFromSchema(schema);
  const payload = tempSerializer.exportPage(tempDoc, pageNode.id);
  payload.entry = schema.entry;
  return payload;
};

/**
 * 构建新页面的 Schema（用于创建时传入 schemaContent）
 * @param {{ name: string, path: string }} pageInfo - 页面信息
 * @returns {Object} schema 内容
 */
const buildNewPageSchema = (pageInfo) => {
  const pageNode = createPageNode({
    name: pageInfo.name,
    path: pageInfo.path,
  });

  const rootNode = createComponentNode("FreeContainer", {
    id: pageNode.rootNodeId,
    label: "画布",
    props: {},
    style: {
      width: "100%",
      height: "100%",
    },
  });

  return {
    page: pageNode,
    nodesById: { [rootNode.id]: rootNode },
    graphicsById: {},
  };
};

export {
  normalizePageList,
  normalizePageSchema,
  isProjectSchemaPayload,
  normalizeNodesById,
  resolveRootNodeId,
  ensureProjectSchemaStructure,
  ensurePagePayloadId,
  resolveProjectSchema,
  ensurePageRootNodes,
  normalizeLayoutSchema,
  applyPageNamePath,
  createBaseSchema,
  buildSchemaFromPagePayload,
  createPageSchemaPayload,
  buildNewPageSchema,
};
