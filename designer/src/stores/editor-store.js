/**
 * 编辑器状态管理（editor-store）
 *
 * 职责：
 * - 文档模型（doc）、历史（history）、选中（selection）
 * - 当前工程、页面、数据绑定系统
 * - 工程/页面 CRUD、Schema 加载与保存
 * - 命令执行（插入、删除、更新、对齐等）
 * - 规范化工具：API 响应解析、Schema 结构修复
 */

import { defineStore } from "pinia";
import { computed, markRaw, ref, shallowRef } from "vue";
import {
  createComponentNode,
  createEmptySchema,
  createPageNode,
  buildPagePathFromName,
  History,
  BatchCommand,
  UpdateNodeCommand,
  UpdateGraphicCommand,
  SelectionModel,
  Serializer,
  UpdatePageCommand,
  UpdateEntryCommand,
  InsertNodeCommand,
  RemoveNodeCommand,
  DuplicateNodeCommand,
  ReorderNodeCommand,
  ToggleNodeVisibilityCommand,
  ToggleNodeLockCommand,
  PageLockManager,
  componentRegistry,
  createSelectableElement,
  AlignElementsCommand,
  DistributeElementsCommand,
  MatchSizeCommand,
} from "@/editor-core";
import { projectApi } from "@/services";
import request from "@/utils/request";
import { Storage } from "@/utils/storage";

/**
 * 解包 API 响应
 * @param {*} payload - 原始响应
 * @returns {*}
 */
const unwrapApiData = (payload) => {
  if (payload && typeof payload === "object" && "success" in payload) {
    return payload.data;
  }
  return payload;
};
const getDefaultGlobalScripts = () => ({
  system: {
    startup: { code: "" },
    shutdown: { code: "" },
  },
  timers: { groups: [], items: [] },
  variableChanges: { groups: [], items: [] },
  custom: { groups: [], items: [] },
});

/**
 * 规范化变量定义
 * 将 source 字符串转为 { type, path }，补齐 dataCenter 类型与 mapped 标记
 * @param {Object} detail - 变量定义
 * @returns {Object}
 */
const normalizeVariableDef = (detail) => {
  if (!detail || typeof detail !== "object") return detail;
  const next = { ...detail };
  // 兼容旧版：source 为字符串时转为 { type: 'dataCenter', path }
  if (typeof next.source === "string") {
    next.source = { type: "dataCenter", path: next.source };
    next.mapped = true;
  }
  const mappedPath = next.mappedPath || next.sourcePath || next.path;
  if (!next.source && mappedPath) {
    next.source = { type: "dataCenter", path: mappedPath };
    next.mapped = true;
  }
  if (next.source && typeof next.source === "object") {
    if (!next.source.type && (next.mapped || next.source.path)) {
      next.source.type = "dataCenter";
    }
    if (!next.mapped && next.source.type === "dataCenter") {
      next.mapped = true;
    }
  }
  return next;
};

/**
 * 规范化全局变量配置
 * 支持 definitions + groups 或扁平对象两种结构
 * @param {*} raw - 原始配置
 * @param {Object} fallbackDefinitions - 兜底定义
 * @returns {{ definitions: Object, groups: Array }}
 */
const normalizeGlobalVariables = (raw, fallbackDefinitions = {}) => {
  if (!raw || typeof raw !== "object") {
    return { definitions: fallbackDefinitions, groups: [] };
  }
  if (raw.definitions || raw.groups) {
    const definitions =
      raw.definitions && typeof raw.definitions === "object"
        ? raw.definitions
        : fallbackDefinitions;
    const normalizedDefinitions = {};
    Object.entries(definitions).forEach(([name, detail]) => {
      normalizedDefinitions[name] = normalizeVariableDef(detail);
    });
    return {
      definitions: normalizedDefinitions,
      groups: Array.isArray(raw.groups) ? raw.groups : [],
    };
  }
  const normalizedDefinitions = {};
  Object.entries(raw).forEach(([name, detail]) => {
    normalizedDefinitions[name] = normalizeVariableDef(detail);
  });
  return { definitions: normalizedDefinitions, groups: [] };
};

/**
 * 规范化全局脚本配置
 * 合并 system、timers、variableChanges、custom 各组，确保结构完整
 * @param {*} raw - 原始配置
 * @returns {Object}
 */
const normalizeGlobalScripts = (raw) => {
  const system = raw && raw.system ? raw.system : {};
  const timers = raw && raw.timers ? raw.timers : {};
  const variableChanges = raw && raw.variableChanges ? raw.variableChanges : {};
  const custom = raw && raw.custom ? raw.custom : {};

  return {
    system: {
      startup: { code: system.startup?.code || "" },
      shutdown: { code: system.shutdown?.code || "" },
    },
    timers: {
      groups: Array.isArray(timers.groups) ? timers.groups : [],
      items: Array.isArray(timers.items) ? timers.items : [],
    },
    variableChanges: {
      groups: Array.isArray(variableChanges.groups)
        ? variableChanges.groups
        : [],
      items: Array.isArray(variableChanges.items) ? variableChanges.items : [],
    },
    custom: {
      groups: Array.isArray(custom.groups) ? custom.groups : [],
      items: Array.isArray(custom.items) ? custom.items : [],
    },
  };
};

/**
 * 获取 Menu 组件默认详细配置
 * @returns {string}
 */
const getMenuDefaultDetailConfig = () =>
  "this.menu({\n" +
  '  id: "menuNav",\n' +
  '  label: "菜单基础配置",\n' +
  '  type: "Menu",\n' +
  "  props: {\n" +
  '    defaultActive: "2",\n' +
  "    items: [\n" +
  '      { index: "1", label: "导航一", icon: "location" },\n' +
  '      { index: "2", label: "导航二", icon: "menu" },\n' +
  '      { index: "3", label: "导航三", icon: "document", disabled: true },\n' +
  '      { index: "4", label: "导航四", icon: "setting" },\n' +
  "    ],\n" +
  "  },\n" +
  "});";

/**
 * 获取 Menu 组件默认属性
 * @returns {Record<string, any>}
 */
const getMenuDefaultProps = () => ({
  defaultActive: "2",
  items: [
    { index: "1", label: "导航一", icon: "location" },
    { index: "2", label: "导航二", icon: "menu" },
    { index: "3", label: "导航三", icon: "document", disabled: true },
    { index: "4", label: "导航四", icon: "setting" },
  ],
});

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
 * @param {Object} schema - 工程 Schema
 */
const ensurePageRootNodes = (schema) => {
  if (!schema || typeof schema !== "object") return;
  if (!schema.pagesById || !schema.nodesById) return;

  for (const page of Object.values(schema.pagesById)) {
    if (!page) continue;
    const rootId = page.rootNodeId;
    let rootNode = rootId ? schema.nodesById[rootId] : null;

    if (!rootNode) {
      rootNode = createComponentNode("FreeContainer", {
        label: "画布",
        props: {},
        style: { width: "100%", height: "100%" },
      });
      schema.nodesById[rootNode.id] = rootNode;
      page.rootNodeId = rootNode.id;
    }

    if (rootNode.type !== "FreeContainer") {
      rootNode.type = "FreeContainer";
    }
    if (!rootNode.props || typeof rootNode.props !== "object") {
      rootNode.props = {};
    }
    rootNode.style = {
      ...(rootNode.style || {}),
      width: "100%",
      height: "100%",
    };

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
/**
 * 编辑器状态管理 Store
 */
export const useEditorStore = defineStore("editor", () => {
  /** @type {import('vue').ShallowRef<import('@/editor-core').DocumentModel | null>} */
  const doc = shallowRef(null);
  /** @type {import('vue').Ref<number>} */
  const docVersion = ref(0);
  /** @type {import('vue').Ref<number>} */
  const selectionVersion = ref(0);
  /** @type {import('vue').ShallowRef<import('@/editor-core').History | null>} */
  const history = shallowRef(null);
  /** @type {import('vue').ShallowRef<import('@/editor-core').SelectionModel | null>} */
  const selection = shallowRef(null);
  /** @type {import('vue').ShallowRef<import('@/editor-core').Serializer>} */
  const serializer = shallowRef(markRaw(new Serializer()));
  const pageDrafts = ref({});
  const pageTabState = ref({ tabs: [], activeId: "" });
  /** @type {import('vue').ShallowRef<import('@/editor-core').PageLockManager | null>} */
  const lockManager = shallowRef(null);
  /** @type {import('vue').Ref<import('@/editor-core').PageLockState | null>} */
  const lockState = ref(null);
  /** @type {import('vue').Ref<import('@/editor-core').EditorReadonlyState>} */
  const readonlyState = ref({ readonly: false });

  const projectId = ref("");
  const projectVariables = ref({});
  const projectVariableGroups = ref([]);
  const globalScripts = ref(getDefaultGlobalScripts());
  const projectName = ref("");
  const currentPageId = ref("");
  const pages = ref([]);
  /** @type {import('vue').Ref<{ homePageId?: string, loginPageId?: string, logoutPageId?: string }>} */
  const entryConfig = ref({});
  const loading = ref(false);
  const saving = ref(false);
  const canUndo = ref(false);
  const canRedo = ref(false);
  const error = ref("");

  let historyUnsubscribe = null;
  let lockUnsubscribe = null;
  let docUnsubscribe = null;
  let selectionUnsubscribe = null;

  /**
   * 同步锁状态
   */
  const syncLockState = () => {
    if (!lockManager.value) {
      lockState.value = null;
      readonlyState.value = { readonly: false };
      return;
    }
    lockState.value = lockManager.value.getLockState();
    readonlyState.value = lockManager.value.getReadonlyState();
  };

  /**
   * 初始化锁管理器
   * @returns {import('@/editor-core').PageLockManager}
   */
  const ensureLockManager = () => {
    if (lockManager.value) return lockManager.value;

    const userInfo = Storage.getUserInfo() || {};
    const userId = userInfo.id || userInfo.userId || userInfo.user_id || "";

    const manager = new PageLockManager({
      api: request,
      currentUserId: userId,
    });

    manager.init();

    if (lockUnsubscribe) {
      lockUnsubscribe();
      lockUnsubscribe = null;
    }

    const unsubs = [
      manager.on("lockAcquired", syncLockState),
      manager.on("lockReleased", syncLockState),
      manager.on("lockStatusChanged", syncLockState),
      manager.on("lockLost", syncLockState),
      manager.on("lockForceReleased", syncLockState),
    ];

    lockUnsubscribe = () => {
      unsubs.forEach((unsub) => {
        if (typeof unsub === "function") {
          unsub();
        }
      });
    };

    lockManager.value = markRaw(manager);
    syncLockState();

    return manager;
  };

  /**
   * 是否只读
   */
  const isReadonly = computed(() => readonlyState.value?.readonly === true);

  /**
   * 是否已锁定
   */
  const isLocked = computed(() => lockState.value?.locked === true);

  /**
   * 是否为锁的持有者
   */
  const isLockOwner = computed(() => lockState.value?.isOwner === true);

  /**
   * 确保可编辑
   * @returns {boolean}
   */
  const ensureEditable = () => {
    if (!isReadonly.value) return true;
    const lockedBy = readonlyState.value?.lockedByName;
    error.value = lockedBy ? `页面已被${lockedBy}锁定` : "当前为只读模式";
    return false;
  };

  /**
   * 初始化编辑器内核
   * @param {import('@/editor-core').ProjectSchema} schema - 工程 Schema
   */
  const initEditor = (schema) => {
    const normalizedSchema = normalizeLayoutSchema(schema);
    const nextDoc = serializer.value.importFromSchema(normalizedSchema);
    const nextHistory = new History(nextDoc);
    const nextSelection = new SelectionModel(nextDoc);

    if (historyUnsubscribe) {
      historyUnsubscribe();
      historyUnsubscribe = null;
    }
    if (docUnsubscribe) {
      docUnsubscribe();
      docUnsubscribe = null;
    }
    if (selectionUnsubscribe) {
      selectionUnsubscribe();
      selectionUnsubscribe = null;
    }

    historyUnsubscribe = nextHistory.on("change", (payload) => {
      canUndo.value = payload.canUndo;
      canRedo.value = payload.canRedo;
    });
    docVersion.value = 0;
    selectionVersion.value = 0;
    docUnsubscribe = nextDoc.on("change", () => {
      docVersion.value += 1;
    });
    selectionUnsubscribe = nextSelection.on("change", () => {
      selectionVersion.value += 1;
    });

    doc.value = markRaw(nextDoc);
    history.value = markRaw(nextHistory);
    selection.value = markRaw(nextSelection);
    projectName.value = nextDoc.project?.name || projectName.value;
    currentPageId.value =
      nextDoc.entry?.homePageId || nextDoc.getAllPages()[0]?.id || "";
    if (!pages.value.length) {
      pages.value = nextDoc.getAllPages();
    }
    canUndo.value = nextHistory.canUndo();
    canRedo.value = nextHistory.canRedo();
  };

  /**
   * 获取页面锁
   * @param {string} pageId - 页面 ID
   * @returns {Promise<import('@/editor-core').LockResult>}
   */
  const acquirePageLock = async (pageId) => {
    if (!pageId) {
      return {
        success: false,
        reason: "error",
        error: new Error("缺少页面信息"),
      };
    }
    const manager = ensureLockManager();
    const result = await manager.acquireLock(pageId);
    syncLockState();
    return result;
  };

  /**
   * 释放页面锁
   * @returns {Promise<void>}
   */
  const releasePageLock = async () => {
    if (!lockManager.value) return;
    await lockManager.value.releaseLock();
    syncLockState();
  };

  /**
   * 切换页面锁状态
   * @returns {Promise<{success: boolean, action?: 'acquire' | 'release', reason?: string, lockedByName?: string, error?: Error}>}
   */
  const togglePageLock = async () => {
    if (!currentPageId.value) {
      return {
        success: false,
        reason: "error",
        error: new Error("缺少页面信息"),
      };
    }

    const manager = ensureLockManager();
    if (manager.isLockOwner()) {
      await manager.releaseLock();
      syncLockState();
      return { success: true, action: "release" };
    }

    const result = await manager.acquireLock(currentPageId.value);
    syncLockState();
    if (result.success) {
      return { success: true, action: "acquire" };
    }
    return { ...result, action: "acquire" };
  };

  /**
   * 加载工程
   * @param {string} id - 工程 ID
   * @returns {Promise<{ok: boolean, error?: Error}>}
   */

  /**
   * 加载工程级变量与脚本设置
   * @returns {Promise<void>}
   */
  const loadProjectSettings = async () => {
    if (!projectId.value) return;

    const results = await Promise.allSettled([
      projectApi.getProjectVariables(projectId.value),
      projectApi.getProjectSettings(projectId.value),
    ]);

    const varsResult =
      results[0].status === "fulfilled"
        ? unwrapApiData(results[0].value)
        : null;
    const settingsResult =
      results[1].status === "fulfilled"
        ? unwrapApiData(results[1].value)
        : null;

    const fallbackDefinitions =
      varsResult && typeof varsResult === "object" ? varsResult : {};

    if (settingsResult && typeof settingsResult === "object") {
      const normalizedVariables = normalizeGlobalVariables(
        settingsResult.globalVariables,
        fallbackDefinitions,
      );
      projectVariables.value = normalizedVariables.definitions;
      projectVariableGroups.value = normalizedVariables.groups;
      globalScripts.value = normalizeGlobalScripts(
        settingsResult.globalScripts || {},
      );
      return;
    }

    projectVariables.value = fallbackDefinitions;
    projectVariableGroups.value = [];
    globalScripts.value = getDefaultGlobalScripts();
  };

  /**
   * 保存工程级变量与脚本设置
   * @returns {Promise<{ok: boolean, error?: Error}>}
   */
  const saveProjectSettings = async () => {
    if (!projectId.value) {
      return { ok: false, error: new Error("缺少工程信息") };
    }

    const payload = {
      globalVariables: {
        definitions: projectVariables.value,
        groups: projectVariableGroups.value,
      },
      globalScripts: globalScripts.value,
    };

    try {
      const results = await Promise.allSettled([
        projectApi.updateProjectSettings(projectId.value, payload),
        projectApi.updateProjectVariables(
          projectId.value,
          projectVariables.value,
        ),
      ]);
      const settingsResult =
        results[0].status === "fulfilled" ? results[0].value : null;
      const data =
        unwrapApiData(settingsResult) || settingsResult?.data || settingsResult;
      const rejected = results.find((res) => res.status === "rejected");
      if (rejected) {
        return {
          ok: false,
          error:
            rejected.reason instanceof Error
              ? rejected.reason
              : new Error("保存失败"),
        };
      }

      if (data && typeof data === "object") {
        if (data.globalVariables) {
          const normalized = normalizeGlobalVariables(
            data.globalVariables,
            projectVariables.value,
          );
          projectVariables.value = normalized.definitions;
          projectVariableGroups.value = normalized.groups;
        }
        if (data.globalScripts) {
          globalScripts.value = normalizeGlobalScripts(data.globalScripts);
        }
      }

      return { ok: true };
    } catch (err) {
      return {
        ok: false,
        error: err instanceof Error ? err : new Error("保存失败"),
      };
    }
  };
  const loadProject = async (id) => {
    projectId.value = id || "";
    await loadProjectSettings();
    loading.value = true;
    error.value = "";

    try {
      await releasePageLock();
      const { pages: pageList, entryConfig: entryConfigResp } =
        await refreshPages();

      if (!pageList.length) {
        const homePageResult = await createHomePage(id);
        if (!homePageResult.ok) {
          initEditor(createBaseSchema(id));
        }
        return { ok: homePageResult.ok };
      }

      const homePageId = entryConfigResp?.homePageId;
      const targetPageId =
        homePageId && pageList.some((p) => p.id === homePageId)
          ? homePageId
          : pageList[0]?.id;

      if (!targetPageId) {
        const homePageResult = await createHomePage(id);
        if (!homePageResult.ok) {
          initEditor(createBaseSchema(id));
        }
        return { ok: homePageResult.ok };
      }

      const pageResponse = await projectApi.getPage(id, targetPageId);
      const pagePayload = unwrapApiData(pageResponse) || {
        page: { id: targetPageId },
      };

      const nextSchema = resolveProjectSchema(pagePayload, id, targetPageId);
      initEditor(nextSchema);

      if (
        entryConfigResp &&
        doc.value &&
        Object.keys(entryConfigResp).length > 0
      ) {
        doc.value._updateEntry(entryConfigResp);
      }

      currentPageId.value = targetPageId;
      return { ok: true };
    } catch (err) {
      const nextError = err instanceof Error ? err : new Error("加载工程失败");
      error.value = nextError.message;
      initEditor(createBaseSchema(id));
      return { ok: false, error: nextError };
    } finally {
      loading.value = false;
    }
  };

  /**
   * 加载指定页面
   * @param {string} pageId - 页面 ID
   * @returns {Promise<{ok: boolean, error?: Error}>}
   */
  const loadPage = async (pageId) => {
    if (!projectId.value) {
      return { ok: false, error: new Error("缺少工程信息") };
    }
    if (!pageId) {
      return { ok: false, error: new Error("缺少页面信息") };
    }

    loading.value = true;
    error.value = "";

    try {
      await releasePageLock();
      const draft = pageDrafts.value?.[pageId];
      if (draft) {
        const nextSchema = resolveProjectSchema(draft, projectId.value, pageId);
        const existingEntry = doc.value?.entry;
        if (existingEntry) {
          nextSchema.entry = { ...nextSchema.entry, ...existingEntry };
        }
        initEditor(nextSchema);
        currentPageId.value = pageId;
        return { ok: true };
      }
      const pageResponse = await projectApi.getPage(projectId.value, pageId);
      const pagePayload = unwrapApiData(pageResponse) || {
        page: { id: pageId },
      };

      const nextSchema = resolveProjectSchema(
        pagePayload,
        projectId.value,
        pageId,
      );
      const existingEntry = doc.value?.entry;

      if (existingEntry) {
        nextSchema.entry = { ...nextSchema.entry, ...existingEntry };
      }

      initEditor(nextSchema);
      currentPageId.value = pageId;
      return { ok: true };
    } catch (err) {
      const nextError = err instanceof Error ? err : new Error("加载页面失败");
      error.value = nextError.message;
      return { ok: false, error: nextError };
    } finally {
      loading.value = false;
    }
  };

  /**
   * 刷新页面列表
   * @returns {Promise<{ pages: Array, entryConfig: Object }>}
   */
  const refreshPages = async () => {
    if (!projectId.value) return { pages: [], entryConfig: {} };
    const pagesResponse = await projectApi.getPages(projectId.value);
    const responseData = unwrapApiData(pagesResponse);

    const pageList = normalizePageList(responseData.pages);
    pages.value = pageList;
    const newEntryConfig = responseData.entryConfig || {};
    entryConfig.value = newEntryConfig;

    if (doc.value && newEntryConfig && Object.keys(newEntryConfig).length > 0) {
      doc.value._updateEntry(newEntryConfig);
    }

    return { pages: pageList, entryConfig: newEntryConfig };
  };

  /**
   * 创建首页
   * @param {string} pid - 工程 ID
   * @returns {Promise<{ok: boolean, pageId?: string, error?: Error}>}
   */
  const createHomePage = async (pid) => {
    try {
      const result = await projectApi.createPage(pid, {
        name: "首页",
        type: "page",
        parentId: null,
      });

      if (!result || !result.data) {
        throw new Error("创建首页失败：API 返回异常");
      }

      const pageId = result.data.id;
      if (!pageId) {
        throw new Error("创建首页失败：未获取到页面ID");
      }

      const schema = createBaseSchema(pid);
      const pageNode = Object.values(schema.pagesById)[0];
      if (!pageNode) {
        throw new Error("创建首页失败：无法获取页面节点");
      }

      const rootNode = schema.nodesById[pageNode.rootNodeId];

      delete schema.pagesById[pageNode.id];
      pageNode.id = pageId;
      schema.pagesById[pageId] = pageNode;
      schema.entry.homePageId = pageId;

      if (rootNode) {
        pageNode.rootNodeId = rootNode.id;
      }

      const pagePayload = {
        page: pageNode,
        nodesById: rootNode ? { [rootNode.id]: rootNode } : {},
        graphicsById: {},
      };
      await projectApi.updatePage(pid, pageId, pagePayload);

      const newEntryConfig = { homePageId: pageId };
      await projectApi.updateEntryConfig(pid, newEntryConfig);
      entryConfig.value = newEntryConfig;

      schema.pagesById = { [pageId]: pageNode };
      initEditor(schema);
      currentPageId.value = pageId;

      await refreshPages();
      return { ok: true, pageId };
    } catch (err) {
      console.error("创建首页失败:", err);
      return {
        ok: false,
        error: err instanceof Error ? err : new Error("创建首页失败"),
      };
    }
  };

  /**
   * 创建页面/分组
   * @param {{ name: string, type: string, parentId?: string | null, schemaContent?: Object }} payload - 创建参数
   * @returns {Promise<Object>}
   */
  const createPage = async (payload) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }

    const result = await projectApi.createPage(projectId.value, payload);
    const data = unwrapApiData(result) || result?.data || result;
    const pageId = data?.id || data?.page?.id;

    if (payload?.schemaContent && pageId) {
      await projectApi.updatePage(
        projectId.value,
        pageId,
        payload.schemaContent,
      );
    }

    await refreshPages();
    return data;
  };

  /**
   * 删除页面/分组
   * @param {string} pageId - 页面 ID
   * @param {"single" | "folder-only" | "cascade"} [mode] - 删除模式
   * @returns {Promise<void>}
   */
  const deletePage = async (pageId, mode) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    if (!pageId) {
      throw new Error("缺少页面信息");
    }

    await projectApi.deletePage(projectId.value, pageId, mode);
    const { pages: pageList, entryConfig: entryConfigResp } =
      await refreshPages();

    if (
      currentPageId.value &&
      pageList.some((page) => page.id === currentPageId.value)
    )
      return;

    const homeId = entryConfigResp?.homePageId;
    const nextId =
      homeId && pageList.some((page) => page.id === homeId)
        ? homeId
        : pageList.find((page) => page.type === "page")?.id;

    if (nextId) {
      await loadPage(nextId);
      return;
    }

    initEditor(createBaseSchema(projectId.value));
    currentPageId.value = "";
  };

  /**
   * 更新页面 Schema
   * @param {string} pageId - 页面 ID
   * @param {Object} [schema] - 页面 Schema
   * @returns {Promise<void>}
   */
  const updatePageSchema = async (pageId, schema) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    if (!pageId) {
      throw new Error("缺少页面信息");
    }

    const payload = schema || serializer.value.exportPage(doc.value, pageId);
    await projectApi.updatePage(projectId.value, pageId, payload);
  };

  /**
   * 移动页面到指定分组
   * @param {string} pageId - 页面 ID
   * @param {string | null} targetGroupId - 分组 ID
   * @param {string} [path] - 页面路径
   * @returns {Promise<void>}
   */
  const movePageToGroup = async (pageId, targetGroupId, path) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    if (!pageId) {
      throw new Error("缺少页面信息");
    }

    await projectApi.movePageToGroup(
      projectId.value,
      pageId,
      targetGroupId,
      path,
    );
    await refreshPages();
  };

  /**
   * 重命名页面
   * @param {string} pageId - 页面 ID
   * @param {string} name - 页面名称
   * @param {string} [path] - 页面路径
   * @returns {Promise<void>}
   */
  const renamePage = async (pageId, name, path) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    if (!pageId) {
      throw new Error("缺少页面信息");
    }

    await projectApi.renamePage(projectId.value, pageId, name, path);

    pages.value = pages.value.map((page) =>
      page.id === pageId
        ? {
            ...page,
            name,
            path: path ?? page.path,
          }
        : page,
    );

    if (doc.value && currentPageId.value === pageId && history.value) {
      const patch = { name };
      if (path !== undefined) {
        patch.path = path;
      }
      history.value.execute(new UpdatePageCommand(pageId, patch));
    }
  };

  /**
   * 更新入口配置（仅更新本地）
   * @param {Object} patch - 更新内容
   */
  const updateEntry = (patch) => {
    if (!doc.value || !history.value) return;
    if (!ensureEditable()) return;

    history.value.execute(new UpdateEntryCommand(patch));
    entryConfig.value = { ...entryConfig.value, ...patch };
  };

  /**
   * 持久化入口配置
   * @returns {Promise<void>}
   */
  const persistEntry = async () => {
    if (!projectId.value) return;
    const payload = doc.value?.entry || entryConfig.value || {};
    await projectApi.updateEntryConfig(projectId.value, payload);
    entryConfig.value = { ...payload };
  };

  /**
   * 保存当前页面
   * @returns {Promise<void>}
   */
  const saveCurrentPage = async () => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    if (!doc.value || !currentPageId.value) {
      throw new Error("缺少页面信息");
    }
    if (!ensureEditable()) {
      throw new Error(error.value || "当前为只读模式");
    }

    saving.value = true;
    try {
      const payload = serializer.value.exportPage(
        doc.value,
        currentPageId.value,
      );
      const pageVars = doc.value?.schema?.vars?.pages?.[currentPageId.value];
      if (pageVars && typeof pageVars === "object") {
        const existingVars =
          payload.vars && typeof payload.vars === "object" ? payload.vars : {};
        const existingPages =
          existingVars.pages && typeof existingVars.pages === "object"
            ? existingVars.pages
            : {};
        payload.vars = {
          ...existingVars,
          pages: {
            ...existingPages,
            [currentPageId.value]: pageVars,
          },
        };
      }
      await projectApi.updatePage(
        projectId.value,
        currentPageId.value,
        payload,
      );
      if (pageDrafts.value[currentPageId.value]) {
        const nextDrafts = { ...(pageDrafts.value || {}) };
        delete nextDrafts[currentPageId.value];
        pageDrafts.value = nextDrafts;
      }
    } finally {
      saving.value = false;
    }
  };

  /**
   * 暂存页面草稿（未保存的编辑内容）
   * @param {string} pageId - 页面 ID
   * @returns {void}
   */
  const savePageDraft = (pageId) => {
    if (!doc.value || !pageId) return;
    try {
      const payload = serializer.value.exportPage(doc.value, pageId);
      pageDrafts.value = { ...(pageDrafts.value || {}), [pageId]: payload };
    } catch (error) {
      // ignore
    }
  };

  /**
   * 保存当前页面草稿
   * @returns {void}
   */
  const saveCurrentPageDraft = () => {
    if (!currentPageId.value) return;
    savePageDraft(currentPageId.value);
  };

  /**
   * 获取页面草稿
   * @param {string} pageId - 页面 ID
   * @returns {Object | null}
   */
  const getPageDraft = (pageId) => {
    if (!pageId) return null;
    return pageDrafts.value?.[pageId] || null;
  };

  /**
   * 更新页面标签状态
   * @param {Array} tabs - 标签列表
   * @param {string} activeId - 当前激活标签 ID
   * @returns {void}
   */
  const setPageTabState = (tabs, activeId) => {
    pageTabState.value = {
      tabs: Array.isArray(tabs) ? tabs.map((item) => ({ ...item })) : [],
      activeId: activeId || "",
    };
  };

  /**
   * 更新当前页面配置（仅更新本地）
   * @param {Partial<import('@/editor-core').PageNode>} patch - 更新内容
   * @returns {boolean}
   */
  const updateCurrentPage = (patch) => {
    if (!doc.value || !currentPageId.value || !history.value) return false;
    if (!ensureEditable()) return false;

    history.value.execute(new UpdatePageCommand(currentPageId.value, patch));
    pages.value = pages.value.map((page) =>
      page.id === currentPageId.value ? { ...page, ...patch } : page,
    );
    return true;
  };

  /**
   * 同步 Element Plus Container 的内置区域容器
   * @param {string} containerId - 容器节点 ID
   * @param {Record<string, any>} nextProps - 最新属性
   */
  const syncElContainerSections = (containerId, nextProps) => {
    if (!doc.value || !history.value) return;
    const containerNode = doc.value.getNode(containerId);
    if (!containerNode || containerNode.type !== "ElContainer") return;

    const sectionDefs = [
      { prop: "showHeader", type: "ElHeader" },
      { prop: "showAside", type: "ElAside" },
      { prop: "showMain", type: "ElMain" },
      { prop: "showFooter", type: "ElFooter" },
    ];
    const sectionSizeMap = {
      ElHeader: {
        prop: "height",
        containerProp: "headerHeight",
        fallback: "60px",
      },
      ElFooter: {
        prop: "height",
        containerProp: "footerHeight",
        fallback: "60px",
      },
      ElAside: {
        prop: "width",
        containerProp: "asideWidth",
        fallback: "200px",
      },
    };
    const sectionOrder = sectionDefs.map((item) => item.type);
    const getOrderIndex = (type) => sectionOrder.indexOf(type);

    const executeCommand = (command) => {
      if (history.value.isInTransaction?.()) {
        history.value.executeInTransaction(command);
        return;
      }
      history.value.execute(command);
    };

    const currentChildren = [...(containerNode.children || [])];
    const existingSectionMap = new Map();
    for (const childId of currentChildren) {
      const childNode = doc.value.getNode(childId);
      if (childNode && sectionOrder.includes(childNode.type)) {
        existingSectionMap.set(childNode.type, childId);
      }
    }

    for (const section of sectionDefs) {
      if (!nextProps?.[section.prop] && existingSectionMap.has(section.type)) {
        executeCommand(
          new RemoveNodeCommand(existingSectionMap.get(section.type)),
        );
      }
    }

    const refreshedNode = doc.value.getNode(containerId);
    if (!refreshedNode) return;
    const refreshedChildren = [...(refreshedNode.children || [])];

    const refreshedSectionMap = new Map();
    for (const childId of refreshedChildren) {
      const childNode = doc.value.getNode(childId);
      if (childNode && sectionOrder.includes(childNode.type)) {
        refreshedSectionMap.set(childNode.type, childId);
      }
    }

    const applySectionSize = (sectionType) => {
      const config = sectionSizeMap[sectionType];
      if (!config) return;
      const sectionId = refreshedSectionMap.get(sectionType);
      if (!sectionId) return;
      const sectionNode = doc.value.getNode(sectionId);
      if (!sectionNode) return;
      const nextValue = nextProps?.[config.containerProp] || config.fallback;
      if (sectionNode.props?.[config.prop] === nextValue) return;
      executeCommand(
        new UpdateNodeCommand(sectionNode.id, {
          props: { ...(sectionNode.props || {}), [config.prop]: nextValue },
        }),
      );
    };

    const resolveInsertIndex = (type) => {
      const orderIndex = getOrderIndex(type);
      let insertIndex = refreshedChildren.length;
      for (let i = 0; i < refreshedChildren.length; i += 1) {
        const childNode = doc.value.getNode(refreshedChildren[i]);
        const childOrder = childNode ? getOrderIndex(childNode.type) : -1;
        if (childOrder !== -1 && childOrder > orderIndex) {
          insertIndex = i;
          break;
        }
      }
      if (type === "ElHeader" && insertIndex === refreshedChildren.length) {
        return 0;
      }
      return insertIndex;
    };

    const buildUniqueLabel = (baseLabel) => {
      const label = baseLabel || "容器";
      if (isLabelUnique(label)) return label;
      let index = 1;
      while (!isLabelUnique(`${label}${index}`)) {
        index += 1;
      }
      return `${label}${index}`;
    };

    for (const section of sectionDefs) {
      if (!nextProps?.[section.prop]) continue;
      if (refreshedSectionMap.has(section.type)) continue;

      const manifest = componentRegistry.get(section.type);
      const sizeConfig = sectionSizeMap[section.type];
      const sizeValue = sizeConfig
        ? nextProps?.[sizeConfig.containerProp] || sizeConfig.fallback
        : null;
      const childNode = createComponentNode(section.type, {
        parentNode: refreshedNode,
        label: buildUniqueLabel(manifest?.name || section.type),
        props: {
          ...(manifest?.defaultProps || {}),
          ...(sizeConfig && sizeValue ? { [sizeConfig.prop]: sizeValue } : {}),
        },
        style: { ...(manifest?.defaultStyle || {}) },
        layoutItem: buildFlexLayoutItem(),
      });

      const insertIndex = resolveInsertIndex(section.type);
      executeCommand(
        new InsertNodeCommand(containerId, insertIndex, childNode),
      );
    }

    Object.keys(sectionSizeMap).forEach((sectionType) => {
      if (!refreshedSectionMap.has(sectionType)) return;
      applySectionSize(sectionType);
    });
  };

  /**
   * 同步 Element Plus Layout 的内置列
   * @param {string} layoutId - 布局节点 ID
   * @param {Record<string, any>} nextProps - 最新属性
   */
  const buildElLayoutUniqueLabel = (baseLabel) => {
    const rootId = currentPage.value?.rootNodeId;
    const existingLabels = new Set();
    if (rootId && doc.value) {
      const stack = [rootId];
      while (stack.length) {
        const id = stack.pop();
        const current = doc.value.getNode(id);
        if (!current) continue;
        if (current.label) existingLabels.add(current.label);
        if (Array.isArray(current.children)) {
          stack.push(...current.children);
        }
      }
    }
    const normalized = baseLabel || "容器";
    if (!existingLabels.has(normalized)) return normalized;
    let index = 1;
    let label = `${normalized}${index}`;
    while (existingLabels.has(label)) {
      index += 1;
      label = `${normalized}${index}`;
    }
    return label;
  };

  /**
   * 同步 Element Plus Layout 行内列
   * @param {string} rowId - 行节点 ID
   * @param {Record<string, any>} nextProps - 最新属性
   * @param {{ forceSpanUpdate?: boolean, anchorColId?: string }} [options] - 同步选项
   */
  const syncElLayoutRowColumns = (rowId, nextProps, options = {}) => {
    if (!doc.value || !history.value) return;
    const rowNode = doc.value.getNode(rowId);
    if (!rowNode || rowNode.type !== "ElLayoutRow") return;

    const columns = Math.max(
      1,
      Math.min(24, Number(nextProps?.columns || rowNode.props?.columns || 3)),
    );
    const baseSpan = Math.max(1, Math.floor(24 / columns));
    const remainder = 24 - baseSpan * columns;
    const getSpanByIndex = (index) => baseSpan + (index < remainder ? 1 : 0);

    const executeCommand = (command) => {
      if (history.value.isInTransaction?.()) {
        history.value.executeInTransaction(command);
        return;
      }
      history.value.execute(command);
    };

    const children = [...(rowNode.children || [])];
    const colIds = children.filter((childId) => {
      const childNode = doc.value.getNode(childId);
      return childNode?.type === "ElCol";
    });

    if (colIds.length > columns) {
      for (let i = colIds.length - 1; i >= columns; i -= 1) {
        executeCommand(new RemoveNodeCommand(colIds[i]));
      }
    }

    let refreshedNode = doc.value.getNode(rowId);
    if (!refreshedNode) return;
    let refreshedChildren = [...(refreshedNode.children || [])];
    let refreshedCols = refreshedChildren.filter((childId) => {
      const childNode = doc.value.getNode(childId);
      return childNode?.type === "ElCol";
    });

    for (let i = refreshedCols.length; i < columns; i += 1) {
      const manifest = componentRegistry.get("ElCol");
      const childNode = createComponentNode("ElCol", {
        parentNode: refreshedNode,
        label: buildElLayoutUniqueLabel(manifest?.name || "Col"),
        props: { ...(manifest?.defaultProps || {}), span: getSpanByIndex(i) },
        style: { ...(manifest?.defaultStyle || {}) },
        layoutItem: buildFlexLayoutItem(),
      });
      const insertIndex = refreshedChildren.length;
      executeCommand(new InsertNodeCommand(rowId, insertIndex, childNode));

      refreshedNode = doc.value.getNode(rowId);
      refreshedChildren = [...(refreshedNode?.children || [])];
      refreshedCols = refreshedChildren.filter((childId) => {
        const childNode = doc.value.getNode(childId);
        return childNode?.type === "ElCol";
      });
    }

    const shouldUpdateSpan =
      options.forceSpanUpdate ||
      Object.prototype.hasOwnProperty.call(nextProps || {}, "columns");
    if (!shouldUpdateSpan) return;
    refreshedCols = refreshedCols.slice(0, columns);
    const anchorColId = options.anchorColId;
    if (anchorColId && refreshedCols.includes(anchorColId)) {
      const anchorIndex = refreshedCols.indexOf(anchorColId);
      const spans = refreshedCols.map((colId) => {
        const colNode = doc.value.getNode(colId);
        const span = Number(colNode?.props?.span) || 1;
        return Math.max(1, Math.min(24, span));
      });
      const offsets = refreshedCols.map((colId) => {
        const colNode = doc.value.getNode(colId);
        const offset = Number(colNode?.props?.offset) || 0;
        return Math.max(0, Math.min(24, offset));
      });
      const fixedSpanTotal = spans
        .slice(0, anchorIndex + 1)
        .reduce((sum, value) => sum + value, 0);
      const totalOffset = offsets.reduce((sum, value) => sum + value, 0);
      const rightCount = Math.max(0, refreshedCols.length - anchorIndex - 1);
      if (rightCount === 0) return;
      const remainingUnits = Math.max(
        rightCount,
        24 - totalOffset - fixedSpanTotal,
      );
      const base = Math.floor(remainingUnits / rightCount);
      const rem = remainingUnits - base * rightCount;
      for (let i = anchorIndex + 1; i < refreshedCols.length; i += 1) {
        const colNode = doc.value.getNode(refreshedCols[i]);
        if (!colNode) continue;
        const rightIndex = i - anchorIndex - 1;
        const nextSpan = Math.max(1, base + (rightIndex < rem ? 1 : 0));
        if (colNode.props?.span !== nextSpan) {
          executeCommand(
            new UpdateNodeCommand(colNode.id, {
              props: { ...(colNode.props || {}), span: nextSpan },
            }),
          );
        }
      }
      return;
    }

    const offsets = refreshedCols.map((colId) => {
      const colNode = doc.value.getNode(colId);
      const offset = Number(colNode?.props?.offset) || 0;
      return Math.max(0, Math.min(24, offset));
    });
    // 按剩余格数等分列宽，避免只压缩右侧区域
    const totalOffset = offsets.reduce((sum, value) => sum + value, 0);
    const remainingUnits = Math.max(columns, 24 - totalOffset);
    const base = Math.floor(remainingUnits / columns);
    const rem = remainingUnits - base * columns;
    for (let i = 0; i < refreshedCols.length; i += 1) {
      const colNode = doc.value.getNode(refreshedCols[i]);
      if (!colNode) continue;
      const nextSpan = Math.max(1, base + (i < rem ? 1 : 0));
      if (colNode.props?.span !== nextSpan) {
        executeCommand(
          new UpdateNodeCommand(colNode.id, {
            props: { ...(colNode.props || {}), span: nextSpan },
          }),
        );
      }
    }
  };

  /**
   * 同步 Element Plus Layout 行节点
   * @param {string} layoutId - 布局节点 ID
   * @param {Record<string, any>} nextProps - 最新属性
   */
  const syncElLayoutRows = (layoutId, nextProps) => {
    if (!doc.value || !history.value) return;
    const layoutNode = doc.value.getNode(layoutId);
    if (!layoutNode || layoutNode.type !== "ElLayout") return;

    const rows = Math.max(1, Math.min(24, Number(nextProps?.rows || 1)));
    const executeCommand = (command) => {
      if (history.value.isInTransaction?.()) {
        history.value.executeInTransaction(command);
        return;
      }
      history.value.execute(command);
    };

    const children = [...(layoutNode.children || [])];
    const rowIds = children.filter((childId) => {
      const childNode = doc.value.getNode(childId);
      return childNode?.type === "ElLayoutRow";
    });

    if (rowIds.length > rows) {
      for (let i = rowIds.length - 1; i >= rows; i -= 1) {
        executeCommand(new RemoveNodeCommand(rowIds[i]));
      }
    }

    let refreshedNode = doc.value.getNode(layoutId);
    if (!refreshedNode) return;
    let refreshedChildren = [...(refreshedNode.children || [])];
    let refreshedRows = refreshedChildren.filter((childId) => {
      const childNode = doc.value.getNode(childId);
      return childNode?.type === "ElLayoutRow";
    });

    for (let i = refreshedRows.length; i < rows; i += 1) {
      const manifest = componentRegistry.get("ElLayoutRow");
      const childNode = createComponentNode("ElLayoutRow", {
        parentNode: refreshedNode,
        label: buildElLayoutUniqueLabel(manifest?.name || "行"),
        props: { ...(manifest?.defaultProps || {}), columns: 1 },
        style: { ...(manifest?.defaultStyle || {}) },
        layoutItem: buildFlexLayoutItem(),
      });
      const insertIndex = refreshedChildren.length;
      executeCommand(new InsertNodeCommand(layoutId, insertIndex, childNode));

      refreshedNode = doc.value.getNode(layoutId);
      refreshedChildren = [...(refreshedNode?.children || [])];
      refreshedRows = refreshedChildren.filter((childId) => {
        const childNode = doc.value.getNode(childId);
        return childNode?.type === "ElLayoutRow";
      });
      syncElLayoutRowColumns(childNode.id, childNode.props || {}, {
        forceSpanUpdate: true,
      });
    }

    refreshedRows = refreshedRows.slice(0, rows);
    refreshedRows.forEach((rowId) => {
      const rowNode = doc.value.getNode(rowId);
      if (!rowNode) return;
      const colCount = (rowNode.children || []).filter((childId) => {
        const childNode = doc.value.getNode(childId);
        return childNode?.type === "ElCol";
      }).length;
      const nextColumns = Math.max(
        1,
        Math.min(24, Number(rowNode.props?.columns || colCount || 3)),
      );
      if (rowNode.props?.columns !== nextColumns) {
        executeCommand(
          new UpdateNodeCommand(rowNode.id, {
            props: { ...(rowNode.props || {}), columns: nextColumns },
          }),
        );
      }
      if (colCount !== nextColumns) {
        syncElLayoutRowColumns(rowNode.id, rowNode.props || {}, {
          forceSpanUpdate: true,
        });
      }
    });
  };

  /**
   * 解析尺寸为像素值
   * @param {string | number | undefined | null} value - 尺寸值
   * @returns {number | undefined}
   */
  const parseSizeToNumber = (value) => {
    if (value === null || value === undefined) return undefined;
    if (typeof value === "number" && Number.isFinite(value)) return value;
    const text = String(value).trim();
    if (!text || text === "auto") return undefined;
    if (text.endsWith("px")) {
      const num = Number.parseFloat(text.slice(0, -2));
      return Number.isFinite(num) ? num : undefined;
    }
    if (/^[\d.]+$/.test(text)) {
      const num = Number.parseFloat(text);
      return Number.isFinite(num) ? num : undefined;
    }
    return undefined;
  };

  /**
   * 计算 Layout 布局最小高度（当前不强制最小高度）
   * @param {Record<string, any> | undefined} layoutProps - 布局属性
   * @returns {number}
   */
  const resolveElLayoutMinHeight = (layoutProps) => {
    const props = layoutProps || {};
    if (!props) return 0;
    return 0;
  };

  /**
   * 同步绝对定位节点的尺寸数据
   * @param {import('@/editor-core').ComponentNode | null} node - 当前节点
   * @param {Partial<import('@/editor-core').ComponentNode>} patch - 更新内容
   * @returns {Partial<import('@/editor-core').ComponentNode>}
   */
  const syncAbsoluteSizePatch = (node, patch) => {
    if (!node || !patch?.style) return patch;
    if (node.positioning !== "absolute" || !node.absolutePos) return patch;

    const widthValue = parseSizeToNumber(patch.style.width);
    const heightValue = parseSizeToNumber(patch.style.height);
    if (widthValue === undefined && heightValue === undefined) return patch;

    const nextAbs = { ...(node.absolutePos || {}) };
    if (widthValue !== undefined) {
      nextAbs.w = Math.max(1, Math.round(widthValue));
    }
    if (heightValue !== undefined) {
      nextAbs.h = Math.max(1, Math.round(heightValue));
    }

    const baseLayoutItem = patch.layoutItem || node.layoutItem;
    let nextLayoutItem = baseLayoutItem;
    if (baseLayoutItem?.free?.abs) {
      nextLayoutItem = {
        ...(baseLayoutItem || {}),
        free: {
          ...(baseLayoutItem.free || {}),
          abs: {
            ...(baseLayoutItem.free.abs || {}),
            ...(widthValue !== undefined ? { w: nextAbs.w } : {}),
            ...(heightValue !== undefined ? { h: nextAbs.h } : {}),
          },
        },
      };
    }

    return {
      ...patch,
      absolutePos: nextAbs,
      ...(nextLayoutItem ? { layoutItem: nextLayoutItem } : {}),
    };
  };

  /**
   * 同步 ElLayout 的最小高度到绝对定位尺寸
   * @param {import('@/editor-core').ComponentNode | null} node - 当前节点
   * @param {Partial<import('@/editor-core').ComponentNode>} patch - 更新内容
   * @param {Record<string, any>} nextProps - 合并后的属性
   * @returns {Partial<import('@/editor-core').ComponentNode>}
   */
  const syncElLayoutMinHeightPatch = (node, patch, nextProps) => {
    if (!node || node.type !== "ElLayout") return patch;
    const nextPositioning = patch.positioning ?? node.positioning;
    const baseAbs =
      patch.absolutePos || node.absolutePos || node.layoutItem?.free?.abs;
    if (nextPositioning !== "absolute" || !baseAbs) return patch;
    const minHeight = resolveElLayoutMinHeight(nextProps);
    if (!minHeight) return patch;
    const currentHeight = Number(baseAbs.h) || 0;
    if (currentHeight >= minHeight) return patch;
    const nextAbs = { ...baseAbs, h: Math.max(1, minHeight) };
    const baseLayoutItem = patch.layoutItem || node.layoutItem;
    let nextLayoutItem = baseLayoutItem;
    if (baseLayoutItem?.free?.abs) {
      nextLayoutItem = {
        ...(baseLayoutItem || {}),
        free: {
          ...(baseLayoutItem.free || {}),
          abs: {
            ...(baseLayoutItem.free.abs || {}),
            h: nextAbs.h,
          },
        },
      };
    }
    return {
      ...patch,
      absolutePos: nextAbs,
      ...(nextLayoutItem ? { layoutItem: nextLayoutItem } : {}),
    };
  };

  /**
   * 限制容器最小尺寸，避免小于内部区域
   * @param {import('@/editor-core').ComponentNode | null} node - 当前节点
   * @param {Partial<import('@/editor-core').ComponentNode>} patch - 更新内容
   * @returns {Partial<import('@/editor-core').ComponentNode>}
   */
  const clampElContainerSizePatch = (node, patch) => {
    if (!node || node.type !== "ElContainer" || !patch?.style) return patch;

    const children = node.children || [];
    let hasHeader = false;
    let hasFooter = false;
    let hasAside = false;
    let hasMain = false;
    for (const childId of children) {
      const childNode = doc.value?.getNode?.(childId);
      if (!childNode) continue;
      if (childNode.type === "ElHeader") hasHeader = true;
      if (childNode.type === "ElFooter") hasFooter = true;
      if (childNode.type === "ElAside") hasAside = true;
      if (childNode.type === "ElMain") hasMain = true;
    }

    const props = node.props || {};
    if (typeof props.showHeader === "boolean") hasHeader = props.showHeader;
    if (typeof props.showFooter === "boolean") hasFooter = props.showFooter;
    if (typeof props.showAside === "boolean") hasAside = props.showAside;
    if (typeof props.showMain === "boolean") hasMain = props.showMain;

    const headerHeight = parseSizeToNumber(props.headerHeight) ?? 60;
    const footerHeight = parseSizeToNumber(props.footerHeight) ?? 60;
    const asideWidth = parseSizeToNumber(props.asideWidth) ?? 200;
    const minBodySize = 40;

    const hasBody = hasAside || hasMain;
    let minWidth = 0;
    if (hasAside && hasMain) {
      minWidth = asideWidth + minBodySize;
    } else if (hasAside) {
      minWidth = asideWidth;
    } else if (hasMain) {
      minWidth = minBodySize;
    }

    let minHeight = 0;
    if (hasHeader) minHeight += headerHeight;
    if (hasFooter) minHeight += footerHeight;
    if (hasBody) minHeight += minBodySize;

    const nextStyle = { ...(patch.style || {}) };
    const widthValue = parseSizeToNumber(nextStyle.width);
    const heightValue = parseSizeToNumber(nextStyle.height);
    if (widthValue !== undefined && minWidth > 0) {
      nextStyle.width = `${Math.max(widthValue, minWidth)}px`;
    }
    if (heightValue !== undefined && minHeight > 0) {
      nextStyle.height = `${Math.max(heightValue, minHeight)}px`;
    }

    return { ...patch, style: nextStyle };
  };

  /**
   * 限制 ElCol 的栅格总和不超过 24
   * @param {import('@/editor-core').ComponentNode | null} node - 当前节点
   * @param {Partial<import('@/editor-core').ComponentNode>} patch - 更新内容
   * @returns {Partial<import('@/editor-core').ComponentNode>}
   */
  const clampElColSpanPatch = (node, patch) => {
    if (!node || node.type !== "ElCol" || !patch?.props) return patch;
    if (!Object.prototype.hasOwnProperty.call(patch.props, "span"))
      return patch;
    const parentNode = doc.value?.getParent?.(node.id);
    if (!parentNode || parentNode.type !== "ElLayoutRow") return patch;

    const colIds = (parentNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElCol";
    });
    const anchorIndex = colIds.indexOf(node.id);
    const leftIds = anchorIndex >= 0 ? colIds.slice(0, anchorIndex) : [];
    const rightCount =
      anchorIndex >= 0 ? Math.max(0, colIds.length - anchorIndex - 1) : 0;
    const leftTotal = leftIds.reduce((sum, colId) => {
      const colNode = doc.value?.getNode?.(colId);
      const span = Number(colNode?.props?.span) || 0;
      return sum + Math.max(1, Math.min(24, span));
    }, 0);
    const maxSpan = Math.max(1, 24 - leftTotal - rightCount);
    const nextSpan = Number(patch.props.span) || 1;
    const clamped = Math.max(1, Math.min(maxSpan, nextSpan));
    if (clamped === nextSpan) return patch;
    return {
      ...patch,
      props: { ...(patch.props || {}), span: clamped },
    };
  };

  /**
   * 限制 ElCol 的偏移不挤出右侧最小栅格
   * @param {import('@/editor-core').ComponentNode | null} node - 当前节点
   * @param {Partial<import('@/editor-core').ComponentNode>} patch - 更新内容
   * @returns {Partial<import('@/editor-core').ComponentNode>}
   */
  const clampElColOffsetPatch = (node, patch) => {
    if (!node || node.type !== "ElCol" || !patch?.props) return patch;
    if (!Object.prototype.hasOwnProperty.call(patch.props, "offset"))
      return patch;
    const parentNode = doc.value?.getParent?.(node.id);
    if (!parentNode || parentNode.type !== "ElLayoutRow") return patch;
    const colIds = (parentNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElCol";
    });
    const anchorIndex = colIds.indexOf(node.id);
    if (anchorIndex < 0) return patch;

    const spans = colIds.map((colId) => {
      const colNode = doc.value?.getNode?.(colId);
      const span = Number(colNode?.props?.span) || 1;
      return Math.max(1, Math.min(24, span));
    });
    if (Object.prototype.hasOwnProperty.call(patch.props, "span")) {
      const nextSpan = Number(patch.props.span) || 1;
      spans[anchorIndex] = Math.max(1, Math.min(24, nextSpan));
    }

    const offsets = colIds.map((colId) => {
      const colNode = doc.value?.getNode?.(colId);
      const offset = Number(colNode?.props?.offset) || 0;
      return Math.max(0, Math.min(24, offset));
    });
    const nextOffset = Math.max(
      0,
      Math.min(24, Number(patch.props.offset) || 0),
    );
    offsets[anchorIndex] = nextOffset;

    const fixedSpanTotal = spans
      .slice(0, anchorIndex + 1)
      .reduce((sum, value) => sum + value, 0);
    const offsetOthers = offsets.reduce(
      (sum, value, index) => (index === anchorIndex ? sum : sum + value),
      0,
    );
    const rightCount = Math.max(0, colIds.length - anchorIndex - 1);
    const maxOffset = Math.max(
      0,
      24 - fixedSpanTotal - offsetOthers - rightCount,
    );
    const clampedOffset = Math.min(nextOffset, maxOffset);
    if (clampedOffset === nextOffset) return patch;
    return {
      ...patch,
      props: { ...(patch.props || {}), offset: clampedOffset },
    };
  };

  /**
   * 限制 ElCol 的 push/pull 不超出当前行宽度
   * @param {import('@/editor-core').ComponentNode | null} node - 当前节点
   * @param {Partial<import('@/editor-core').ComponentNode>} patch - 更新内容
   * @returns {Partial<import('@/editor-core').ComponentNode>}
   */
  const clampElColShiftPatch = (node, patch) => {
    if (!node || node.type !== "ElCol" || !patch?.props) return patch;
    const hasPush = Object.prototype.hasOwnProperty.call(patch.props, "push");
    const hasPull = Object.prototype.hasOwnProperty.call(patch.props, "pull");
    if (!hasPush && !hasPull) return patch;
    const parentNode = doc.value?.getParent?.(node.id);
    if (!parentNode || parentNode.type !== "ElLayoutRow") return patch;

    const mergedProps = {
      ...(node.props || {}),
      ...(patch.props || {}),
    };
    const span = Math.max(1, Math.min(24, Number(mergedProps.span) || 1));
    const offset = Math.max(0, Math.min(24, Number(mergedProps.offset) || 0));
    const prevPush = Math.max(0, Math.min(24, Number(node.props?.push) || 0));
    const prevPull = Math.max(0, Math.min(24, Number(node.props?.pull) || 0));
    const push = Math.max(0, Math.min(24, Number(mergedProps.push) || 0));
    const pull = Math.max(0, Math.min(24, Number(mergedProps.pull) || 0));
    const colIds = (parentNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElCol";
    });
    const colIndex = colIds.indexOf(node.id);
    let leftEdge = offset;
    if (colIndex > 0) {
      leftEdge = colIds.slice(0, colIndex).reduce((sum, colId) => {
        const colNode = doc.value?.getNode?.(colId);
        if (!colNode) return sum;
        const colSpan = Number(colNode.props?.span) || 1;
        const colOffset = Number(colNode.props?.offset) || 0;
        return (
          sum +
          Math.max(1, Math.min(24, colSpan)) +
          Math.max(0, Math.min(24, colOffset))
        );
      }, 0);
      leftEdge += offset;
    }
    const minShift = -leftEdge;
    const maxShift = 24 - leftEdge - span;
    const desiredShift = push - pull;
    const clampedShift = Math.min(maxShift, Math.max(minShift, desiredShift));

    const changedPush = hasPush && push !== prevPush;
    const changedPull = hasPull && pull !== prevPull;
    let nextPush = push;
    let nextPull = pull;
    if (changedPush && !changedPull) {
      nextPush = Math.max(0, clampedShift + pull);
      nextPull = pull;
    } else if (changedPull && !changedPush) {
      nextPull = Math.max(0, push - clampedShift);
      nextPush = push;
    } else if (changedPush && changedPull) {
      nextPush = Math.max(0, clampedShift + pull);
      nextPull = pull;
    } else {
      return patch;
    }

    if (nextPush === push && nextPull === pull) return patch;
    return {
      ...patch,
      props: { ...(patch.props || {}), push: nextPush, pull: nextPull },
    };
  };

  /**
   * 限制 Layout 行列数范围
   * @param {import('@/editor-core').ComponentNode} node - 当前节点
   * @param {Partial<import('@/editor-core').ComponentNode>} patch - 更新内容
   * @returns {Partial<import('@/editor-core').ComponentNode>}
   */
  const clampElLayoutRowColumnsPatch = (node, patch) => {
    if (!node || node.type !== "ElLayoutRow" || !patch?.props) return patch;
    if (!Object.prototype.hasOwnProperty.call(patch.props, "columns"))
      return patch;
    const raw = Number(patch.props.columns);
    if (!Number.isFinite(raw)) return patch;
    const clamped = Math.max(1, Math.min(24, raw));
    if (clamped === raw) return patch;
    return {
      ...patch,
      props: { ...(patch.props || {}), columns: clamped },
    };
  };

  /**
   * 更新组件节点
   * @param {string} nodeId - 节点 ID
   * @param {Partial<import('@/editor-core').ComponentNode>} patch - 更新内容
   * @returns {boolean}
   */
  const updateNode = (nodeId, patch) => {
    if (!doc.value || !history.value || !nodeId) return false;
    if (!ensureEditable()) return false;

    const node = doc.value.getNode(nodeId);
    const limitedPatch = clampElContainerSizePatch(node, patch);
    const clampedColumnsPatch = clampElLayoutRowColumnsPatch(
      node,
      limitedPatch,
    );
    const clampedSpanPatch = clampElColSpanPatch(node, clampedColumnsPatch);
    const clampedOffsetPatch = clampElColOffsetPatch(node, clampedSpanPatch);
    const clampedShiftPatch = clampElColShiftPatch(node, clampedOffsetPatch);
    const nextPatch = syncAbsoluteSizePatch(node, clampedShiftPatch);
    const hasPropPatch = nextPatch?.props && typeof nextPatch === "object";
    const mergedProps = hasPropPatch
      ? { ...(node.props || {}), ...(nextPatch.props || {}) }
      : { ...(node.props || {}) };
    const mergedPatch = hasPropPatch
      ? { ...nextPatch, props: mergedProps }
      : nextPatch;
    const layoutAdjustedPatch = hasPropPatch
      ? syncElLayoutMinHeightPatch(node, mergedPatch, mergedProps || {})
      : mergedPatch;
    const shouldSyncContainer = node?.type === "ElContainer" && hasPropPatch;
    const shouldSyncLayout = node?.type === "ElLayout" && hasPropPatch;
    const shouldSyncLayoutRow = node?.type === "ElLayoutRow" && hasPropPatch;
    const shouldSyncLayoutRowFromCol =
      node?.type === "ElCol" &&
      hasPropPatch &&
      (Object.prototype.hasOwnProperty.call(nextPatch.props, "offset") ||
        Object.prototype.hasOwnProperty.call(nextPatch.props, "span"));

    if (
      !shouldSyncContainer &&
      !shouldSyncLayout &&
      !shouldSyncLayoutRow &&
      !shouldSyncLayoutRowFromCol
    ) {
      history.value.execute(new UpdateNodeCommand(nodeId, layoutAdjustedPatch));
      return true;
    }

    const finalPatch = layoutAdjustedPatch;
    const shouldCommit = !history.value.isInTransaction?.();
    if (shouldCommit) {
      history.value.beginTransaction();
    }

    history.value.executeInTransaction(
      new UpdateNodeCommand(nodeId, finalPatch),
    );
    if (shouldSyncContainer) {
      syncElContainerSections(nodeId, mergedProps || {});
    }
    if (shouldSyncLayout) {
      syncElLayoutRows(nodeId, mergedProps || {});
    }
    if (
      shouldSyncLayoutRow &&
      Object.prototype.hasOwnProperty.call(nextPatch.props || {}, "columns")
    ) {
      syncElLayoutRowColumns(nodeId, mergedProps || {}, {
        forceSpanUpdate: true,
      });
    }
    if (shouldSyncLayoutRowFromCol) {
      const parentNode = doc.value.getParent(nodeId);
      if (parentNode?.type === "ElLayoutRow") {
        syncElLayoutRowColumns(parentNode.id, parentNode.props || {}, {
          forceSpanUpdate: true,
          anchorColId: nodeId,
        });
      }
    }

    if (shouldCommit) {
      history.value.commitTransaction("更新容器布局");
    }
    const executeParentUpdate = (parentId, patch) => {
      if (history.value.isInTransaction?.()) {
        history.value.executeInTransaction(
          new UpdateNodeCommand(parentId, patch),
        );
        return;
      }
      history.value.execute(new UpdateNodeCommand(parentId, patch));
    };
    if (node?.type === "ElHeader") {
      const parentNode = doc.value.getParent(nodeId);
      if (parentNode?.type === "ElContainer" && mergedProps.height) {
        executeParentUpdate(parentNode.id, {
          props: {
            ...(parentNode.props || {}),
            headerHeight: mergedProps.height,
          },
        });
      }
    }
    if (node?.type === "ElFooter") {
      const parentNode = doc.value.getParent(nodeId);
      if (parentNode?.type === "ElContainer" && mergedProps.height) {
        executeParentUpdate(parentNode.id, {
          props: {
            ...(parentNode.props || {}),
            footerHeight: mergedProps.height,
          },
        });
      }
    }
    if (node?.type === "ElAside") {
      const parentNode = doc.value.getParent(nodeId);
      if (parentNode?.type === "ElContainer" && mergedProps.width) {
        executeParentUpdate(parentNode.id, {
          props: {
            ...(parentNode.props || {}),
            asideWidth: mergedProps.width,
          },
        });
      }
    }
    return true;
  };

  /**
   * 更新图形节点
   * @param {string} graphicId - 图形 ID
   * @param {Partial<import('@/editor-core').GraphicNode>} patch - 更新内容
   * @returns {boolean}
   */
  const updateGraphic = (graphicId, patch) => {
    if (!doc.value || !history.value || !graphicId) return false;
    if (!ensureEditable()) return false;

    history.value.execute(new UpdateGraphicCommand(graphicId, patch));
    return true;
  };

  /**
   * 撤销
   * @returns {boolean}
   */
  const undo = () => {
    if (readonlyState.value?.readonly) return false;
    return history.value?.undo() || false;
  };

  /**
   * 重做
   * @returns {boolean}
   */
  const redo = () => {
    if (readonlyState.value?.readonly) return false;
    return history.value?.redo() || false;
  };

  /**
   * 当前页面
   */
  const currentPage = computed(() => {
    if (!doc.value || !currentPageId.value) return null;
    return doc.value.getPage(currentPageId.value);
  });

  /**
   * 切换当前页面
   * @param {string} pageId - 页面 ID
   * @returns {Promise<{ok: boolean, error?: Error}>}
   */
  const setCurrentPage = async (pageId) => {
    if (!pageId || pageId === currentPageId.value) {
      return { ok: true };
    }
    return loadPage(pageId);
  };

  /**
   * 解析默认尺寸
   * @param {string} type - 组件类型
   * @param {Object | undefined} manifest - 组件清单
   * @returns {{ width: number, height: number }}
   */
  const resolveDefaultSize = (type, manifest) => {
    if (manifest?.defaultSize) {
      return {
        width: manifest.defaultSize.width || 120,
        height: manifest.defaultSize.height || 32,
      };
    }

    const sizeMap = {
      FlexContainer: { width: 360, height: 200 },
      FreeContainer: { width: 360, height: 200 },
      GridContainer: { width: 360, height: 200 },
      ElContainer: { width: 360, height: 240 },
      ElLayout: { width: 360, height: 200 },
      Text: { width: 120, height: 32 },
      Button: { width: 120, height: 36 },
    };

    return sizeMap[type] || { width: 160, height: 80 };
  };

  /**
   * 判断是否为布局容器
   * @param {string} type - 组件类型
   * @returns {boolean}
   */
  const isLayoutContainerType = (type) => {
    return [
      "FlexContainer",
      "GridContainer",
      "FreeContainer",
      "ResponsiveLayout",
      "ColumnLayout1",
      "ColumnLayout2",
      "ColumnLayout4",
      "ElContainer",
      "ElLayout",
      "ElLayoutRow",
    ].includes(type);
  };

  /**
   * 解析 Grid 列数
   * @param {string | number | undefined} value - 列配置
   * @returns {number}
   */
  const resolveGridCount = (value) => {
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
  };

  /**
   * 构建布局配置
   * @param {import('@/editor-core').ComponentNode | null} parentNode - 父节点
   * @param {{ x: number, y: number, width: number, height: number }} dropInfo - 放置信息
   * @returns {import('@/editor-core').LayoutItem | null}
   */
  const buildLayoutItem = (parentNode, dropInfo) => {
    if (!parentNode) {
      return buildFreeLayoutItem(dropInfo);
    }

    if (
      parentNode.type === "GridContainer" ||
      parentNode.type === "ColumnLayout1" ||
      parentNode.type === "ColumnLayout2" ||
      parentNode.type === "ColumnLayout4"
    ) {
      return buildGridLayoutItem(parentNode);
    }

    if (
      parentNode.type === "FlexContainer" ||
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

    if (parentNode.type === "FreeContainer") {
      return buildFreeLayoutItem(dropInfo);
    }

    return buildFreeLayoutItem(dropInfo);
  };

  /**
   * 构建自由布局配置
   * @param {{ x: number, y: number, width: number, height: number }} dropInfo - 放置信息
   * @returns {import('@/editor-core').LayoutItem}
   */
  const buildFreeLayoutItem = (dropInfo) => {
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
  };

  /**
   * 构建 Flex 布局配置
   * @returns {import('@/editor-core').LayoutItem}
   */
  const buildFlexLayoutItem = () => {
    return {
      flex: {
        grow: 0,
        shrink: 0,
        basis: "auto",
      },
    };
  };

  /**
   * 构建 Grid 布局配置
   * @param {import('@/editor-core').ComponentNode} parentNode - 父节点
   * @returns {import('@/editor-core').LayoutItem}
   */
  const buildGridLayoutItem = (parentNode) => {
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
  };

  /**
   * 插入组件节点
   * @param {string} type - 组件类型
   * @param {string} parentId - 父节点 ID
   * @param {number | undefined} index - 插入索引
   * @param {{ dropPosition?: { x: number, y: number } }} [options] - 插入选项
   * @returns {import('@/editor-core').ComponentNode | null}
   */
  const insertNode = (type, parentId, index, options = {}) => {
    if (!doc.value || !history.value || !parentId) return null;
    if (!ensureEditable()) return null;

    let parentNode = doc.value.getNode(parentId);
    if (!parentNode) return null;
    let resolvedParentId = parentId;
    const isElContainerRegionType = (nodeType) =>
      nodeType === "ElHeader" ||
      nodeType === "ElAside" ||
      nodeType === "ElMain" ||
      nodeType === "ElFooter";
    let shouldReplaceRegionChildren = false;
    if (parentNode.type === "ElContainer" && !isElContainerRegionType(type)) {
      const mainChildId = (parentNode.children || []).find((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElMain";
      });
      if (mainChildId) {
        const mainNode = doc.value.getNode(mainChildId);
        if (mainNode) {
          parentNode = mainNode;
          resolvedParentId = mainNode.id;
          shouldReplaceRegionChildren = true;
        }
      }
    }
    if (parentNode.type === "ElCol" && (parentNode.children || []).length > 0) {
      return null;
    }

    const manifest = componentRegistry.get(type);
    const defaultSize = resolveDefaultSize(type, manifest);
    const insertIndex = Number.isInteger(index)
      ? index
      : (parentNode.children?.length ?? 0);

    const dropPosition = options.dropPosition || { x: 0, y: 0 };
    const dropInfo = {
      x: dropPosition.x ?? 0,
      y: dropPosition.y ?? 0,
      width: defaultSize.width,
      height: defaultSize.height,
    };

    const isLayoutContainer = isLayoutContainerType(type);
    let layoutItem = null;
    let nodeStyle = { ...(manifest?.defaultStyle || {}) };

    if (isLayoutContainer) {
      layoutItem = buildFlexLayoutItem();
      nodeStyle = {
        ...nodeStyle,
        width: "100%",
        minHeight: "120px",
      };
    } else {
      layoutItem = buildLayoutItem(parentNode, dropInfo);
    }

    const isRegionContainer = [
      "ElHeader",
      "ElAside",
      "ElMain",
      "ElFooter",
      "ElCol",
    ].includes(parentNode.type);
    const isTabsContainer = parentNode.type === "Tabs";
    if (isRegionContainer) {
      // 区域容器内默认填满
      if (parentNode.type === "ElCol") {
        nodeStyle = {
          ...nodeStyle,
          width: "100%",
        };
        if (isLayoutContainer || manifest?.isContainer) {
          nodeStyle.height = "100%";
        }
      } else {
        nodeStyle = {
          ...nodeStyle,
          width: "100%",
          height: "100%",
        };
      }
      if (isLayoutContainer) {
        delete nodeStyle.minHeight;
        delete nodeStyle.minWidth;
      }
    }
    if (isTabsContainer) {
      nodeStyle = {
        ...nodeStyle,
        width: "100%",
        height: "100%",
      };
      if (isLayoutContainer) {
        delete nodeStyle.minHeight;
        delete nodeStyle.minWidth;
      }
    }

    const baseLabel = manifest?.name || type;
    const existingLabels = new Set();
    const rootId = currentPage.value?.rootNodeId;
    if (rootId && doc.value) {
      const stack = [rootId];
      while (stack.length) {
        const id = stack.pop();
        const current = doc.value.getNode(id);
        if (!current) continue;
        if (current.label) existingLabels.add(current.label);
        if (Array.isArray(current.children)) {
          stack.push(...current.children);
        }
      }
    }
    let uniqueLabel = `${baseLabel}1`;
    let labelIndex = 2;
    while (existingLabels.has(uniqueLabel)) {
      uniqueLabel = `${baseLabel}${labelIndex}`;
      labelIndex += 1;
    }

    const node = createComponentNode(type, {
      label: uniqueLabel,
      props: { ...(manifest?.defaultProps || {}) },
      style: nodeStyle,
      layoutItem,
    });
    if (type === "Menu") {
      node.detailConfig = getMenuDefaultDetailConfig();
      node.props = { ...(node.props || {}), ...getMenuDefaultProps() };
    }
    if (type === "ElLayoutRow") {
      const rawColumns = Number(node.props?.columns);
      const normalizedColumns =
        Number.isFinite(rawColumns) && rawColumns > 0
          ? Math.min(24, Math.floor(rawColumns))
          : 3;
      node.props = { ...(node.props || {}), columns: normalizedColumns };
    }
    if (parentNode.type === "FreeContainer") {
      node.positioning = "absolute";
      node.absolutePos = {
        x: Math.max(0, Math.round(dropInfo.x)),
        y: Math.max(0, Math.round(dropInfo.y)),
        w: dropInfo.width,
        h: dropInfo.height,
        z: 1,
      };
    } else if (
      parentNode.type === "FlexContainer" ||
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
      node.positioning = "flow";
      if (layoutItem?.flex) {
        node.flowLayout = { ...layoutItem.flex };
      }
    } else if (
      parentNode.type === "GridContainer" ||
      parentNode.type === "ColumnLayout1" ||
      parentNode.type === "ColumnLayout2" ||
      parentNode.type === "ColumnLayout4"
    ) {
      node.positioning = "flow";
      if (layoutItem?.grid) {
        node.flowLayout = { ...layoutItem.grid };
      }
    }

    const shouldWrapTransaction =
      (type === "ElContainer" ||
        type === "ElLayout" ||
        type === "ElLayoutRow") &&
      !history.value.isInTransaction?.();
    if (shouldWrapTransaction) {
      history.value.beginTransaction();
    }

    if (type === "ElContainer") {
      history.value.executeInTransaction(
        new InsertNodeCommand(parentId, insertIndex, node),
      );
      syncElContainerSections(node.id, node.props || {});
    } else if (type === "ElLayout") {
      history.value.executeInTransaction(
        new InsertNodeCommand(parentId, insertIndex, node),
      );
      syncElLayoutRows(node.id, node.props || {});
    } else if (type === "ElLayoutRow") {
      history.value.executeInTransaction(
        new InsertNodeCommand(parentId, insertIndex, node),
      );
      syncElLayoutRowColumns(node.id, node.props || {}, {
        forceSpanUpdate: true,
      });
    } else {
      const shouldCommit =
        shouldReplaceRegionChildren && !history.value.isInTransaction?.();
      if (shouldCommit) {
        history.value.beginTransaction();
      }
      if (shouldReplaceRegionChildren) {
        const existingChildren = [...(parentNode.children || [])];
        for (const childId of existingChildren) {
          history.value.executeInTransaction?.(new RemoveNodeCommand(childId));
          if (!history.value.executeInTransaction) {
            history.value.execute(new RemoveNodeCommand(childId));
          }
        }
      }
      history.value.execute(
        new InsertNodeCommand(resolvedParentId, insertIndex, node),
      );
      if (shouldCommit) {
        history.value.commitTransaction("更新Main区域");
      }
    }

    if (shouldWrapTransaction) {
      history.value.commitTransaction("插入容器布局");
    }
    selection.value?.select(createSelectableElement("node", node.id));

    return node;
  };

  /**
   * 在 ElLayoutRow 中按左右插入列
   * @param {"left" | "right"} direction - 插入方向
   * @param {string} [colId] - 参考列节点 ID
   * @returns {boolean}
   */
  const insertElColByDirection = (direction, colId) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;

    const targetId = colId || resolveLayerTarget();
    if (!targetId) return false;
    const targetNode = doc.value.getNode(targetId);
    if (targetNode?.type !== "ElCol") return false;

    const rowNode = doc.value.getParent(targetId);
    if (!rowNode || rowNode.type !== "ElLayoutRow") return false;

    const children = [...(rowNode.children || [])];
    const currentIndex = children.indexOf(targetId);
    if (currentIndex < 0) return false;

    const colCount = children.filter((childId) => {
      const childNodeItem = doc.value.getNode(childId);
      return childNodeItem?.type === "ElCol";
    }).length;
    if (colCount >= 24) return false;

    const manifest = componentRegistry.get("ElCol");
    const childNode = createComponentNode("ElCol", {
      parentNode: rowNode,
      label: buildElLayoutUniqueLabel(manifest?.name || "Col"),
      props: { ...(manifest?.defaultProps || {}) },
      style: { ...(manifest?.defaultStyle || {}) },
      layoutItem: buildFlexLayoutItem(),
    });

    const insertIndex = direction === "left" ? currentIndex : currentIndex + 1;
    const shouldCommit = !history.value.isInTransaction?.();
    if (shouldCommit) {
      history.value.beginTransaction();
    }

    history.value.executeInTransaction(
      new InsertNodeCommand(rowNode.id, insertIndex, childNode),
    );
    const nextColumns = Math.max(1, colCount + 1);
    updateNode(rowNode.id, {
      props: { ...(rowNode.props || {}), columns: nextColumns },
    });

    selection.value?.select(createSelectableElement("node", childNode.id));
    if (shouldCommit) {
      history.value.commitTransaction("新增布局列");
    }
    return true;
  };

  /**
   * 在当前列左侧新增一列
   * @param {string} [colId] - 参考列节点 ID
   * @returns {boolean}
   */
  const insertElColLeft = (colId) => {
    return insertElColByDirection("left", colId);
  };

  /**
   * 在当前列右侧新增一列
   * @param {string} [colId] - 参考列节点 ID
   * @returns {boolean}
   */
  const insertElColRight = (colId) => {
    return insertElColByDirection("right", colId);
  };

  /**
   * 在 ElLayoutRow 上下插入行
   * @param {"up" | "down"} direction - 插入方向
   * @param {string} [rowId] - 参考行节点 ID
   * @returns {boolean}
   */
  const insertElLayoutRowByDirection = (direction, rowId) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;

    const targetId = rowId || resolveLayerTarget();
    if (!targetId) return false;
    const targetNode = doc.value.getNode(targetId);
    if (targetNode?.type !== "ElLayoutRow") return false;

    const layoutNode = doc.value.getParent(targetId);
    if (!layoutNode || layoutNode.type !== "ElLayout") return false;

    const children = [...(layoutNode.children || [])];
    const currentIndex = children.indexOf(targetId);
    if (currentIndex < 0) return false;

    const manifest = componentRegistry.get("ElLayoutRow");
    const rowNode = createComponentNode("ElLayoutRow", {
      parentNode: layoutNode,
      label: buildElLayoutUniqueLabel(manifest?.name || "行"),
      props: { ...(manifest?.defaultProps || {}) },
      style: { ...(manifest?.defaultStyle || {}) },
      layoutItem: buildFlexLayoutItem(),
    });
    rowNode.props = { ...(rowNode.props || {}), columns: 1 };

    const insertIndex = direction === "up" ? currentIndex : currentIndex + 1;
    const shouldCommit = !history.value.isInTransaction?.();
    if (shouldCommit) {
      history.value.beginTransaction();
    }

    history.value.executeInTransaction(
      new InsertNodeCommand(layoutNode.id, insertIndex, rowNode),
    );
    syncElLayoutRowColumns(rowNode.id, rowNode.props || {}, {
      forceSpanUpdate: true,
    });

    const refreshedLayout = doc.value.getNode(layoutNode.id);
    const rowCount = (refreshedLayout?.children || []).filter((childId) => {
      const childNode = doc.value.getNode(childId);
      return childNode?.type === "ElLayoutRow";
    }).length;
    const nextRows = Math.max(1, rowCount);
    if ((refreshedLayout?.props?.rows || 0) !== nextRows) {
      updateNode(layoutNode.id, {
        props: { ...(refreshedLayout?.props || {}), rows: nextRows },
      });
    }

    selection.value?.select(createSelectableElement("node", rowNode.id));
    if (shouldCommit) {
      history.value.commitTransaction("新增布局行");
    }
    return true;
  };

  /**
   * 在当前行上方新增一行
   * @param {string} [rowId] - 参考行节点 ID
   * @returns {boolean}
   */
  const insertElLayoutRowUp = (rowId) => {
    return insertElLayoutRowByDirection("up", rowId);
  };

  /**
   * 在当前行下方新增一行
   * @param {string} [rowId] - 参考行节点 ID
   * @returns {boolean}
   */
  const insertElLayoutRowDown = (rowId) => {
    return insertElLayoutRowByDirection("down", rowId);
  };

  /**
   * 校验当前页面组件名称是否唯一
   * @param {string} name - 组件名称
   * @param {string} [excludeId] - 排除的节点ID
   * @returns {boolean}
   */
  const isLabelUnique = (name, excludeId) => {
    if (!doc.value || !currentPage.value?.rootNodeId) return true;
    const stack = [currentPage.value.rootNodeId];
    while (stack.length) {
      const id = stack.pop();
      const node = doc.value.getNode(id);
      if (!node) continue;
      if (node.label === name && node.id !== excludeId) {
        return false;
      }
      if (Array.isArray(node.children)) {
        stack.push(...node.children);
      }
    }
    return true;
  };

  /**
   * 删除选中的节点
   * @returns {boolean}
   */
  const removeSelectedNodes = () => {
    if (!doc.value || !history.value || !selection.value) return false;
    if (!ensureEditable()) return false;

    const selected = selection.value.getSelectedElements?.() || [];
    const nodeIds = selected
      .filter((el) => el.kind === "node")
      .map((el) => el.id);

    if (!nodeIds.length) return false;

    const rootId = currentPage.value?.rootNodeId;
    const nodeSet = new Set(nodeIds);
    const deletable = nodeIds.filter((nodeId) => {
      if (nodeId === rootId) return false;
      const ancestors = doc.value.getAncestors(nodeId) || [];
      return !ancestors.some((ancestor) => nodeSet.has(ancestor.id));
    });

    if (!deletable.length) return false;

    const affectedRowIds = new Set();
    const affectedLayoutIds = new Set();
    deletable.forEach((nodeId) => {
      const node = doc.value.getNode(nodeId);
      if (node?.type !== "ElCol") return;
      const parentNode = doc.value.getParent(nodeId);
      if (parentNode?.type === "ElLayoutRow") {
        affectedRowIds.add(parentNode.id);
      }
    });
    deletable.forEach((nodeId) => {
      const node = doc.value.getNode(nodeId);
      if (node?.type !== "ElLayoutRow") return;
      const parentNode = doc.value.getParent(nodeId);
      if (parentNode?.type === "ElLayout") {
        affectedLayoutIds.add(parentNode.id);
      }
    });
    const shouldWrapTransaction =
      (affectedRowIds.size > 0 || affectedLayoutIds.size > 0) &&
      !history.value.isInTransaction?.();
    if (shouldWrapTransaction) {
      history.value.beginTransaction();
    }

    deletable.forEach((nodeId) => {
      history.value.execute(new RemoveNodeCommand(nodeId));
    });
    affectedRowIds.forEach((rowId) => {
      const rowNode = doc.value.getNode(rowId);
      if (!rowNode) return;
      const colCount = (rowNode.children || []).filter((childId) => {
        const childNode = doc.value.getNode(childId);
        return childNode?.type === "ElCol";
      }).length;
      const nextColumns = Math.max(1, colCount || 1);
      if ((rowNode.props?.columns || 0) !== nextColumns) {
        updateNode(rowId, {
          props: { ...(rowNode.props || {}), columns: nextColumns },
        });
      }
    });
    affectedLayoutIds.forEach((layoutId) => {
      const layoutNode = doc.value.getNode(layoutId);
      if (!layoutNode) return;
      const rowCount = (layoutNode.children || []).filter((childId) => {
        const childNode = doc.value.getNode(childId);
        return childNode?.type === "ElLayoutRow";
      }).length;
      const nextRows = Math.max(1, rowCount || 1);
      if ((layoutNode.props?.rows || 0) !== nextRows) {
        updateNode(layoutId, {
          props: { ...(layoutNode.props || {}), rows: nextRows },
        });
      }
    });
    if (shouldWrapTransaction) {
      history.value.commitTransaction("调整布局列");
    }
    selection.value.clearSelection?.();
    return true;
  };

  /**
   * 删除指定节点
   * @param {string} nodeId - 节点 ID
   * @returns {boolean}
   */
  const removeNode = (nodeId) => {
    if (!doc.value || !history.value || !nodeId) return false;
    if (!ensureEditable()) return false;

    const rootId = currentPage.value?.rootNodeId;
    if (nodeId === rootId) return false;

    const node = doc.value.getNode(nodeId);
    const colParentNode =
      node?.type === "ElCol" ? doc.value.getParent(nodeId) : null;
    const rowParentNode =
      node?.type === "ElLayoutRow" ? doc.value.getParent(nodeId) : null;
    const shouldSyncRow = colParentNode?.type === "ElLayoutRow";
    const shouldSyncLayout = rowParentNode?.type === "ElLayout";
    const shouldWrapTransaction =
      (shouldSyncRow || shouldSyncLayout) && !history.value.isInTransaction?.();
    if (shouldWrapTransaction) {
      history.value.beginTransaction();
    }

    history.value.execute(new RemoveNodeCommand(nodeId));
    if (shouldSyncRow) {
      const rowNode = doc.value.getNode(colParentNode.id);
      if (rowNode) {
        const colCount = (rowNode.children || []).filter((childId) => {
          const childNode = doc.value.getNode(childId);
          return childNode?.type === "ElCol";
        }).length;
        const nextColumns = Math.max(1, colCount || 1);
        if ((rowNode.props?.columns || 0) !== nextColumns) {
          updateNode(rowNode.id, {
            props: { ...(rowNode.props || {}), columns: nextColumns },
          });
        }
      }
    }
    if (shouldSyncLayout) {
      const layoutNode = doc.value.getNode(rowParentNode.id);
      if (layoutNode) {
        const rowCount = (layoutNode.children || []).filter((childId) => {
          const childNode = doc.value.getNode(childId);
          return childNode?.type === "ElLayoutRow";
        }).length;
        const nextRows = Math.max(1, rowCount || 1);
        if ((layoutNode.props?.rows || 0) !== nextRows) {
          updateNode(layoutNode.id, {
            props: { ...(layoutNode.props || {}), rows: nextRows },
          });
        }
      }
    }
    if (shouldWrapTransaction) {
      history.value.commitTransaction("调整布局列");
    }
    selection.value?.clearSelection?.();
    return true;
  };

  /**
   * 获取用于图层操作的节点 ID
   * @param {string | undefined} nodeId - 指定节点
   * @returns {string}
   */
  const resolveLayerTarget = (nodeId) => {
    if (nodeId) return nodeId;
    const primary = selection.value?.getPrimaryElement?.();
    return primary?.kind === "node" ? primary.id : "";
  };

  /**
   * 上移节点
   * @param {string} [nodeId] - 节点 ID
   * @returns {boolean}
   */
  const moveNodeUp = (nodeId) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;

    const targetId = resolveLayerTarget(nodeId);
    if (!targetId) return false;
    if (targetId === currentPage.value?.rootNodeId) return false;

    history.value.execute(new ReorderNodeCommand(targetId, "up"));
    return true;
  };

  /**
   * 下移节点
   * @param {string} [nodeId] - 节点 ID
   * @returns {boolean}
   */
  const moveNodeDown = (nodeId) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;

    const targetId = resolveLayerTarget(nodeId);
    if (!targetId) return false;
    if (targetId === currentPage.value?.rootNodeId) return false;

    history.value.execute(new ReorderNodeCommand(targetId, "down"));
    return true;
  };

  /**
   * 置顶节点
   * @param {string} [nodeId] - 节点 ID
   * @returns {boolean}
   */
  const moveNodeToTop = (nodeId) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;

    const targetId = resolveLayerTarget(nodeId);
    if (!targetId) return false;
    if (targetId === currentPage.value?.rootNodeId) return false;

    history.value.execute(new ReorderNodeCommand(targetId, "top"));
    return true;
  };

  /**
   * 置底节点
   * @param {string} [nodeId] - 节点 ID
   * @returns {boolean}
   */
  const moveNodeToBottom = (nodeId) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;

    const targetId = resolveLayerTarget(nodeId);
    if (!targetId) return false;
    if (targetId === currentPage.value?.rootNodeId) return false;

    history.value.execute(new ReorderNodeCommand(targetId, "bottom"));
    return true;
  };

  /**
   * 切换节点可见性
   * @param {string} nodeId - 节点 ID
   * @returns {boolean}
   */
  const toggleNodeVisibility = (nodeId) => {
    if (!doc.value || !history.value || !nodeId) return false;
    if (!ensureEditable()) return false;

    history.value.execute(new ToggleNodeVisibilityCommand(nodeId));
    return true;
  };

  /**
   * 切换节点锁定状态
   * @param {string} nodeId - 节点 ID
   * @returns {boolean}
   */
  const toggleNodeLock = (nodeId) => {
    if (!doc.value || !history.value || !nodeId) return false;
    if (!ensureEditable()) return false;

    history.value.execute(new ToggleNodeLockCommand(nodeId));
    return true;
  };

  /**
   * 对齐选中元素
   * @param {'left' | 'centerH' | 'right' | 'top' | 'centerV' | 'bottom'} alignType
   * @returns {boolean}
   */
  const alignElements = (alignType) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;
    const elements = selection.value?.getSelectedElements?.() || [];
    if (elements.length < 2) return false;
    history.value.execute(new AlignElementsCommand(elements, alignType));
    return true;
  };

  /**
   * 等距分布选中元素
   * @param {'horizontal' | 'vertical'} direction
   * @returns {boolean}
   */
  const distributeElements = (direction) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;
    const elements = selection.value?.getSelectedElements?.() || [];
    if (elements.length < 3) return false;
    history.value.execute(new DistributeElementsCommand(elements, direction));
    return true;
  };

  /**
   * 统一选中元素尺寸（以主选中元素为基准）
   * @param {'width' | 'height' | 'both'} mode
   * @returns {boolean}
   */
  const matchElementSize = (mode) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;
    const elements = selection.value?.getSelectedElements?.() || [];
    if (elements.length < 2) return false;
    const primary = selection.value?.getPrimaryElement?.();
    if (!primary) return false;
    history.value.execute(new MatchSizeCommand(elements, mode, primary.id));
    return true;
  };

  /**
   * 按偏移量移动选中的节点（支持方向键 + Shift 微调）
   * @param {number} dx - 水平偏移
   * @param {number} dy - 垂直偏移
   * @returns {boolean}
   */
  const moveSelectedByDelta = (dx, dy) => {
    if (!doc.value || !history.value || !selection.value) return false;
    if (!ensureEditable()) return false;

    const elements = selection.value.getSelectedElements?.() || [];
    const nodeElements = elements.filter((el) => el.kind === "node");
    if (!nodeElements.length) return false;

    const commands = [];
    for (const el of nodeElements) {
      const node = doc.value.getNode(el.id);
      if (!node || node.positioning === "flow") continue;

      const p = node.absolutePos;
      const a = node.layoutItem?.free?.abs;
      let patch = null;

      if (
        p &&
        (node.positioning === "absolute" ||
          Number.isFinite(p.x) ||
          Number.isFinite(p.y))
      ) {
        const newX = (Number.isFinite(p.x) ? p.x : 0) + dx;
        const newY = (Number.isFinite(p.y) ? p.y : 0) + dy;
        patch = { absolutePos: { ...p, x: newX, y: newY } };
      } else if (a) {
        const newX = (Number.isFinite(a.x) ? a.x : 0) + dx;
        const newY = (Number.isFinite(a.y) ? a.y : 0) + dy;
        const nextLayoutItem = JSON.parse(
          JSON.stringify(node.layoutItem || {}),
        );
        if (!nextLayoutItem.free) nextLayoutItem.free = {};
        if (!nextLayoutItem.free.abs) nextLayoutItem.free.abs = {};
        nextLayoutItem.free.abs = {
          ...nextLayoutItem.free.abs,
          x: newX,
          y: newY,
        };
        patch = { layoutItem: nextLayoutItem };
      } else if (node.style) {
        const left = typeof node.style.left === "number" ? node.style.left : 0;
        const top = typeof node.style.top === "number" ? node.style.top : 0;
        patch = { style: { ...node.style, left: left + dx, top: top + dy } };
      }

      if (patch) {
        commands.push(new UpdateNodeCommand(el.id, patch));
      }
    }

    if (!commands.length) return false;
    history.value.execute(new BatchCommand(commands, "移动选中节点"));
    return true;
  };

  /** @type {import('vue').Ref<Array|null>} 内存剪贴板 */
  const _clipboard = ref(null);

  /** 剪贴板是否有内容 */
  const hasClipboard = computed(() => !!_clipboard.value?.length);

  /**
   * 复制选中元素到内存剪贴板
   * @returns {boolean}
   */
  const copyNodes = () => {
    if (!doc.value || !selection.value) return false;
    const elements = selection.value.getSelectedElements?.() || [];
    if (!elements.length) return false;

    const snapshots = [];
    for (const el of elements) {
      if (el.kind === "node") {
        const node = doc.value.getNode(el.id);
        if (node)
          snapshots.push({
            kind: "node",
            data: JSON.parse(JSON.stringify(node)),
          });
      } else if (el.kind === "graphic") {
        const graphic = doc.value.getGraphic(el.id);
        if (graphic)
          snapshots.push({
            kind: "graphic",
            data: JSON.parse(JSON.stringify(graphic)),
          });
      }
    }
    if (!snapshots.length) return false;
    _clipboard.value = snapshots;
    return true;
  };

  /**
   * 从节点数据获取包围盒（用于计算粘贴偏移）
   * @param {Object} node - 节点数据
   * @returns {{ x: number, y: number, w: number, h: number } | null}
   */
  const getNodeBoundsFromData = (node) => {
    if (!node) return null;
    const p = node.absolutePos;
    const a = node.layoutItem?.free?.abs;
    if (
      p &&
      (node.positioning === "absolute" ||
        Number.isFinite(p.x) ||
        Number.isFinite(p.y))
    ) {
      return {
        x: Number.isFinite(p.x) ? p.x : 0,
        y: Number.isFinite(p.y) ? p.y : 0,
        w: Number.isFinite(p.w) ? p.w : 100,
        h: Number.isFinite(p.h) ? p.h : 100,
      };
    }
    if (a) {
      return {
        x: Number.isFinite(a.x) ? a.x : 0,
        y: Number.isFinite(a.y) ? a.y : 0,
        w: Number.isFinite(a.w) ? a.w : 100,
        h: Number.isFinite(a.h) ? a.h : 100,
      };
    }
    return {
      x: typeof node.style?.left === "number" ? node.style.left : 0,
      y: typeof node.style?.top === "number" ? node.style.top : 0,
      w: node.style?.width ?? 100,
      h: node.style?.height ?? 100,
    };
  };

  /**
   * 将偏移应用到节点数据的位置
   * @param {Object} node - 节点数据（会被修改）
   * @param {number} dx - 水平偏移
   * @param {number} dy - 垂直偏移
   */
  const applyOffsetToNodeData = (node, dx, dy) => {
    const p = node.absolutePos;
    const a = node.layoutItem?.free?.abs;
    if (
      p &&
      (node.positioning === "absolute" ||
        Number.isFinite(p.x) ||
        Number.isFinite(p.y))
    ) {
      node.absolutePos = {
        ...p,
        x: (Number.isFinite(p.x) ? p.x : 0) + dx,
        y: (Number.isFinite(p.y) ? p.y : 0) + dy,
      };
    } else if (a) {
      const next = JSON.parse(JSON.stringify(node.layoutItem || {}));
      if (!next.free) next.free = {};
      if (!next.free.abs) next.free.abs = {};
      next.free.abs = {
        ...next.free.abs,
        x: (Number.isFinite(a.x) ? a.x : 0) + dx,
        y: (Number.isFinite(a.y) ? a.y : 0) + dy,
      };
      node.layoutItem = next;
    } else if (node.style) {
      node.style = {
        ...node.style,
        left: (typeof node.style.left === "number" ? node.style.left : 0) + dx,
        top: (typeof node.style.top === "number" ? node.style.top : 0) + dy,
      };
    }
  };

  /**
   * 从剪贴板粘贴元素
   * @param {{ x: number, y: number } | null} [targetPos=null] - 画布坐标，粘贴到该位置（包围盒中心对齐）；null 时使用默认 20px 偏移
   * @returns {boolean}
   */
  const pasteNodes = (targetPos = null) => {
    if (!doc.value || !history.value || !_clipboard.value?.length) return false;
    if (!ensureEditable()) return false;

    const rootId = currentPage.value?.rootNodeId;
    if (!rootId) return false;

    const nodeItems = _clipboard.value.filter((item) => item.kind === "node");
    let offset = { x: 20, y: 20 };

    if (targetPos && nodeItems.length > 0) {
      let minX = Infinity;
      let minY = Infinity;
      let maxX = -Infinity;
      let maxY = -Infinity;
      for (const item of nodeItems) {
        const b = getNodeBoundsFromData(item.data);
        if (b) {
          minX = Math.min(minX, b.x);
          minY = Math.min(minY, b.y);
          maxX = Math.max(maxX, b.x + b.w);
          maxY = Math.max(maxY, b.y + b.h);
        }
      }
      if (Number.isFinite(minX) && Number.isFinite(minY)) {
        const centerX = (minX + maxX) / 2;
        const centerY = (minY + maxY) / 2;
        offset = { x: targetPos.x - centerX, y: targetPos.y - centerY };
      }
    }

    for (const item of _clipboard.value) {
      if (item.kind === "node") {
        const sourceId = item.data.id;
        const existsInDoc = !!doc.value.getNode(sourceId);
        if (existsInDoc) {
          history.value.execute(
            new DuplicateNodeCommand(sourceId, undefined, offset),
          );
        } else {
          const cloned = JSON.parse(JSON.stringify(item.data));
          cloned.id = crypto.randomUUID().replace(/-/g, "").substring(0, 12);
          applyOffsetToNodeData(cloned, offset.x, offset.y);
          history.value.execute(new InsertNodeCommand(rootId, -1, cloned));
        }
      }
    }
    return true;
  };

  /**
   * 复制选中元素（复制 + 粘贴一步完成）
   * @returns {boolean}
   */
  const duplicateNodes = () => {
    if (!doc.value || !history.value || !selection.value) return false;
    if (!ensureEditable()) return false;

    const elements = selection.value.getSelectedElements?.() || [];
    const nodeIds = elements
      .filter((el) => el.kind === "node")
      .map((el) => el.id);
    if (!nodeIds.length) return false;

    const rootId = currentPage.value?.rootNodeId;
    for (const nodeId of nodeIds) {
      if (nodeId === rootId) continue;
      history.value.execute(
        new DuplicateNodeCommand(nodeId, undefined, { x: 20, y: 20 }),
      );
    }
    return true;
  };

  return {
    doc,
    docVersion,
    selectionVersion,
    history,
    selection,
    serializer: serializer.value,
    projectId,
    projectName,
    projectVariables,
    projectVariableGroups,
    globalScripts,
    currentPageId,
    currentPage,
    pages,
    entryConfig,
    loading,
    saving,
    canUndo,
    canRedo,
    error,
    readonlyState,
    isReadonly,
    isLocked,
    isLockOwner,
    initEditor,
    loadProject,
    saveProjectSettings,
    loadProjectSettings,
    loadPage,
    setCurrentPage,
    refreshPages,
    createHomePage,
    createPage,
    deletePage,
    updatePageSchema,
    movePageToGroup,
    renamePage,
    updateEntry,
    persistEntry,
    saveCurrentPage,
    savePageDraft,
    saveCurrentPageDraft,
    getPageDraft,
    pageDrafts,
    pageTabState,
    setPageTabState,
    updateCurrentPage,
    updateNode,
    isLabelUnique,
    updateGraphic,
    insertNode,
    insertElColLeft,
    insertElColRight,
    insertElLayoutRowUp,
    insertElLayoutRowDown,
    removeSelectedNodes,
    removeNode,
    moveNodeUp,
    moveNodeDown,
    moveNodeToTop,
    moveNodeToBottom,
    toggleNodeVisibility,
    toggleNodeLock,
    undo,
    redo,
    acquirePageLock,
    releasePageLock,
    togglePageLock,
    ensureEditable,
    buildNewPageSchema,
    createPageSchemaPayload,
    alignElements,
    distributeElements,
    matchElementSize,
    moveSelectedByDelta,
    hasClipboard,
    copyNodes,
    pasteNodes,
    duplicateNodes,
  };
});

export default useEditorStore;
