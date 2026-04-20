<!--
  DesignerView - 设计器主视图
  布局：顶部工具栏 + 左侧工具轨 + 左侧面板（可停靠/浮动）+ 画布区 + 右侧面板
  功能：页面管理、画布编辑、属性面板、预览、保存、导出等
-->
<script setup lang="ts">
import type { Ref } from "vue";
import type {
  DesignerPageTab,
  DesignerRouteProjectMeta,
  DesignerStorePageRow,
} from "./designer-view-types";
import type { ToolRailItem } from "./tool-rail-types";
import type { DesignerPageTabsStore } from "./use-designer-page-tabs";
import type { ViewPreset } from "@/constants";
import type { PageConfig } from "@/editor-core/document/types";
import type { CreatePageForStoreResult } from "@/stores/editor-store.types";
import { storeToRefs } from "pinia";
import {
  computed,
  defineAsyncComponent,
  nextTick,
  onBeforeUnmount,
  onMounted,
  provide,
  ref,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import IconEpDocument from "~icons/ep/document";
import IconEpPlus from "~icons/ep/plus";
import IconEpUpload from "~icons/ep/upload";
import IconEpWarning from "~icons/ep/warning";
import IconLucideBox from "~icons/lucide/box";
import IconLucideBraces from "~icons/lucide/braces";
import IconLucideDatabase from "~icons/lucide/database";
import IconLucideFileCode from "~icons/lucide/file-code";
import IconLucideFileText from "~icons/lucide/file-text";
import IconLucideLanguages from "~icons/lucide/languages";
import IconLucideList from "~icons/lucide/list";
import IconLucideSettings from "~icons/lucide/settings";
import IconLucideSlidersHorizontal from "~icons/lucide/sliders-horizontal";
import IconLucideUsers from "~icons/lucide/users";
import { VIEW_PRESETS } from "@/constants";
import { useEditorStore } from "@/stores/editor-store";
import { getEditorUiStore } from "@/stores/editor-ui-store";
import { CanvasContainer } from "@/ui/editors/page/canvas";
import SelectionToolbar from "@/ui/editors/page/canvas/SelectionToolbar.vue";
import { MaterialPanel, OutlineTree } from "@/ui/editors/page/sidebar-panels/left";
import { I18nPanel, RolePanel } from "@/ui/shared/tool-panels";
import { DockPanel } from "@/ui/shell/DockPanel";
import { ToolRail } from "@/ui/shell/ToolRail";
import { TopToolbar } from "@/ui/shell/TopToolbar";
import { DESIGNER_DEFAULT_PAGE_CONFIG_DIMS } from "./designer-view-types";
import { ElMessage } from "./el-message-compat";
import { useDesignerAutoSave } from "./use-designer-auto-save";
import { useDesignerPageTabs } from "./use-designer-page-tabs";

/** 重型面板异步加载，减轻 DesignerView 首 chunk */
const DataPanel = defineAsyncComponent(() => import("@/ui/editors/page/sidebar-panels/left/DataPanel.vue"));
const AdvancedPanel = defineAsyncComponent(
  () => import("@/ui/editors/page/sidebar-panels/right/AdvancedPanel.vue"),
);
const PropertyPanel = defineAsyncComponent(
  () => import("@/ui/editors/page/sidebar-panels/right/PropertyPanel.vue"),
);
const PageTree = defineAsyncComponent(() => import("@/ui/shared/tool-panels/page-tree/PageTree.vue"));
const ScriptVarsPanel = defineAsyncComponent(
  () => import("@/ui/shared/tool-panels/ScriptVarsPanel.vue"),
);
const VariablesPanel = defineAsyncComponent(() => import("@/ui/shared/tool-panels/VariablesPanel.vue"));

const route = useRoute();
const router = useRouter();
const editorUi = getEditorUiStore();
const { t } = useI18n();

function mergePageConfig(
  base: DesignerStorePageRow["config"] | undefined,
  patch: Partial<PageConfig>,
): PageConfig {
  return {
    ...DESIGNER_DEFAULT_PAGE_CONFIG_DIMS,
    ...(base ?? {}),
    ...patch,
  } as PageConfig;
}

/** storeToRefs 会把部分 ref 标成可能 undefined，此处收窄为壳层实际用到的形状 */
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
  pages: Ref<DesignerStorePageRow[]>;
  currentPageId: Ref<string>;
  currentPage: Ref<DesignerStorePageRow | null | undefined>;
  isLocked: Ref<boolean>;
  readonlyState: Ref<{ readonly?: boolean } | undefined>;
  pageTabState: Ref<{ tabs?: DesignerPageTab[]; activeId?: string } | undefined>;
  canvasMousePos: Ref<{ x: number; y: number } | null>;
  hoveredNodeType: Ref<string>;
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
  canvasMousePos,
  hoveredNodeType,
} = storeToRefs(editorStore) as unknown as EditorShellStoreRefs;

const leftActiveKey = ref("pages");

const zoom = ref(1);
const showRuler = ref(true);
const viewResetToken = ref(0);
const activeViewKey = ref("pc");
const viewPresets: readonly ViewPreset[] = VIEW_PRESETS;
const AUTO_FIT_PADDING = 48;
const autoZoomEnabled = ref(true);
const autoFitFrame = ref(0);
const canvasHostResizeObserver = ref<ResizeObserver | null>(null);

const drawingTool = ref("");

function setDrawingTool(value: string) {
  drawingTool.value = value;
}

const { pageTabs, activePageTabId, openPageTab, handleClosePageTab } = useDesignerPageTabs({
  editorStore: editorStore as unknown as DesignerPageTabsStore,
  pageTabState,
  currentPageId,
  canUndo,
  leftActiveKey,
});

const { saveSettings, handleSaveSettingsChange } = useDesignerAutoSave({
  editorStore,
  pageTabs,
  currentPageId,
  readonlyState,
  isSaving,
});

provide("openPageTab", openPageTab);

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
  const rootEl = rootId ? document.querySelector(`[data-node-id="${rootId}"]`) : null;
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
  return editorStore.pages.some((p: DesignerStorePageRow) => p.id === currentPageId.value);
});
// ==================== 页面标签页系统结束 ====================
const canUndoEnabled = computed(() => canUndo.value && !readonlyState.value?.readonly);
const canRedoEnabled = computed(() => canRedo.value && !readonlyState.value?.readonly);

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
const rightActiveKey = ref("props");
const leftFloating = ref(false);
const rightFloating = ref(false);
const DEFAULT_DOCK_PANEL_WIDTH = 272;
const MIN_DOCK_PANEL_WIDTH = 220;
const MAX_DOCK_PANEL_WIDTH = 520;
const DOCK_PANEL_WIDTH_STORAGE_KEY = "designer:dock-panel-width:v1";
const leftPanelWidth = ref(DEFAULT_DOCK_PANEL_WIDTH);
const rightPanelWidth = ref(DEFAULT_DOCK_PANEL_WIDTH);
const leftPanelRef = ref<{ openCreateDialog?: () => void } | null>(null);
const canvasHostRef = ref<HTMLElement | null>(null);

const pageName = computed(() => {
  // 如果没有页面，返回空
  if (!hasPages.value) return "";
  // 优先从 pages 列表获取名称（更可靠）
  const pageFromList = editorStore.pages.find(
    (p: DesignerStorePageRow) => p.id === currentPageId.value,
  );
  if (pageFromList?.name) return pageFromList.name;
  // 其次从 doc 中获取
  const page = currentPage.value;
  return page?.name || "";
});

const isDirty = computed(() => canUndo.value);

const currentPageSnapshot = computed(() => {
  const page = pages.value.find((item: DesignerStorePageRow) => item.id === currentPageId.value);
  return page || currentPage.value || null;
});
const defaultViewPreset = computed(
  () =>
    viewPresets.find((preset) => preset.key === "pc") ||
    viewPresets[0] || { width: 1366, height: 768 },
);
function normalizeCanvasDimension(value: unknown, fallback: number): number {
  const next = Number(value);
  return Number.isFinite(next) && next > 0 ? Math.round(next) : fallback;
}
const resolvedPageWidth = computed(() =>
  normalizeCanvasDimension(currentPageSnapshot.value?.config?.width, defaultViewPreset.value.width),
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
        preset.width === resolvedPageWidth.value && preset.height === resolvedPageHeight.value,
    ) || null,
);
const canvasWidth = computed(() =>
  normalizeCanvasDimension(currentPageSnapshot.value?.config?.width, defaultViewPreset.value.width),
);
const canvasHeight = computed(() =>
  normalizeCanvasDimension(
    currentPageSnapshot.value?.config?.height,
    defaultViewPreset.value.height,
  ),
);
const isCustomView = computed(() => !matchedViewPreset.value);
const showGrid = computed(() => Boolean(currentPageSnapshot.value?.config?.showGrid));
const enableSnap = computed(() => currentPageSnapshot.value?.config?.enableSnap ?? true);

const leftRailItems = computed<ToolRailItem[]>(() => [
  { key: "pages", label: t("shell.pages"), icon: IconLucideFileText },
  { key: "outline", label: t("shell.outline"), icon: IconLucideList },
  { key: "material", label: t("shell.material"), icon: IconLucideBox },
  { key: "data", label: t("shell.data"), icon: IconLucideDatabase },
  { key: "i18n", label: t("shell.i18n"), icon: IconLucideLanguages },
  { key: "script", label: t("shell.script"), icon: IconLucideFileCode },
  { key: "role", label: t("shell.role"), icon: IconLucideUsers, placement: "bottom" },
]);

const rightRailItems = computed<ToolRailItem[]>(() => [
  { key: "props", label: t("shell.props"), icon: IconLucideSlidersHorizontal },
  { key: "advanced", label: t("shell.advanced"), icon: IconLucideSettings },
  { key: "variables", label: t("shell.variables"), icon: IconLucideBraces },
]);

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
  const item = leftRailItems.value.find((entry) => entry.key === leftActiveKey.value);
  return item?.label || t("shell.panel");
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
  const item = rightRailItems.value.find((entry) => entry.key === rightActiveKey.value);
  return item?.label || t("shell.panel");
});

/**
 * 左侧停靠面板当前占用宽度（关闭或悬浮时不占位）。
 */
const leftDockWidth = computed(() => (leftActiveKey.value && !leftFloating.value ? leftPanelWidth.value : 0));

/**
 * 右侧停靠面板当前占用宽度（关闭或悬浮时不占位）。
 */
const rightDockWidth = computed(() =>
  rightActiveKey.value && !rightFloating.value ? rightPanelWidth.value : 0,
);

/**
 * 布局 CSS 变量：用于顶部工具栏中间区与面板宽度同步。
 */
const layoutStyleVars = computed(() => ({
  "--designer-left-panel-width": `${leftDockWidth.value}px`,
  "--designer-right-panel-width": `${rightDockWidth.value}px`,
}));

/**
 * 约束面板宽度，避免拖拽过窄或过宽。
 * @param {number} width - 目标宽度
 * @returns {number}
 */
function clampPanelWidth(width: number): number {
  const next = Number(width);
  if (!Number.isFinite(next)) return DEFAULT_DOCK_PANEL_WIDTH;
  return Math.min(MAX_DOCK_PANEL_WIDTH, Math.max(MIN_DOCK_PANEL_WIDTH, Math.round(next)));
}

/**
 * 左侧面板宽度变更。
 * @param {number} width - 目标宽度
 */
function handleLeftPanelResize(width: number): void {
  leftPanelWidth.value = clampPanelWidth(width);
}

/**
 * 右侧面板宽度变更。
 * @param {number} width - 目标宽度
 */
function handleRightPanelResize(width: number): void {
  rightPanelWidth.value = clampPanelWidth(width);
}

/**
 * 从本地存储恢复左右面板宽度。
 */
function restoreDockPanelWidths(): void {
  if (typeof window === "undefined") return;
  try {
    const raw = window.localStorage.getItem(DOCK_PANEL_WIDTH_STORAGE_KEY);
    if (!raw) return;
    const parsed = JSON.parse(raw) as { left?: number; right?: number } | null;
    if (!parsed || typeof parsed !== "object") return;
    if (parsed.left !== undefined) {
      leftPanelWidth.value = clampPanelWidth(parsed.left);
    }
    if (parsed.right !== undefined) {
      rightPanelWidth.value = clampPanelWidth(parsed.right);
    }
  } catch {
    // 本地数据异常时忽略，保持默认宽度
  }
}

/**
 * 持久化左右面板宽度到本地存储。
 */
function persistDockPanelWidths(): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(
      DOCK_PANEL_WIDTH_STORAGE_KEY,
      JSON.stringify({
        left: leftPanelWidth.value,
        right: rightPanelWidth.value,
      }),
    );
  } catch {
    // 存储失败时静默，不影响编辑器使用
  }
}

/**
 * 撤销操作
 */
function handleUndo() {
  if (!editorStore.undo()) {
    ElMessage.info(t("message.noUndo"));
  }
}

/**
 * 重做操作
 */
function handleRedo() {
  if (!editorStore.redo()) {
    ElMessage.info(t("message.noRedo"));
  }
}

/**
 * 预览
 */
function handlePreview() {
  const projectId = (route.meta as DesignerRouteProjectMeta).project?.id;
  router.push({
    path: "/preview",
    query: { pid: projectId, pageId: currentPageId.value || "" },
  });
}

/**
 * 应用预览（占位）
 */
function handlePreviewApp() {
  ElMessage.info(t("message.appPreviewWip"));
}

/**
 * 保存
 */
async function handleSave() {
  try {
    await editorStore.saveCurrentPage();

    // 保存成功后清除当前标签页的脏状态
    const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
    if (tab) {
      tab.isDirty = false;
    }

    ElMessage.success(t("message.saveSuccess"));
  } catch (err) {
    const message = err instanceof Error ? err.message : t("message.unknownError");
    ElMessage.error(t("message.saveFailed", { message }) as string);
  }
}

function handleViewChange(key: string) {
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
}

function handleApplyCustomSize({ width, height }: { width: number; height: number }) {
  const page = currentPageSnapshot.value;
  if (!page) return;
  const nextConfig = mergePageConfig(page.config, {
    width: Math.round(width),
    height: Math.round(height),
  });
  activeViewKey.value = "custom";
  editorStore.updateCurrentPage({ config: nextConfig });
}

/**
 * 切换锁定状态
 */
async function handleToggleLock() {
  const result = await editorStore.togglePageLock();
  if (!result) return;

  if (result.success) {
    const message = result.action === "release" ? t("message.pageUnlocked") : t("message.pageLocked");
    ElMessage.success(message);
    return;
  }

  if (result.reason === "locked") {
    ElMessage.warning(
      t("message.pageLockedBy", { user: result.lockedByName || t("message.otherUser") }) as string,
    );
    return;
  }

  const err = result.error;
  ElMessage.error(err instanceof Error ? err.message : String(err ?? t("message.pageLockFailed")));
}

/**
 * 导出页面 Schema
 */
function handleExport() {
  if (!editorStore.doc || !currentPageId.value) {
    ElMessage.warning(t("message.noExportablePage"));
    return;
  }
  const payload = editorStore.serializer.exportPage(editorStore.doc, currentPageId.value);
  const json = JSON.stringify(payload, null, 2);
  const blob = new Blob([json], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const name = `${pageName.value || "page"}.json`;
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
  ElMessage.success(t("message.pageExported"));
}

function handleLeftSelect(key: string) {
  leftActiveKey.value = leftActiveKey.value === key ? "" : key;
}

function handleRightSelect(key: string) {
  rightActiveKey.value = rightActiveKey.value === key ? "" : key;
}

function handleLeftClose() {
  leftActiveKey.value = "";
}

function handleRightClose() {
  rightActiveKey.value = "";
}

function toggleLeftFloating() {
  leftFloating.value = !leftFloating.value;
}

function toggleRightFloating() {
  rightFloating.value = !rightFloating.value;
}

function handleZoomChange(value: number) {
  autoZoomEnabled.value = false;
  zoom.value = value;
}

function clampZoom(value: number): number {
  const next = Number.isFinite(value) ? value : 1;
  return Math.min(5, Math.max(0.1, Number(next.toFixed(2))));
}

/**
 * 工具栏：缩小
 * @returns {void}
 */
function handleZoomOut() {
  autoZoomEnabled.value = false;
  zoom.value = clampZoom(zoom.value - 0.1);
}

/**
 * 工具栏：放大
 * @returns {void}
 */
function handleZoomIn() {
  autoZoomEnabled.value = false;
  zoom.value = clampZoom(zoom.value + 0.1);
}

/**
 * 工具栏：适配画布（100%）
 * @returns {void}
 */
function handleFitCanvas() {
  autoZoomEnabled.value = false;
  zoom.value = 1;
  viewResetToken.value += 1;
}

/**
 * 计算适配当前工作区的推荐缩放比例
 * 规则与参考页保持一致：能 100% 展示时保持 100%，不足时自动缩放到刚好适配
 * @returns {number}
 */
function getRecommendedZoom() {
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
}

/**
 * 应用推荐缩放比例
 * @returns {void}
 */
function applyRecommendedZoom() {
  const nextZoom = getRecommendedZoom();
  if (Math.abs(nextZoom - zoom.value) < 0.001) {
    if (viewResetToken.value === 0) {
      viewResetToken.value += 1;
    }
    return;
  }
  zoom.value = nextZoom;
  viewResetToken.value += 1;
}

/**
 * 在布局稳定后重新计算自动缩放
 * @returns {void}
 */
function scheduleAutoFit() {
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
}

/**
 * 工具栏：适配屏幕
 * @returns {void}
 */
function handleFitScreen() {
  autoZoomEnabled.value = true;
  applyRecommendedZoom();
}

/**
 * 切换标尺显示
 * @returns {void}
 */
function handleToggleRuler() {
  showRuler.value = !showRuler.value;
}

/**
 * 切换网格显示
 * @returns {void}
 */
function handleToggleGrid() {
  const page = currentPageSnapshot.value;
  if (!page) return;
  const nextConfig = mergePageConfig(page.config, {
    showGrid: !showGrid.value,
  });
  editorStore.updateCurrentPage({ config: nextConfig });
}

/**
 * 切换吸附开关
 * @returns {void}
 */
function handleToggleSnap() {
  const page = currentPageSnapshot.value;
  if (!page) return;
  const nextConfig = mergePageConfig(page.config, {
    enableSnap: !enableSnap.value,
  });
  editorStore.updateCurrentPage({ config: nextConfig });
}

/**
 * 打开页面新建弹窗
 * @returns {void}
 */
async function handlePageCreate() {
  if (leftActiveKey.value !== "pages") {
    leftActiveKey.value = "pages";
    leftFloating.value = false;
    await nextTick();
  }
  leftPanelRef.value?.openCreateDialog?.();
}

function getUniquePageName(name: string | undefined) {
  const base = (name || t("shell.importPage")).trim() || t("shell.importPage");
  const existingNames = editorStore.pages
    .map((page: DesignerStorePageRow) => page.name)
    .filter(Boolean) as string[];
  if (!existingNames.includes(base)) return base;
  let index = 1;
  let next = `${base}_${index}`;
  while (existingNames.includes(next)) {
    index += 1;
    next = `${base}_${index}`;
  }
  return next;
}

function handlePageImport() {
  if (!editorStore.doc) {
    ElMessage.warning(t("message.noImportablePage"));
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
        ElMessage.error(t("message.invalidPageData"));
        return;
      }
      const serializer = editorStore.serializer;
      const docModel = editorStore.doc;
      if (!docModel) {
        ElMessage.error(t("message.importFailed"));
        return;
      }
      const baseSchema = serializer.exportToSchema(docModel);
      const tempDoc = serializer.importFromSchema(baseSchema);
      const tempPageId = serializer.importPage(tempDoc, payload, {
        generateNewIds: true,
      });
      const imported = serializer.exportPage(tempDoc, tempPageId);
      const uniqueName = getUniquePageName(payload.page?.name);
      imported.page.name = uniqueName;

      const importedPage = imported.page as { type?: string };
      const result: CreatePageForStoreResult = await editorStore.createPage({
        name: uniqueName,
        type: importedPage.type || "page",
        parentId: null,
      });
      const pageId = result?.id || result?.page?.id;
      if (!pageId) {
        ElMessage.error(t("message.importFailed"));
        return;
      }
      imported.page.id = pageId;
      await editorStore.updatePageSchema(pageId, imported);
      await editorStore.loadPage(pageId);
      openPageTab(pageId);
      ElMessage.success(t("message.pageImported"));
    } catch {
      ElMessage.error(t("message.importFailed"));
    }
  };
  input.click();
}

/**
 * 工具栏:上移图层
 */
function handleLayerMoveUp() {
  if (editorStore.moveNodeUp()) {
    ElMessage.success(t("message.movedUp"));
  }
}

/**
 * 工具栏:下移图层
 */
function handleLayerMoveDown() {
  if (editorStore.moveNodeDown()) {
    ElMessage.success(t("message.movedDown"));
  }
}

/**
 * 工具栏:置顶
 */
function handleLayerMoveToTop() {
  if (editorStore.moveNodeToTop()) {
    ElMessage.success(t("message.movedToTop"));
  }
}

/**
 * 工具栏:置底
 */
function handleLayerMoveToBottom() {
  if (editorStore.moveNodeToBottom()) {
    ElMessage.success(t("message.movedToBottom"));
  }
}
function handleCopy() {
  if (editorStore.copyNodes()) {
    ElMessage.success(t("message.copied"));
  }
}
const handlePaste = () => editorStore.pasteNodes();
const handleDeleteSelected = () => editorStore.removeSelectedNodes();

/**
 * 更多设置：多人协作（占位）
 */
function handleOpenCollaboration() {
  ElMessage.info(t("message.collaborationWip"));
}

/**
 * 工具栏：AI 助手（占位）
 */
function handleOpenAi() {
  ElMessage.info(t("message.aiAssistantWip"));
}

/**
 * 工具栏：主题切换（占位）
 */
function handleToggleTheme() {
  editorUi.setTheme(editorUi.theme.value === "dark" ? "light" : "dark");
}

/**
 * 更多设置：刷新画布（占位）
 */
function handleRefreshCanvas() {
  ElMessage.info(t("message.refreshCanvasWip"));
}

/**
 * 更多设置：中英文切换（占位）
 */
function handleToggleLocale() {
  editorUi.setLocale(editorUi.locale.value === "zh" ? "en" : "zh");
}

/**
 * 工具栏：清除当前界面（占位）
 */
function handleClearCanvas() {
  ElMessage.info(t("message.clearCanvasWip"));
}

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

watch([leftPanelWidth, rightPanelWidth], () => {
  persistDockPanelWidths();
});

/**
 * 加载工程数据
 */
async function loadProject() {
  const project = (route.meta as DesignerRouteProjectMeta).project;
  if (!project?.id) return;
  if (editorStore.projectId === project.id && editorStore.doc) {
    return;
  }
  const result = await editorStore.loadProject(project.id);
  if (!result.ok) {
    ElMessage.error(result.error?.message || t("message.loadProjectFailed"));
    return;
  }
  const targetPageId = String(route.query.pageId || "");
  if (targetPageId) {
    await editorStore.setCurrentPage(targetPageId);
  }
}

onMounted(() => {
  restoreDockPanelWidths();
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
  void editorStore.releasePageLock();
});
</script>

<template>
  <div class="designer-layout" :style="layoutStyleVars">
    <!-- 顶部工具栏 -->
    <TopToolbar
      :page-name="pageName"
      :is-locked="isLocked"
      :is-dirty="isDirty"
      :view-presets="viewPresets"
      :active-view-key="activeViewKey"
      :canvas-width="canvasWidth"
      :canvas-height="canvasHeight"
      :is-custom-view="isCustomView"
      :can-undo="canUndoEnabled"
      :can-redo="canRedoEnabled"
      :can-move-layer="canMoveLayer"
      :zoom="zoom"
      :show-ruler="showRuler"
      :show-grid="showGrid"
      :enable-snap="enableSnap"
      :is-saving="isSaving"
      :save-settings="saveSettings"
      :has-selection="hasSelection"
      :has-clipboard="hasClipboard"
      @update:active-view-key="handleViewChange"
      @undo="handleUndo"
      @redo="handleRedo"
      @preview="handlePreview"
      @preview-app="handlePreviewApp"
      @save="handleSave"
      @export="handleExport"
      @toggle-lock="handleToggleLock"
      @move-up="handleLayerMoveUp"
      @move-down="handleLayerMoveDown"
      @move-to-top="handleLayerMoveToTop"
      @move-to-bottom="handleLayerMoveToBottom"
      @open-collaboration="handleOpenCollaboration"
      @refresh-canvas="handleRefreshCanvas"
      @toggle-locale="handleToggleLocale"
      @open-ai="handleOpenAi"
      @toggle-theme="handleToggleTheme"
      @clear-canvas="handleClearCanvas"
      @apply-custom-size="handleApplyCustomSize"
      @save-settings-change="handleSaveSettingsChange"
      @zoom-in="handleZoomIn"
      @zoom-out="handleZoomOut"
      @fit-canvas="handleFitCanvas"
      @fit-screen="handleFitScreen"
      @toggle-ruler="handleToggleRuler"
      @toggle-grid="handleToggleGrid"
      @toggle-snap="handleToggleSnap"
      @copy="handleCopy"
      @paste="handlePaste"
      @delete-selected="handleDeleteSelected"
    />

    <SelectionToolbar v-if="hasSelection && hasPages" />

    <!-- 主体区域 -->
    <div class="designer-main">
      <ToolRail
        side="left"
        :items="leftRailItems"
        :active-key="leftActiveKey"
        @select="handleLeftSelect"
      />

      <div class="designer-workspace">
        <div class="designer-workspace-main">
          <DockPanel
            v-if="leftActiveKey && !leftFloating"
            side="left"
            :title="leftPanelTitle"
            :floating="leftFloating"
            :width="leftPanelWidth"
            :min-width="MIN_DOCK_PANEL_WIDTH"
            :max-width="MAX_DOCK_PANEL_WIDTH"
            @close="handleLeftClose"
            @toggle-floating="toggleLeftFloating"
            @resize="handleLeftPanelResize"
          >
            <template #actions>
              <el-tooltip v-if="leftActiveKey === 'pages'" :content="t('shell.newPage')">
                <el-button size="small" text @click="handlePageCreate">
                  <IconEpPlus />
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="leftActiveKey === 'pages'" :content="t('shell.importPage')">
                <el-button size="small" text @click="handlePageImport">
                  <IconEpUpload />
                </el-button>
              </el-tooltip>
            </template>
            <component
              :is="leftPanelComponent"
              :key="`left-panel-${leftActiveKey}`"
              v-bind="leftPanelProps"
              ref="leftPanelRef"
              @update:drawing-tool="setDrawingTool"
            />
          </DockPanel>

          <div ref="canvasHostRef" class="designer-canvas">
            <!-- 画布容器 -->
            <template v-if="hasPages">
              <CanvasContainer
                :width="canvasWidth"
                :height="canvasHeight"
                :zoom="zoom"
                :show-ruler="showRuler"
                :view-reset-token="viewResetToken"
                @zoom-change="handleZoomChange"
              />
            </template>

            <!-- 空页面提示 -->
            <div v-else class="empty-canvas-placeholder">
              <div class="empty-content">
                <IconEpDocument class="empty-icon" />
                <h3 class="empty-title">{{ t("shell.noPages") }}</h3>
                <p class="empty-desc">{{ t("shell.createPageHint") }}</p>
                <el-button type="primary" @click="handlePageCreate">
                  <IconEpPlus class="mr-1" />
                  {{ t("shell.newPage") }}
                </el-button>
              </div>
            </div>

            <DockPanel
              v-if="leftActiveKey && leftFloating"
              side="left"
              :title="leftPanelTitle"
              :floating="leftFloating"
              :width="leftPanelWidth"
              :min-width="MIN_DOCK_PANEL_WIDTH"
              :max-width="MAX_DOCK_PANEL_WIDTH"
              @close="handleLeftClose"
              @toggle-floating="toggleLeftFloating"
              @resize="handleLeftPanelResize"
            >
              <template #actions>
                <el-tooltip v-if="leftActiveKey === 'pages'" :content="t('shell.newPage')">
                  <el-button size="small" text @click="handlePageCreate">
                    <IconEpPlus />
                  </el-button>
                </el-tooltip>
                <el-tooltip v-if="leftActiveKey === 'pages'" :content="t('shell.importPage')">
                  <el-button size="small" text @click="handlePageImport">
                    <IconEpUpload />
                  </el-button>
                </el-tooltip>
              </template>
              <component
                :is="leftPanelComponent"
                :key="`left-floating-panel-${leftActiveKey}`"
                v-bind="leftPanelProps"
                ref="leftPanelRef"
                @update:drawing-tool="setDrawingTool"
              />
            </DockPanel>

            <DockPanel
              v-if="rightActiveKey && rightFloating"
              side="right"
              :title="rightPanelTitle"
              :floating="rightFloating"
              :width="rightPanelWidth"
              :min-width="MIN_DOCK_PANEL_WIDTH"
              :max-width="MAX_DOCK_PANEL_WIDTH"
              @close="handleRightClose"
              @toggle-floating="toggleRightFloating"
              @resize="handleRightPanelResize"
            >
              <component :is="rightPanelComponent" />
            </DockPanel>
          </div>

          <DockPanel
            v-if="rightActiveKey && !rightFloating"
            side="right"
            :title="rightPanelTitle"
            :floating="rightFloating"
            :width="rightPanelWidth"
            :min-width="MIN_DOCK_PANEL_WIDTH"
            :max-width="MAX_DOCK_PANEL_WIDTH"
            @close="handleRightClose"
            @toggle-floating="toggleRightFloating"
            @resize="handleRightPanelResize"
          >
            <component :is="rightPanelComponent" />
          </DockPanel>
        </div>

        <div class="designer-bottom-toolbar">
          <div class="page-tabs-bar">
            <el-tabs
              v-if="pageTabs.length > 0"
              v-model="activePageTabId"
              type="card"
              closable
              addable
              @tab-remove="handleClosePageTab"
              @tab-add="handlePageCreate"
            >
              <el-tab-pane v-for="tab in pageTabs" :key="tab.id" :name="tab.id" closable>
                <template #label>
                  <span class="page-tab-label">
                    <IconEpDocument class="tab-icon" />
                    <span class="tab-name">{{ tab.name }}</span>
                    <IconEpWarning
                      v-if="tab.isDirty"
                      class="tab-dirty-icon"
                      :title="t('toolbar.saveStatus.dirty')"
                    />
                  </span>
                </template>
              </el-tab-pane>
            </el-tabs>
            <div v-else class="page-tabs-empty">
              <span>{{ t("shell.noPages") }}</span>
              <el-button class="page-tabs-add-btn" text @click="handlePageCreate">
                <IconEpPlus />
              </el-button>
            </div>
          </div>
          <!-- 底部右侧状态信息区 -->
          <div class="status-info-bar">
            <span class="status-item status-mouse">
              {{
                canvasMousePos
                  ? t("shell.mousePosition", {
                      x: Math.round(canvasMousePos.x),
                      y: Math.round(canvasMousePos.y),
                    })
                  : t("shell.mousePositionEmpty")
              }}
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
                {{ t("shell.nodePosition", { x: selectedNodePos.x, y: selectedNodePos.y }) }}
              </span>
              <span class="status-sep">|</span>
            </template>
            <template v-if="selectedNodeSize">
              <span class="status-item status-mouse">
                {{ t("shell.nodeSize", { w: selectedNodeSize.w, h: selectedNodeSize.h }) }}
              </span>
              <span class="status-sep">|</span>
            </template>
            <span class="status-item">{{ t("shell.selectedCount", { count: selectionCount }) }}</span>
            <span class="status-sep">|</span>
            <span class="status-item">{{ t("shell.totalCount", { count: totalNodeCount }) }}</span>
            <template v-if="hoveredNodeType">
              <span class="status-sep">|</span>
              <span class="status-item status-hover">{{ hoveredNodeType }}</span>
            </template>
          </div>
        </div>
      </div>

      <ToolRail
        side="right"
        :items="rightRailItems"
        :active-key="rightActiveKey"
        @select="handleRightSelect"
      />
    </div>
  </div>
</template>

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
