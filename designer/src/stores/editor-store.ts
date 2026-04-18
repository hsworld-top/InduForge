/**
 * 编辑器状态管理（editor-store）
 *
 * 职责：
 * - 文档模型（doc）、历史（history）、选中（selection）
 * - 当前工程、页面、数据绑定系统
 * - 工程/页面 CRUD、Schema 加载与保存
 * - 命令执行（插入、删除、更新、对齐等）
 * - 规范化工具：见 ./editor/normalize-settings.ts、./editor/normalize-schema.ts
 */

import type {
  EditorClipboardItem,
  EditorClipboardRefValue,
  EditorCreateHomePageResult,
  EditorDeletePageMode,
  EditorEntryConfigPatch,
  EditorLoadPageResult,
  EditorPageDraftEntry,
  EditorPageDraftsMap,
  EditorPageSchemaPayload,
  EditorPageTabItem,
  EditorPageTabState,
  EditorPasteTargetPosition,
  EditorSaveProjectSettingsResult,
  EditorUpdateCurrentPagePatch,
  LockResult,
  TogglePageLockResult,
} from "./editor-store.contracts";
import type { CreatePageForStoreResult, CreatePagePayload } from "./editor/page-create-actions";
import type { EntryConfigFromApi } from "./editor/pages-sync-types";
import type { LoadProjectForStoreResult } from "./editor/project-load-actions";
import type { Command } from "@/editor-core/commands/Command";
import type { DocumentModel } from "@/editor-core/document/DocumentModel";
import type {
  ComponentNode,
  EditorReadonlyState,
  GraphicNode,
  LayoutItem,
  PageLockState,
  PageNode,
  ProjectSchema,
} from "@/editor-core/document/types";
import { defineStore } from "pinia";
import { computed, markRaw, ref, shallowRef } from "vue";
import {
  getDescriptor,
  isLayoutContainerType,
  isRegionType,
} from "@/editor-core/descriptors/registry";
import {
  AlignElementsCommand,
  DistributeElementsCommand,
  MatchSizeCommand,
} from "@/editor-core/commands/alignCommands";
import { BatchCommand } from "@/editor-core/commands/Command";
import { UpdateEntryCommand } from "@/editor-core/commands/entryCommands";
import { UpdateGraphicCommand } from "@/editor-core/commands/graphicCommands";
import { History } from "@/editor-core/commands/History";
import {
  DuplicateNodeCommand,
  InsertNodeCommand,
  RemoveNodeCommand,
  ReorderNodeCommand,
  ToggleNodeLockCommand,
  ToggleNodeVisibilityCommand,
  UpdateNodeCommand,
} from "@/editor-core/commands/nodeCommands";
import { UpdatePageCommand } from "@/editor-core/commands/pageCommands";
import { createComponentNode } from "@/editor-core/document/factory";
import { Serializer } from "@/editor-core/document/Serializer";
import { createSelectableElement } from "@/editor-core/document/types";
import { PageLockManager } from "@/editor-core/lock/PageLockManager";
import { componentRegistry } from "@/editor-core/registry/component-registry";
import { SelectionModel } from "@/editor-core/selection/SelectionModel";
import { projectApi } from "@/services";
import request from "@/utils/request";

import { Storage } from "@/utils/storage";
import { normalizeEditorPageTabsForState } from "./editor-store.contracts";
import {
  buildFlexLayoutItem,
  buildLayoutItem,
  buildUniqueNodeLabel,
  isNodeLabelUnique,
  resolveDefaultSize,
  resolveLayerTargetFromSelection,
} from "./editor/editor-node-layout-helpers";
import { createHomePageForStore } from "./editor/home-page-actions";
import {
  clampElColOffsetPatch,
  clampElColShiftPatch,
  clampElColSpanPatch,
  clampElContainerSizePatch,
  clampElLayoutRowColumnsPatch,
  syncAbsoluteSizePatch,
  syncElLayoutMinHeightPatch,
} from "./editor/node-update-layout-patches";
import {
  buildNewPageSchema,
  createBaseSchema,
  createPageSchemaPayload,
  normalizeLayoutSchema,
  resolveProjectSchema,
} from "./editor/normalize-schema";
import {
  getDefaultGlobalScripts,
  getMenuDefaultDetailConfig,
  getMenuDefaultProps,
} from "./editor/normalize-settings";
import { createPageForStore } from "./editor/page-create-actions";
import { refreshPagesForStore } from "./editor/page-list-sync-actions";
import {
  deletePageForStore,
  movePageToGroupForStore,
  renamePageForStore,
} from "./editor/page-mutation-actions";
import { resolvePageSchemaOnLoad } from "./editor/page-navigation-actions";
import { resolveLandingPageId } from "./editor/page-navigation-helpers";
import {
  persistEntryConfigForStore,
  saveCurrentPageForStore,
  saveEntryPatchForStore,
  savePageDraftForStore,
} from "./editor/page-persist-actions";
import { updatePageSchemaForStore } from "./editor/page-schema-remote-actions";
import { loadProjectForStore } from "./editor/project-load-actions";
import {
  loadProjectSettingsForStore,
  saveProjectSettingsForStore,
} from "./editor/project-settings-actions";

const UUID_DASH_REGEX = /-/g;
const EL_CONTAINER_REGION_PRESET_TOP_MAIN = "top-main";
const EL_CONTAINER_REGION_PRESET_ASIDE_FULL_HEIGHT = "aside-full-height";
const EL_CONTAINER_REGION_PRESET_ASIDE_BETWEEN = "aside-between";
const EL_CONTAINER_REGION_PRESET_DEFAULT = EL_CONTAINER_REGION_PRESET_ASIDE_BETWEEN;

type ElContainerRegionPreset =
  | typeof EL_CONTAINER_REGION_PRESET_TOP_MAIN
  | typeof EL_CONTAINER_REGION_PRESET_ASIDE_FULL_HEIGHT
  | typeof EL_CONTAINER_REGION_PRESET_ASIDE_BETWEEN;

function normalizeElContainerRegionPreset(value: unknown): ElContainerRegionPreset {
  if (
    value === EL_CONTAINER_REGION_PRESET_TOP_MAIN ||
    value === EL_CONTAINER_REGION_PRESET_ASIDE_FULL_HEIGHT ||
    value === EL_CONTAINER_REGION_PRESET_ASIDE_BETWEEN
  ) {
    return value;
  }
  return EL_CONTAINER_REGION_PRESET_DEFAULT;
}

function resolveElContainerPresetVisibility(
  preset: ElContainerRegionPreset,
): Record<string, boolean> {
  if (preset === EL_CONTAINER_REGION_PRESET_TOP_MAIN) {
    return {
      showHeader: true,
      showAside: false,
      showMain: true,
      showFooter: false,
    };
  }
  return {
    showHeader: true,
    showAside: true,
    showMain: true,
    showFooter: true,
  };
}

function applyElContainerPresetProps(props: Record<string, unknown>): Record<string, unknown> {
  const preset = normalizeElContainerRegionPreset(props?.regionPreset);
  return {
    ...props,
    regionPreset: preset,
    ...resolveElContainerPresetVisibility(preset),
  };
}

/**
 * 编辑器状态管理 Store
 */
export const useEditorStore = defineStore("editor", () => {
  const doc = shallowRef<DocumentModel | null>(null);
  const docVersion = ref(0);
  const selectionVersion = ref(0);
  const history = shallowRef<History | null>(null);
  const selection = shallowRef<SelectionModel | null>(null);
  const serializer = shallowRef<Serializer>(markRaw(new Serializer()));
  const pageDrafts = ref<EditorPageDraftsMap>({});
  const pageTabState = ref<EditorPageTabState>({
    tabs: [],
    activeId: "",
  });
  const lockManager = shallowRef<PageLockManager | null>(null);
  const lockState = ref<PageLockState | null>(null);
  const readonlyState = ref<EditorReadonlyState>({ readonly: false });

  const projectId = ref("");
  const projectVariables = ref<Record<string, unknown>>({});
  const projectVariableGroups = ref<unknown[]>([]);
  const globalScripts = ref(getDefaultGlobalScripts());
  const projectName = ref("");
  const currentPageId = ref("");
  const pages = ref<PageNode[]>([]);
  const entryConfig = ref<EntryConfigFromApi>({});
  const isLoading = ref(false);
  const isSaving = ref(false);
  const canUndo = ref(false);
  const canRedo = ref(false);
  const error = ref("");
  const canvasMousePos = ref<{ x: number; y: number } | null>(null);
  const hoveredNodeType = ref("");

  /**
   * 解析当前页面根节点 ID
   * @returns {string | undefined}
   */
  const resolveCurrentRootNodeId = () => {
    if (!doc.value || !currentPageId.value) return undefined;
    return doc.value.getPage(currentPageId.value)?.rootNodeId;
  };

  let historyUnsubscribe: (() => void) | null = null;
  let lockUnsubscribe: (() => void) | null = null;
  let docUnsubscribe: (() => void) | null = null;
  let selectionUnsubscribe: (() => void) | null = null;

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
    readonlyState.value = lockManager.value.getReadonlyState() as EditorReadonlyState;
  };

  /**
   * 初始化锁管理器
   */
  const ensureLockManager = (): PageLockManager => {
    if (lockManager.value) return lockManager.value;

    const rawUser = Storage.getUserInfo();
    const userInfo =
      rawUser && typeof rawUser === "object" ? (rawUser as Record<string, unknown>) : {};
    const userId = String(userInfo.id ?? userInfo.userId ?? userInfo.user_id ?? "");

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
   */
  const initEditor = (schema: ProjectSchema) => {
    const normalizedSchema = normalizeLayoutSchema(schema) ?? schema;
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

    historyUnsubscribe = nextHistory.on("change", (payload: unknown) => {
      const p = payload as { canUndo: boolean; canRedo: boolean };
      canUndo.value = p.canUndo;
      canRedo.value = p.canRedo;
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
    canvasMousePos.value = null;
    hoveredNodeType.value = "";
    projectName.value = nextDoc.project?.name || projectName.value;
    currentPageId.value = nextDoc.entry?.homePageId || nextDoc.getAllPages()[0]?.id || "";
    if (!pages.value.length) {
      pages.value = nextDoc.getAllPages();
    }
    canUndo.value = nextHistory.canUndo();
    canRedo.value = nextHistory.canRedo();
  };

  /**
   * 获取页面锁
   */
  const acquirePageLock = async (pageId: string): Promise<LockResult> => {
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
  const togglePageLock = async (): Promise<TogglePageLockResult> => {
    if (!currentPageId.value) {
      return {
        success: false as const,
        reason: "error" as const,
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
    await loadProjectSettingsForStore(projectId.value, projectApi, {
      projectVariables,
      projectVariableGroups,
      globalScripts,
    });
  };

  /**
   * 保存工程级变量与脚本设置
   * @returns {Promise<{ok: boolean, error?: Error}>}
   */
  const saveProjectSettings = async (): Promise<EditorSaveProjectSettingsResult> => {
    if (!projectId.value) {
      return { ok: false, error: new Error("缺少工程信息") };
    }
    return saveProjectSettingsForStore(projectId.value, projectApi, {
      projectVariables,
      projectVariableGroups,
      globalScripts,
    });
  };

  /**
   * 刷新页面列表
   * @returns {Promise<{pages: Array, entryConfig: object}>}
   */
  const refreshPages = async () => {
    return refreshPagesForStore({
      projectId: projectId.value,
      pages,
      entryConfig,
      doc,
      projectApi,
    });
  };

  /**
   * 创建首页
   * @param {string} pid - 工程 ID
   * @returns {Promise<{ok: boolean, pageId?: string, error?: Error}>}
   */
  const createHomePage = async (pid: string): Promise<EditorCreateHomePageResult> => {
    const result = await createHomePageForStore(
      {
        entryConfig,
        currentPageId,
        initEditor,
        createBaseSchema,
        projectApi,
      },
      pid,
      refreshPages,
    );
    if (result.ok) {
      return { ok: true, pageId: result.pageId };
    }
    return { ok: false, error: result.error };
  };

  const loadProject = async (id: string): Promise<LoadProjectForStoreResult> => {
    return loadProjectForStore(id, {
      projectId,
      isLoading,
      error,
      doc,
      currentPageId,
      loadProjectSettings,
      releasePageLock,
      refreshPages,
      createHomePage,
      initEditor,
      createBaseSchema,
      resolveLandingPageId,
      projectApi,
      resolveProjectSchema,
    });
  };

  /**
   * 加载指定页面
   * @param {string} pageId - 页面 ID
   * @returns {Promise<{ok: boolean, error?: Error}>}
   */
  const loadPage = async (pageId: string): Promise<EditorLoadPageResult> => {
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
      const nextSchema = await resolvePageSchemaOnLoad({
        projectId: projectId.value,
        pageId,
        pageDrafts: pageDrafts.value || {},
        doc: doc.value,
        projectApi,
        resolveProjectSchema,
      });
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
   * 创建页面/分组
   * @param {{name: string, type: string, parentId?: string | null, schemaContent?: object}} payload - 创建参数
   * @returns {Promise<object>}
   */
  const createPage = async (payload: CreatePagePayload): Promise<CreatePageForStoreResult> => {
    return createPageForStore({ projectId, projectApi, refreshPages }, payload);
  };

  /**
   * 删除页面/分组
   * @param {string} pageId - 页面 ID
   * @param {"single" | "folder-only" | "cascade"} [mode] - 删除模式
   * @returns {Promise<void>}
   */
  const deletePage = async (pageId: string, mode?: EditorDeletePageMode) => {
    const mutationCtx = {
      projectId,
      currentPageId,
      pages,
      doc,
      history,
      projectApi,
      refreshPages,
      loadPage,
      initEditor,
      createBaseSchema,
    };
    return deletePageForStore(mutationCtx, pageId, mode);
  };

  /**
   * 更新页面 Schema
   * @param {string} pageId - 页面 ID
   * @param {object} [schema] - 页面 Schema
   * @returns {Promise<void>}
   */
  const updatePageSchema = async (pageId: string, schema?: EditorPageSchemaPayload) => {
    return updatePageSchemaForStore({ projectId, doc, serializer, projectApi }, pageId, schema);
  };

  /**
   * 移动页面到指定分组
   * @param {string} pageId - 页面 ID
   * @param {string | null} targetGroupId - 分组 ID
   * @param {string} [path] - 页面路径
   * @returns {Promise<void>}
   */
  const movePageToGroup = async (pageId: string, targetGroupId: string | null, path?: string) => {
    return movePageToGroupForStore(
      { projectId, projectApi, refreshPages },
      pageId,
      targetGroupId,
      path,
    );
  };

  /**
   * 重命名页面
   * @param {string} pageId - 页面 ID
   * @param {string} name - 页面名称
   * @param {string} [path] - 页面路径
   * @returns {Promise<void>}
   */
  const renamePage = async (pageId: string, name: string, path?: string) => {
    const mutationCtx = {
      projectId,
      currentPageId,
      pages,
      doc,
      history,
      projectApi,
      refreshPages,
      loadPage,
      initEditor,
      createBaseSchema,
    };
    return renamePageForStore(mutationCtx, pageId, name, path);
  };

  /**
   * 更新入口配置（仅更新本地）
   * @param {object} patch - 更新内容
   */
  const updateEntry = (patch: EditorEntryConfigPatch) => {
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
    await persistEntryConfigForStore({
      projectId,
      doc,
      entryConfig,
      projectApi,
    });
  };

  /**
   * 直接保存入口配置补丁（绕过命令栈限制）。
   * @param {EditorEntryConfigPatch} patch - 入口配置补丁
   * @returns {Promise<void>}
   */
  const saveEntryPatch = async (patch: EditorEntryConfigPatch) => {
    await saveEntryPatchForStore({
      projectId,
      doc,
      entryConfig,
      projectApi,
      patch,
    });
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
      await saveCurrentPageForStore({
        projectId,
        currentPageId,
        doc,
        serializer,
        pageDrafts,
        projectApi,
      });
    } finally {
      isSaving.value = false;
    }
  };

  /**
   * 暂存页面草稿（未保存的编辑内容）
   * @param {string} pageId - 页面 ID
   * @returns {void}
   */
  const savePageDraft = (pageId: string) => {
    savePageDraftForStore({ doc, serializer, pageDrafts }, pageId);
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
   * @returns {object | null}
   */
  const getPageDraft = (pageId: string): EditorPageDraftEntry | null => {
    if (!pageId) return null;
    return pageDrafts.value?.[pageId] ?? null;
  };

  /**
   * 更新页面标签状态
   * @param {Array} tabs - 标签列表
   * @param {string} activeId - 当前激活标签 ID
   * @returns {void}
   */
  const setPageTabState = (tabs: EditorPageTabItem[], activeId: string) => {
    pageTabState.value = {
      tabs: normalizeEditorPageTabsForState(tabs),
      activeId: activeId || "",
    };
  };

  /**
   * 更新当前页面配置（仅更新本地）
   */
  const updateCurrentPage = (patch: EditorUpdateCurrentPagePatch) => {
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
  const syncElContainerSections = (containerId: string, nextProps: Record<string, unknown>) => {
    if (!doc.value || !history.value) return;
    const d = doc.value;
    const h = history.value;
    const containerNode = d.getNode(containerId);
    if (!containerNode || containerNode.type !== "ElContainer") return;

    const sectionDefs = [
      { prop: "showHeader", type: "ElHeader", label: "Header区域" },
      { prop: "showAside", type: "ElAside", label: "Aside区域" },
      { prop: "showMain", type: "ElMain", label: "Main区域" },
      { prop: "showFooter", type: "ElFooter", label: "Footer区域" },
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
    const getOrderIndex = (type: string) => sectionOrder.indexOf(type);

    const executeCommand = (command: Command) => {
      if (h.isInTransaction?.()) {
        h.executeInTransaction(command);
        return;
      }
      h.execute(command);
    };

    const currentChildren = [...(containerNode.children || [])];
    const existingSectionMap = new Map();
    for (const childId of currentChildren) {
      const childNode = d.getNode(childId);
      if (childNode && sectionOrder.includes(childNode.type)) {
        existingSectionMap.set(childNode.type, childId);
      }
    }

    for (const section of sectionDefs) {
      if (!nextProps?.[section.prop] && existingSectionMap.has(section.type)) {
        executeCommand(new RemoveNodeCommand(existingSectionMap.get(section.type)!));
      }
    }

    const refreshedNode = d.getNode(containerId);
    if (!refreshedNode) return;
    const refreshedChildren = [...(refreshedNode.children || [])];

    const refreshedSectionMap = new Map();
    for (const childId of refreshedChildren) {
      const childNode = d.getNode(childId);
      if (childNode && sectionOrder.includes(childNode.type)) {
        refreshedSectionMap.set(childNode.type, childId);
      }
    }

    type SectionSizeKey = keyof typeof sectionSizeMap;
    const applySectionSize = (sectionType: SectionSizeKey) => {
      const config = sectionSizeMap[sectionType];
      if (!config) return;
      const sectionId = refreshedSectionMap.get(sectionType);
      if (!sectionId) return;
      const sectionNode = d.getNode(sectionId);
      if (!sectionNode) return;
      const nextValue = nextProps?.[config.containerProp] || config.fallback;
      if (sectionNode.props?.[config.prop] === nextValue) return;
      executeCommand(
        new UpdateNodeCommand(sectionNode.id, {
          props: { ...(sectionNode.props || {}), [config.prop]: nextValue },
        }),
      );
    };

    const resolveInsertIndex = (type: string) => {
      const orderIndex = getOrderIndex(type);
      let insertIndex = refreshedChildren.length;
      for (let i = 0; i < refreshedChildren.length; i += 1) {
        const childId = refreshedChildren[i]!;
        const childNode = d.getNode(childId);
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

    const buildUniqueLabel = (baseLabel: string) => {
      const label = baseLabel || "容器";
      const rootNodeId = resolveCurrentRootNodeId();
      if (isNodeLabelUnique(doc.value, rootNodeId, label)) return label;
      let index = 1;
      while (!isNodeLabelUnique(doc.value, rootNodeId, `${label}${index}`)) {
        index += 1;
      }
      return `${label}${index}`;
    };

    for (const section of sectionDefs) {
      if (!nextProps?.[section.prop]) continue;
      if (refreshedSectionMap.has(section.type)) continue;

      const manifest = componentRegistry.get(section.type);
      const sizeConfig = sectionSizeMap[section.type as SectionSizeKey];
      const sizeValue = sizeConfig
        ? nextProps?.[sizeConfig.containerProp] || sizeConfig.fallback
        : null;
      const childNode = createComponentNode(section.type, {
        parentNode: refreshedNode,
        label: buildUniqueLabel(manifest?.name || section.label || section.type),
        props: {
          ...(manifest?.defaultProps || {}),
          ...(sizeConfig && sizeValue ? { [sizeConfig.prop]: sizeValue } : {}),
        },
        style: { ...(manifest?.defaultStyle || {}) },
        layoutItem: buildFlexLayoutItem(),
      });

      const insertIndex = resolveInsertIndex(section.type);
      executeCommand(new InsertNodeCommand(containerId, insertIndex, childNode));
    }

    (Object.keys(sectionSizeMap) as SectionSizeKey[]).forEach((sectionType) => {
      if (!refreshedSectionMap.has(sectionType)) return;
      applySectionSize(sectionType);
    });
  };

  /**
   * 构建 Element Plus Layout 的唯一标签
   * @param {string} baseLabel - 标签前缀
   */
  const buildElLayoutUniqueLabel = (baseLabel: string) => {
    return buildUniqueNodeLabel(doc.value, resolveCurrentRootNodeId(), baseLabel);
  };

  /**
   * 同步 Element Plus Layout 行内列
   * @param {string} rowId - 行节点 ID
   * @param {Record<string, any>} nextProps - 最新属性
   * @param {{ forceSpanUpdate?: boolean, anchorColId?: string }} [options] - 同步选项
   */
  const syncElLayoutRowColumns = (
    rowId: string,
    nextProps: Record<string, unknown>,
    options: { forceSpanUpdate?: boolean; anchorColId?: string } = {},
  ) => {
    if (!doc.value || !history.value) return;
    const d = doc.value;
    const h = history.value;
    const rowNode = d.getNode(rowId);
    if (!rowNode || rowNode.type !== "ElLayoutRow") return;

    const columns = Math.max(
      1,
      Math.min(24, Number(nextProps?.columns || rowNode.props?.columns || 3)),
    );
    const baseSpan = Math.max(1, Math.floor(24 / columns));
    const remainder = 24 - baseSpan * columns;
    const getSpanByIndex = (index: number) => baseSpan + (index < remainder ? 1 : 0);

    const executeCommand = (command: Command) => {
      if (h.isInTransaction?.()) {
        h.executeInTransaction(command);
        return;
      }
      h.execute(command);
    };

    const children = [...(rowNode.children || [])];
    const colIds = children.filter((childId) => {
      const childNode = d.getNode(childId);
      return childNode?.type === "ElCol";
    });

    if (colIds.length > columns) {
      for (let i = colIds.length - 1; i >= columns; i -= 1) {
        executeCommand(new RemoveNodeCommand(colIds[i]!));
      }
    }

    let refreshedNode = d.getNode(rowId);
    if (!refreshedNode) return;
    let refreshedChildren = [...(refreshedNode.children || [])];
    let refreshedCols = refreshedChildren.filter((childId) => {
      const childNode = d.getNode(childId);
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

      refreshedNode = d.getNode(rowId);
      refreshedChildren = [...(refreshedNode?.children || [])];
      refreshedCols = refreshedChildren.filter((childId) => {
        const childNode = d.getNode(childId);
        return childNode?.type === "ElCol";
      });
    }

    const shouldUpdateSpan = options.forceSpanUpdate || Object.hasOwn(nextProps || {}, "columns");
    if (!shouldUpdateSpan) return;
    refreshedCols = refreshedCols.slice(0, columns);
    const anchorColId = options.anchorColId;
    if (anchorColId && refreshedCols.includes(anchorColId)) {
      const anchorIndex = refreshedCols.indexOf(anchorColId);
      const spans = refreshedCols.map((colId) => {
        const colNode = d.getNode(colId);
        const span = Number(colNode?.props?.span) || 1;
        return Math.max(1, Math.min(24, span));
      });
      const offsets = refreshedCols.map((colId) => {
        const colNode = d.getNode(colId);
        const offset = Number(colNode?.props?.offset) || 0;
        return Math.max(0, Math.min(24, offset));
      });
      const fixedSpanTotal = spans.slice(0, anchorIndex + 1).reduce((sum, value) => sum + value, 0);
      const totalOffset = offsets.reduce((sum, value) => sum + value, 0);
      const rightCount = Math.max(0, refreshedCols.length - anchorIndex - 1);
      if (rightCount === 0) return;
      const remainingUnits = Math.max(rightCount, 24 - totalOffset - fixedSpanTotal);
      const base = Math.floor(remainingUnits / rightCount);
      const rem = remainingUnits - base * rightCount;
      for (let i = anchorIndex + 1; i < refreshedCols.length; i += 1) {
        const colId = refreshedCols[i];
        if (!colId) continue;
        const colNode = d.getNode(colId);
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
      const colNode = d.getNode(colId);
      const offset = Number(colNode?.props?.offset) || 0;
      return Math.max(0, Math.min(24, offset));
    });
    // 按剩余格数等分列宽，避免只压缩右侧区域
    const totalOffset = offsets.reduce((sum, value) => sum + value, 0);
    const remainingUnits = Math.max(columns, 24 - totalOffset);
    const base = Math.floor(remainingUnits / columns);
    const rem = remainingUnits - base * columns;
    for (let i = 0; i < refreshedCols.length; i += 1) {
      const colId = refreshedCols[i];
      if (!colId) continue;
      const colNode = d.getNode(colId);
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
  const syncElLayoutRows = (layoutId: string, nextProps: Record<string, unknown>) => {
    if (!doc.value || !history.value) return;
    const d = doc.value;
    const h = history.value;
    const layoutNode = d.getNode(layoutId);
    if (!layoutNode || layoutNode.type !== "ElLayout") return;

    const rows = Math.max(1, Math.min(24, Number(nextProps?.rows || 1)));
    const executeCommand = (command: Command) => {
      if (h.isInTransaction?.()) {
        h.executeInTransaction(command);
        return;
      }
      h.execute(command);
    };

    const children = [...(layoutNode.children || [])];
    const rowIds = children.filter((childId) => {
      const childNode = d.getNode(childId);
      return childNode?.type === "ElLayoutRow";
    });

    if (rowIds.length > rows) {
      for (let i = rowIds.length - 1; i >= rows; i -= 1) {
        const rowId = rowIds[i];
        if (!rowId) continue;
        executeCommand(new RemoveNodeCommand(rowId));
      }
    }

    let refreshedNode = d.getNode(layoutId);
    if (!refreshedNode) return;
    let refreshedChildren = [...(refreshedNode.children || [])];
    let refreshedRows = refreshedChildren.filter((childId) => {
      const childNode = d.getNode(childId);
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

      refreshedNode = d.getNode(layoutId);
      refreshedChildren = [...(refreshedNode?.children || [])];
      refreshedRows = refreshedChildren.filter((childId) => {
        const childNode = d.getNode(childId);
        return childNode?.type === "ElLayoutRow";
      });
      syncElLayoutRowColumns(childNode.id, childNode.props || {}, {
        forceSpanUpdate: true,
      });
    }

    refreshedRows = refreshedRows.slice(0, rows);
    refreshedRows.forEach((rowId) => {
      const rowNode = d.getNode(rowId);
      if (!rowNode) return;
      const colCount = (rowNode.children || []).filter((childId) => {
        const childNode = d.getNode(childId);
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
   */
  const updateNode = (nodeId: string, patch: Partial<ComponentNode>) => {
    if (!doc.value || !history.value || !nodeId) return false;
    if (!ensureEditable()) return false;

    const d = doc.value;
    const h = history.value;
    const node = d.getNode(nodeId);
    if (!node) return false;
    const limitedPatch = clampElContainerSizePatch(d, node, patch);
    const clampedColumnsPatch = clampElLayoutRowColumnsPatch(node, limitedPatch);
    const clampedSpanPatch = clampElColSpanPatch(d, node, clampedColumnsPatch);
    const clampedOffsetPatch = clampElColOffsetPatch(d, node, clampedSpanPatch);
    const clampedShiftPatch = clampElColShiftPatch(d, node, clampedOffsetPatch);
    const nextPatch = syncAbsoluteSizePatch(node, clampedShiftPatch);
    const hasPropPatch = nextPatch?.props && typeof nextPatch === "object";
    const hasContainerPresetPatch =
      node?.type === "ElContainer" &&
      hasPropPatch &&
      Object.hasOwn(nextPatch.props || {}, "regionPreset");
    const mergedPropsBase = hasPropPatch
      ? { ...(node.props || {}), ...(nextPatch.props || {}) }
      : { ...(node.props || {}) };
    const mergedProps = hasContainerPresetPatch
      ? applyElContainerPresetProps(mergedPropsBase)
      : mergedPropsBase;
    const mergedPatch = hasPropPatch ? { ...nextPatch, props: mergedProps } : nextPatch;
    const layoutAdjustedPatch = hasPropPatch
      ? syncElLayoutMinHeightPatch(node, mergedPatch, mergedProps || {})
      : mergedPatch;
    const shouldSyncContainer = node?.type === "ElContainer" && hasPropPatch;
    const shouldSyncLayout = node?.type === "ElLayout" && hasPropPatch;
    const shouldSyncLayoutRow = node?.type === "ElLayoutRow" && hasPropPatch;
    const shouldSyncLayoutRowFromCol =
      node?.type === "ElCol" &&
      hasPropPatch &&
      (Object.hasOwn(nextPatch?.props || {}, "offset") ||
        Object.hasOwn(nextPatch?.props || {}, "span"));

    if (
      !shouldSyncContainer &&
      !shouldSyncLayout &&
      !shouldSyncLayoutRow &&
      !shouldSyncLayoutRowFromCol
    ) {
      h.execute(new UpdateNodeCommand(nodeId, layoutAdjustedPatch));
      return true;
    }

    const finalPatch = layoutAdjustedPatch;
    const shouldCommit = !h.isInTransaction?.();
    if (shouldCommit) {
      h.beginTransaction();
    }

    h.executeInTransaction(new UpdateNodeCommand(nodeId, finalPatch));
    if (shouldSyncContainer) {
      syncElContainerSections(nodeId, mergedProps || {});
    }
    if (shouldSyncLayout) {
      syncElLayoutRows(nodeId, mergedProps || {});
    }
    if (shouldSyncLayoutRow && Object.hasOwn(nextPatch.props || {}, "columns")) {
      syncElLayoutRowColumns(nodeId, mergedProps || {}, {
        forceSpanUpdate: true,
      });
    }
    if (shouldSyncLayoutRowFromCol) {
      const parentNode = d.getParent(nodeId);
      if (parentNode?.type === "ElLayoutRow") {
        syncElLayoutRowColumns(parentNode.id, parentNode.props || {}, {
          forceSpanUpdate: true,
          anchorColId: nodeId,
        });
      }
    }

    if (shouldCommit) {
      h.commitTransaction("更新容器布局");
    }
    const executeParentUpdate = (parentId: string, patch: { props: Record<string, unknown> }) => {
      if (h.isInTransaction?.()) {
        h.executeInTransaction(new UpdateNodeCommand(parentId, patch));
        return;
      }
      h.execute(new UpdateNodeCommand(parentId, patch));
    };
    if (node?.type === "ElHeader") {
      const parentNode = d.getParent(nodeId);
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
      const parentNode = d.getParent(nodeId);
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
      const parentNode = d.getParent(nodeId);
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
   */
  const updateGraphic = (graphicId: string, patch: Partial<GraphicNode>) => {
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
  const currentPage = computed((): PageNode | null => {
    if (!doc.value || !currentPageId.value) return null;
    return doc.value.getPage(currentPageId.value);
  });

  /**
   * 获取用于图层操作的节点 ID
   * @param {string | undefined} nodeId - 指定节点
   * @returns {string}
   */
  const resolveLayerTarget = (nodeId?: string) =>
    resolveLayerTargetFromSelection(selection.value, nodeId);

  /**
   * 切换当前页面
   * @param {string} pageId - 页面 ID
   * @returns {Promise<{ok: boolean, error?: Error}>}
   */
  const setCurrentPage = async (pageId: string) => {
    if (!pageId || pageId === currentPageId.value) {
      return { ok: true };
    }
    return loadPage(pageId);
  };

  /**
   * 插入组件节点
   */
  const insertNode = (
    type: string,
    parentId: string,
    index: number | undefined,
    options: { dropPosition?: { x: number; y: number }; autoSelectInserted?: boolean } = {},
  ): ComponentNode | null => {
    if (!doc.value || !history.value || !parentId) return null;
    if (!ensureEditable()) return null;
    const d = doc.value;
    const h = history.value;

    let parentNode = d.getNode(parentId);
    if (!parentNode) return null;
    let resolvedParentId = parentId;
    // isRegionType 且排除 ElCol：ElContainer 内可落点区域不含 ElCol
    const isElContainerRegion = isRegionType(type) && type !== "ElCol";
    let shouldReplaceRegionChildren = false;
    if (parentNode.type === "ElContainer" && !isElContainerRegion) {
      const mainChildId = (parentNode.children || []).find((childId) => {
        const childNode = d.getNode(childId);
        return childNode?.type === "ElMain";
      });
      if (mainChildId) {
        const mainNode = d.getNode(mainChildId);
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
    const insertIndex =
      typeof index === "number" && Number.isInteger(index)
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
    let layoutItem: LayoutItem | null = null;
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
    const isCollapseContainer = parentNode.type === "Collapse";
    const isTabbedLikeContainer = isTabsContainer || isCollapseContainer;
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
    if (isTabbedLikeContainer) {
      const shouldFillContainer = Boolean(isLayoutContainer || manifest?.isContainer);
      nodeStyle = {
        ...nodeStyle,
        ...(shouldFillContainer ? { width: "100%", height: "100%" } : { height: "auto" }),
      };
      if (shouldFillContainer && isLayoutContainer) {
        delete nodeStyle.minHeight;
        delete nodeStyle.minWidth;
      }
    }

    const baseLabel = manifest?.name || type;
    const existingLabels = new Set<string>();
    const rootId = currentPage.value?.rootNodeId;
    if (rootId && d) {
      const stack = [rootId];
      while (stack.length) {
        const id = stack.pop()!;
        const current = d.getNode(id);
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
    if (type === "ElContainer") {
      node.props = applyElContainerPresetProps({ ...(node.props || {}) });
    }
    if (type === "Menu") {
      node.detailConfig = getMenuDefaultDetailConfig();
      node.props = { ...(node.props || {}), ...getMenuDefaultProps() };
    }
    if (type === "ElLayoutRow") {
      const rawColumns = Number(node.props?.columns);
      const normalizedColumns =
        Number.isFinite(rawColumns) && rawColumns > 0 ? Math.min(24, Math.floor(rawColumns)) : 3;
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
      parentNode.type === "FormLayout" ||
      parentNode.type === "Tabs" ||
      parentNode.type === "Collapse" ||
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
        node.style = {
          ...(node.style || {}),
          ...(parentDescriptor.childStyle(parentNode.type) as Record<string, unknown>),
        };
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
      (type === "ElContainer" || type === "ElLayout" || type === "ElLayoutRow") &&
      !h.isInTransaction?.();
    if (shouldWrapTransaction) {
      h.beginTransaction();
    }

    if (type === "ElContainer") {
      h.executeInTransaction(new InsertNodeCommand(parentId, insertIndex, node));
      syncElContainerSections(node.id, node.props || {});
    } else if (type === "ElLayout") {
      h.executeInTransaction(new InsertNodeCommand(parentId, insertIndex, node));
      syncElLayoutRows(node.id, node.props || {});
    } else if (type === "ElLayoutRow") {
      h.executeInTransaction(new InsertNodeCommand(parentId, insertIndex, node));
      syncElLayoutRowColumns(node.id, node.props || {}, {
        forceSpanUpdate: true,
      });
    } else {
      const shouldCommit = shouldReplaceRegionChildren && !h.isInTransaction?.();
      if (shouldCommit) {
        h.beginTransaction();
      }
      if (shouldReplaceRegionChildren) {
        const existingChildren = [...(parentNode.children || [])];
        for (const childId of existingChildren) {
          h.executeInTransaction?.(new RemoveNodeCommand(childId));
          if (!h.executeInTransaction) {
            h.execute(new RemoveNodeCommand(childId));
          }
        }
      }
      h.execute(new InsertNodeCommand(resolvedParentId, insertIndex, node));
      if (shouldCommit) {
        h.commitTransaction("更新Main区域");
      }
    }

    if (shouldWrapTransaction) {
      h.commitTransaction("插入容器布局");
    }
    const shouldAutoSelect = options.autoSelectInserted !== false;
    if (shouldAutoSelect) {
      selection.value?.select(createSelectableElement("node", node.id));
    }

    return node;
  };

  /**
   * 在 ElLayoutRow 中按左右插入列
   * @param {"left" | "right"} direction - 插入方向
   * @param {string} [colId] - 参考列节点 ID
   * @returns {boolean}
   */
  const insertElColByDirection = (direction: "left" | "right", colId?: string) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;
    const d = doc.value;
    const h = history.value;

    const targetId = colId || resolveLayerTarget();
    if (!targetId) return false;
    const targetNode = d.getNode(targetId);
    if (targetNode?.type !== "ElCol") return false;

    const rowNode = d.getParent(targetId);
    if (!rowNode || rowNode.type !== "ElLayoutRow") return false;

    const children = [...(rowNode.children || [])];
    const currentIndex = children.indexOf(targetId);
    if (currentIndex < 0) return false;

    const colCount = children.filter((childId) => {
      const childNodeItem = d.getNode(childId);
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
    const shouldCommit = !h.isInTransaction?.();
    if (shouldCommit) {
      h.beginTransaction();
    }

    h.executeInTransaction(new InsertNodeCommand(rowNode.id, insertIndex, childNode));
    const nextColumns = Math.max(1, colCount + 1);
    updateNode(rowNode.id, {
      props: { ...(rowNode.props || {}), columns: nextColumns },
    });

    selection.value?.select(createSelectableElement("node", childNode.id));
    if (shouldCommit) {
      h.commitTransaction("新增布局列");
    }
    return true;
  };

  /**
   * 在当前列左侧新增一列
   * @param {string} [colId] - 参考列节点 ID
   * @returns {boolean}
   */
  const insertElColLeft = (colId?: string) => {
    return insertElColByDirection("left", colId);
  };

  /**
   * 在当前列右侧新增一列
   * @param {string} [colId] - 参考列节点 ID
   * @returns {boolean}
   */
  const insertElColRight = (colId?: string) => {
    return insertElColByDirection("right", colId);
  };

  /**
   * 在 ElLayoutRow 上下插入行
   * @param {"up" | "down"} direction - 插入方向
   * @param {string} [rowId] - 参考行节点 ID
   * @returns {boolean}
   */
  const insertElLayoutRowByDirection = (direction: "up" | "down", rowId?: string) => {
    if (!doc.value || !history.value) return false;
    if (!ensureEditable()) return false;
    const d = doc.value;
    const h = history.value;

    const targetId = rowId || resolveLayerTarget();
    if (!targetId) return false;
    const targetNode = d.getNode(targetId);
    if (targetNode?.type !== "ElLayoutRow") return false;

    const layoutNode = d.getParent(targetId);
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
    const shouldCommit = !h.isInTransaction?.();
    if (shouldCommit) {
      h.beginTransaction();
    }

    h.executeInTransaction(new InsertNodeCommand(layoutNode.id, insertIndex, rowNode));
    syncElLayoutRowColumns(rowNode.id, rowNode.props || {}, {
      forceSpanUpdate: true,
    });

    const refreshedLayout = d.getNode(layoutNode.id);
    const rowCount = (refreshedLayout?.children || []).filter((childId) => {
      const childNode = d.getNode(childId);
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
      h.commitTransaction("新增布局行");
    }
    return true;
  };

  /**
   * 在当前行上方新增一行
   * @param {string} [rowId] - 参考行节点 ID
   * @returns {boolean}
   */
  const insertElLayoutRowUp = (rowId?: string) => {
    return insertElLayoutRowByDirection("up", rowId);
  };

  /**
   * 在当前行下方新增一行
   * @param {string} [rowId] - 参考行节点 ID
   * @returns {boolean}
   */
  const insertElLayoutRowDown = (rowId?: string) => {
    return insertElLayoutRowByDirection("down", rowId);
  };

  /**
   * 校验当前页面组件名称是否唯一
   * @param {string} name - 组件名称
   * @param {string} [excludeId] - 排除的节点ID
   * @returns {boolean}
   */
  const isLabelUnique = (name: string, excludeId?: string) =>
    isNodeLabelUnique(doc.value, currentPage.value?.rootNodeId, name, excludeId);

  /**
   * 删除选中的节点
   * @returns {boolean}
   */
  const removeSelectedNodes = () => {
    if (!doc.value || !history.value || !selection.value) return false;
    if (!ensureEditable()) return false;
    const d = doc.value;
    const h = history.value;

    const rootId = currentPage.value?.rootNodeId;
    const selected = selection.value.getSelectedElements?.() || [];
    const nodeIds = selected.filter((el) => el.kind === "node").map((el) => el.id);
    if (!nodeIds.length) return false;

    // 根节点不可删除，且不应阻断其子节点的删除
    const nodeSet = new Set(nodeIds.filter((nodeId) => nodeId !== rootId));
    const deletable = [...nodeSet].filter((nodeId) => {
      const ancestors = d.getAncestors(nodeId) || [];
      return !ancestors.some((ancestor) => ancestor.id !== rootId && nodeSet.has(ancestor.id));
    });

    if (!deletable.length) return false;

    const affectedRowIds = new Set<string>();
    const affectedLayoutIds = new Set<string>();
    deletable.forEach((nodeId) => {
      const node = d.getNode(nodeId);
      if (node?.type !== "ElCol") return;
      const parentNode = d.getParent(nodeId);
      if (parentNode?.type === "ElLayoutRow") {
        affectedRowIds.add(parentNode.id);
      }
    });
    deletable.forEach((nodeId) => {
      const node = d.getNode(nodeId);
      if (node?.type !== "ElLayoutRow") return;
      const parentNode = d.getParent(nodeId);
      if (parentNode?.type === "ElLayout") {
        affectedLayoutIds.add(parentNode.id);
      }
    });
    const shouldWrapTransaction =
      (affectedRowIds.size > 0 || affectedLayoutIds.size > 0) && !h.isInTransaction?.();
    if (shouldWrapTransaction) {
      h.beginTransaction();
    }

    deletable.forEach((nodeId) => {
      h.execute(new RemoveNodeCommand(nodeId));
    });
    affectedRowIds.forEach((rowId) => {
      const rowNode = d.getNode(rowId);
      if (!rowNode) return;
      const colCount = (rowNode.children || []).filter((childId) => {
        const childNode = d.getNode(childId);
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
      const layoutNode = d.getNode(layoutId);
      if (!layoutNode) return;
      const rowCount = (layoutNode.children || []).filter((childId) => {
        const childNode = d.getNode(childId);
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
      h.commitTransaction("调整布局列");
    }
    selection.value.clearSelection?.();
    return true;
  };

  /**
   * 删除指定节点
   * @param {string} nodeId - 节点 ID
   * @returns {boolean}
   */
  const removeNode = (nodeId: string) => {
    if (!doc.value || !history.value || !nodeId) return false;
    if (!ensureEditable()) return false;

    const d = doc.value;
    const rootId = currentPage.value?.rootNodeId;
    if (nodeId === rootId) return false;

    const node = d.getNode(nodeId);
    const colParentNode = node?.type === "ElCol" ? d.getParent(nodeId) : null;
    const rowParentNode = node?.type === "ElLayoutRow" ? d.getParent(nodeId) : null;
    const shouldSyncRow = colParentNode?.type === "ElLayoutRow";
    const shouldSyncLayout = rowParentNode?.type === "ElLayout";
    const shouldWrapTransaction =
      (shouldSyncRow || shouldSyncLayout) && !history.value.isInTransaction?.();
    if (shouldWrapTransaction) {
      history.value.beginTransaction();
    }

    history.value.execute(new RemoveNodeCommand(nodeId));
    if (shouldSyncRow) {
      const rowNode = d.getNode(colParentNode.id);
      if (rowNode) {
        const colCount = (rowNode.children || []).filter((childId) => {
          const childNode = d.getNode(childId);
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
      const layoutNode = d.getNode(rowParentNode.id);
      if (layoutNode) {
        const rowCount = (layoutNode.children || []).filter((childId) => {
          const childNode = d.getNode(childId);
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
   * 上移节点
   * @param {string} [nodeId] - 节点 ID
   * @returns {boolean}
   */
  const moveNodeUp = (nodeId?: string) => {
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
  const moveNodeDown = (nodeId?: string) => {
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
  const moveNodeToTop = (nodeId?: string) => {
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
  const moveNodeToBottom = (nodeId?: string) => {
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
  const toggleNodeVisibility = (nodeId: string) => {
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
  const toggleNodeLock = (nodeId: string) => {
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
  const alignElements = (
    alignType: "left" | "centerH" | "right" | "top" | "centerV" | "bottom",
  ) => {
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
  const distributeElements = (direction: "horizontal" | "vertical") => {
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
  const matchElementSize = (mode: "width" | "height" | "both") => {
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
  const moveSelectedByDelta = (dx: number, dy: number) => {
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
        const nextLayoutItem = JSON.parse(JSON.stringify(node.layoutItem || {}));
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

  const _clipboard = ref<EditorClipboardRefValue>(null);

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

    const snapshots: EditorClipboardItem[] = [];
    for (const el of elements) {
      if (el.kind === "node") {
        const node = doc.value.getNode(el.id);
        if (node) {
          snapshots.push({
            kind: "node",
            data: JSON.parse(JSON.stringify(node)) as ComponentNode,
          });
        }
      } else if (el.kind === "graphic") {
        const graphic = doc.value.getGraphic(el.id);
        if (graphic) {
          snapshots.push({
            kind: "graphic",
            data: JSON.parse(JSON.stringify(graphic)) as GraphicNode,
          });
        }
      }
    }
    if (!snapshots.length) return false;
    _clipboard.value = snapshots;
    return true;
  };

  /**
   * 从节点数据获取包围盒（用于计算粘贴偏移）
   * @param {object} node - 节点数据
   * @returns {{ x: number, y: number, w: number, h: number } | null}
   */
  const getNodeBoundsFromData = (
    node: (Partial<ComponentNode> & Record<string, unknown>) | null | undefined,
  ): { x: number; y: number; w: number; h: number } | null => {
    if (node == null) return null;
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
    const styleW = node.style?.width;
    const styleH = node.style?.height;
    const toDim = (v: unknown, fallback: number) => {
      if (typeof v === "number" && Number.isFinite(v)) return v;
      if (typeof v === "string") {
        const n = Number.parseFloat(v);
        return Number.isFinite(n) ? n : fallback;
      }
      return fallback;
    };
    return {
      x: typeof node.style?.left === "number" ? node.style.left : 0,
      y: typeof node.style?.top === "number" ? node.style.top : 0,
      w: toDim(styleW, 100),
      h: toDim(styleH, 100),
    };
  };

  /**
   * 将偏移应用到节点数据的位置
   * @param {object} node - 节点数据（会被修改）
   * @param {number} dx - 水平偏移
   * @param {number} dy - 垂直偏移
   */
  const applyOffsetToNodeData = (
    node: Partial<ComponentNode> & Record<string, unknown>,
    dx: number,
    dy: number,
  ) => {
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
   * @param {{ x: number, y: number } | null} [targetPos] - 画布坐标，粘贴到该位置（包围盒中心对齐）；null 时使用默认 20px 偏移
   * @returns {boolean}
   */
  const pasteNodes = (targetPos: EditorPasteTargetPosition | null = null) => {
    if (!doc.value || !history.value || !_clipboard.value?.length) return false;
    if (!ensureEditable()) return false;

    const rootId = currentPage.value?.rootNodeId;
    if (!rootId) return false;

    const nodeItems = _clipboard.value.filter((item) => item.kind === "node");
    let offset: { x: number; y: number } = { x: 20, y: 20 };

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
          history.value.execute(new DuplicateNodeCommand(sourceId, undefined, offset));
        } else {
          const cloned = JSON.parse(JSON.stringify(item.data));
          cloned.id = crypto.randomUUID().replace(UUID_DASH_REGEX, "").substring(0, 12);
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
    const nodeIds = elements.filter((el) => el.kind === "node").map((el) => el.id);
    if (!nodeIds.length) return false;

    const rootId = currentPage.value?.rootNodeId;
    for (const nodeId of nodeIds) {
      if (nodeId === rootId) continue;
      history.value.execute(new DuplicateNodeCommand(nodeId, undefined, { x: 20, y: 20 }));
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
    canvasMousePos,
    hoveredNodeType,
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
    saveEntryPatch,
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
