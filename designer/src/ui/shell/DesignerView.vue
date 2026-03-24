<!--
  DesignerView - 设计器主视图
  布局：顶部工具栏 + 左侧工具轨 + 左侧面板（可停靠/浮动）+ 画布区 + 右侧面板
  功能：页面管理、画布编辑、属性面板、预览、保存、导出等
-->
<template>
  <div class="designer-layout">
    <!-- 顶部工具栏 -->
    <TopToolbar :page-name="pageName" :is-locked="isLocked" :is-dirty="isDirty" :view-presets="viewPresets"
      :active-view-key="activeViewKey" :canvas-width="canvasWidth" :canvas-height="canvasHeight"
      :is-custom-view="isCustomView" :can-undo="canUndoEnabled" :can-redo="canRedoEnabled"
      :can-move-layer="canMoveLayer" :zoom="zoom" :show-ruler="showRuler" :show-grid="showGrid"
      :enable-snap="enableSnap" :is-saving="isSaving" :save-settings="saveSettings" :has-selection="hasSelection"
      :has-clipboard="hasClipboard" @update:activeViewKey="handleViewChange" @undo="handleUndo" @redo="handleRedo"
      @preview="handlePreview" @previewApp="handlePreviewApp" @save="handleSave" @export="handleExport"
      @toggleLock="handleToggleLock" @moveUp="handleLayerMoveUp" @moveDown="handleLayerMoveDown"
      @moveToTop="handleLayerMoveToTop" @moveToBottom="handleLayerMoveToBottom"
      @openCollaboration="handleOpenCollaboration" @refreshCanvas="handleRefreshCanvas"
      @toggleLocale="handleToggleLocale" @openAi="handleOpenAi" @toggleTheme="handleToggleTheme"
      @clearCanvas="handleClearCanvas" @applyCustomSize="handleApplyCustomSize"
      @saveSettingsChange="handleSaveSettingsChange" @zoomIn="handleZoomIn" @zoomOut="handleZoomOut"
      @fitCanvas="handleFitCanvas" @fitScreen="handleFitScreen" @toggleRuler="handleToggleRuler"
      @toggleGrid="handleToggleGrid" @toggleSnap="handleToggleSnap" @copy="handleCopy" @paste="handlePaste"
      @deleteSelected="handleDeleteSelected" />

    <SelectionToolbar v-if="hasSelection && hasPages" />

    <!-- 主体区域 -->
    <div class="designer-main">
      <ToolRail side="left" :items="leftRailItems" :active-key="leftActiveKey" @select="handleLeftSelect" />

      <div class="designer-workspace">
        <div class="designer-workspace-main">
          <DockPanel v-if="leftActiveKey && !leftFloating" side="left" :title="leftPanelTitle" :floating="leftFloating"
            @close="handleLeftClose" @toggleFloating="toggleLeftFloating">
            <template #actions>
              <el-tooltip v-if="leftActiveKey === 'pages'" content="新建页面">
                <el-button size="small" text @click="handlePageCreate">
                  <IconEpPlus />
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="leftActiveKey === 'pages'" content="导入页面">
                <el-button size="small" text @click="handlePageImport">
                  <IconEpUpload />
                </el-button>
              </el-tooltip>
            </template>
            <component :is="leftPanelComponent" :key="`left-panel-${leftActiveKey}`" v-bind="leftPanelProps"
              ref="leftPanelRef" @update:drawingTool="setDrawingTool" />
          </DockPanel>

          <div ref="canvasHostRef" class="designer-canvas">
            <!-- 画布容器 -->
            <template v-if="hasPages">
              <CanvasContainer :width="canvasWidth" :height="canvasHeight" :zoom="zoom" :show-ruler="showRuler"
                :view-reset-token="viewResetToken" @zoomChange="handleZoomChange" />
            </template>

            <!-- 空页面提示 -->
            <div v-else class="empty-canvas-placeholder">
              <div class="empty-content">
                <IconEpDocument class="empty-icon" />
                <h3 class="empty-title">暂无页面</h3>
                <p class="empty-desc">创建一个新页面开始设计</p>
                <el-button type="primary" @click="handlePageCreate">
                  <IconEpPlus class="mr-1" />
                  新建页面
                </el-button>
              </div>
            </div>

            <DockPanel v-if="leftActiveKey && leftFloating" side="left" :title="leftPanelTitle" :floating="leftFloating"
              @close="handleLeftClose" @toggleFloating="toggleLeftFloating">
              <template #actions>
                <el-tooltip v-if="leftActiveKey === 'pages'" content="新建页面">
                  <el-button size="small" text @click="handlePageCreate">
                    <IconEpPlus />
                  </el-button>
                </el-tooltip>
                <el-tooltip v-if="leftActiveKey === 'pages'" content="导入页面">
                  <el-button size="small" text @click="handlePageImport">
                    <IconEpUpload />
                  </el-button>
                </el-tooltip>
              </template>
              <component :is="leftPanelComponent" :key="`left-floating-panel-${leftActiveKey}`" v-bind="leftPanelProps"
                ref="leftPanelRef" @update:drawingTool="setDrawingTool" />
            </DockPanel>

            <DockPanel v-if="rightActiveKey && rightFloating" side="right" :title="rightPanelTitle"
              :floating="rightFloating" @close="handleRightClose" @toggleFloating="toggleRightFloating">
              <component :is="rightPanelComponent" />
            </DockPanel>
          </div>

          <DockPanel v-if="rightActiveKey && !rightFloating" side="right" :title="rightPanelTitle"
            :floating="rightFloating" @close="handleRightClose" @toggleFloating="toggleRightFloating">
            <component :is="rightPanelComponent" />
          </DockPanel>
        </div>

        <div class="designer-bottom-toolbar">
          <div class="page-tabs-bar">
            <el-tabs v-if="pageTabs.length > 0" v-model="activePageTabId" type="card" closable addable
              @tab-remove="handleClosePageTab" @tab-add="handlePageCreate">
              <el-tab-pane v-for="tab in pageTabs" :key="tab.id" :name="tab.id" closable>
                <template #label>
                  <span class="page-tab-label">
                    <IconEpDocument class="tab-icon" />
                    <span class="tab-name">{{ tab.name }}</span>
                    <IconEpWarning v-if="tab.isDirty" class="tab-dirty-icon" title="未保存" />
                  </span>
                </template>
              </el-tab-pane>
            </el-tabs>
            <div v-else class="page-tabs-empty">
              <span>暂无页面</span>
              <el-button class="page-tabs-add-btn" text @click="handlePageCreate">
                <IconEpPlus />
              </el-button>
            </div>
          </div>
          <!-- 底部右侧状态信息区 -->
          <div class="status-info-bar">
            <span class="status-item status-mouse">
              {{ canvasMousePos ? `X: ${Math.round(canvasMousePos.x)}  Y: ${Math.round(canvasMousePos.y)}` : 'X: -  Y: -' }}
            </span>
            <span class="status-sep">|</span>
            <template v-if="selectedNodeName">
              <span class="status-item status-node-name" :title="selectedNodeName">
                {{ selectedNodeName }}
              </span>
              <span class="status-sep">|</span>
            </template>
            <template v-if="selectedNodePos">
              <span class="status-item status-mouse">
                {{ selectedNodePos.x }}, {{ selectedNodePos.y }}
              </span>
              <span class="status-sep">|</span>
            </template>
            <template v-if="selectedNodeSize">
              <span class="status-item status-mouse">
                {{ selectedNodeSize.w }} × {{ selectedNodeSize.h }}
              </span>
              <span class="status-sep">|</span>
            </template>
            <span class="status-item">选中: {{ selectionCount }}</span>
            <span class="status-sep">|</span>
            <span class="status-item">共 {{ totalNodeCount }} 个</span>
            <template v-if="hoveredNodeType">
              <span class="status-sep">|</span>
              <span class="status-item status-hover">{{ hoveredNodeType }}</span>
            </template>
          </div>
        </div>
      </div>

      <ToolRail side="right" :items="rightRailItems" :active-key="rightActiveKey" @select="handleRightSelect" />
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  provide,
  inject,
  type Ref,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessageBox } from "element-plus";
import { ElMessage } from "./el-message-compat";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { CanvasContainer } from "@/ui/editors/page/canvas";
import SelectionToolbar from "@/ui/editors/page/canvas/SelectionToolbar.vue";
import { TopToolbar } from "@/ui/shell/TopToolbar";
import { ToolRail } from "@/ui/shell/ToolRail";
import { DockPanel } from "@/ui/shell/DockPanel";
import { OutlineTree, MaterialPanel, DataPanel } from "@/ui/editors/page/panels/left";
import {
  PageTree,
  I18nPanel,
  ScriptVarsPanel,
  RolePanel,
  VariablesPanel,
} from "@/ui/shared/panels";
import { PropertyPanel, AdvancedPanel } from "@/ui/editors/page/panels/right";
import { VIEW_PRESETS, type ViewPreset } from "@/constants";
import type { ToolRailItem } from "./tool-rail-types";
import IconEpDocument from "~icons/ep/document";
import IconEpPlus from "~icons/ep/plus";
import IconEpWarning from "~icons/ep/warning";
import IconEpUpload from "~icons/ep/upload";
import IconLucideFileText from "~icons/lucide/file-text";
import IconLucideList from "~icons/lucide/list";
import IconLucideBox from "~icons/lucide/box";
import IconLucideDatabase from "~icons/lucide/database";
import IconLucideLanguages from "~icons/lucide/languages";
import IconLucideFileCode from "~icons/lucide/file-code";
import IconLucideUsers from "~icons/lucide/users";
import IconLucideSlidersHorizontal from "~icons/lucide/sliders-horizontal";
import IconLucideSettings from "~icons/lucide/settings";
import IconLucideBraces from "~icons/lucide/braces";
import { Storage } from "@/utils/storage";

const route = useRoute();
const router = useRouter();

/** 路由 meta 中的工程信息（由路由守卫注入） */
interface RouteProjectMeta {
  project?: { id: string };
}

/** store 中页面列表项（editor-store 尚未 TS 化时的最小形状） */
interface StorePageRow {
  id: string;
  name?: string;
  type?: string;
  parentId?: string | null;
  config?: {
    width?: number;
    height?: number;
    showGrid?: boolean;
    enableSnap?: boolean;
  };
  rootNodeId?: string;
}

interface PageTab {
  id: string;
  name: string;
  isDirty: boolean;
}

/** storeToRefs(StoreGeneric) 会把 ref 标成可能 undefined，此处收窄为壳层实际用到的形状 */
interface EditorShellStoreRefs {
  canUndo: Ref<boolean>;
  canRedo: Ref<boolean>;
  isSaving: Ref<boolean>;
  selection: Ref<
    | {
        getSelectionCount?: () => number;
        getPrimaryElement: () => { kind: string; id: string } | null;
      }
    | undefined
  >;
  doc: Ref<
    | {
        nodesById?: Record<string, unknown>;
        getNode?: (id: string) => {
          absolutePos?: { x: number; y: number };
          label?: string;
          type?: string;
          props?: { width?: number; height?: number };
          style?: { width?: number; height?: number };
        } | null;
      }
    | undefined
  >;
  pages: Ref<StorePageRow[]>;
  currentPageId: Ref<string>;
  currentPage: Ref<StorePageRow | null | undefined>;
  isLocked: Ref<boolean>;
  readonlyState: Ref<{ readonly?: boolean } | undefined>;
  pageTabState: Ref<{ tabs?: PageTab[]; activeId?: string } | undefined>;
}

const editorStore = useEditorStore();
const {
  canUndo,
  canRedo,
  isSaving,
  selection,
  doc,
  pages,
  currentPageId,
  currentPage,
  isLocked,
  readonlyState,
  pageTabState,
} = storeToRefs(editorStore) as unknown as EditorShellStoreRefs;

const zoom = ref(1);
const showRuler = ref(true);
const viewResetToken = ref(0);
const activeViewKey = ref("pc");
const viewPresets: readonly ViewPreset[] = VIEW_PRESETS;
const SAVE_SETTINGS_STORAGE_KEY = "designer_save_settings";
const AUTO_FIT_PADDING = 48;
const saveSettings = ref({
  autoSave: false,
  intervalMinutes: 5,
});
const autoSaveTimer = ref<ReturnType<typeof setInterval> | null>(null);
const autoSaving = ref(false);
const autoZoomEnabled = ref(true);
const autoFitFrame = ref(0);
const canvasHostResizeObserver = ref<ResizeObserver | null>(null);

const drawingTool = ref("");

const setDrawingTool = (value: string) => {
  drawingTool.value = value;
};

// ==================== 页面标签页系统 ====================
const pageTabs = ref<PageTab[]>([]);
const activePageTabId = ref("");

{
  const initialTabState = pageTabState.value;
  if (initialTabState?.tabs?.length) {
    pageTabs.value = initialTabState.tabs.map((item: PageTab) => ({
      ...item,
    }));
    activePageTabId.value = initialTabState.activeId || "";
  }
}

const openPageTab = (pageId: string) => {
  const page = editorStore.pages.find((p: StorePageRow) => p.id === pageId);
  if (!page) return;

  // 检查是否已打开
  const existingTab = pageTabs.value.find((t) => t.id === pageId);
  if (existingTab) {
    // 只更新 activePageTabId，watch 会自动调用 setCurrentPage
    activePageTabId.value = pageId;
    return;
  }

  // 添加新标签页
  pageTabs.value.push({
    id: pageId,
    name: page.name || "未命名页面",
    isDirty: false,
  });

  // 只更新 activePageTabId，watch 会自动调用 setCurrentPage
  activePageTabId.value = pageId;
};

const handleClosePageTab = async (tabId: string) => {
  const index = pageTabs.value.findIndex((t) => t.id === tabId);
  if (index === -1) return;

  const tab = pageTabs.value[index];
  if (!tab) return;

  // 如果有未保存的修改，弹窗确认
  if (tab.isDirty) {
    try {
      const action = await ElMessageBox.confirm(
        `页面 "${tab.name}" 有未保存的修改，是否保存后关闭？`,
        "关闭确认",
        {
          distinguishCancelAndClose: true,
          confirmButtonText: "保存并关闭",
          cancelButtonText: "不保存",
          type: "warning",
        },
      );

      if (action === "confirm") {
        // 先切换到该页面再保存
        if (currentPageId.value !== tabId) {
          editorStore.setCurrentPage(tabId);
          await new Promise((resolve) => setTimeout(resolve, 100));
        }
        await editorStore.saveCurrentPage();
        ElMessage.success("页面已保存");
      }
    } catch (action) {
      if (action === "close") {
        // 用户点击关闭按钮，取消操作
        return;
      }
      // action === 'cancel'，不保存直接关闭
    }
  }

  // 执行关闭
  pageTabs.value.splice(index, 1);

  // 如果关闭的是当前标签页
  if (activePageTabId.value === tabId) {
    if (pageTabs.value.length > 0) {
      // 切换到其他标签页
      const newActiveTab =
        pageTabs.value[Math.min(index, pageTabs.value.length - 1)];
      if (!newActiveTab) {
        activePageTabId.value = "";
        return;
      }
      activePageTabId.value = newActiveTab.id;
      editorStore.setCurrentPage(newActiveTab.id);
    } else {
      // 没有标签页了，清空状态
      activePageTabId.value = "";
    }
  }
};

/**
 * 更新标签页脏状态
 */
const updateTabDirtyState = () => {
  const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
  if (tab) {
    tab.isDirty = canUndo.value;
  }
};

/**
 * 更新标签页名称
 */
const updateTabName = () => {
  const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
  if (tab) {
    // 从 pages 列表获取最新的页面名称
    const page = editorStore.pages.find(
      (p: StorePageRow) => p.id === currentPageId.value,
    );
    tab.name = page?.name || "未命名页面";
  }
};

/**
 * 同步底部页面标签，清理已被删除的页面标签
 * @returns {void}
 */
const syncPageTabsWithPages = () => {
  const pageIdSet = new Set(
    editorStore.pages.map((page: StorePageRow) => page.id),
  );
  const nextTabs = pageTabs.value.filter((tab) => pageIdSet.has(tab.id));
  if (nextTabs.length !== pageTabs.value.length) {
    pageTabs.value = nextTabs;
  }
  if (activePageTabId.value && !pageIdSet.has(activePageTabId.value)) {
    activePageTabId.value =
      currentPageId.value && pageIdSet.has(currentPageId.value)
        ? currentPageId.value
        : nextTabs[0]?.id || "";
  }
};

// 监听 canUndo 变化，更新标签页脏状态
watch(canUndo, updateTabDirtyState);

// 监听页面列表变化，更新标签页名称
watch(
  [() => editorStore.pages, currentPageId],
  () => {
    syncPageTabsWithPages();
    updateTabName();
  },
  { deep: true },
);

// 监听标签页切换，同步到 currentPageId
watch(
  activePageTabId,
  async (newTabId, oldTabId) => {
    // 标签页切换时，直接同步到 store
    if (newTabId && newTabId !== oldTabId) {
      if (canUndo.value && currentPageId.value) {
        editorStore.saveCurrentPageDraft();
      }
      await editorStore.setCurrentPage(newTabId);
    }
  },
  { flush: "sync" },
);

watch(
  [pageTabs, activePageTabId],
  () => {
    editorStore.setPageTabState(pageTabs.value, activePageTabId.value);
  },
  { deep: true },
);

// 初始化时打开当前页面
watch(
  currentPageId,
  (newPageId, oldPageId) => {
    if (newPageId && !pageTabs.value.find((t) => t.id === newPageId)) {
      // 验证页面是否真的存在
      const pageExists = editorStore.pages.some(
        (p: StorePageRow) => p.id === newPageId,
      );
      if (pageExists) {
        openPageTab(newPageId);
      }
    }
    if (newPageId && oldPageId && newPageId !== oldPageId) {
      activateMaterialPanel();
    }
  },
  { immediate: true },
);

// 提供 openPageTab 方法给子组件
provide("openPageTab", openPageTab);

// ==================== 底部状态栏数据 ====================
/** 从 DesignCanvas 注入画布鼠标坐标（provide in DesignCanvas.vue） */
const canvasMousePos = inject<Ref<{ x: number; y: number } | null>>(
  "canvasMousePos",
  ref(null),
);
/** 从 DesignCanvas 注入悬停节点类型（provide in DesignCanvas.vue） */
const hoveredNodeType = inject<Ref<string>>("hoveredNodeType", ref(""));

/**
 * 当前选中节点数量
 */
const selectionCount = computed(() => {
  void editorStore.selectionVersion;
  return selection.value?.getSelectionCount?.() ?? 0;
});

/**
 * 当前页面内的总节点数（排除根节点）
 */
const totalNodeCount = computed(() => {
  void editorStore.docVersion;
  if (!doc.value) return 0;
  const root = currentPage.value?.rootNodeId;
  const allIds = Object.keys(doc.value?.nodesById || {});
  // 减去根节点本身
  return root ? Math.max(0, allIds.length - 1) : allIds.length;
});

/**
 * 主选中节点的位置（absolutePos 或 DOM 坐标）
 */
const selectedNodePos = computed(() => {
  void editorStore.selectionVersion;
  void editorStore.docVersion;
  const primary = selection.value?.getPrimaryElement();
  if (!primary || primary.kind !== "node") return null;
  const node = doc.value?.getNode?.(primary.id);
  if (!node) return null;
  if (node.absolutePos && Number.isFinite(node.absolutePos.x)) {
    return {
      x: Math.round(node.absolutePos.x),
      y: Math.round(node.absolutePos.y),
    };
  }
  // flow 定位：从 DOM 读取相对于根节点的坐标
  const rootId = currentPage.value?.rootNodeId;
  const rootEl = rootId
    ? document.querySelector(`[data-node-id="${rootId}"]`)
    : null;
  const nodeEl = document.querySelector(`[data-node-id="${primary.id}"]`);
  if (rootEl && nodeEl) {
    const rootRect = rootEl.getBoundingClientRect();
    const nodeRect = nodeEl.getBoundingClientRect();
    const zoomValue = zoom.value || 1;
    return {
      x: Math.round((nodeRect.left - rootRect.left) / zoomValue),
      y: Math.round((nodeRect.top - rootRect.top) / zoomValue),
    };
  }
  return null;
});

/**
 * 主选中节点的显示名称（label 优先，fallback 到 type）
 */
const selectedNodeName = computed(() => {
  void editorStore.selectionVersion;
  void editorStore.docVersion;
  const primary = selection.value?.getPrimaryElement();
  if (!primary || primary.kind !== "node") return "";
  const node = doc.value?.getNode?.(primary.id);
  if (!node) return "";
  return node.label || node.type || "";
});

/**
 * 主选中节点的尺寸（优先从 props/style 读取，fallback 到 DOM 实际渲染尺寸）
 */
const selectedNodeSize = computed(() => {
  void editorStore.selectionVersion;
  void editorStore.docVersion;
  const primary = selection.value?.getPrimaryElement();
  if (!primary || primary.kind !== "node") return null;
  const node = doc.value?.getNode?.(primary.id);
  if (!node) return null;
  const pw = node.props?.width ?? node.style?.width;
  const ph = node.props?.height ?? node.style?.height;
  if (pw != null && ph != null) {
    return { w: pw, h: ph };
  }
  const nodeEl = document.querySelector(`[data-node-id="${primary.id}"]`);
  if (nodeEl instanceof HTMLElement) {
    const zoomValue = zoom.value || 1;
    return {
      w: Math.round(nodeEl.offsetWidth / zoomValue),
      h: Math.round(nodeEl.offsetHeight / zoomValue),
    };
  }
  return null;
});
// ==================== 底部状态栏数据结束 ====================

/**
 * 是否有页面
 */
const hasPages = computed(() => {
  // 确保页面列表不为空且当前页面确实存在
  if (editorStore.pages.length === 0) return false;
  if (!currentPageId.value) return false;
  return editorStore.pages.some(
    (p: StorePageRow) => p.id === currentPageId.value,
  );
});
// ==================== 页面标签页系统结束 ====================
const canUndoEnabled = computed(
  () => canUndo.value && !readonlyState.value?.readonly,
);
const canRedoEnabled = computed(
  () => canRedo.value && !readonlyState.value?.readonly,
);

/**
 * 是否可以移动图层
 */
const canMoveLayer = computed(() => {
  if (readonlyState.value?.readonly) return false;
  const primary = selection.value?.getPrimaryElement();
  if (!primary || primary.kind !== "node") return false;
  // 不能移动根节点
  const rootNodeId = currentPage.value?.rootNodeId;
  return primary.id !== rootNodeId;
});
const hasSelection = computed(() => {
  void editorStore.selectionVersion;
  return (selection.value?.getSelectionCount?.() || 0) >= 1;
});
const hasClipboard = computed(() => editorStore.hasClipboard);
const leftActiveKey = ref("pages");
const rightActiveKey = ref("props");
const leftFloating = ref(false);
const rightFloating = ref(false);
const leftPanelRef = ref<{ openCreateDialog?: () => void } | null>(null);
const canvasHostRef = ref<HTMLElement | null>(null);

const pageName = computed(() => {
  // 如果没有页面，返回空
  if (!hasPages.value) return "";
  // 优先从 pages 列表获取名称（更可靠）
  const pageFromList = editorStore.pages.find(
    (p: StorePageRow) => p.id === currentPageId.value,
  );
  if (pageFromList?.name) return pageFromList.name;
  // 其次从 doc 中获取
  const page = currentPage.value;
  return page?.name || "";
});

const isDirty = computed(() => canUndo.value);

const currentPageSnapshot = computed(() => {
  const page = pages.value.find(
    (item: StorePageRow) => item.id === currentPageId.value,
  );
  return page || currentPage.value || null;
});
const activeView = computed(() =>
  viewPresets.find((preset) => preset.key === activeViewKey.value),
);
const defaultViewPreset = computed(
  () =>
    viewPresets.find((preset) => preset.key === "pc") ||
    viewPresets[0] || { width: 1366, height: 768 },
);
const normalizeCanvasDimension = (value: unknown, fallback: number): number => {
  const next = Number(value);
  return Number.isFinite(next) && next > 0 ? Math.round(next) : fallback;
};
const resolvedPageWidth = computed(() =>
  normalizeCanvasDimension(
    currentPageSnapshot.value?.config?.width,
    defaultViewPreset.value.width,
  ),
);
const resolvedPageHeight = computed(() =>
  normalizeCanvasDimension(
    currentPageSnapshot.value?.config?.height,
    defaultViewPreset.value.height,
  ),
);
const matchedViewPreset = computed(
  () =>
    viewPresets.find(
      (preset) =>
        preset.width === resolvedPageWidth.value &&
        preset.height === resolvedPageHeight.value,
    ) || null,
);
const canvasWidth = computed(() =>
  normalizeCanvasDimension(
    currentPageSnapshot.value?.config?.width,
    defaultViewPreset.value.width,
  ),
);
const canvasHeight = computed(() =>
  normalizeCanvasDimension(
    currentPageSnapshot.value?.config?.height,
    defaultViewPreset.value.height,
  ),
);
const isCustomView = computed(() => !matchedViewPreset.value);
const showGrid = computed(() =>
  Boolean(currentPageSnapshot.value?.config?.showGrid),
);
const enableSnap = computed(
  () => currentPageSnapshot.value?.config?.enableSnap ?? true,
);

const leftRailItems: ToolRailItem[] = [
  { key: "pages", label: "页面", icon: IconLucideFileText },
  { key: "outline", label: "大纲", icon: IconLucideList },
  { key: "material", label: "物料", icon: IconLucideBox },
  { key: "data", label: "数据", icon: IconLucideDatabase },
  { key: "i18n", label: "国际", icon: IconLucideLanguages },
  { key: "script", label: "脚本", icon: IconLucideFileCode },
  { key: "role", label: "角色", icon: IconLucideUsers, placement: "bottom" },
];

const rightRailItems: ToolRailItem[] = [
  { key: "props", label: "属性", icon: IconLucideSlidersHorizontal },
  { key: "advanced", label: "高级", icon: IconLucideSettings },
  { key: "variables", label: "变量", icon: IconLucideBraces },
];

const leftPanelComponent = computed(() => {
  switch (leftActiveKey.value) {
    case "pages":
      return PageTree;
    case "outline":
      return OutlineTree;
    case "material":
      return MaterialPanel;
    case "data":
      return DataPanel;
    case "i18n":
      return I18nPanel;
    case "script":
      return ScriptVarsPanel;
    case "role":
      return RolePanel;
    default:
      return PageTree;
  }
});

const leftPanelTitle = computed(() => {
  const item = leftRailItems.find((entry) => entry.key === leftActiveKey.value);
  return item?.label || "面板";
});

const leftPanelProps = computed(() => {
  if (leftActiveKey.value === "material") {
    return { drawingTool: drawingTool.value };
  }
  return {};
});

const rightPanelComponent = computed(() => {
  switch (rightActiveKey.value) {
    case "props":
      return PropertyPanel;
    case "advanced":
      return AdvancedPanel;
    case "variables":
      return VariablesPanel;
    default:
      return PropertyPanel;
  }
});

const rightPanelTitle = computed(() => {
  const item = rightRailItems.find(
    (entry) => entry.key === rightActiveKey.value,
  );
  return item?.label || "面板";
});

/**
 * 撤销操作
 */
const handleUndo = () => {
  if (!editorStore.undo()) {
    ElMessage.info("没有可撤销的操作");
  }
};

/**
 * 重做操作
 */
const handleRedo = () => {
  if (!editorStore.redo()) {
    ElMessage.info("没有可重做的操作");
  }
};

/**
 * 预览
 */
const handlePreview = () => {
  const projectId = (route.meta as RouteProjectMeta).project?.id;
  router.push({
    path: "/preview",
    query: { pid: projectId, pageId: currentPageId.value || "" },
  });
};

/**
 * 应用预览（占位）
 */
const handlePreviewApp = () => {
  ElMessage.info("应用预览功能开发中");
};

/**
 * 保存
 */
const handleSave = async () => {
  try {
    await editorStore.saveCurrentPage();

    // 保存成功后清除当前标签页的脏状态
    const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
    if (tab) {
      tab.isDirty = false;
    }

    ElMessage.success("保存成功");
  } catch (error) {
    const message = error instanceof Error ? error.message : "未知错误";
    ElMessage.error("保存失败: " + message);
  }
};

const handleViewChange = (key: string) => {
  if (key === "custom") {
    activeViewKey.value = "custom";
    return;
  }
  activeViewKey.value = key;
  const targetView = viewPresets.find((preset) => preset.key === key);
  if (!targetView || !currentPage.value) return;
  const nextConfig = {
    ...currentPage.value.config,
    width: targetView.width,
    height: targetView.height,
  };
  editorStore.updateCurrentPage({ config: nextConfig });
};

const handleApplyCustomSize = ({
  width,
  height,
}: {
  width: number;
  height: number;
}) => {
  const page = currentPageSnapshot.value;
  if (!page) return;
  const nextConfig = {
    ...(page.config || {}),
    width: Math.round(width),
    height: Math.round(height),
  };
  activeViewKey.value = "custom";
  editorStore.updateCurrentPage({ config: nextConfig });
};

/**
 * 切换锁定状态
 */
const handleToggleLock = async () => {
  const result = await editorStore.togglePageLock();
  if (!result) return;

  if (result.success) {
    const message =
      result.action === "release" ? "已释放页面锁" : "已获取页面锁";
    ElMessage.success(message);
    return;
  }

  if (result.reason === "locked") {
    ElMessage.warning(`页面已被${result.lockedByName || "其他用户"}锁定`);
    return;
  }

  ElMessage.error(result.error?.message || "页面锁操作失败");
};

/**
 * 导出页面 Schema
 */
const handleExport = () => {
  if (!editorStore.doc || !currentPageId.value) {
    ElMessage.warning("暂无可导出的页面");
    return;
  }
  const payload = editorStore.serializer.exportPage(
    editorStore.doc,
    currentPageId.value,
  );
  const json = JSON.stringify(payload, null, 2);
  const blob = new Blob([json], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const name = `${pageName.value || "page"}.json`;
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
  ElMessage.success("已导出页面");
};

const handleLeftSelect = (key: string) => {
  leftActiveKey.value = leftActiveKey.value === key ? "" : key;
};

const handleRightSelect = (key: string) => {
  rightActiveKey.value = rightActiveKey.value === key ? "" : key;
};

const handleLeftClose = () => {
  leftActiveKey.value = "";
};

const handleRightClose = () => {
  rightActiveKey.value = "";
};

/**
 * 页面切换后默认回到物料面板，便于继续拖拽组件
 * @returns {void}
 */
const activateMaterialPanel = () => {
  leftActiveKey.value = "material";
};

const toggleLeftFloating = () => {
  leftFloating.value = !leftFloating.value;
};

const toggleRightFloating = () => {
  rightFloating.value = !rightFloating.value;
};

const handleZoomChange = (value: number) => {
  autoZoomEnabled.value = false;
  zoom.value = value;
};

const clampZoom = (value: number): number => {
  const next = Number.isFinite(value) ? value : 1;
  return Math.min(5, Math.max(0.1, Number(next.toFixed(2))));
};

/**
 * 工具栏：缩小
 * @returns {void}
 */
const handleZoomOut = () => {
  autoZoomEnabled.value = false;
  zoom.value = clampZoom(zoom.value - 0.1);
};

/**
 * 工具栏：放大
 * @returns {void}
 */
const handleZoomIn = () => {
  autoZoomEnabled.value = false;
  zoom.value = clampZoom(zoom.value + 0.1);
};

/**
 * 工具栏：适配画布（100%）
 * @returns {void}
 */
const handleFitCanvas = () => {
  autoZoomEnabled.value = false;
  zoom.value = 1;
  viewResetToken.value += 1;
};

/**
 * 计算适配当前工作区的推荐缩放比例
 * 规则与参考页保持一致：能 100% 展示时保持 100%，不足时自动缩放到刚好适配
 * @returns {number}
 */
const getRecommendedZoom = () => {
  const host = canvasHostRef.value;
  if (!host) {
    return 1;
  }
  const rect = host.getBoundingClientRect();
  const availableWidth = Math.max(1, rect.width - AUTO_FIT_PADDING);
  const availableHeight = Math.max(1, rect.height - AUTO_FIT_PADDING);
  const fitZoom = Math.min(
    1,
    availableWidth / canvasWidth.value,
    availableHeight / canvasHeight.value,
  );
  return clampZoom(fitZoom);
};

/**
 * 应用推荐缩放比例
 * @returns {void}
 */
const applyRecommendedZoom = () => {
  const nextZoom = getRecommendedZoom();
  if (Math.abs(nextZoom - zoom.value) < 0.001) {
    if (viewResetToken.value === 0) {
      viewResetToken.value += 1;
    }
    return;
  }
  zoom.value = nextZoom;
  viewResetToken.value += 1;
};

/**
 * 在布局稳定后重新计算自动缩放
 * @returns {void}
 */
const scheduleAutoFit = () => {
  if (!autoZoomEnabled.value) return;
  if (typeof window === "undefined") return;
  if (autoFitFrame.value) {
    window.cancelAnimationFrame(autoFitFrame.value);
  }
  autoFitFrame.value = window.requestAnimationFrame(() => {
    autoFitFrame.value = 0;
    if (!autoZoomEnabled.value) return;
    applyRecommendedZoom();
  });
};

/**
 * 工具栏：适配屏幕
 * @returns {void}
 */
const handleFitScreen = () => {
  autoZoomEnabled.value = true;
  applyRecommendedZoom();
};

/**
 * 切换标尺显示
 * @returns {void}
 */
const handleToggleRuler = () => {
  showRuler.value = !showRuler.value;
};

/**
 * 切换网格显示
 * @returns {void}
 */
const handleToggleGrid = () => {
  const page = currentPageSnapshot.value;
  if (!page) return;
  const nextConfig = {
    ...(page.config || {}),
    showGrid: !showGrid.value,
  };
  editorStore.updateCurrentPage({ config: nextConfig });
};

/**
 * 切换吸附开关
 * @returns {void}
 */
const handleToggleSnap = () => {
  const page = currentPageSnapshot.value;
  if (!page) return;
  const nextConfig = {
    ...(page.config || {}),
    enableSnap: !enableSnap.value,
  };
  editorStore.updateCurrentPage({ config: nextConfig });
};

/**
 * 打开页面新建弹窗
 * @returns {void}
 */
const handlePageCreate = async () => {
  if (leftActiveKey.value !== "pages") {
    leftActiveKey.value = "pages";
    leftFloating.value = false;
    await nextTick();
  }
  leftPanelRef.value?.openCreateDialog?.();
};

const getUniquePageName = (name: string | undefined) => {
  const base = (name || "导入页面").trim() || "导入页面";
  const existingNames = editorStore.pages
    .map((page: StorePageRow) => page.name)
    .filter(Boolean) as string[];
  if (!existingNames.includes(base)) return base;
  let index = 1;
  let next = `${base}_${index}`;
  while (existingNames.includes(next)) {
    index += 1;
    next = `${base}_${index}`;
  }
  return next;
};

const handlePageImport = () => {
  if (!editorStore.doc) {
    ElMessage.warning("暂无可导入的页面");
    return;
  }
  const input = document.createElement("input");
  input.type = "file";
  input.accept = ".json,application/json";
  input.onchange = async (event: Event) => {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (!file) return;
    try {
      const text = await file.text();
      const payload = JSON.parse(text);
      if (!payload?.page || !payload?.nodesById) {
        ElMessage.error("页面数据格式不正确");
        return;
      }
      const serializer = editorStore.serializer;
      const baseSchema = serializer.exportToSchema(editorStore.doc);
      const tempDoc = serializer.importFromSchema(baseSchema);
      const tempPageId = serializer.importPage(tempDoc, payload, {
        generateNewIds: true,
      });
      const imported = serializer.exportPage(tempDoc, tempPageId);
      const uniqueName = getUniquePageName(payload.page?.name);
      imported.page.name = uniqueName;

      const result = await editorStore.createPage({
        name: uniqueName,
        type: imported.page?.type || "page",
        parentId: null,
      });
      const pageId = result?.id || result?.page?.id;
      if (!pageId) {
        ElMessage.error("导入页面失败");
        return;
      }
      imported.page.id = pageId;
      await editorStore.updatePageSchema(pageId, imported);
      await editorStore.loadPage(pageId);
      openPageTab(pageId);
      ElMessage.success("页面已导入");
    } catch (error) {
      ElMessage.error("导入页面失败");
    }
  };
  input.click();
};

/**
 * 工具栏:上移图层
 */
const handleLayerMoveUp = () => {
  if (editorStore.moveNodeUp()) {
    ElMessage.success("已上移");
  }
};

/**
 * 工具栏:下移图层
 */
const handleLayerMoveDown = () => {
  if (editorStore.moveNodeDown()) {
    ElMessage.success("已下移");
  }
};

/**
 * 工具栏:置顶
 */
const handleLayerMoveToTop = () => {
  if (editorStore.moveNodeToTop()) {
    ElMessage.success("已置顶");
  }
};

/**
 * 工具栏:置底
 */
const handleLayerMoveToBottom = () => {
  if (editorStore.moveNodeToBottom()) {
    ElMessage.success("已置底");
  }
};
const handleCopy = () => {
  if (editorStore.copyNodes()) {
    ElMessage.success("已复制");
  }
};
const handlePaste = () => editorStore.pasteNodes();
const handleDeleteSelected = () => editorStore.removeSelectedNodes();

/**
 * 自动保存当前页面
 * @returns {Promise<void>}
 */
const handleAutoSave = async () => {
  if (!saveSettings.value.autoSave) return;
  if (readonlyState.value?.readonly) return;
  if (!currentPageId.value || autoSaving.value || isSaving.value) return;
  try {
    autoSaving.value = true;
    await editorStore.saveCurrentPage();
    const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
    if (tab) {
      tab.isDirty = false;
    }
  } catch (error) {
    console.warn("自动保存失败:", error);
  } finally {
    autoSaving.value = false;
  }
};

/**
 * 清理自动保存定时器
 * @returns {void}
 */
const clearAutoSaveTimer = () => {
  if (autoSaveTimer.value) {
    clearInterval(autoSaveTimer.value);
    autoSaveTimer.value = null;
  }
};

const syncAutoSaveTimer = () => {
  clearAutoSaveTimer();
  if (!saveSettings.value.autoSave) return;
  const interval = Number(saveSettings.value.intervalMinutes) || 5;
  autoSaveTimer.value = setInterval(
    () => {
      void handleAutoSave();
    },
    interval * 60 * 1000,
  );
};

const handleSaveSettingsChange = (settings: {
  autoSave?: boolean;
  intervalMinutes?: number;
}) => {
  const nextSettings = {
    autoSave: Boolean(settings?.autoSave),
    intervalMinutes: Number(settings?.intervalMinutes) || 5,
  };
  saveSettings.value = nextSettings;
  Storage.set(SAVE_SETTINGS_STORAGE_KEY, nextSettings);
  syncAutoSaveTimer();
};

/**
 * 更多设置：多人协作（占位）
 */
const handleOpenCollaboration = () => {
  ElMessage.info("多人协作功能开发中");
};

/**
 * 工具栏：AI 助手（占位）
 */
const handleOpenAi = () => {
  ElMessage.info("AI 助手功能开发中");
};

/**
 * 工具栏：主题切换（占位）
 */
const handleToggleTheme = () => {
  ElMessage.info("主题设置功能开发中");
};

/**
 * 更多设置：刷新画布（占位）
 */
const handleRefreshCanvas = () => {
  ElMessage.info("画布刷新功能开发中");
};

/**
 * 更多设置：中英文切换（占位）
 */
const handleToggleLocale = () => {
  ElMessage.info("中英文切换功能开发中");
};

/**
 * 工具栏：清除当前界面（占位）
 */
const handleClearCanvas = () => {
  ElMessage.info("清除当前界面功能开发中");
};

watch(
  [
    currentPageId,
    () => currentPageSnapshot.value?.config?.width,
    () => currentPageSnapshot.value?.config?.height,
  ],
  () => {
    activeViewKey.value = matchedViewPreset.value?.key || "custom";
    autoZoomEnabled.value = true;
    nextTick(() => {
      scheduleAutoFit();
    });
  },
  { immediate: true },
);

/**
 * 加载工程数据
 */
const loadProject = async () => {
  const project = (route.meta as RouteProjectMeta).project;
  if (!project?.id) return;
  if (editorStore.projectId === project.id && editorStore.doc) {
    return;
  }
  const result = await editorStore.loadProject(project.id);
  if (!result.ok) {
    ElMessage.error(result.error?.message || "加载工程失败");
    return;
  }
  const targetPageId = String(route.query.pageId || "");
  if (targetPageId) {
    await editorStore.setCurrentPage(targetPageId);
  }
};

onMounted(() => {
  const cached = Storage.get(SAVE_SETTINGS_STORAGE_KEY, null) as {
    autoSave?: boolean;
    intervalMinutes?: number;
  } | null;
  if (cached && typeof cached === "object") {
    saveSettings.value = {
      autoSave: Boolean(cached.autoSave),
      intervalMinutes: Number(cached.intervalMinutes) || 5,
    };
  }
  syncAutoSaveTimer();
  void loadProject();
  nextTick(() => {
    scheduleAutoFit();
    if (canvasHostRef.value && typeof ResizeObserver !== "undefined") {
      canvasHostResizeObserver.value = new ResizeObserver(() => {
        scheduleAutoFit();
      });
      canvasHostResizeObserver.value.observe(canvasHostRef.value);
    }
  });
});

onBeforeUnmount(() => {
  if (typeof window !== "undefined" && autoFitFrame.value) {
    window.cancelAnimationFrame(autoFitFrame.value);
    autoFitFrame.value = 0;
  }
  canvasHostResizeObserver.value?.disconnect?.();
  canvasHostResizeObserver.value = null;
  clearAutoSaveTimer();
  void editorStore.releasePageLock();
});
</script>

<style scoped>
.designer-workspace {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.designer-workspace-main {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: #eef1f5;
}

.designer-canvas {
  background: #f5f7fb;
}

/* ====== 底部工具栏容器 ====== */
.designer-bottom-toolbar {
  flex-shrink: 0;
  height: 34px;
  border-top: 1px solid #e2e8f0;
  background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
  display: flex;
  align-items: stretch;
  overflow: hidden;
}

.dark .designer-bottom-toolbar {
  background: linear-gradient(180deg, #1e293b 0%, #0f172a 100%);
  border-top-color: #334155;
}

.designer-bottom-toolbar .page-tabs-bar {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
}

/* ====== 页面标签栏 ====== */
.page-tabs-bar {
  width: 100%;
  min-width: 0;
  height: 100%;
  border: 0;
  background: transparent;
}

.dark .page-tabs-bar {
  background: transparent;
}

.page-tabs-bar :deep(.el-tabs__header) {
  margin: 0;
  border-bottom: none;
  height: 100%;
}

.page-tabs-bar :deep(.el-tabs__nav-wrap) {
  padding: 0 2px 0 0;
}

.page-tabs-bar :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.page-tabs-bar :deep(.el-tabs__nav-wrap.is-scrollable) {
  padding: 0 24px;
}

.page-tabs-bar :deep(.el-tabs__nav-prev),
.page-tabs-bar :deep(.el-tabs__nav-next) {
  width: 22px;
  height: 100%;
  line-height: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  background: transparent;
  z-index: 2;
  transition: color 0.15s;
}

.page-tabs-bar :deep(.el-tabs__nav-prev:hover),
.page-tabs-bar :deep(.el-tabs__nav-next:hover) {
  color: #3b82f6;
}

.page-tabs-bar :deep(.el-tabs__nav-prev .el-icon),
.page-tabs-bar :deep(.el-tabs__nav-next .el-icon) {
  width: 14px;
  height: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.page-tabs-bar :deep(.el-tabs__nav) {
  border: none;
  height: 100%;
  display: flex;
  align-items: flex-end;
}

.page-tabs-bar :deep(.el-tabs__item) {
  height: 28px;
  line-height: 28px;
  font-size: 12px;
  border: 1px solid transparent !important;
  border-bottom: none !important;
  background: transparent;
  color: #64748b;
  padding: 0 14px !important;
  border-radius: 6px 6px 0 0;
  margin-right: 2px;
  transition: all 0.18s ease;
  position: relative;
}

.page-tabs-bar :deep(.el-tabs__item.is-active) {
  background: #ffffff;
  color: #1e293b;
  border-color: #e2e8f0 !important;
  border-bottom-color: transparent !important;
  font-weight: 500;
  box-shadow: 0 -1px 3px rgba(0, 0, 0, 0.04);
}

.page-tabs-bar :deep(.el-tabs__item:not(.is-active):hover) {
  color: #475569;
  background: rgba(148, 163, 184, 0.12);
}

.dark .page-tabs-bar :deep(.el-tabs__item) {
  color: #94a3b8;
}

.dark .page-tabs-bar :deep(.el-tabs__item.is-active) {
  background: #1e293b;
  color: #e2e8f0;
  border-color: #334155 !important;
  box-shadow: 0 -1px 3px rgba(0, 0, 0, 0.2);
}

.dark .page-tabs-bar :deep(.el-tabs__item:not(.is-active):hover) {
  background: rgba(148, 163, 184, 0.08);
  color: #cbd5e1;
}

/* ====== 标签页内部元素 ====== */
.page-tab-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.page-tab-label .tab-icon {
  width: 13px;
  height: 13px;
  color: #94a3b8;
  flex-shrink: 0;
}

.page-tabs-bar :deep(.el-tabs__item.is-active) .page-tab-label .tab-icon {
  color: #3b82f6;
}

.page-tab-label .tab-name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.page-tab-label .tab-dirty-icon {
  width: 8px;
  height: 8px;
  color: #f59e0b;
  flex-shrink: 0;
}

/* ====== 状态信息栏 ====== */
.status-info-bar {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 0;
  padding: 0 12px;
  border-left: 1px solid #e2e8f0;
  height: 100%;
  white-space: nowrap;
  font-size: 11px;
  color: #64748b;
  font-variant-numeric: tabular-nums;
  user-select: none;
  letter-spacing: 0.01em;
}

.dark .status-info-bar {
  border-left-color: #334155;
  color: #94a3b8;
}

.status-info-bar .status-item {
  padding: 0 6px;
}

.status-info-bar .status-mouse {
  font-family: "SF Mono", "Cascadia Code", "Consolas", monospace;
  font-size: 10px;
  letter-spacing: 0.03em;
}

.status-info-bar .status-node-name {
  color: #1e40af;
  font-weight: 500;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dark .status-info-bar .status-node-name {
  color: #93c5fd;
}

.status-info-bar .status-hover {
  color: #3b82f6;
  font-weight: 500;
}

.dark .status-info-bar .status-hover {
  color: #60a5fa;
}

.status-info-bar .status-sep {
  color: #cbd5e1;
  padding: 0 1px;
  font-size: 10px;
}

.dark .status-info-bar .status-sep {
  color: #334155;
}

/* ====== 新增页面按钮 / 空状态 ====== */
.page-tabs-add-btn {
  width: 22px;
  height: 22px;
  border: 1px dashed #cbd5e1;
  border-radius: 5px;
  background: transparent;
  color: #94a3b8;
  padding: 0;
  transition: all 0.15s ease;
}

.page-tabs-add-btn:hover {
  border-color: #3b82f6;
  border-style: solid;
  color: #3b82f6;
  background: rgba(59, 130, 246, 0.06);
}

.page-tabs-empty {
  height: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #94a3b8;
  font-size: 12px;
  padding: 0 8px;
}

.page-tabs-bar :deep(.el-tabs__new-tab) {
  margin: 0 4px 0 2px;
  width: 22px;
  height: 22px;
  line-height: 20px;
  border-radius: 5px;
  border: 1px dashed #cbd5e1;
  color: #94a3b8;
  background: transparent;
  transition: all 0.15s ease;
}

.page-tabs-bar :deep(.el-tabs__new-tab:hover) {
  border-color: #3b82f6;
  border-style: solid;
  color: #3b82f6;
  background: rgba(59, 130, 246, 0.06);
}

/* 空页面提示 */
.empty-canvas-placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e7ed 100%);
}

.dark .empty-canvas-placeholder {
  background: linear-gradient(135deg, #1a1a1a 0%, #2a2a2a 100%);
}

.empty-content {
  text-align: center;
}

.empty-icon {
  width: 80px;
  height: 80px;
  color: #c0c4cc;
  margin-bottom: 20px;
}

.dark .empty-icon {
  color: #4a4a4a;
}

.empty-title {
  font-size: 20px;
  font-weight: 500;
  color: #606266;
  margin: 0 0 8px 0;
}

.dark .empty-title {
  color: #d1d5db;
}

.empty-desc {
  font-size: 14px;
  color: #909399;
  margin: 0 0 24px 0;
}
</style>
