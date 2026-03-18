<template>
  <div class="designer-layout">
    <!-- 顶部工具栏 -->
    <TopToolbar :page-name="pageName" :is-locked="isLocked" :is-dirty="isDirty" :view-presets="viewPresets"
      :active-view-key="activeViewKey" :canvas-width="canvasWidth" :canvas-height="canvasHeight"
      :is-custom-view="isCustomView" :can-undo="canUndoEnabled" :can-redo="canRedoEnabled"
      :can-move-layer="canMoveLayer" :zoom="zoom" :show-ruler="showRuler" :show-grid="showGrid"
      :enable-snap="enableSnap" :saving="saving" :save-settings="saveSettings"
      @update:activeViewKey="handleViewChange" @undo="handleUndo" @redo="handleRedo" @preview="handlePreview"
      @previewApp="handlePreviewApp" @save="handleSave" @export="handleExport" @toggleLock="handleToggleLock"
      @moveUp="handleLayerMoveUp" @moveDown="handleLayerMoveDown" @moveToTop="handleLayerMoveToTop"
      @moveToBottom="handleLayerMoveToBottom" @openCollaboration="handleOpenCollaboration"
      @refreshCanvas="handleRefreshCanvas" @toggleLocale="handleToggleLocale" @openAi="handleOpenAi"
      @toggleTheme="handleToggleTheme" @clearCanvas="handleClearCanvas" @applyCustomSize="handleApplyCustomSize"
      @saveSettingsChange="handleSaveSettingsChange" @zoomIn="handleZoomIn" @zoomOut="handleZoomOut"
      @fitCanvas="handleFitCanvas" @fitScreen="handleFitScreen" @toggleRuler="handleToggleRuler"
      @toggleGrid="handleToggleGrid" @toggleSnap="handleToggleSnap" />

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
            <component :is="leftPanelComponent" :key="`left-panel-${leftActiveKey}`" v-bind="leftPanelProps" ref="leftPanelRef"
              @update:drawingTool="(value) => (drawingTool.value = value)" />
          </DockPanel>

          <div ref="canvasHostRef" class="designer-canvas">
            <!-- 画布容器 -->
            <template v-if="hasPages">
              <CanvasContainer :width="canvasWidth" :height="canvasHeight" :zoom="zoom" :show-ruler="showRuler"
                :view-reset-token="viewResetToken"
                @zoomChange="handleZoomChange" />
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
              <component :is="leftPanelComponent" :key="`left-floating-panel-${leftActiveKey}`" v-bind="leftPanelProps" ref="leftPanelRef"
                @update:drawingTool="(value) => (drawingTool.value = value)" />
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
        </div>
      </div>

      <ToolRail side="right" :items="rightRailItems" :active-key="rightActiveKey" @select="handleRightSelect" />
    </div>
  </div>
</template>

<script setup>
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  provide,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { CanvasContainer } from "@/ui/Canvas";
import { TopToolbar } from "@/ui/TopToolbar";
import { ToolRail } from "@/ui/ToolRail";
import { DockPanel } from "@/ui/DockPanel";
import {
  PageTree,
  OutlineTree,
  MaterialPanel,
  DataPanel,
  I18nPanel,
  ScriptVarsPanel,
  RolePanel,
} from "@/ui/LeftPanel";
import {
  PropertyPanel,
  AdvancedPanel,
  VariablesPanel,
} from "@/ui/RightPanel";
import { VIEW_PRESETS } from "@/constants";
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

const editorStore = useEditorStore();
const {
  canUndo,
  canRedo,
  saving,
  selection,
  pages,
  currentPageId,
  currentPage,
  isLocked,
  readonlyState,
  pageTabState,
} = storeToRefs(editorStore);

const zoom = ref(1);
const showRuler = ref(true);
const viewResetToken = ref(0);
const activeViewKey = ref("pc");
const viewPresets = VIEW_PRESETS;
const SAVE_SETTINGS_STORAGE_KEY = "designer_save_settings";
const AUTO_FIT_PADDING = 48;
const saveSettings = ref({
  autoSave: false,
  intervalMinutes: 5,
});
const autoSaveTimer = ref(null);
const autoSaving = ref(false);
const autoZoomEnabled = ref(true);
const autoFitFrame = ref(0);
const canvasHostResizeObserver = ref(null);

const drawingTool = ref("");

// ==================== 页面标签页系统 ====================
/**
 * @typedef {Object} PageTab
 * @property {string} id - 页面ID
 * @property {string} name - 页面名称
 * @property {boolean} isDirty - 是否有未保存的修改
 */

/** @type {import('vue').Ref<PageTab[]>} */
const pageTabs = ref([]);
const activePageTabId = ref("");

if (pageTabState.value?.tabs?.length) {
  pageTabs.value = pageTabState.value.tabs.map((item) => ({ ...item }));
  activePageTabId.value = pageTabState.value.activeId || "";
}

/**
 * 打开页面标签页
 * @param {string} pageId - 页面ID
 */
const openPageTab = (pageId) => {
  const page = editorStore.pages.find((p) => p.id === pageId);
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

/**
 * 关闭页面标签页
 * @param {string} tabId - 标签页ID
 */
const handleClosePageTab = async (tabId) => {
  const index = pageTabs.value.findIndex((t) => t.id === tabId);
  if (index === -1) return;

  const tab = pageTabs.value[index];

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
        }
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
    const page = editorStore.pages.find((p) => p.id === currentPageId.value);
    tab.name = page?.name || "未命名页面";
  }
};

/**
 * 同步底部页面标签，清理已被删除的页面标签
 * @returns {void}
 */
const syncPageTabsWithPages = () => {
  const pageIdSet = new Set(editorStore.pages.map((page) => page.id));
  const nextTabs = pageTabs.value.filter((tab) => pageIdSet.has(tab.id));
  if (nextTabs.length !== pageTabs.value.length) {
    pageTabs.value = nextTabs;
  }
  if (activePageTabId.value && !pageIdSet.has(activePageTabId.value)) {
    activePageTabId.value = currentPageId.value && pageIdSet.has(currentPageId.value)
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
  { deep: true }
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
  { flush: "sync" }
);

watch(
  [pageTabs, activePageTabId],
  () => {
    editorStore.setPageTabState(pageTabs.value, activePageTabId.value);
  },
  { deep: true }
);

// 初始化时打开当前页面
watch(
  currentPageId,
  (newPageId, oldPageId) => {
    if (newPageId && !pageTabs.value.find((t) => t.id === newPageId)) {
      // ✅ 验证页面是否真的存在
      const pageExists = editorStore.pages.some((p) => p.id === newPageId);
      if (pageExists) {
        openPageTab(newPageId);
      }
    }
    if (newPageId && oldPageId && newPageId !== oldPageId) {
      activateMaterialPanel();
    }
  },
  { immediate: true }
);

// 提供 openPageTab 方法给子组件
provide("openPageTab", openPageTab);

/**
 * 是否有页面
 */
const hasPages = computed(() => {
  // ✅ 确保页面列表不为空且当前页面确实存在
  if (editorStore.pages.length === 0) return false;
  if (!currentPageId.value) return false;
  return editorStore.pages.some((p) => p.id === currentPageId.value);
});
// ==================== 页面标签页系统结束 ====================
const canUndoEnabled = computed(
  () => canUndo.value && !readonlyState.value?.readonly
);
const canRedoEnabled = computed(
  () => canRedo.value && !readonlyState.value?.readonly
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
const leftActiveKey = ref("pages");
const rightActiveKey = ref("props");
const leftFloating = ref(false);
const rightFloating = ref(false);
const leftPanelRef = ref(null);
const canvasHostRef = ref(null);

const pageName = computed(() => {
  // ✅ 如果没有页面，返回空
  if (!hasPages.value) return "";
  // 优先从 pages 列表获取名称（更可靠）
  const pageFromList = editorStore.pages.find(
    (p) => p.id === currentPageId.value
  );
  if (pageFromList?.name) return pageFromList.name;
  // 其次从 doc 中获取
  const page = currentPage.value;
  return page?.name || "";
});

const isDirty = computed(() => canUndo.value);

const currentPageSnapshot = computed(() => {
  const page = pages.value.find((item) => item.id === currentPageId.value);
  return page || currentPage.value || null;
});
const activeView = computed(() =>
  viewPresets.find((preset) => preset.key === activeViewKey.value)
);
const defaultViewPreset = computed(
  () => viewPresets.find((preset) => preset.key === "pc") || viewPresets[0] || { width: 1366, height: 768 }
);
const normalizeCanvasDimension = (value, fallback) => {
  const next = Number(value);
  return Number.isFinite(next) && next > 0 ? Math.round(next) : fallback;
};
const resolvedPageWidth = computed(() =>
  normalizeCanvasDimension(currentPageSnapshot.value?.config?.width, defaultViewPreset.value.width)
);
const resolvedPageHeight = computed(() =>
  normalizeCanvasDimension(currentPageSnapshot.value?.config?.height, defaultViewPreset.value.height)
);
const matchedViewPreset = computed(() =>
  viewPresets.find(
    (preset) =>
      preset.width === resolvedPageWidth.value && preset.height === resolvedPageHeight.value
  ) || null
);
const canvasWidth = computed(() =>
  normalizeCanvasDimension(currentPageSnapshot.value?.config?.width, defaultViewPreset.value.width)
);
const canvasHeight = computed(() =>
  normalizeCanvasDimension(currentPageSnapshot.value?.config?.height, defaultViewPreset.value.height)
);
const isCustomView = computed(() => !matchedViewPreset.value);
const showGrid = computed(() =>
  Boolean(currentPageSnapshot.value?.config?.showGrid)
);
const enableSnap = computed(
  () => currentPageSnapshot.value?.config?.enableSnap ?? true
);

const leftRailItems = [
  { key: "pages", label: "页面", icon: IconLucideFileText },
  { key: "outline", label: "大纲", icon: IconLucideList },
  { key: "material", label: "物料", icon: IconLucideBox },
  { key: "data", label: "数据", icon: IconLucideDatabase },
  { key: "i18n", label: "国际", icon: IconLucideLanguages },
  { key: "script", label: "脚本", icon: IconLucideFileCode },
  { key: "role", label: "角色", icon: IconLucideUsers, placement: "bottom" },
];

const rightRailItems = [
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
    (entry) => entry.key === rightActiveKey.value
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
  const projectId = route.meta.project?.id;
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

/**
 * 切换视图
 * @param {string} key - 视图键值
 */
const handleViewChange = (key) => {
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

/**
 * 应用自定义画布尺寸
 * @param {{width:number,height:number}} size - 自定义尺寸
 * @returns {void}
 */
const handleApplyCustomSize = ({ width, height }) => {
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
    currentPageId.value
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

const handleLeftSelect = (key) => {
  leftActiveKey.value = leftActiveKey.value === key ? "" : key;
};

const handleRightSelect = (key) => {
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

/**
 * 更新缩放比例
 * @param {number} value - 缩放比例
 */
const handleZoomChange = (value) => {
  autoZoomEnabled.value = false;
  zoom.value = value;
};

/**
 * 限制缩放范围
 * @param {number} value - 缩放值
 * @returns {number}
 */
const clampZoom = (value) => {
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
    availableHeight / canvasHeight.value
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

const getUniquePageName = (name) => {
  const base = (name || "导入页面").trim() || "导入页面";
  const existingNames = editorStore.pages.map((page) => page.name).filter(Boolean);
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
  input.onchange = async (event) => {
    const file = event.target.files?.[0];
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
      openPageTab?.(pageId);
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

/**
 * 自动保存当前页面
 * @returns {Promise<void>}
 */
const handleAutoSave = async () => {
  if (!saveSettings.value.autoSave) return;
  if (readonlyState.value?.readonly) return;
  if (!currentPageId.value || autoSaving.value || saving.value) return;
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

/**
 * 根据当前设置更新自动保存定时器
 * @returns {void}
 */
const syncAutoSaveTimer = () => {
  clearAutoSaveTimer();
  if (!saveSettings.value.autoSave) return;
  const interval = Number(saveSettings.value.intervalMinutes) || 5;
  autoSaveTimer.value = setInterval(() => {
    void handleAutoSave();
  }, interval * 60 * 1000);
};

/**
 * 应用并持久化保存设置
 * @param {{autoSave?: boolean, intervalMinutes?: number}} settings - 保存设置
 * @returns {void}
 */
const handleSaveSettingsChange = (settings) => {
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
  [currentPageId, () => currentPageSnapshot.value?.config?.width, () => currentPageSnapshot.value?.config?.height],
  () => {
    activeViewKey.value = matchedViewPreset.value?.key || "custom";
    autoZoomEnabled.value = true;
    nextTick(() => {
      scheduleAutoFit();
    });
  },
  { immediate: true }
);

/**
 * 加载工程数据
 */
const loadProject = async () => {
  const project = route.meta.project;
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
  const cached = Storage.get(SAVE_SETTINGS_STORAGE_KEY, null);
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

/* 页面标签栏样式 */
.page-tabs-bar {
  width: 100%;
  min-width: 0;
  height: 100%;
  border: 0;
  background: transparent;
}

.dark .page-tabs-bar {
  background: transparent;
  border-color: transparent;
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
  color: #6b7280;
  background: #f3f6fa;
  z-index: 2;
}

.page-tabs-bar :deep(.el-tabs__nav-prev:hover),
.page-tabs-bar :deep(.el-tabs__nav-next:hover) {
  color: #2563eb;
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
}

.page-tabs-bar :deep(.el-tabs__item) {
  height: 30px;
  line-height: 30px;
  font-size: 12px;
  border: 1px solid transparent !important;
  background: #eef2f7;
  color: #6b7280;
  padding: 0 14px;
  border-radius: 8px 8px 0 0;
  margin-right: 6px;
  transition: all 0.15s ease;
}

.page-tabs-bar :deep(.el-tabs__item.is-active) {
  background: #ffffff;
  color: #1f2937;
  border-color: #d5deea !important;
  border-bottom-color: #ffffff !important;
  box-shadow: 0 -1px 0 #ffffff inset;
}

.page-tabs-bar :deep(.el-tabs__item:hover) {
  color: #374151;
  background: #e8edf5;
}

.dark .page-tabs-bar :deep(.el-tabs__item.is-active) {
  background: #2a2a2a;
}

.page-tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
}

.page-tab-label .tab-icon {
  width: 14px;
  height: 14px;
  color: #909399;
}

.page-tab-label .tab-name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.page-tab-label .tab-dirty-icon {
  width: 12px;
  height: 12px;
  color: #e6a23c;
}

.designer-bottom-toolbar {
  flex-shrink: 0;
  border-top: 1px solid #dfe6ef;
  background: #f3f6fa;
  display: flex;
  align-items: center;

}

.dark .designer-bottom-toolbar {
  background: #0f172a;
  border-top-color: #374151;
}

.designer-bottom-toolbar .page-tabs-bar {
  width: 100%;
  min-width: 0;
}

.page-tabs-add-btn {
  width: 26px;
  height: 26px;
  border: 1px solid #d1d9e5;
  border-radius: 7px;
  background: #ffffff;
  color: #4b5563;
  padding: 0;
}

.page-tabs-add-btn:hover {
  border-color: #bfd2ee;
  color: #2563eb;
  background: #f8fbff;
}

.page-tabs-empty {
  height: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #9ca3af;
  font-size: 12px;
  padding: 0 6px;
}

.page-tabs-bar :deep(.el-tabs__new-tab) {
  margin: 0 4px 0 2px;
  width: 26px;
  height: 26px;
  line-height: 24px;
  border-radius: 7px;
  border: 1px solid #d1d9e5;
  color: #4b5563;
  background: #ffffff;
}

.page-tabs-bar :deep(.el-tabs__new-tab:hover) {
  border-color: #bfd2ee;
  color: #2563eb;
  background: #f8fbff;
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
