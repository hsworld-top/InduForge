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
  componentRegistry,
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
  return buildSchemaFromPagePayload(
    { page: { id: fallbackPageId } },
    projectId,
  );
};

/**
 * 确保页面根节点存在并修复异常标签
 * 注意：只修复缺失、非法或历史脏数据，不静默重写合法页面结构
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

const ensurePageRootNodes = (schema) => {
  if (!schema || typeof schema !== "object") return;
  if (!schema.pagesById || !schema.nodesById) return;

  for (const page of Object.values(schema.pagesById)) {
    if (!page) continue;
    const rootId = page.rootNodeId;
    let rootNode = rootId ? schema.nodesById[rootId] : null;

    // 只在根节点完全缺失时才创建新的 FreeContainer
    if (!rootNode) {
      rootNode = createComponentNode("FreeContainer", {
        label: "画布",
        props: {},
        style: { width: "100%", height: "100%" },
      });
      schema.nodesById[rootNode.id] = rootNode;
      page.rootNodeId = rootNode.id;
      continue;
    }

    // 只在根节点类型未知或非法时才修复，不强制改写为 FreeContainer
    if (typeof rootNode.type !== "string" || !rootNode.type) {
      rootNode.type = "FreeContainer";
    } else if (!KNOWN_LAYOUT_TYPES.has(rootNode.type)) {
      // 未知类型（非布局容器类型）改为 FreeContainer，合法布局容器保持不变
      rootNode.type = "FreeContainer";
    }

    // 补齐缺失的属性
    if (!rootNode.props || typeof rootNode.props !== "object") {
      rootNode.props = {};
    }
    rootNode.style = {
      ...(rootNode.style || {}),
      width: "100%",
      height: "100%",
    };

    // 修复乱码标签
    if (!rootNode.label || rootNode.label.includes("\uFFFD")) {
      rootNode.label = "画布";
    }
  }
};

/**
 * 规范化布局容器类型
 * @param {import('@/editor-core').ProjectSchema} schema - 工程 Schema
 * @returns {import('@/editor-core').ProjectSchema}
 */
const normalizeLayoutSchema = (schema) => {
  if (!schema || typeof schema !== "object") return schema;
  if (!schema.nodesById) return schema;

  ensurePageRootNodes(schema);
  const resolveLayoutCount = (value, fallback, max = 24) => {
    const num = Number(value);
    if (Number.isFinite(num) && num > 0) {
      return Math.min(max, Math.floor(num));
    }
    return fallback;
  };
  const buildColumnSpans = (columns) => {
    const count = resolveLayoutCount(columns, 1, 24);
    const base = Math.max(1, Math.floor(24 / count));
    const remainder = 24 - base * (count - 1);
    return Array.from({ length: count }, (_, index) =>
      index === count - 1 ? Math.max(1, remainder) : base,
    );
  };
  const ensureRowColumns = (rowNode) => {
    if (!rowNode) return;
    if (!rowNode.props || typeof rowNode.props !== "object") {
      rowNode.props = {};
    }
    const children = Array.isArray(rowNode.children) ? rowNode.children : [];
    const colIds = children.filter((childId) => {
      const childNode = schema.nodesById?.[childId];
      return childNode?.type === "ElCol";
    });
    const resolvedColumns = resolveLayoutCount(
      rowNode.props.columns,
      colIds.length || 1,
      24,
    );
    rowNode.props.columns = resolvedColumns;
    if (colIds.length >= resolvedColumns) return;
    const colManifest = componentRegistry.get("ElCol");
    const spans = buildColumnSpans(resolvedColumns);
    const nextChildren = [...children];
    for (let index = colIds.length; index < resolvedColumns; index += 1) {
      const span = spans[index];
      const colNode = createComponentNode("ElCol", {
        parentNode: rowNode,
        label: colManifest?.name || "Col",
        props: { ...(colManifest?.defaultProps || {}), span },
        style: { ...(colManifest?.defaultStyle || {}) },
      });
      schema.nodesById[colNode.id] = colNode;
      nextChildren.push(colNode.id);
    }
    rowNode.children = nextChildren;
  };

  for (const node of Object.values(schema.nodesById)) {
    if (!node || !node.type) continue;

    // 兼容历史拼写错误的布局类型
    if (node.type === "Elayout" || node.type === "EILayout") {
      node.type = "ElLayout";
    } else if (node.type === "ElayoutRow" || node.type === "EILayoutRow") {
      node.type = "ElLayoutRow";
    } else if (node.type === "Elcol" || node.type === "EICol") {
      node.type = "ElCol";
    }

    // 补齐布局组件默认属性
    if (
      node.type === "ElLayout" ||
      node.type === "ElLayoutRow" ||
      node.type === "ElCol"
    ) {
      const manifest = componentRegistry.get(node.type);
      if (manifest?.defaultProps) {
        if (!node.props || typeof node.props !== "object") {
          node.props = {};
        }
        Object.entries(manifest.defaultProps).forEach(([key, value]) => {
          if (node.props[key] === undefined) {
            node.props[key] = value;
          }
        });
      }
    }

    if (node.type === "Container") {
      node.type = "FlexContainer";
      node.props = {
        direction: node.props?.direction || "column",
        wrap: node.props?.wrap || "nowrap",
        justify: node.props?.justify || "flex-start",
        align: node.props?.align || "stretch",
        gap: node.props?.gap ?? 0,
      };
    } else if (node.type === "Row") {
      node.type = "FlexContainer";
      node.props = {
        direction: "row",
        wrap: node.props?.wrap || "wrap",
        justify: node.props?.justify || "flex-start",
        align: node.props?.align || "stretch",
        gap: node.props?.gap ?? 0,
      };
    } else if (node.type === "Col") {
      node.type = "FlexContainer";
      node.props = {
        direction: "column",
        wrap: node.props?.wrap || "nowrap",
        justify: node.props?.justify || "flex-start",
        align: node.props?.align || "stretch",
        gap: node.props?.gap ?? 0,
      };
      if (!node.layoutItem) {
        node.layoutItem = {
          flex: {
            grow: 1,
            shrink: 1,
            basis: "0%",
          },
        };
      }
    } else if (node.type === "ElLayout") {
      if (!node.props || typeof node.props !== "object") {
        node.props = {};
      }
      if (node.props.columns !== undefined) {
        delete node.props.columns;
      }
      const desiredRows = resolveLayoutCount(node.props.rows, 1, 24);
      const children = Array.isArray(node.children) ? node.children : [];
      const rowIds = [];
      const orphanIds = [];
      for (const childId of children) {
        const childNode = schema.nodesById?.[childId];
        if (!childNode) continue;
        if (childNode.type === "ElLayoutRow") {
          rowIds.push(childId);
        } else {
          orphanIds.push(childId);
        }
      }

      let normalizedRowIds = [...rowIds];
      if (normalizedRowIds.length === 0 && orphanIds.length > 0) {
        const rowManifest = componentRegistry.get("ElLayoutRow");
        const rowNode = createComponentNode("ElLayoutRow", {
          parentNode: node,
          label: rowManifest?.name || "行",
          props: {
            ...(rowManifest?.defaultProps || {}),
            columns: orphanIds.length,
          },
          style: { ...(rowManifest?.defaultStyle || {}) },
        });
        rowNode.children = orphanIds;
        schema.nodesById[rowNode.id] = rowNode;
        normalizedRowIds = [rowNode.id];
      } else if (normalizedRowIds.length > 0 && orphanIds.length > 0) {
        const firstRow = schema.nodesById?.[normalizedRowIds[0]];
        if (firstRow) {
          firstRow.children = [...orphanIds, ...(firstRow.children || [])];
        }
      }

      const rowManifest = componentRegistry.get("ElLayoutRow");
      while (normalizedRowIds.length < desiredRows) {
        const rowNode = createComponentNode("ElLayoutRow", {
          parentNode: node,
          label: rowManifest?.name || "行",
          props: { ...(rowManifest?.defaultProps || {}), columns: 1 },
          style: { ...(rowManifest?.defaultStyle || {}) },
        });
        schema.nodesById[rowNode.id] = rowNode;
        normalizedRowIds.push(rowNode.id);
      }

      node.children = normalizedRowIds;
      node.props.rows = Math.max(1, normalizedRowIds.length);
      normalizedRowIds.forEach((rowId) => {
        ensureRowColumns(schema.nodesById?.[rowId]);
      });
    }
  }

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

  let rootId = resolveRootNodeId(page, nodesById);
  if (!rootId) {
    const rootNode = createComponentNode("FreeContainer", {
      label: "画布",
      props: {},
      style: {
        width: "100%",
        height: "100%",
      },
    });
    nodesById[rootNode.id] = rootNode;
    rootId = rootNode.id;
  }

  if (!nodesById[rootId]) {
    nodesById[rootId] = createComponentNode("FreeContainer", {
      id: rootId,
      label: "画布",
      props: {},
      style: {
        width: "100%",
        height: "100%",
      },
    });
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
