/**
 * 编辑器状态管理
 * 负责加载工程、初始化编辑器内核、管理撤销重做与保存
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
 * 解包 API 响应数据
 * @param {*} payload - 原始响应
 * @returns {*} 业务数据
 */
const unwrapApiData = (payload) => {
  if (payload && typeof payload === "object" && "success" in payload) {
    return payload.data;
  }
  return payload;
};

/**
 * 规范化页面列表返回值
 * @param {*} payload - 原始响应数据
 * @returns {Array} 页面数组
 */
const normalizePageList = (payload) => {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.items)) return payload.items;
  if (Array.isArray(payload?.list)) return payload.list;
  return [];
};

/**
 * 规范化页面 Schema
 * @param {*} payload - 原始响应数据
 * @returns {Object}
 */
const normalizePageSchema = (payload) => {
  if (!payload || typeof payload !== "object") return payload;
  if (payload.schema) return payload.schema;
  return payload;
};

/**
 * 规范化布局容器类型（对齐布局系统文档）
 * @param {import('@/editor-core').ProjectSchema} schema - 工程 Schema
 * @returns {import('@/editor-core').ProjectSchema}
 */
const normalizeLayoutSchema = (schema) => {
  if (!schema || typeof schema !== "object") return schema;
  if (!schema.nodesById) return schema;

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
 * 更新页面 Schema 中的名称与路由
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

  // ✅ 根容器使用FreeContainer，允许自由放置组件
  const rootNode = createComponentNode("FreeContainer", {
    id: page.rootNodeId,
    label: "根容器",
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
  const nodesById = payload.nodesById || {};
  const graphicsById = payload.graphicsById || {};

  // 确保根节点存在
  if (!nodesById[page.rootNodeId]) {
    nodesById[page.rootNodeId] = createComponentNode("FlexContainer", {
      id: page.rootNodeId,
      label: "根容器",
      props: {
        direction: "column",
        wrap: "nowrap",
        justify: "flex-start",
        align: "stretch",
        gap: 0,
      },
      style: {
        width: "100%",
        height: "100%",
      },
    });
  }

  schema.pagesById[page.id] = page;
  schema.nodesById = nodesById;
  schema.graphicsById = graphicsById;
  schema.entry.homePageId = page.id;

  return schema;
};

/**
 * 编辑器状态管理 Store
 */
export const useEditorStore = defineStore("editor", () => {
  /** @type {import('vue').ShallowRef<import('@/editor-core').DocumentModel | null>} */
  const doc = shallowRef(null);
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

    historyUnsubscribe = nextHistory.on("change", (payload) => {
      canUndo.value = payload.canUndo;
      canRedo.value = payload.canRedo;
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
      const { pages: pageList, entryConfig } = await refreshPages();

      // ✅ 如果没有页面，自动创建首页并持久化
      if (!pageList.length) {
        const homePageResult = await createHomePage(id);
        if (!homePageResult.ok) {
          initEditor(createBaseSchema(id));
        }
        // createHomePage 内部已经初始化了编辑器
        return { ok: homePageResult.ok };
      }

      // 优先使用 entryConfig 中的 homePageId，否则使用第一个页面
      const homePageId = entryConfig?.homePageId;
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

      // 如果返回的是完整工程 Schema，直接使用
      if (pagePayload?.page) {
        initEditor(pagePayload);
      } else if (pagePayload?.schema) {
        initEditor(pagePayload.schema);
      } else {
        initEditor(buildSchemaFromPagePayload(pagePayload || {}, id));
      }

      // 使用 entryConfig 更新 doc.entry
      if (entryConfig && doc.value && Object.keys(entryConfig).length > 0) {
        doc.value._updateEntry(entryConfig);
      }

      currentPageId.value = targetPageId;
      // 锁定改为手动操作，移除自动锁定

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

      let nextSchema = null;
      // 保留原有的 entry 配置，避免切换页面时覆盖 homePageId
      const existingEntry = doc.value?.entry;

      if (pagePayload?.page) {
        nextSchema = pagePayload;
        // 合并原有的 entry 配置
        if (existingEntry && !nextSchema.entry) {
          nextSchema.entry = existingEntry;
        }
      } else if (pagePayload?.schema) {
        nextSchema = pagePayload.schema;
        // 合并原有的 entry 配置
        if (existingEntry && !nextSchema.entry) {
          nextSchema.entry = existingEntry;
        }
      } else {
        nextSchema = buildSchemaFromPagePayload(
          pagePayload || {},
          projectId.value
        );
        // 合并原有的 entry 配置
        if (existingEntry) {
          nextSchema.entry = { ...nextSchema.entry, ...existingEntry };
        }
      }

      initEditor(nextSchema);
      // 无论如何都要更新 currentPageId
      currentPageId.value = pageId;
      // 锁定改为手动操作，移除自动锁定

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

    // 更新响应式 entryConfig
    entryConfig.value = newEntryConfig;

    // 同步 entryConfig 到 doc（使用内部方法）
    if (doc.value && newEntryConfig && Object.keys(newEntryConfig).length > 0) {
      doc.value._updateEntry(newEntryConfig);
    }

    return { pages: pageList, entryConfig: newEntryConfig };
  };

  /**
   * 创建首页（项目初始化时自动调用）
   * @param {string} pid - 工程 ID
   * @returns {Promise<{ok: boolean, pageId?: string, error?: Error}>}
   */
  const createHomePage = async (pid) => {
    try {
      // 1. 创建页面记录
      const result = await projectApi.createPage(pid, {
        name: "首页",
        type: "page",
        parentId: null,
      });

      // 调试日志
      console.log("创建页面返回结果:", result);

      // 确保 result 存在
      if (!result) {
        throw new Error("创建首页失败：API 返回空结果");
      }

      // 确保 data 存在
      if (!result.data) {
        throw new Error("创建首页失败：API 返回结果缺少 data 字段");
      }

      const pageId = result.data.id;

      if (!pageId) {
        throw new Error("创建首页失败：未获取到页面ID");
      }

      // 2. 构建并保存页面 Schema
      const schema = createBaseSchema(pid);
      const pageNode = Object.values(schema.pagesById)[0];

      // 确保 pageNode 存在
      if (!pageNode) {
        throw new Error("创建首页失败：无法获取页面节点");
      }

      const rootNode = schema.nodesById[pageNode.rootNodeId];

      // 更新 pageNode 使用实际的 pageId
      delete schema.pagesById[pageNode.id];
      pageNode.id = pageId;
      schema.pagesById[pageId] = pageNode;
      schema.entry.homePageId = pageId;

      if (rootNode) {
        pageNode.rootNodeId = rootNode.id;
      }

      // 3. 保存页面 Schema
      const pagePayload = {
        page: pageNode,
        nodesById: rootNode ? { [rootNode.id]: rootNode } : {},
        graphicsById: {},
      };
      await projectApi.updatePage(pid, pageId, pagePayload);

      // 4. 保存项目级别的 entryConfig（设置首页）
      const newEntryConfig = { homePageId: pageId };
      await projectApi.updateEntryConfig(pid, newEntryConfig);
      entryConfig.value = newEntryConfig;

      // 5. 初始化编辑器
      schema.pagesById = { [pageId]: pageNode };
      initEditor(schema);
      currentPageId.value = pageId;

      // 6. 刷新页面列表，确保页面树正确渲染
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
   * 创建页面/页面组
   * @param {{ name: string, type?: string, parentId?: string | null, schemaContent?: Object }} payload - 创建参数
   * @returns {Promise<{ id: string, name: string }>}
   */
  const createPage = async (payload) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    const response = await projectApi.createPage(projectId.value, payload);
    await refreshPages();
    // 统一返回格式，提取 data 中的内容
    const data = response?.data || response;
    return data;
  };

  /**
   * 删除页面
   * @param {string} pageId - 页面 ID
   * @returns {Promise<void>}
   */
  const deletePage = async (pageId) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    await projectApi.deletePage(projectId.value, pageId);
    await refreshPages();
  };

  /**
   * 更新指定页面 Schema
   * @param {string} pageId - 页面 ID
   * @param {Object} schema - 页面 Schema
   * @returns {Promise<void>}
   */
  const updatePageSchema = async (pageId, schema) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    if (doc.value?.entry && schema && typeof schema === "object") {
      schema.entry = { ...doc.value.entry };
    }
    await projectApi.updatePage(projectId.value, pageId, schema);
  };

  /**
   * 构建页面 Schema Payload
   * @param {{ id: string, name: string, path: string }} page - 页面信息
   * @returns {Object}
   */
  const createPageSchemaPayload = (page) => {
    const schema = createEmptySchema({
      projectId: projectId.value,
      name: projectName.value || "工程",
    });
    const pageNode = createPageNode({
      id: page.id,
      name: page.name,
      path: page.path,
    });
    const rootNode = createComponentNode("FlexContainer", {
      id: pageNode.rootNodeId,
      label: "根容器",
      props: {
        direction: "column",
        wrap: "nowrap",
        justify: "flex-start",
        align: "stretch",
        gap: 0,
      },
      style: {
        width: "100%",
        height: "100%",
      },
    });

    schema.pagesById[pageNode.id] = pageNode;
    schema.nodesById[rootNode.id] = rootNode;
    // 保留现有的 entry 配置，不要覆盖 homePageId
    schema.entry = { ...schema.entry, ...(doc.value?.entry || {}) };
    // 只在还没有设置首页时才设置（用于初始化）
    if (!schema.entry.homePageId) {
      schema.entry.homePageId = pageNode.id;
    }

    const tempDoc = serializer.value.importFromSchema(schema);
    const payload = serializer.value.exportPage(tempDoc, pageNode.id);
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
    // 使用 FreeContainer 作为根容器
    const rootNode = createComponentNode("FreeContainer", {
      id: pageNode.rootNodeId,
      label: "根容器",
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
   * 移动页面到指定分组
   * @param {string} pageId - 页面 ID
   * @param {string|null} parentId - 分组 ID
   * @returns {Promise<void>}
   */
  const movePageToGroup = async (pageId, parentId) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    await projectApi.movePageToGroup(projectId.value, pageId, parentId ?? null);
    await refreshPages();
  };

  /**
   * 重命名页面/分组
   * @param {string} pageId - 页面 ID
   * @param {string} name - 新名称
   * @param {string} [path] - 页面路径
   * @returns {Promise<void>}
   */
  const renamePage = async (pageId, name, path) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    await projectApi.renamePage(projectId.value, pageId, name);

    if (path) {
      if (doc.value && currentPageId.value === pageId) {
        updateCurrentPage({ name, path });
        const payload = serializer.value.exportPage(doc.value, pageId);
        await projectApi.updatePage(projectId.value, pageId, payload);
      } else {
        const pageResponse = await projectApi.getPage(projectId.value, pageId);
        const pagePayload = normalizePageSchema(unwrapApiData(pageResponse));
        const nextSchema = applyPageNamePath(pagePayload, pageId, name, path);
        await projectApi.updatePage(projectId.value, pageId, nextSchema);
      }
    }

    await refreshPages();
  };

  /**
   * 更新入口配置
   * @param {Partial<import('@/editor-core').EntryConfig>} patch - 更新内容
   * @returns {boolean}
   */
  const updateEntry = (patch) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;
    history.value.execute(new UpdateEntryCommand(patch));
    // 同步更新响应式 entryConfig
    entryConfig.value = { ...entryConfig.value, ...patch };
    return true;
  };

  /**
   * 持久化入口配置
   * 使用项目级别的 entryConfig API
   * @returns {Promise<void>}
   */
  const persistEntry = async () => {
    if (!projectId.value) return;

    const entryData = entryConfig.value || {};
    try {
      await projectApi.updateEntryConfig(projectId.value, entryData);
      await refreshPages();
    } catch (err) {
      console.error("持久化 entry 失败:", err);
      throw err;
    }
  };

  /**
   * 保存当前页面
   * @returns {Promise<void>}
   */
  const saveCurrentPage = async () => {
    if (!doc.value || !currentPageId.value || !projectId.value) {
      throw new Error("缺少工程或页面信息，无法保存");
    }
    if (!ensureEditable()) {
      throw new Error(error.value || "当前为只读模式");
    }

    saving.value = true;
    error.value = "";

    try {
      const payload = serializer.value.exportPage(
        doc.value,
        currentPageId.value
      );
      payload.entry = doc.value.entry;
      await projectApi.updatePage(
        projectId.value,
        currentPageId.value,
        payload
      );
    } catch (err) {
      const nextError = err instanceof Error ? err : new Error("保存失败");
      error.value = nextError.message;
      throw nextError;
    } finally {
      saving.value = false;
    }
  };

  /**
   * 更新当前页面配置
   * @param {Partial<import('@/editor-core').PageNode>} patch - 更新内容
   * @returns {boolean}
   */
  const updateCurrentPage = (patch) => {
    if (!doc.value || !history.value || !currentPageId.value) return false;
    if (!ensureEditable()) return false;
    history.value.execute(new UpdatePageCommand(currentPageId.value, patch));
    return true;
  };

  /**
   * 更新节点
   * @param {string} nodeId - 节点 ID
   * @param {Partial<import('@/editor-core').ComponentNode>} patch - 更新内容
   * @returns {boolean}
   */
  const updateNode = (nodeId, patch) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;
    history.value.execute(new UpdateNodeCommand(nodeId, patch));
    return true;
  };

  /**
   * 更新图形
   * @param {string} graphicId - 图形 ID
   * @param {Partial<import('@/editor-core').GraphicNode>} patch - 更新内容
   * @returns {boolean}
   */
  const updateGraphic = (graphicId, patch) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;
    history.value.execute(new UpdateGraphicCommand(graphicId, patch));
    return true;
  };

  /**
   * 执行撤销
   * @returns {boolean} 是否成功撤销
   */
  const undo = () => {
    if (!history.value) return false;
    return history.value.undo();
  };

  /**
   * 执行重做
   * @returns {boolean} 是否成功重做
   */
  const redo = () => {
    if (!history.value) return false;
    return history.value.redo();
  };

  /**
   * 当前页面配置
   */
  const currentPage = computed(() => {
    if (!doc.value || !currentPageId.value) return null;
    return doc.value.getPage(currentPageId.value);
  });

  /**
   * 切换当前页面
   * @param {string} pageId - 页面 ID
   */
  const setCurrentPage = async (pageId) => {
    if (!pageId) return;

    // 如果已经是当前页面，只重置选择但不重新加载
    if (currentPageId.value === pageId) {
      selection.value?.reset();
      return;
    }

    const currentPageData = doc.value?.getPage(pageId);

    // 如果 doc 中已有该页面数据，直接切换
    if (currentPageData) {
      currentPageId.value = pageId;
      selection.value?.reset();
      return;
    }

    // 否则需要从后端加载页面数据
    try {
      loading.value = true;
      const result = await loadPage(pageId);
      if (result.ok) {
        return;
      }
    } catch (err) {
      console.error("加载页面失败:", err);
    } finally {
      loading.value = false;
    }

    // 加载失败时仍然更新 currentPageId（保持 UI 同步）
    currentPageId.value = pageId;
    console.log(
      "[EditorStore] Fallback updated currentPageId to:",
      currentPageId.value
    );
    selection.value?.reset();
  };

  /**
   * 解析默认尺寸
   * @param {string} type - 组件类型
   * @param {Object | undefined} manifest - 组件清单
   * @returns {{width: number, height: number}}
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
   * 插入组件节点
   * @param {string} componentType - 组件类型
   * @param {string} [parentId] - 父节点 ID，默认为当前页面根节点
   * @param {number} [index] - 插入位置，默认追加到末尾
   * @param {{ dropPosition?: { x: number, y: number } }} [options] - 额外参数
   * @returns {string | null} 新节点 ID
   */
  const insertNode = (componentType, parentId, index, options = {}) => {
    if (!doc.value || !history.value || !currentPageId.value) return null;
    if (!ensureEditable()) return null;

    // 获取组件清单
    const manifest = componentRegistry.get(componentType);
    if (!manifest) {
      console.warn(`未找到组件类型: ${componentType}`);
      return null;
    }

    // 确定父节点
    const targetParentId = parentId || currentPage.value?.rootNodeId;
    if (!targetParentId) return null;

    const parent = doc.value.getNode(targetParentId);
    if (!parent) return null;

    // 确定插入位置
    const insertIndex = index ?? (parent.children?.length || 0);

    // 创建新节点
    const newNode = createComponentNode(componentType, {
      parentNode: parent,
      label: manifest.name,
      props: { ...manifest.defaultProps },
      style: { ...manifest.defaultStyle },
    });

    // ✅ 判断是否为布局容器组件
    const isLayoutComponent =
      manifest.category === "布局" ||
      manifest.category === "layout" ||
      ["FlexContainer", "GridContainer", "FreeContainer"].includes(
        componentType
      );

    // FreeContainer 下：布局组件走流式，占据整行；普通组件走绝对定位
    if (parent.type === "FreeContainer") {
      if (isLayoutComponent) {
        const minHeightValue = Number.parseFloat(
          String(newNode.style?.minHeight || 0)
        );
        const nextMinHeight = Number.isFinite(minHeightValue)
          ? Math.max(minHeightValue, 120)
          : 120;
        newNode.positioning = "flow";
        newNode.absolutePos = undefined;
        newNode.flowLayout = undefined;
        newNode.layoutItem = null;
        newNode.style = {
          ...newNode.style,
          width: newNode.style?.width || "100%",
          minHeight: `${nextMinHeight}px`,
        };
      } else if (options.dropPosition) {
        const { width, height } = resolveDefaultSize(componentType, manifest);
        const nextX = Math.max(0, Math.round(options.dropPosition.x));
        const nextY = Math.max(0, Math.round(options.dropPosition.y));

        newNode.positioning = "absolute";
        newNode.absolutePos = {
          x: nextX,
          y: nextY,
          w: width,
          h: height,
          z: 1,
        };
        newNode.layoutItem = {
          free: {
            mode: "abs",
            abs: {
              x: nextX,
              y: nextY,
              w: width,
              h: height,
              z: 1,
            },
          },
        };
      }
    }

    // 执行插入命令
    history.value.execute(
      new InsertNodeCommand(targetParentId, insertIndex, newNode)
    );

    // 选中新节点
    const element = createSelectableElement("node", newNode.id);
    selection.value?.select(element);

    return newNode.id;
  };

  /**
   * 删除选中的节点
   * @returns {boolean} 是否成功删除
   */
  const removeSelectedNodes = () => {
    if (!doc.value || !history.value || !selection.value) return false;
    if (!ensureEditable()) return false;

    const selectedNodeIds = selection.value.getSelectedNodeIds();
    if (!selectedNodeIds.length) return false;

    // 过滤掉根节点
    const rootNodeId = currentPage.value?.rootNodeId;
    const nodesToRemove = selectedNodeIds.filter((id) => id !== rootNodeId);
    if (!nodesToRemove.length) return false;

    // 依次删除节点
    for (const nodeId of nodesToRemove) {
      history.value.execute(new RemoveNodeCommand(nodeId));
    }

    // 清除选中
    selection.value.clearSelection();

    return true;
  };

  /**
   * 上移图层
   * @param {string} [nodeId] - 节点 ID,不传则使用当前选中节点
   * @returns {boolean} 是否成功
   */
  const moveNodeUp = (nodeId) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;

    const targetId = nodeId || selection.value?.getPrimaryElement()?.id;
    if (!targetId) return false;

    const node = doc.value.getNode(targetId);
    if (!node) return false;

    // 不能移动根节点
    const rootNodeId = currentPage.value?.rootNodeId;
    if (targetId === rootNodeId) return false;

    history.value.execute(new ReorderNodeCommand(targetId, "up"));
    return true;
  };

  /**
   * 下移图层
   * @param {string} [nodeId] - 节点 ID,不传则使用当前选中节点
   * @returns {boolean} 是否成功
   */
  const moveNodeDown = (nodeId) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;

    const targetId = nodeId || selection.value?.getPrimaryElement()?.id;
    if (!targetId) return false;

    const node = doc.value.getNode(targetId);
    if (!node) return false;

    // 不能移动根节点
    const rootNodeId = currentPage.value?.rootNodeId;
    if (targetId === rootNodeId) return false;

    history.value.execute(new ReorderNodeCommand(targetId, "down"));
    return true;
  };

  /**
   * 置顶图层
   * @param {string} [nodeId] - 节点 ID,不传则使用当前选中节点
   * @returns {boolean} 是否成功
   */
  const moveNodeToTop = (nodeId) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;

    const targetId = nodeId || selection.value?.getPrimaryElement()?.id;
    if (!targetId) return false;

    const node = doc.value.getNode(targetId);
    if (!node) return false;

    // 不能移动根节点
    const rootNodeId = currentPage.value?.rootNodeId;
    if (targetId === rootNodeId) return false;

    history.value.execute(new ReorderNodeCommand(targetId, "top"));
    return true;
  };

  /**
   * 置底图层
   * @param {string} [nodeId] - 节点 ID,不传则使用当前选中节点
   * @returns {boolean} 是否成功
   */
  const moveNodeToBottom = (nodeId) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;

    const targetId = nodeId || selection.value?.getPrimaryElement()?.id;
    if (!targetId) return false;

    const node = doc.value.getNode(targetId);
    if (!node) return false;

    // 不能移动根节点
    const rootNodeId = currentPage.value?.rootNodeId;
    if (targetId === rootNodeId) return false;

    history.value.execute(new ReorderNodeCommand(targetId, "bottom"));
    return true;
  };

  /**
   * 切换节点显示/隐藏
   * @param {string} nodeId - 节点 ID
   * @param {boolean} [hidden] - 目标状态,不传则自动切换
   * @returns {boolean} 是否成功
   */
  const toggleNodeVisibility = (nodeId, hidden) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;
    if (!nodeId) return false;

    const node = doc.value.getNode(nodeId);
    if (!node) return false;

    history.value.execute(new ToggleNodeVisibilityCommand(nodeId, hidden));
    return true;
  };

  /**
   * 切换节点锁定/解锁
   * @param {string} nodeId - 节点 ID
   * @param {boolean} [locked] - 目标状态,不传则自动切换
   * @returns {boolean} 是否成功
   */
  const toggleNodeLock = (nodeId, locked) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;
    if (!nodeId) return false;

    const node = doc.value.getNode(nodeId);
    if (!node) return false;

    history.value.execute(new ToggleNodeLockCommand(nodeId, locked));
    return true;
  };

  return {
    doc,
    history,
    selection,
    serializer,
    lockState,
    readonlyState,
    isLocked,
    isLockOwner,
    isReadonly,
    projectId,
    projectName,
    currentPageId,
    pages,
    entryConfig,
    loading,
    saving,
    canUndo,
    canRedo,
    error,
    currentPage,
    initEditor,
    loadProject,
    loadPage,
    refreshPages,
    createPage,
    deletePage,
    updatePageSchema,
    createPageSchemaPayload,
    buildNewPageSchema,
    movePageToGroup,
    renamePage,
    saveCurrentPage,
    updateCurrentPage,
    persistEntry,
    updateNode,
    updateGraphic,
    updateEntry,
    undo,
    redo,
    setCurrentPage,
    insertNode,
    removeSelectedNodes,
    togglePageLock,
    releasePageLock,
    moveNodeUp,
    moveNodeDown,
    moveNodeToTop,
    moveNodeToBottom,
    toggleNodeVisibility,
    toggleNodeLock,
  };
});

export default useEditorStore;
