/**
 * 编辑器状态管理（editor-store）
 *
 * 职责：
 * - 文档模型（doc）、历史（history）、选中（selection）
 * - 当前工程、页面、数据绑定系统
 * - 工程/页面 CRUD、Schema 加载与保存
 * - 命令执行（插入、删除、更新、对齐等）
 * - 规范化工具：见 ./editor/normalize-settings.js、./editor/normalize-schema.js
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
import { getDescriptor, isLayoutContainerType, isRegionType } from "@/components/descriptors/registry.js";
import { Storage } from "@/utils/storage";

import {
  unwrapApiData,
  getDefaultGlobalScripts,
  normalizeVariableDef,
  normalizeGlobalVariables,
  normalizeGlobalScripts,
  getMenuDefaultDetailConfig,
  getMenuDefaultProps,
} from "./editor/normalize-settings.js";
import {
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
} from "./editor/normalize-schema.js";
import {
  syncAbsoluteSizePatch,
  syncElLayoutMinHeightPatch,
  clampElContainerSizePatch,
  clampElColSpanPatch,
  clampElColOffsetPatch,
  clampElColShiftPatch,
  clampElLayoutRowColumnsPatch,
} from "./editor/node-update-layout-patches.js";

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
  const isLoading = ref(false);
  const isSaving = ref(false);
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
      const rejected = results.find((entry) => entry.status === "rejected");
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
    } catch (error) {
      return {
        ok: false,
        error: error instanceof Error ? error : new Error("保存失败"),
      };
    }
  };
  const loadProject = async (id) => {
    projectId.value = id || "";
    await loadProjectSettings();
    isLoading.value = true;
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
    } catch (cause) {
      const nextError = cause instanceof Error ? cause : new Error("加载工程失败");
      error.value = nextError.message;
      initEditor(createBaseSchema(id));
      return { ok: false, error: nextError };
    } finally {
      isLoading.value = false;
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

    isLoading.value = true;
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
    } catch (cause) {
      const nextError = cause instanceof Error ? cause : new Error("加载页面失败");
      error.value = nextError.message;
      return { ok: false, error: nextError };
    } finally {
      isLoading.value = false;
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
    } catch (error) {
      console.error("创建首页失败:", error);
      return {
        ok: false,
        error: error instanceof Error ? error : new Error("创建首页失败"),
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

    isSaving.value = true;
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
      isSaving.value = false;
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
   * 更新组件节点
   * @param {string} nodeId - 节点 ID
   * @param {Partial<import('@/editor-core').ComponentNode>} patch - 更新内容
   * @returns {boolean}
   */
  const updateNode = (nodeId, patch) => {
    if (!doc.value || !history.value || !nodeId) return false;
    if (!ensureEditable()) return false;

    const node = doc.value.getNode(nodeId);
    const limitedPatch = clampElContainerSizePatch(doc.value, node, patch);
    const clampedColumnsPatch = clampElLayoutRowColumnsPatch(
      node,
      limitedPatch,
    );
    const clampedSpanPatch = clampElColSpanPatch(
      doc.value,
      node,
      clampedColumnsPatch,
    );
    const clampedOffsetPatch = clampElColOffsetPatch(
      doc.value,
      node,
      clampedSpanPatch,
    );
    const clampedShiftPatch = clampElColShiftPatch(
      doc.value,
      node,
      clampedOffsetPatch,
    );
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
      HorizontalLayout: { width: 400, height: 160 },
      VerticalLayout: { width: 240, height: 240 },
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
      parentNode.type === "HorizontalLayout" ||
      parentNode.type === "VerticalLayout" ||
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
    // isRegionType 且排除 ElCol：ElContainer 内可落点区域不含 ElCol
    const isElContainerRegion = isRegionType(type) && type !== "ElCol";
    let shouldReplaceRegionChildren = false;
    if (parentNode.type === "ElContainer" && !isElContainerRegion) {
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

    const isRegionContainer = isRegionType(parentNode.type);
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
    const isRootCanvas = parentNode.id === currentPage.value?.rootNodeId;
    if (parentNode.type === "FreeContainer" || isRootCanvas) {
      node.positioning = "absolute";
      node.absolutePos = {
        x: Math.round(dropInfo.x),
        y: Math.round(dropInfo.y),
        w: dropInfo.width,
        h: dropInfo.height,
        z: 1,
      };
    } else if (
      parentNode.type === "FlexContainer" ||
      parentNode.type === "HorizontalLayout" ||
      parentNode.type === "VerticalLayout" ||
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
      // 优先从 descriptor 读取子项策略
      const parentDescriptor = getDescriptor(parentNode.type);
      if (parentDescriptor?.childFlowLayout) {
        node.flowLayout = { ...parentDescriptor.childFlowLayout };
      } else if (layoutItem?.flex) {
        node.flowLayout = { ...layoutItem.flex };
      }
      if (parentDescriptor?.childStyle) {
        node.style = { ...(node.style || {}), ...parentDescriptor.childStyle(parentNode.type) };
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

      const absolutePos = node.absolutePos;
      const freeAbsLayout = node.layoutItem?.free?.abs;
      let patch = null;

      if (
        absolutePos &&
        (node.positioning === "absolute" ||
          Number.isFinite(absolutePos.x) ||
          Number.isFinite(absolutePos.y))
      ) {
        const newX = (Number.isFinite(absolutePos.x) ? absolutePos.x : 0) + dx;
        const newY = (Number.isFinite(absolutePos.y) ? absolutePos.y : 0) + dy;
        patch = { absolutePos: { ...absolutePos, x: newX, y: newY } };
      } else if (freeAbsLayout) {
        const newX = (Number.isFinite(freeAbsLayout.x) ? freeAbsLayout.x : 0) + dx;
        const newY = (Number.isFinite(freeAbsLayout.y) ? freeAbsLayout.y : 0) + dy;
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
    const absolutePos = node.absolutePos;
    const freeAbsLayout = node.layoutItem?.free?.abs;
    if (
      absolutePos &&
      (node.positioning === "absolute" ||
        Number.isFinite(absolutePos.x) ||
        Number.isFinite(absolutePos.y))
    ) {
      return {
        x: Number.isFinite(absolutePos.x) ? absolutePos.x : 0,
        y: Number.isFinite(absolutePos.y) ? absolutePos.y : 0,
        w: Number.isFinite(absolutePos.w) ? absolutePos.w : 100,
        h: Number.isFinite(absolutePos.h) ? absolutePos.h : 100,
      };
    }
    if (freeAbsLayout) {
      return {
        x: Number.isFinite(freeAbsLayout.x) ? freeAbsLayout.x : 0,
        y: Number.isFinite(freeAbsLayout.y) ? freeAbsLayout.y : 0,
        w: Number.isFinite(freeAbsLayout.w) ? freeAbsLayout.w : 100,
        h: Number.isFinite(freeAbsLayout.h) ? freeAbsLayout.h : 100,
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
    const absolutePos = node.absolutePos;
    const freeAbsLayout = node.layoutItem?.free?.abs;
    if (
      absolutePos &&
      (node.positioning === "absolute" ||
        Number.isFinite(absolutePos.x) ||
        Number.isFinite(absolutePos.y))
    ) {
      node.absolutePos = {
        ...absolutePos,
        x: (Number.isFinite(absolutePos.x) ? absolutePos.x : 0) + dx,
        y: (Number.isFinite(absolutePos.y) ? absolutePos.y : 0) + dy,
      };
    } else if (freeAbsLayout) {
      const next = JSON.parse(JSON.stringify(node.layoutItem || {}));
      if (!next.free) next.free = {};
      if (!next.free.abs) next.free.abs = {};
      next.free.abs = {
        ...next.free.abs,
        x: (Number.isFinite(freeAbsLayout.x) ? freeAbsLayout.x : 0) + dx,
        y: (Number.isFinite(freeAbsLayout.y) ? freeAbsLayout.y : 0) + dy,
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
        const bounds = getNodeBoundsFromData(item.data);
        if (bounds) {
          minX = Math.min(minX, bounds.x);
          minY = Math.min(minY, bounds.y);
          maxX = Math.max(maxX, bounds.x + bounds.w);
          maxY = Math.max(maxY, bounds.y + bounds.h);
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
    isLoading,
    isSaving,
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
