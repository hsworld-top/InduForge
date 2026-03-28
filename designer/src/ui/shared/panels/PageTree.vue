<!--
  PageTree - 页面树
  展示工程页面层级（基础页面、自定义页面），支持搜索、新建、重命名、删除、拖拽排序
-->
<script setup lang="ts">
import { ElCheckbox, ElMessage, ElMessageBox } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, h, inject, ref, watch, watchEffect } from "vue";
import IconEpArrowDown from "~icons/ep/arrow-down";
import IconEpArrowRight from "~icons/ep/arrow-right";
import IconEpDocument from "~icons/ep/document";
import IconEpFolder from "~icons/ep/folder";
import IconEpMoreFilled from "~icons/ep/more-filled";
import { useEditorStore } from "@/stores/editor-store";
import {
  getPageTreeOrderStorageKey,
  moveIdBefore,
  ROOT_CONTAINER_KEY,
  validatePageName,
} from "./page-tree-utils";
import PageTreeBasicPagesSection from "./PageTreeBasicPagesSection.vue";
import PageTreeCreateDialog from "./PageTreeCreateDialog.vue";
import PageTreePageSearch from "./PageTreePageSearch.vue";

type FixedSystemType = "home" | "login" | "logout";
type CreateType = "page" | "folder" | FixedSystemType;
type PageNodeLike = any;
type SelectedNodeLike = any;
type DragTargetLike = any;
type BasicPageMetaLike = any;
type BasicSlotLike = any;
type MoveTargetLike = any;
interface CreateTypeOptionLike {
  value: CreateType;
  label: string;
  desc: string;
  icon: any;
}

type OpenPageTabLike = (pageId: string) => void;

const showSuccess = (message: string): void => {
  (ElMessage as any).success({ message });
};

const showWarning = (message: string): void => {
  (ElMessage as any).warning({ message });
};

const showError = (message: string): void => {
  (ElMessage as any).error({ message });
};

// 注入打开标签页的方法
const openPageTab = inject<OpenPageTabLike | null>("openPageTab", null) as any;

const editorStore = useEditorStore();
const { pages, currentPageId, entryConfig, canUndo, projectId } = storeToRefs(editorStore);
const selectedNode = ref<any>(null);
const createDialogVisible = ref(false);
const createForm = ref<any>({ type: "page", name: "", parentId: null });
const creating = ref(false);
const skipSwitchPrompt = ref(false);
const pageSwitchPromptKey = "designer.pageSwitchPrompt.disabled";
const pageOrderMap = ref<Record<string, any>>({});
const dragState = ref<any>(null);
const dragOverTarget = ref<any>(null);
let dragOverFrameId = 0;
let pendingDragOverTarget: any = null;

// 新建弹窗类型选项
const createTypeOptions: CreateTypeOptionLike[] = [
  {
    value: "page",
    label: "业务页面",
    desc: "创建一个普通业务页面",
    icon: IconEpDocument,
  },
  {
    value: "folder",
    label: "分组",
    desc: "创建一个分组来组织页面",
    icon: IconEpFolder,
  },
];

// 新建弹窗标题
const createDialogTitle = computed<string>(() => {
  const option = createTypeOptions.find((o) => o.value === createForm.value.type);
  return `新建${option?.label || "页面"}`;
});

// 表单验证规则
const createFormRules: Record<string, any> = {
  name: [
    { required: true, message: "请输入名称", trigger: "blur" },
    { min: 1, max: 50, message: "名称长度在 1 到 50 个字符", trigger: "blur" },
    {
      validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
        const result = validatePageName(value);
        if (!result.valid) {
          callback(new Error(result.message));
          return;
        }
        callback();
      },
      trigger: "blur",
    },
  ],
};

const BASIC_PAGE_META: Record<FixedSystemType, BasicPageMetaLike> = {
  home: { label: "首页", path: "/" },
  login: { label: "登录页", path: "/login" },
  logout: { label: "登出页", path: "/logout" },
};
const PAGE_TREE_SEGMENT_SPACE_RE = /\s+/g;
const PAGE_TREE_SEGMENT_RE = /[/?#\\]+/g;

/**
 * 获取基础页面固定配置
 * @param {"home" | "login" | "logout"} type - 基础页面类型
 * @returns {{ label: string, path: string }}
 */
function getBasicPageMeta(type: FixedSystemType): BasicPageMetaLike {
  return BASIC_PAGE_META[type] || { label: "", path: "/" };
}

/**
 * 判断是否为固定基础页创建类型
 * @param {string} type - 创建类型
 * @returns {boolean}
 */
function isFixedBasicCreateType(type: string): boolean {
  return ["home", "login", "logout"].includes(type);
}

/**
 * 获取基础页面类型
 * @param {import('@/editor-core').PageNode | undefined | null} page - 页面
 * @returns {"home" | "login" | "logout" | null}
 */
function getFixedSystemType(page: PageNodeLike | null | undefined): FixedSystemType | null {
  if (!page) return null;
  if (entryConfig.value?.homePageId === page.id) return "home";
  if (entryConfig.value?.loginPageId === page.id || page.path === "/login") return "login";
  if (entryConfig.value?.logoutPageId === page.id || page.path === "/logout") return "logout";
  return null;
}

/**
 * 在分组中创建页面
 * @param {string} folderId - 分组ID
 */
function handleCreateInFolder(folderId: string): void {
  createForm.value = { type: "page", name: "", parentId: folderId };
  createDialogVisible.value = true;
}

// 搜索和展开状态
const searchText = ref("");
const folderStates = ref<Record<string, boolean>>({});

/**
 * 切换分组展开状态
 * @param {string} folderId - 分组ID
 */
function toggleFolder(folderId: string): void {
  folderStates.value[folderId] = !folderStates.value[folderId];
}

/**
 * 点击分组
 * @param {Object} folder - 分组
 */
function handleFolderClick(folder: PageNodeLike): void {
  selectedNode.value = {
    id: folder?.id,
    type: "folder",
    label: folder?.name || "",
    parentId: null,
  };
  toggleFolder(folder.id);
}

function getProjectOrderStorageKey(): string {
  return getPageTreeOrderStorageKey(projectId.value || "");
}

function loadPageOrderMap(): void {
  try {
    const raw = localStorage.getItem(getProjectOrderStorageKey());
    pageOrderMap.value = raw ? JSON.parse(raw) || {} : {};
  } catch {
    pageOrderMap.value = {};
  }
}

function persistPageOrderMap(): void {
  try {
    localStorage.setItem(getProjectOrderStorageKey(), JSON.stringify(pageOrderMap.value || {}));
  } catch {
    // ignore
  }
}

function getContainerOrderKey(parentId: string | null = null): string {
  return parentId || ROOT_CONTAINER_KEY;
}

function updateContainerOrder(
  containerKey: string,
  orderedIds: Array<string | null | undefined>,
): void {
  const nextMap = { ...(pageOrderMap.value || {}) };
  nextMap[containerKey] = Array.from(new Set((orderedIds || []).filter(Boolean)));
  pageOrderMap.value = nextMap;
  persistPageOrderMap();
}

function sortItemsByStoredOrder(items: Array<any>, containerKey: string): Array<any> {
  const source = Array.isArray(items) ? [...items] : [];
  const stored = Array.isArray(pageOrderMap.value?.[containerKey])
    ? pageOrderMap.value[containerKey]
    : [];
  const orderIndexMap = new Map(stored.map((id, index) => [id, index]));
  return source.sort((a, b) => {
    const aIndex = orderIndexMap.has(a.id) ? orderIndexMap.get(a.id) : Number.MAX_SAFE_INTEGER;
    const bIndex = orderIndexMap.has(b.id) ? orderIndexMap.get(b.id) : Number.MAX_SAFE_INTEGER;
    if (aIndex !== bIndex) {
      return Number(aIndex) - Number(bIndex);
    }
    return (a.name || "").localeCompare(b.name || "", "zh-CN");
  });
}

function ensureContainerOrder(containerKey: string, items: Array<any>): void {
  const nextIds = (items || []).map((item) => item.id);
  const currentIds = Array.isArray(pageOrderMap.value?.[containerKey])
    ? pageOrderMap.value[containerKey].filter((id) => nextIds.includes(id))
    : [];
  const missingIds = nextIds.filter((id) => !currentIds.includes(id));
  if (missingIds.length === 0 && currentIds.length === nextIds.length) return;
  updateContainerOrder(containerKey, [...currentIds, ...missingIds]);
}

watch(
  () => createForm.value.type,
  (type) => {
    if (type !== "page") {
      createForm.value.parentId = null;
    }
  },
);

/**
 * 获取页面显示名称
 * @param {import('@/editor-core').PageNode} page - 页面
 * @returns {string}
 */
function getPageLabel(page: PageNodeLike): string {
  return page.name || page.title || page.id;
}

function matchesKeyword(page: PageNodeLike): boolean {
  const keyword = searchText.value.trim().toLowerCase();
  if (!keyword) return true;
  return (
    (page.name || "").toLowerCase().includes(keyword) ||
    (page.path || "").toLowerCase().includes(keyword)
  );
}

/**
 * 创建活跃页面ID映射（computed 确保响应式更新）
 */
const activePageMap = computed(() => {
  const map = new Map<string, boolean>();
  if (currentPageId.value) {
    map.set(currentPageId.value, true);
  }
  return map;
});

/**
 * 判断页面是否选中
 * @param {string} pageId - 页面ID
 * @returns {boolean}
 */
function isPageActive(pageId: string): boolean {
  return activePageMap.value.has(pageId);
}

/**
 * 同步页面选中状态（使用 watchEffect 确保响应式更新）
 */
watchEffect(() => {
  const pageId = currentPageId.value;
  const pageList = (pages.value || []) as any[];

  if (!pageId || !pageList.length) return;

  const page = pageList.find((p: any) => p.id === pageId) as any;
  if (page) {
    selectedNode.value = {
      id: page.id,
      type: "page",
      label: getPageLabel(page),
    };

    // 如果页面在分组中，自动展开该分组
    if (page.parentId) {
      folderStates.value[page.parentId] = true;
    }
  }
});

/**
 * 所有可见页面（包含登录页、登出页等系统页面）
 */
const appPages = computed<PageNodeLike[]>(() =>
  ((pages.value || []) as any[] as PageNodeLike[]).filter((page: any) => page.type !== "dialog"),
);
const systemPages = computed<Record<string, any>>(() => ({
  home:
    appPages.value.find((page: any) => page.id === (entryConfig.value as any)?.homePageId) || null,
  login:
    appPages.value.find((page: any) => page.id === (entryConfig.value as any)?.loginPageId) ||
    appPages.value.find((page: any) => getFixedSystemPath(page) === "/login") ||
    null,
  logout:
    appPages.value.find((page: any) => page.id === (entryConfig.value as any)?.logoutPageId) ||
    appPages.value.find((page: any) => getFixedSystemPath(page) === "/logout") ||
    null,
}));
const basicPageIdSet = computed(
  () =>
    new Set(
      [
        systemPages.value.home?.id,
        systemPages.value.login?.id,
        systemPages.value.logout?.id,
      ].filter(Boolean),
    ),
);

const basicPageMap = computed(() => {
  const pageMap = new Map<string, PageNodeLike>();
  appPages.value.forEach((page) => {
    pageMap.set(page.id, page);
  });
  return pageMap;
});

const basicPageSlots = computed<any[]>(() => {
  const slots: any[] = [
    {
      type: "home",
      label: "首页",
      pageId: systemPages.value.home?.id || "",
    },
    {
      type: "login",
      label: "登录页",
      pageId: systemPages.value.login?.id || "",
    },
    {
      type: "logout",
      label: "登出页",
      pageId: systemPages.value.logout?.id || "",
    },
  ];
  return slots.map((slot: any) => ({
    ...slot,
    path: getBasicPageMeta(slot.type).path,
    page: slot.pageId ? basicPageMap.value.get(slot.pageId) || null : null,
  }));
});

const filteredBasicSlots = computed(() => {
  const keyword = searchText.value.trim().toLowerCase();
  if (!keyword) {
    return basicPageSlots.value;
  }
  return basicPageSlots.value.filter((slot) => {
    const label = slot.label.toLowerCase();
    const pageName = slot.page ? getPageLabel(slot.page).toLowerCase() : "";
    return label.includes(keyword) || pageName.includes(keyword);
  });
});

const businessPages = computed(() =>
  appPages.value.filter((page) => !basicPageIdSet.value.has(page.id)),
);

const businessFolders = computed(() =>
  businessPages.value.filter((page) => page.type === "folder"),
);

const businessRootItems = computed(() =>
  sortItemsByStoredOrder(
    businessPages.value.filter(
      (page) => page.type === "folder" || (page.type === "page" && !page.parentId),
    ),
    ROOT_CONTAINER_KEY,
  ).sort((a, b) => {
    if (a.type === b.type) return 0;
    return a.type === "folder" ? -1 : 1;
  }),
);

function getChildrenCount(folderId: string): number {
  return businessPages.value.filter((page) => page.type === "page" && page.parentId === folderId)
    .length;
}

function getBusinessChildren(folderId: string): PageNodeLike[] {
  return sortItemsByStoredOrder(
    businessPages.value.filter(
      (page) => page.type === "page" && page.parentId === folderId && matchesKeyword(page),
    ),
    getContainerOrderKey(folderId),
  );
}

const filteredBusinessRootItems = computed(() =>
  businessRootItems.value.filter((item) => {
    if (item.type === "folder") {
      return matchesKeyword(item) || getBusinessChildren(item.id).length > 0;
    }
    return matchesKeyword(item);
  }),
);

function appendIdToContainer(
  containerKey: string,
  sourceId: string,
  visibleItems: Array<PageNodeLike>,
): void {
  const nextIds = (visibleItems || []).map((item) => item.id).filter((id) => id !== sourceId);
  nextIds.push(sourceId);
  updateContainerOrder(containerKey, nextIds);
}

/**
 * 仅在目标发生变化时更新拖拽高亮，避免 dragover 频繁触发整树重渲染
 * @param {{ id: string, mode: string, parentId: string | null }} nextTarget - 新目标
 */
function setDragOverTarget(nextTarget: DragTargetLike): void {
  pendingDragOverTarget = nextTarget;
  if (dragOverFrameId) return;
  dragOverFrameId = requestAnimationFrame(() => {
    dragOverFrameId = 0;
    const current = dragOverTarget.value;
    const target = pendingDragOverTarget;
    if (!target) return;
    if (
      current?.id === target.id &&
      current?.mode === target.mode &&
      (current?.parentId || null) === (target.parentId || null)
    ) {
      return;
    }
    dragOverTarget.value = target;
  });
}

function handleDragStart(item: PageNodeLike, parentId: string | null): void {
  dragState.value = {
    id: item.id,
    type: item.type,
    parentId: parentId || null,
  };
}

function handleDragEnd(): void {
  if (dragOverFrameId) {
    cancelAnimationFrame(dragOverFrameId);
    dragOverFrameId = 0;
  }
  pendingDragOverTarget = null;
  dragState.value = null;
  dragOverTarget.value = null;
}

function handleDragLeave(targetId: string): void {
  if (dragOverTarget.value?.id === targetId) {
    dragOverTarget.value = null;
  }
  if (pendingDragOverTarget?.id === targetId) {
    pendingDragOverTarget = null;
  }
}

function handleNodeDragOver(event: DragEvent, item: PageNodeLike, parentId: string | null): void {
  if (!dragState.value || dragState.value.id === item.id) return;
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = "move";
  }
  setDragOverTarget({
    id: item.id,
    mode: "before",
    parentId: parentId || null,
  });
}

function handleFolderDragOver(event: DragEvent, folder: PageNodeLike): void {
  if (!dragState.value || dragState.value.id === folder.id) return;
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = "move";
  }
  setDragOverTarget({
    id: folder.id,
    mode: "append",
    parentId: folder.id,
  });
}

function handleContainerDragOver(event: DragEvent, parentId: string | null): void {
  if (!dragState.value) return;
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = "move";
  }
  setDragOverTarget({
    id: getContainerOrderKey(parentId),
    mode: "append",
    parentId: parentId || null,
  });
}

async function handleNodeDrop(targetItem: PageNodeLike, parentId: string | null): Promise<void> {
  if (!dragState.value || dragState.value.id === targetItem.id) return;
  const source = { ...dragState.value };
  const targetParentId = parentId || null;
  const sourceParentId = source.parentId || null;

  if (source.type === "page" && sourceParentId !== targetParentId) {
    await editorStore.movePageToGroup(source.id, targetParentId);
  }

  const visibleItems =
    targetParentId === null
      ? businessRootItems.value
      : businessPages.value.filter(
          (page) => page.type === "page" && page.parentId === targetParentId,
        );
  const nextOrder = moveIdBefore(
    visibleItems.map((item) => item.id),
    source.id,
    targetItem.id,
  );
  updateContainerOrder(getContainerOrderKey(targetParentId), nextOrder);

  if (sourceParentId !== targetParentId) {
    const sourceItems =
      sourceParentId === null
        ? businessRootItems.value.filter((item) => item.id !== source.id)
        : businessPages.value.filter(
            (page) =>
              page.type === "page" && page.parentId === sourceParentId && page.id !== source.id,
          );
    updateContainerOrder(
      getContainerOrderKey(sourceParentId),
      sourceItems.map((item) => item.id),
    );
  }
  handleDragEnd();
}

async function handleFolderDrop(folder: PageNodeLike): Promise<void> {
  if (!dragState.value || dragState.value.id === folder.id) return;
  const source = { ...dragState.value };
  if (source.type !== "page") {
    handleDragEnd();
    return;
  }
  if (source.parentId !== folder.id) {
    await editorStore.movePageToGroup(source.id, folder.id);
  }
  appendIdToContainer(
    getContainerOrderKey(folder.id),
    source.id,
    businessPages.value.filter((page) => page.type === "page" && page.parentId === folder.id),
  );
  if (!folderStates.value[folder.id]) {
    folderStates.value[folder.id] = true;
  }
  if ((source.parentId || null) !== folder.id) {
    const sourceItems =
      source.parentId === null
        ? businessRootItems.value.filter((item) => item.id !== source.id)
        : businessPages.value.filter(
            (page) =>
              page.type === "page" && page.parentId === source.parentId && page.id !== source.id,
          );
    updateContainerOrder(
      getContainerOrderKey(source.parentId),
      sourceItems.map((item) => item.id),
    );
  }
  handleDragEnd();
}

async function handleContainerDrop(parentId: string | null): Promise<void> {
  if (!dragState.value) return;
  const source = { ...dragState.value };
  const targetParentId = parentId || null;

  if (source.type === "page" && (source.parentId || null) !== targetParentId) {
    await editorStore.movePageToGroup(source.id, targetParentId);
  }

  const visibleItems =
    targetParentId === null
      ? businessRootItems.value.filter((item) => item.id !== source.id)
      : businessPages.value.filter(
          (page) =>
            page.type === "page" && page.parentId === targetParentId && page.id !== source.id,
        );
  appendIdToContainer(getContainerOrderKey(targetParentId), source.id, visibleItems);

  if ((source.parentId || null) !== targetParentId) {
    const sourceItems =
      source.parentId === null
        ? businessRootItems.value.filter((item) => item.id !== source.id)
        : businessPages.value.filter(
            (page) =>
              page.type === "page" && page.parentId === source.parentId && page.id !== source.id,
          );
    updateContainerOrder(
      getContainerOrderKey(source.parentId),
      sourceItems.map((item) => item.id),
    );
  }
  handleDragEnd();
}

watch(
  projectId,
  () => {
    loadPageOrderMap();
  },
  { immediate: true },
);

watch(
  [businessRootItems, businessFolders],
  () => {
    ensureContainerOrder(ROOT_CONTAINER_KEY, businessRootItems.value);
    businessFolders.value.forEach((folder) => {
      ensureContainerOrder(
        getContainerOrderKey(folder.id),
        businessPages.value.filter((page) => page.type === "page" && page.parentId === folder.id),
      );
    });
  },
  { immediate: true, deep: true },
);

/**
 * 初始化切换提示配置
 * @returns {void}
 */
function initSwitchPromptState() {
  try {
    skipSwitchPrompt.value = localStorage.getItem(pageSwitchPromptKey) === "1";
  } catch {
    skipSwitchPrompt.value = false;
  }
}

/**
 * 设置切换提示是否禁用
 * @param {boolean} disabled - 是否禁用
 * @returns {void}
 */
function setSwitchPromptDisabled(disabled: any) {
  skipSwitchPrompt.value = disabled;
  try {
    if (disabled) {
      localStorage.setItem(pageSwitchPromptKey, "1");
    } else {
      localStorage.removeItem(pageSwitchPromptKey);
    }
  } catch {
    // 存储异常时保持内存状态
  }
}

initSwitchPromptState();

/**
 * 选中页面（只选中，不切换页面）
 * @param {{ id: string }} node - 点击的节点
 */
function handleNodeClick(node: SelectedNodeLike): void {
  selectedNode.value = node;
  // 点击只选中节点，不切换页面
  // 页面切换通过双击 handleNodeDoubleClick 实现
}

/**
 * 处理基础页面槽位点击
 * @param {{ page?: import('@/editor-core').PageNode | null }} slot - 槽位
 * @returns {void}
 */
function handleBasicSlotClick(slot: BasicSlotLike): void {
  if (!slot?.page) {
    selectedNode.value = null;
    return;
  }
  handleNodeClick({ id: slot.page.id, type: "page" });
}

/**
 * 处理基础页面槽位双击
 * @param {{ page?: import('@/editor-core').PageNode | null }} slot - 槽位
 * @returns {void}
 */
function handleBasicSlotDoubleClick(slot: BasicSlotLike): void {
  if (!slot?.page) return;
  void handleNodeDoubleClick({
    id: slot.page.id,
    type: "page",
    label: slot.label,
  });
}

/**
 * 处理页面双击切换
 * @param {{ id: string, type: string, label?: string }} node - 双击的节点
 * @returns {Promise<void>}
 */
async function handleNodeDoubleClick(node: SelectedNodeLike): Promise<void> {
  if (!node || node.type !== "page") return;

  // 使用标签页系统打开页面
  if (openPageTab) {
    openPageTab(node.id);
    return;
  }

  // 兼容旧逻辑（如果没有注入 openPageTab）
  if (node.id === currentPageId.value) return;

  const targetPage = pages.value.find((page) => page.id === node.id);
  const targetName = targetPage?.name || node.label || node.id;

  const canSwitch = await confirmPageSwitch(targetName);
  if (!canSwitch) return;

  const safeToSwitch = await ensureUnsavedSwitch(targetName);
  if (!safeToSwitch) return;

  try {
    const result = await editorStore.loadPage(node.id);
    if (!result.ok) {
      showError(result.error?.message || "切换页面失败");
    }
  } catch {
    showError("切换页面失败");
  }
}

/**
 * 确认是否切换页面
 * @param {string} targetName - 目标页面名称
 * @returns {Promise<boolean>}
 */
async function confirmPageSwitch(targetName: string): Promise<boolean> {
  if (skipSwitchPrompt.value) return true;

  const skipPrompt = ref(false);
  const message = h("div", { class: "flex flex-col gap-2" }, [
    h("div", `是否切换到${targetName}页面？`),
    h(
      ElCheckbox as any,
      {
        modelValue: skipPrompt.value,
        "onUpdate:modelValue": (value: boolean) => {
          skipPrompt.value = value;
        },
      },
      () => "不再提示",
    ),
  ]);

  try {
    await (ElMessageBox as any)({
      title: "切换页面",
      message,
      showCancelButton: true,
      confirmButtonText: "切换",
      cancelButtonText: "取消",
      distinguishCancelAndClose: true,
      closeOnClickModal: false,
    });
    if (skipPrompt.value) {
      setSwitchPromptDisabled(true);
    }
    return true;
  } catch {
    return false;
  }
}

/**
 * 处理未保存切换提示
 * @param {string} targetName - 目标页面名称
 * @returns {Promise<boolean>}
 */
async function ensureUnsavedSwitch(targetName: string): Promise<boolean> {
  if (!canUndo.value) return true;
  try {
    const action = await ElMessageBox.confirm(
      `当前页面未保存，是否切换到${targetName}页面？`,
      "切换页面",
      {
        confirmButtonText: "保存并切换",
        cancelButtonText: "不保存切换",
        distinguishCancelAndClose: true,
        closeOnClickModal: false,
      },
    );
    if (action === "confirm") {
      await editorStore.saveCurrentPage();
    }
    return true;
  } catch (error) {
    if (error === "cancel") {
      return true;
    }
    if (error && error !== "close") {
      showError("切换页面失败");
    }
    return false;
  }
}

const buildMoveCommand = (targetId: string | null): string => `move:${targetId || "root"}`;

/**
 * 创建基础页面
 * @param {"home" | "login" | "logout"} basicType - 基础页面类型
 * @returns {Promise<void>}
 */
async function createBasicPage(basicType: FixedSystemType): Promise<void> {
  if (basicType === "home") {
    if (entryConfig.value?.homePageId) {
      showWarning("首页已存在");
      return;
    }
    if (!projectId.value) {
      showError("缺少工程信息");
      return;
    }
    const result = (await editorStore.createHomePage(projectId.value)) as any;
    if (!result?.ok || !result.pageId) {
      showError(result?.error?.message || "首页创建失败");
      return;
    }
    await openCreatedPageTab(result.pageId);
    showSuccess("首页创建成功");
    return;
  }

  if (basicType === "login") {
    if (entryConfig.value?.loginPageId || pages.value.some((page) => page.path === "/login")) {
      showWarning("登录页已存在");
      return;
    }
    try {
      const meta = getBasicPageMeta("login");
      const schemaContent = editorStore.buildNewPageSchema({
        name: meta.label,
        path: meta.path,
      });
      const result = await editorStore.createPage({
        name: meta.label,
        type: "page",
        parentId: null,
        schemaContent,
      });
      const pageId = result?.id;
      if (!pageId) {
        throw new Error("登录页创建失败");
      }
      editorStore.updateEntry({ loginPageId: pageId });
      await editorStore.persistEntry();
      await openCreatedPageTab(pageId);
      showSuccess("登录页创建成功");
    } catch (error: any) {
      showError(error?.message || "登录页创建失败");
    }
    return;
  }

  if (basicType === "logout") {
    if (entryConfig.value?.logoutPageId || pages.value.some((page) => page.path === "/logout")) {
      showWarning("登出页已存在");
      return;
    }
    try {
      const meta = getBasicPageMeta("logout");
      const schemaContent = editorStore.buildNewPageSchema({
        name: meta.label,
        path: meta.path,
      });
      const result = await editorStore.createPage({
        name: meta.label,
        type: "page",
        parentId: null,
        schemaContent,
      });
      const pageId = result?.id;
      if (!pageId) {
        throw new Error("登出页创建失败");
      }
      editorStore.updateEntry({ logoutPageId: pageId });
      await editorStore.persistEntry();
      await openCreatedPageTab(pageId);
      showSuccess("登出页创建成功");
    } catch (error: any) {
      showError(error?.message || "登出页创建失败");
    }
  }
}

/**
 * 基础页面行尾操作
 * @param {string} command - 命令
 * @param {{ type: "home" | "login" | "logout", page?: import('@/editor-core').PageNode | null }} slot - 槽位
 * @returns {void}
 */
function handleBasicRowAction(command: string, slot: BasicSlotLike): void {
  if (!slot) return;
  if (command === "create") {
    void createBasicPage(slot.type);
    return;
  }
  if (!slot.page) return;
  selectedNode.value = {
    id: slot.page.id,
    type: "page",
    label: slot.label,
    parentId: null,
  };
  handleRowAction(command, slot.page);
}

/**
 * 行尾操作按钮
 * @param {string} command - 命令
 * @param {import('@/editor-core').PageNode} node - 节点
 * @returns {void}
 */
function handleRowAction(command: string, node: PageNodeLike): void {
  if (!node) return;
  selectedNode.value = {
    id: node.id,
    type: node.type,
    label: getPageLabel(node),
    parentId: node.parentId || null,
  };

  if (command === "open" && node.type === "page") {
    void handleNodeDoubleClick({
      id: node.id,
      type: "page",
      label: getPageLabel(node),
    });
    return;
  }
  if (command === "rename") {
    if (getFixedSystemType(node)) {
      showWarning("基础页面名称固定，不支持重命名");
      return;
    }
    void handleRename();
    return;
  }
  if (command === "export" && node.type === "page") {
    handleExportPage(node.id, getPageLabel(node));
    return;
  }
  if (command === "createPage" && node.type === "folder") {
    handleCreateInFolder(node.id);
    return;
  }
  if (command === "moveToRoot" && node.type === "page") {
    void handleMove(node.id, null);
    return;
  }
  if (command.startsWith("move:") && node.type === "page") {
    const targetId = command.slice(5);
    void handleMove(node.id, targetId === "root" ? null : targetId);
    return;
  }
  if (command === "delete") {
    void handleDelete();
  }
}

/**
 * 导出页面 Schema
 * @param {string} pageId - 页面 ID
 * @param {string} label - 页面名称
 */
function handleExportPage(pageId: string, label: string): void {
  if (!editorStore.doc || !pageId) {
    showWarning("暂无可导出的页面");
    return;
  }
  try {
    const payload = editorStore.serializer.exportPage(editorStore.doc, pageId);
    const json = JSON.stringify(payload, null, 2);
    const blob = new Blob([json], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const name = `${label || "page"}.json`;
    const link = document.createElement("a");
    link.href = url;
    link.download = name;
    link.click();
    URL.revokeObjectURL(url);
    showSuccess("已导出页面");
  } catch {
    showError("导出页面失败");
  }
}

/**
 * 打开新建弹窗
 */
function openCreateDialog(): void {
  createForm.value = {
    type: "page",
    name: "",
    parentId: selectedNode.value?.type === "folder" ? selectedNode.value.id : null,
  };
  createDialogVisible.value = true;
}

defineExpose({ openCreateDialog });

/**
 * 创建完成后打开对应标签并切换到页面
 * @param {string} pageId - 页面 ID
 */
async function openCreatedPageTab(pageId: string): Promise<void> {
  if (!pageId) return;
  selectedNode.value = { id: pageId, type: "page", label: "", parentId: null };
  if (openPageTab) {
    openPageTab(pageId);
    return;
  }
  await editorStore.setCurrentPage(pageId);
}

/**
 * 移动页面到分组
 * @param {string} pageId - 页面 ID
 * @param {string | null} parentId - 分组 ID
 */
async function handleMove(pageId: string, parentId: string | null): Promise<void> {
  try {
    const page = ((pages.value || []) as any[]).find((item: any) => item.id === pageId) as any;
    const currentParentId = page?.parentId || null;
    if (currentParentId === (parentId || null)) {
      return;
    }
    if (getFixedSystemType(page) && parentId) {
      showWarning("基础页面不支持移动到分组");
      return;
    }
    const nextPath =
      page?.type === "page"
        ? getFixedSystemPath(page) || buildBusinessPagePath(getPageLabel(page), parentId)
        : undefined;
    await editorStore.movePageToGroup(pageId, parentId, nextPath);
    const nextPage = ((pages.value || []) as any[]).find((item: any) => item.id === pageId) as any;
    if (nextPage?.type === "page") {
      await editorStore.renamePage(pageId, getPageLabel(nextPage), nextPath);
    }
    showSuccess("移动成功");
  } catch {
    showError("移动失败");
  }
}

/**
 * 校验名称唯一
 * @param {string} name - 名称
 * @param {string} [excludeId] - 排除的页面 ID
 * @returns {boolean}
 */
function isNameUnique(name: any, excludeId: any) {
  const lowerName = name.trim().toLowerCase();
  return !((pages.value || []) as any[]).some(
    (page: any) => page.id !== excludeId && (page.name || "").trim().toLowerCase() === lowerName,
  );
}

/**
 * 规范化路由片段
 * @param {string} value - 名称
 * @returns {string}
 */
function toPathSegment(value: any) {
  const normalized = value.trim().replace(PAGE_TREE_SEGMENT_SPACE_RE, "-");
  const sanitized = normalized.replace(PAGE_TREE_SEGMENT_RE, "-");
  return sanitized || "page";
}

/**
 * 获取直属分组路由片段
 * @param {string | null | undefined} parentId - 分组 ID
 * @returns {string[]}
 */
function getFolderPathSegments(parentId: any) {
  const folder = ((pages.value || []) as any[]).find(
    (page: any) => page.id === (parentId || null) && page.type === "folder",
  );
  if (!folder) {
    return [];
  }
  return [toPathSegment(folder.name || folder.title || folder.id)];
}

/**
 * 生成业务页面路径
 * @param {string} name - 页面名称
 * @param {string | null | undefined} parentId - 分组 ID
 * @returns {string}
 */
function buildBusinessPagePath(name: any, parentId: any) {
  const segments = [...getFolderPathSegments(parentId), toPathSegment(name)];
  return `/${segments.filter(Boolean).join("/")}`;
}

/**
 * 同步分组直接子页面路径
 * @param {string} folderId - 分组 ID
 * @returns {Promise<void>}
 */
async function syncFolderDescendantPaths(folderId: any) {
  const descendants = ((pages.value || []) as any[]).filter(
    (page: any) => page.type === "page" && page.parentId === folderId,
  );
  for (const page of descendants as any[]) {
    const fixedPath = getFixedSystemPath(page);
    if (fixedPath) continue;
    const nextPath = buildBusinessPagePath(getPageLabel(page), page.parentId || null);
    if (page.path !== nextPath) {
      await editorStore.renamePage(page.id, getPageLabel(page), nextPath);
    }
  }
}

/**
 * 获取登录/登出固定路径
 * @param {import('@/editor-core').PageNode | undefined} page - 页面
 * @returns {string | null}
 */
function getFixedSystemPath(page: any) {
  const systemType = getFixedSystemType(page);
  return systemType ? getBasicPageMeta(systemType).path : null;
}

/**
 * 创建页面/分组
 */
function onCreateFormUpdate(next: any) {
  createForm.value = next;
}

async function handleCreateConfirm() {
  const name = createForm.value.name?.trim();
  if (!name) {
    showWarning("名称不能为空");
    return;
  }
  const nameValidation = validatePageName(name);
  if (!nameValidation.valid) {
    showWarning(nameValidation.message);
    return;
  }
  if (!isNameUnique(name, undefined)) {
    showWarning("页面或分组名称已存在");
    return;
  }

  const type = createForm.value.type;
  const parentId = type === "page" ? createForm.value.parentId || null : null;

  creating.value = true;
  try {
    if (type === "folder") {
      await editorStore.createPage({ name, type: "folder", parentId });
      showSuccess("分组创建成功");
      createDialogVisible.value = false;
      creating.value = false;
      return;
    }

    // 普通页面：一次性创建带 schema 的页面
    const schemaContent = editorStore.buildNewPageSchema({
      name,
      path: buildBusinessPagePath(name, parentId),
    });
    const result = await editorStore.createPage({
      name,
      type: "page",
      parentId,
      schemaContent,
    });
    const pageId = result?.id;
    if (!pageId) throw new Error("页面创建失败");
    await openCreatedPageTab(pageId);
    showSuccess("页面创建成功");
    createDialogVisible.value = false;
  } catch (error: any) {
    console.error("创建失败:", error);
    showError(error?.message || "创建失败");
  } finally {
    creating.value = false;
  }
}

/**
 * 重命名页面/分组
 */
async function handleRename() {
  const target = selectedNode.value;
  if (!target) {
    showWarning("请先选择要重命名的页面或分组");
    return;
  }
  try {
    const result = await ElMessageBox.prompt("请输入新名称", "重命名", {
      inputValue: target.label || "",
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      closeOnClickModal: false,
    });
    const name = result?.value?.trim();
    if (!name) {
      showWarning("名称不能为空");
      return;
    }
    const nameValidation = validatePageName(name);
    if (!nameValidation.valid) {
      showWarning(nameValidation.message);
      return;
    }
    if (!isNameUnique(name, target.id)) {
      showWarning("页面或分组名称已存在");
      return;
    }
    const page = ((pages.value || []) as any[]).find((item: any) => item.id === target.id);
    const fixedPath = getFixedSystemPath(page);
    const path =
      page?.type === "page"
        ? fixedPath || buildBusinessPagePath(name, page?.parentId || null)
        : undefined;
    await editorStore.renamePage(target.id, name, path);
    if (page?.type === "folder") {
      await syncFolderDescendantPaths(target.id);
    }
    showSuccess("重命名成功");
  } catch (error: any) {
    if (error !== "cancel") {
      showError("重命名失败");
    }
  }
}

/**
 * 判断是否为首页
 * @param {string} pageId - 页面 ID
 * @returns {boolean}
 */
function isHomePage(pageId: any) {
  return (entryConfig.value as any)?.homePageId === pageId;
}

/**
 * 获取分组下的全部后代节点数量
 * @param {string} folderId - 分组 ID
 * @returns {number}
 */
function getFolderDescendantCount(folderId: any) {
  let count = 0;
  const stack = [folderId];
  while (stack.length) {
    const currentFolderId = stack.pop();
    const children = ((pages.value || []) as any[]).filter(
      (page: any) => page.parentId === currentFolderId,
    );
    count += children.length;
    children.filter((page) => page.type === "folder").forEach((folder) => stack.push(folder.id));
  }
  return count;
}

/**
 * 删除页面/分组
 */
async function handleDelete() {
  const target = selectedNode.value;
  if (!target) {
    showWarning("请先选择要删除的页面或分组");
    return;
  }

  // 首页保护：不允许删除首页
  if (target.type === "page" && isHomePage(target.id)) {
    showWarning("首页不可删除，请先设置其他页面为首页");
    return;
  }

  try {
    if (target.type === "folder") {
      const childCount = getFolderDescendantCount(target.id);
      try {
        await ElMessageBox.confirm(
          childCount > 0
            ? `分组 "${target.label}" 下还有 ${childCount} 个子项。建议优先选择“仅删除分组”，组下页面会自动移到根目录；“删除组和页面”会一并删除全部子项，且不可恢复。`
            : `确定删除分组 "${target.label}" 吗？`,
          "删除分组",
          {
            confirmButtonText: childCount > 0 ? "仅删除分组" : "删除分组",
            cancelButtonText: childCount > 0 ? "删除组和页面" : "取消",
            type: "warning",
            closeOnClickModal: false,
            distinguishCancelAndClose: true,
          },
        );
        await editorStore.deletePage(target.id, childCount > 0 ? "folder-only" : "single");
      } catch (error) {
        if (error === "cancel" && childCount > 0) {
          await editorStore.deletePage(target.id, "cascade");
        } else if (error === "close") {
          return;
        } else if (error !== "cancel") {
          throw error;
        } else {
          return;
        }
      }

      selectedNode.value = null;
      showSuccess("删除成功");
      return;
    }

    await ElMessageBox.confirm(`确定删除 "${target.label}" 吗？此操作不可恢复。`, "删除确认", {
      confirmButtonText: "删除",
      cancelButtonText: "取消",
      type: "warning",
      confirmButtonClass: "el-button--danger",
    });

    // 如果删除的是当前页面，先切换到首页
    if (target.type === "page" && target.id === currentPageId.value) {
      const homePageId = (entryConfig.value as any)?.homePageId;
      if (homePageId && homePageId !== target.id) {
        if (openPageTab) {
          openPageTab(homePageId);
        } else {
          await editorStore.setCurrentPage(homePageId);
        }
      }
    }

    await editorStore.deletePage(target.id);
    selectedNode.value = null;
    showSuccess("删除成功");
  } catch (error: any) {
    if (error !== "cancel") {
      showError("删除失败");
    }
  }
}

/**
 * 分组选项
 */
const folderOptions = computed<any[]>(() =>
  businessFolders.value.map((page: Record<string, any>) => ({
    id: page.id,
    name: page.name || "",
  })),
);

/**
 * 可移动目标列表
 * @param {{ parentId?: string | null, id?: string }} node - 节点
 * @returns {Array<{ id: string | null, label: string }>}
 */
function getMoveTargets(node: any): MoveTargetLike[] {
  if (getFixedSystemType(node)) {
    return [];
  }

  const targets: MoveTargetLike[] = [];
  const groups = folderOptions.value.filter((group: any) => group.id !== node.parentId);
  for (const group of groups) {
    targets.push({ id: group.id, label: group.name });
  }
  return targets;
}
</script>

<template>
  <div class="page-tree-container">
    <PageTreePageSearch v-model="searchText" />

    <!-- 页面树 -->
    <div class="page-tree-content">
      <PageTreeBasicPagesSection
        :basic-slots="filteredBasicSlots"
        :is-page-active="isPageActive"
        @slot-click="handleBasicSlotClick"
        @slot-dblclick="handleBasicSlotDoubleClick"
        @row-action="handleBasicRowAction"
      />

      <section
        class="page-section"
        @dragover.prevent="handleContainerDragOver($event, null)"
        @drop.prevent="handleContainerDrop(null)"
      >
        <div class="page-section__header">
          <span class="page-section__title">普通页面</span>
        </div>
        <div v-if="filteredBusinessRootItems.length" class="page-section__body">
          <template v-for="item in filteredBusinessRootItems" :key="item.id">
            <div
              v-if="item.type === 'folder'"
              class="tree-node folder-node"
              :class="{
                'is-active': selectedNode?.id === item.id,
                'is-drop-target':
                  dragOverTarget?.id === item.id && dragOverTarget?.mode === 'append',
              }"
              draggable="true"
              @dragstart="handleDragStart(item, null)"
              @dragend="handleDragEnd"
              @dragover.prevent="handleFolderDragOver($event, item)"
              @dragleave="handleDragLeave(item.id)"
              @drop.prevent="handleFolderDrop(item)"
              @click="handleFolderClick(item)"
            >
              <div class="node-indent node-indent--folder" />
              <component
                :is="folderStates[item.id] ? IconEpArrowDown : IconEpArrowRight"
                class="node-arrow"
                @click.stop="toggleFolder(item.id)"
              />
              <IconEpFolder class="node-icon folder" />
              <span class="node-label">{{ item.name }}</span>
              <span class="node-count">{{ getChildrenCount(item.id) }}</span>
              <el-dropdown
                trigger="click"
                placement="bottom-end"
                @command="(command: any) => handleRowAction(command, item)"
              >
                <el-button class="node-action-btn" text @click.stop>
                  <IconEpMoreFilled />
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="rename">重命名</el-dropdown-item>
                    <el-dropdown-item command="createPage">新建页面</el-dropdown-item>
                    <el-dropdown-item command="delete" divided>删除分组</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            <div
              v-else
              class="tree-node page-node"
              :class="{
                'is-active': isPageActive(item.id),
                'is-drop-target':
                  dragOverTarget?.id === item.id && dragOverTarget?.mode === 'before',
              }"
              draggable="true"
              @dragstart="handleDragStart(item, null)"
              @dragend="handleDragEnd"
              @dragover.prevent="handleNodeDragOver($event, item, null)"
              @dragleave="handleDragLeave(item.id)"
              @drop.prevent="handleNodeDrop(item, null)"
              @click="handleNodeClick({ id: item.id, type: 'page' })"
              @dblclick="
                handleNodeDoubleClick({
                  id: item.id,
                  type: 'page',
                  label: getPageLabel(item),
                })
              "
            >
              <div class="node-indent" />
              <IconEpDocument class="node-icon page" />
              <span class="node-label">{{ getPageLabel(item) }}</span>
              <el-dropdown
                trigger="click"
                placement="bottom-end"
                @command="(command: any) => handleRowAction(command, item)"
              >
                <el-button class="node-action-btn" text @click.stop>
                  <IconEpMoreFilled />
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="open">打开</el-dropdown-item>
                    <el-dropdown-item command="rename">重命名</el-dropdown-item>
                    <el-dropdown-item command="export">导出页面</el-dropdown-item>
                    <el-dropdown-item v-if="item.parentId" command="moveToRoot" divided>
                      移到根目录
                    </el-dropdown-item>
                    <el-dropdown-item
                      v-for="target in getMoveTargets(item)"
                      :key="`root-${item.id}-${target.id ?? 'root'}`"
                      :command="buildMoveCommand(target.id)"
                    >
                      {{ target.label }}
                    </el-dropdown-item>
                    <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>

            <div v-if="item.type === 'folder' && folderStates[item.id]" class="tree-children">
              <div
                v-for="page in getBusinessChildren(item.id)"
                :key="page.id"
                class="tree-node page-node level-1"
                :class="{
                  'is-active': isPageActive(page.id),
                  'is-drop-target':
                    dragOverTarget?.id === page.id && dragOverTarget?.mode === 'before',
                }"
                draggable="true"
                @dragstart="handleDragStart(page, item.id)"
                @dragend="handleDragEnd"
                @dragover.prevent="handleNodeDragOver($event, page, item.id)"
                @dragleave="handleDragLeave(page.id)"
                @drop.prevent="handleNodeDrop(page, item.id)"
                @click="handleNodeClick({ id: page.id, type: 'page' })"
                @dblclick="
                  handleNodeDoubleClick({
                    id: page.id,
                    type: 'page',
                    label: getPageLabel(page),
                  })
                "
              >
                <div class="node-indent">
                  <span class="tree-line horizontal" />
                </div>
                <IconEpDocument class="node-icon page" />
                <span class="node-label">{{ getPageLabel(page) }}</span>
                <el-dropdown
                  trigger="click"
                  placement="bottom-end"
                  @command="(command: any) => handleRowAction(command, page)"
                >
                  <el-button class="node-action-btn" text @click.stop>
                    <IconEpMoreFilled />
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="open">打开</el-dropdown-item>
                      <el-dropdown-item command="rename">重命名</el-dropdown-item>
                      <el-dropdown-item command="export">导出页面</el-dropdown-item>
                      <el-dropdown-item v-if="page.parentId" command="moveToRoot" divided>
                        移到根目录
                      </el-dropdown-item>
                      <el-dropdown-item
                        v-for="target in getMoveTargets(page)"
                        :key="`${page.id}-${target.id ?? 'root'}`"
                        :command="buildMoveCommand(target.id)"
                      >
                        {{ target.label }}
                      </el-dropdown-item>
                      <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
              <div
                v-if="!getBusinessChildren(item.id).length && !searchText.trim()"
                class="folder-dropzone"
                @dragover.prevent="handleFolderDragOver($event, item)"
                @dragleave="handleDragLeave(item.id)"
                @drop.prevent="handleFolderDrop(item)"
              >
                拖拽页面到此分组
              </div>
            </div>
          </template>
        </div>
        <div v-else class="page-section__empty">暂无普通页面</div>
      </section>

      <div
        v-if="filteredBasicSlots.length === 0 && filteredBusinessRootItems.length === 0"
        class="empty-state"
      >
        <IconEpDocument class="empty-icon" />
        <p class="empty-text">暂无页面</p>
        <p class="empty-hint">点击上方按钮新建页面</p>
      </div>
    </div>
  </div>
  <PageTreeCreateDialog
    v-model="createDialogVisible"
    :title="createDialogTitle"
    :create-type-options="createTypeOptions as any"
    :form="createForm as any"
    :rules="createFormRules"
    :folder-options="folderOptions as any"
    :creating="creating"
    :is-fixed-basic-type="isFixedBasicCreateType"
    @update:form="onCreateFormUpdate"
    @confirm="handleCreateConfirm"
  />
</template>

<style scoped>
/* ==================== 容器 ==================== */
.page-tree-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* 树内容区 */
.page-tree-content {
  flex: 1;
  overflow-y: auto;
  padding: 2px 6px 8px;
}

.page-section {
  margin-bottom: 10px;
}

.page-section__header {
  display: flex;
  align-items: center;
  min-height: 22px;
  padding: 3px 8px 5px;
}

.page-section__title {
  font-size: 12px;
  font-weight: 600;
  color: var(--designer-text-primary, #303133);
}

.page-section__body {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.page-section__empty {
  padding: 4px 10px 2px;
  font-size: 11px;
  color: var(--designer-text-muted, #8c8c8c);
}

/* ==================== 树节点通用样式 ==================== */
.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  padding: 5px 8px;
  margin: 1px 0;
  border-radius: 4px;
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
  position: relative;
}

.tree-node:hover {
  background-color: rgba(15, 23, 42, 0.04);
}

.tree-node.is-active {
  background-color: rgba(15, 23, 42, 0.08);
}

.tree-node.is-active::before {
  content: none;
}

.tree-node.is-drop-target {
  background-color: rgba(59, 130, 246, 0.12);
  box-shadow: inset 0 0 0 1px rgba(59, 130, 246, 0.28);
}

.page-node--basic {
  background: rgba(15, 23, 42, 0.024);
}

.page-node--basic.is-empty {
  color: var(--designer-text-muted, #8c8c8c);
}

.folder-dropzone {
  margin: 2px 0 2px 26px;
  padding: 6px 8px;
  border: 1px dashed #d0d7de;
  border-radius: 4px;
  font-size: 11px;
  color: var(--designer-text-muted, #8c8c8c);
}

.dark .tree-node:hover {
  background-color: rgba(255, 255, 255, 0.06);
}

.dark .tree-node.is-active {
  background-color: rgba(255, 255, 255, 0.1);
}

/* 节点缩进区域 */
.node-indent {
  width: 16px;
  height: 20px;
  flex-shrink: 0;
  position: relative;
}

.node-indent--folder {
  width: 6px;
}

.node-indent--basic {
  width: 10px;
}

/* 层级连接线 */
.tree-line {
  position: absolute;
  background-color: #dcdfe6;
}

.dark .tree-line {
  background-color: #4a4a4a;
}

.tree-line.vertical {
  width: 1px;
  height: 100%;
  left: 6px;
  top: -12px;
}

.tree-line.horizontal {
  height: 1px;
  width: 14px;
  left: 2px;
  top: 50%;
}

/* 子节点容器 */
.tree-children {
  position: relative;
  margin-left: 4px;
  padding-left: 12px;
}

.tree-children::before {
  content: "";
  position: absolute;
  left: 20px;
  top: 0;
  bottom: 12px;
  width: 1px;
  background-color: #dcdfe6;
}

.dark .tree-children::before {
  background-color: #4a4a4a;
}

/* 节点箭头 */
.node-arrow {
  width: 12px;
  height: 12px;
  color: #909399;
  flex-shrink: 0;
  cursor: pointer;
  transition: transform 0.2s;
}

.node-arrow:hover {
  color: #606266;
}

/* 节点图标 */
.node-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.node-icon.folder {
  color: #e6a23c;
}

.node-icon.page {
  color: #909399;
}

.tree-node.is-active .node-icon.page {
  color: #606266;
}

/* 节点标签 */
.node-label {
  flex: 1;
  font-size: 12px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dark .node-label {
  color: #d1d5db;
}

.tree-node.is-active .node-label {
  color: #303133;
  font-weight: 500;
}

.node-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.node-main .node-label {
  flex: none;
}

.node-meta {
  font-size: 11px;
  line-height: 1.2;
  color: var(--designer-text-muted, #8c8c8c);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-action-btn {
  width: 18px;
  height: 18px;
  padding: 0;
  color: #909399;
  opacity: 0;
  transition:
    opacity 0.15s ease,
    color 0.15s ease,
    background-color 0.15s ease;
}

.tree-node:hover .node-action-btn,
.tree-node.is-active .node-action-btn,
.node-action-btn:focus-visible {
  opacity: 1;
}

.node-action-btn:hover {
  color: #606266;
  background: rgba(15, 23, 42, 0.05);
}

/* 节点计数 */
.node-count {
  font-size: 11px;
  color: #8c8c8c;
  background: rgba(15, 23, 42, 0.04);
  padding: 0 5px;
  line-height: 16px;
  border-radius: 10px;
}

.dark .node-count {
  background: #3a3a3a;
  color: #a0a0a0;
}

/* 节点标签 */
.node-badge {
  flex-shrink: 0;
  font-size: 10px;
  margin-left: 4px;
}

/* ==================== 分组节点 ==================== */
.folder-node:hover {
  background-color: rgba(15, 23, 42, 0.04);
}

.folder-node.is-active {
  background-color: rgba(15, 23, 42, 0.08);
}

.dark .folder-node:hover {
  background-color: rgba(255, 255, 255, 0.06);
}

.dark .folder-node.is-active {
  background-color: rgba(255, 255, 255, 0.1);
}

/* ==================== 空状态 ==================== */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 28px 16px;
  color: #909399;
}

.empty-icon {
  width: 32px;
  height: 32px;
  color: #dcdfe6;
  margin-bottom: 8px;
}

.dark .empty-icon {
  color: #4a4a4a;
}

.empty-text {
  font-size: 12px;
  color: #909399;
  margin: 0 0 2px 0;
}

.empty-hint {
  font-size: 11px;
  color: #c0c4cc;
  margin: 0;
}
</style>
