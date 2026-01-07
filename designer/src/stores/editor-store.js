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
  History,
  SelectionModel,
  Serializer,
  UpdatePageCommand,
  UpdateEntryCommand,
} from "@/editor-core";
import { projectApi } from "@/services";

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

  const rootNode = createComponentNode("Container", {
    id: page.rootNodeId,
    label: "根容器",
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
  const nodesById = payload.nodesById || {};
  const graphicsById = payload.graphicsById || {};

  // 确保根节点存在
  if (!nodesById[page.rootNodeId]) {
    nodesById[page.rootNodeId] = createComponentNode("Container", {
      id: page.rootNodeId,
      label: "根容器",
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

  const projectId = ref("");
  const projectName = ref("");
  const currentPageId = ref("");
  const pages = ref([]);
  const loading = ref(false);
  const saving = ref(false);
  const canUndo = ref(false);
  const canRedo = ref(false);
  const error = ref("");

  let historyUnsubscribe = null;

  /**
   * 初始化编辑器内核
   * @param {import('@/editor-core').ProjectSchema} schema - 工程 Schema
   */
  const initEditor = (schema) => {
    const nextDoc = serializer.value.importFromSchema(schema);
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
    pages.value = nextDoc.getAllPages();
    canUndo.value = nextHistory.canUndo();
    canRedo.value = nextHistory.canRedo();
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
      const pageList = await refreshPages();

      if (!pageList.length) {
        initEditor(createBaseSchema(id));
        return { ok: true };
      }

      const targetPage = pageList[0];
      const pageId = targetPage?.id || targetPage?.pageId;
      if (!pageId) {
        initEditor(createBaseSchema(id));
        return { ok: true };
      }

      const pageResponse = await projectApi.getPage(id, pageId);
      const pagePayload =
        unwrapApiData(pageResponse) || { page: { id: pageId } };

      // 如果返回的是完整工程 Schema，直接使用
      if (pagePayload?.schemaVersion) {
        initEditor(pagePayload);
      } else if (pagePayload?.schema) {
        initEditor(pagePayload.schema);
      } else {
        initEditor(buildSchemaFromPagePayload(pagePayload || {}, id));
      }

      currentPageId.value =
        doc.value?.entry?.homePageId ||
        doc.value?.getAllPages()[0]?.id ||
        "";

      return { ok: true };
    } catch (err) {
      const nextError =
        err instanceof Error ? err : new Error("加载工程失败");
      error.value = nextError.message;
      initEditor(createBaseSchema(id));
      return { ok: false, error: nextError };
    } finally {
      loading.value = false;
    }
  };

  /**
   * 刷新页面列表
   * @returns {Promise<Array>}
   */
  const refreshPages = async () => {
    if (!projectId.value) return [];
    const pagesResponse = await projectApi.getPages(projectId.value);
    const pageList = normalizePageList(unwrapApiData(pagesResponse));
    pages.value = pageList;
    return pageList;
  };

  /**
   * 创建页面/页面组
   * @param {{ name: string, type?: string, parentId?: string | null }} payload - 创建参数
   * @returns {Promise<Object>}
   */
  const createPage = async (payload) => {
    if (!projectId.value) {
      throw new Error("缺少工程信息");
    }
    const data = await projectApi.createPage(projectId.value, payload);
    await refreshPages();
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
    const rootNode = createComponentNode("Container", {
      id: pageNode.rootNodeId,
      label: "根容器",
    });

    schema.pagesById[pageNode.id] = pageNode;
    schema.nodesById[rootNode.id] = rootNode;
    schema.entry.homePageId = pageNode.id;

    const tempDoc = serializer.value.importFromSchema(schema);
    return serializer.value.exportPage(tempDoc, pageNode.id);
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
    history.value.execute(new UpdateEntryCommand(patch));
    return true;
  };

  /**
   * 保存当前页面
   * @returns {Promise<void>}
   */
  const saveCurrentPage = async () => {
    if (!doc.value || !currentPageId.value || !projectId.value) {
      throw new Error("缺少工程或页面信息，无法保存");
    }

    saving.value = true;
    error.value = "";

    try {
      const payload = serializer.value.exportPage(doc.value, currentPageId.value);
      await projectApi.updatePage(projectId.value, currentPageId.value, payload);
    } catch (err) {
      const nextError =
        err instanceof Error ? err : new Error("保存失败");
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
    history.value.execute(new UpdatePageCommand(currentPageId.value, patch));
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
  const setCurrentPage = (pageId) => {
    if (!doc.value || !pageId) return;
    const targetPage = doc.value.getPage(pageId);
    if (!targetPage) return;

    currentPageId.value = pageId;
    selection.value?.reset();
  };

  return {
    doc,
    history,
    selection,
    serializer,
    projectId,
    projectName,
    currentPageId,
    pages,
    loading,
    saving,
    canUndo,
    canRedo,
    error,
    currentPage,
    initEditor,
    loadProject,
    refreshPages,
    createPage,
    deletePage,
    updatePageSchema,
    createPageSchemaPayload,
    movePageToGroup,
    renamePage,
    saveCurrentPage,
    updateCurrentPage,
    updateEntry,
    undo,
    redo,
    setCurrentPage,
  };
});

export default useEditorStore;
