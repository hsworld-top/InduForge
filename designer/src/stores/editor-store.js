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
      const pagesResponse = await projectApi.getPages(id);
      const pageList = normalizePageList(unwrapApiData(pagesResponse));
      pages.value = pageList;

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
    saveCurrentPage,
    undo,
    redo,
  };
});

export default useEditorStore;
