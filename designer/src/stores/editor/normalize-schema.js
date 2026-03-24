/**
 * 文档 Schema 规范化：工程/页面结构、节点映射、布局修复
 *
 * @module stores/editor/normalize-schema
 */

import {
  createComponentNode,
  createEmptySchema,
  createPageNode,
  Serializer,
} from "@/editor-core";

/**
 * 规范化页面列表（仅接受数组，与 PagesListPayload.pages 一致）
 * @param {*} payload - pages 字段原始值
 * @returns {Array}
 */
const normalizePageList = (payload) => {
  if (payload == null) {
    throw new Error("页面列表缺失：期望 pages 为数组");
  }
  if (!Array.isArray(payload)) {
    throw new Error(
      "页面列表格式无效：期望数组（不再兼容 items/list 等历史字段）",
    );
  }
  return payload;
};

/**
 * 规范化页面 Schema：仅支持 envelope `{ schema }` 或裸 schema，二者互斥由调用方数据决定
 * @param {*} payload - 原始响应
 * @returns {Object}
 */
const normalizePageSchema = (payload) => {
  if (payload == null || typeof payload !== "object") return payload;
  if (Object.prototype.hasOwnProperty.call(payload, "schema")) {
    const inner = payload.schema;
    if (inner != null && typeof inner === "object") return inner;
    throw new Error("页面详情 envelope 中 schema 无效或缺失");
  }
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
 * 规范化页面节点映射；禁止键名与 node.id 不一致，不再改写 children
 * @param {Object | Array} nodesById - 节点映射或数组
 * @returns {{ nodesById: Object, idMap: Map<string, string> }}
 */
const normalizeNodesById = (nodesById) => {
  const idMap = new Map();

  if (Array.isArray(nodesById)) {
    const normalized = {};
    for (const node of nodesById) {
      if (!node || typeof node !== "object" || !node.id) {
        throw new Error("nodesById 数组项必须为带 id 的对象");
      }
      if (Object.prototype.hasOwnProperty.call(normalized, node.id)) {
        throw new Error(`nodesById 重复 id: ${node.id}`);
      }
      normalized[node.id] = node;
    }
    return { nodesById: normalized, idMap };
  }

  if (nodesById == null || typeof nodesById !== "object") {
    throw new Error("nodesById 必须为对象或数组");
  }

  const normalized = {};
  for (const [key, node] of Object.entries(nodesById)) {
    if (!node || typeof node !== "object" || !node.id) {
      throw new Error(`nodesById 项无效: ${key}`);
    }
    if (key !== node.id) {
      throw new Error(`nodesById 键与 node.id 不一致: ${key} !== ${node.id}`);
    }
    normalized[node.id] = node;
  }
  return { nodesById: normalized, idMap };
};

/**
 * 解析根节点 ID（必须显式有效，禁止由图推断兜底）
 * @param {Object} page - 页面数据
 * @param {Object} nodesById - 节点映射
 * @returns {string}
 */
const resolveRootNodeId = (page, nodesById) => {
  const rid = page?.rootNodeId;
  if (!rid || typeof rid !== "string") {
    throw new Error("页面缺少有效的 rootNodeId");
  }
  if (!nodesById?.[rid]) {
    throw new Error(`rootNodeId 在 nodesById 中不存在: ${rid}`);
  }
  return rid;
};

/**
 * 补齐工程级 Schema 结构
 * @param {Object} schema - 工程 Schema
 * @returns {Object}
 */
const ensureProjectSchemaStructure = (schema) => {
  if (!schema || typeof schema !== "object") return schema;
  if (!schema.pagesById || typeof schema.pagesById !== "object") return schema;

  if (schema.nodesById == null || typeof schema.nodesById !== "object") {
    throw new Error("工程 Schema 缺少 nodesById");
  }
  if (schema.graphicsById == null || typeof schema.graphicsById !== "object") {
    throw new Error("工程 Schema 缺少 graphicsById");
  }
  if (schema.entry == null || typeof schema.entry !== "object") {
    throw new Error("工程 Schema 缺少 entry");
  }

  const { nodesById } = normalizeNodesById(schema.nodesById);
  schema.nodesById = nodesById;

  for (const page of Object.values(schema.pagesById)) {
    if (!page) continue;
    page.rootNodeId = resolveRootNodeId(page, nodesById);
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

  const { nodesById } = normalizeNodesById(payload.nodesById);
  page.rootNodeId = resolveRootNodeId(page, nodesById);

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
  const { nodesById } = normalizeNodesById(payload.nodesById);
  if (payload.graphicsById == null || typeof payload.graphicsById !== "object") {
    throw new Error("页面载荷缺少 graphicsById 对象");
  }
  const graphicsById = payload.graphicsById;

  page.rootNodeId = resolveRootNodeId(page, nodesById);

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
