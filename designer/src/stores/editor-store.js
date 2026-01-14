/**
 * 编辑器状态管理
 */

import { defineStore } from "pinia";
import { computed, markRaw, ref, shallowRef } from "vue";
import {
  createComponentNode,
  createEmptySchema,
  createPageNode,
  buildPagePathFromName,
  History,
  UpdateNodeCommand,
  UpdateGraphicCommand,
  SelectionModel,
  Serializer,
  UpdatePageCommand,
  UpdateEntryCommand,
  InsertNodeCommand,
  RemoveNodeCommand,
  ReorderNodeCommand,
  ToggleNodeVisibilityCommand,
  ToggleNodeLockCommand,
  PageLockManager,
  componentRegistry,
  createSelectableElement,
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
      payload.nodesById
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
  return buildSchemaFromPagePayload({ page: { id: fallbackPageId } }, projectId);
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

  for (const node of Object.values(schema.nodesById)) {
    if (!node || !node.type) continue;

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
  /** @type {import('vue').ShallowRef<import('@/editor-core').PageLockManager | null>} */
  const lockManager = shallowRef(null);
  /** @type {import('vue').Ref<import('@/editor-core').PageLockState | null>} */
  const lockState = ref(null);
  /** @type {import('vue').Ref<import('@/editor-core').EditorReadonlyState>} */
  const readonlyState = ref({ readonly: false });

  const projectId = ref("");
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
  const loadProject = async (id) => {
    projectId.value = id || "";
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

      if (entryConfigResp && doc.value && Object.keys(entryConfigResp).length > 0) {
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
      const pageResponse = await projectApi.getPage(projectId.value, pageId);
      const pagePayload = unwrapApiData(pageResponse) || {
        page: { id: pageId },
      };

      let nextSchema = resolveProjectSchema(pagePayload, projectId.value, pageId);
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
      await projectApi.updatePage(projectId.value, pageId, payload.schemaContent);
    }

    await refreshPages();
    return data;
  };

  /**
   * 删除页面/分组
   * @param {string} pageId - 页面 ID
   * @returns {Promise<void>}
   */
  const deletePage = async (pageId) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    if (!pageId) {
      throw new Error("缺少页面信息");
    }

    await projectApi.deletePage(projectId.value, pageId);
    const { pages: pageList, entryConfig: entryConfigResp } =
      await refreshPages();

    if (currentPageId.value !== pageId) return;

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
   * @returns {Promise<void>}
   */
  const movePageToGroup = async (pageId, targetGroupId) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    if (!pageId) {
      throw new Error("缺少页面信息");
    }

    await projectApi.movePageToGroup(projectId.value, pageId, targetGroupId);
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

    await projectApi.renamePage(projectId.value, pageId, name);

    pages.value = pages.value.map((page) =>
      page.id === pageId
        ? {
            ...page,
            name,
            path: path ?? page.path,
          }
        : page
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
        currentPageId.value
      );
      await projectApi.updatePage(projectId.value, currentPageId.value, payload);
    } finally {
      saving.value = false;
    }
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
      page.id === currentPageId.value ? { ...page, ...patch } : page
    );
    return true;
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

    history.value.execute(new UpdateNodeCommand(nodeId, patch));
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

    if (parentNode.type === "FlexContainer" || parentNode.type === "ResponsiveLayout") {
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

    const parentNode = doc.value.getNode(parentId);
    if (!parentNode) return null;

    const manifest = componentRegistry.get(type);
    const defaultSize = resolveDefaultSize(type, manifest);
    const insertIndex = Number.isInteger(index)
      ? index
      : parentNode.children?.length ?? 0;

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

    const node = createComponentNode(type, {
      label: manifest?.name || type,
      props: { ...(manifest?.defaultProps || {}) },
      style: nodeStyle,
      layoutItem,
    });

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
      parentNode.type === "ResponsiveLayout"
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

    history.value.execute(new InsertNodeCommand(parentId, insertIndex, node));
    selection.value?.select(createSelectableElement("node", node.id));

    return node;
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

    deletable.forEach((nodeId) => {
      history.value.execute(new RemoveNodeCommand(nodeId));
    });
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

    history.value.execute(new RemoveNodeCommand(nodeId));
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

  return {
    doc,
    docVersion,
    selectionVersion,
    history,
    selection,
    serializer: serializer.value,
    projectId,
    projectName,
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
    updateCurrentPage,
    updateNode,
    updateGraphic,
    insertNode,
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
  };
});

export default useEditorStore;
