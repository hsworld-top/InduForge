<!--
  PreviewView - 预览视图
  预览模式主界面：返回编辑、视图预设、刷新、NodeRenderer 只读渲染
-->
<script setup lang="ts">
import type {
  PreviewPageLifecycleShape,
  PreviewRuntimeHandle,
  PreviewRuntimeInitOptions,
} from "./preview-runtime.types";
import type { ViewPreset } from "@/constants";
import type { PageConfig } from "@/editor-core/document/types";
import type { EditorRouteProjectMeta } from "@/stores/editor-store.types";
import { storeToRefs } from "pinia";
import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { VIEW_PRESETS } from "@/constants";

import { useEditorStore } from "@/stores/editor-store";
import {
  canvasZoomKey,
  runtimeAccessContextKey,
} from "@/ui/editors/page/canvas/injection-keys";
import NodeRenderer from "@/ui/editors/page/canvas/NodeRenderer.vue";
import { createRuntimeAccessContext } from "@/ui/editors/page/canvas/runtime-access";
import { clearPreviewRuntime, initPreviewRuntime } from "./previewRuntime";

/** 预览宿主读取的页面配置视图：优先 grouped 配置，兼容旧平铺字段。 */
type PreviewCanvasPageConfig = Partial<PageConfig> & {
  viewport?: PreviewViewportConfig;
  background?: PreviewBackgroundConfig | null;
  backgroundColor?: string;
  backgroundImage?: string;
  backgroundSize?: string;
  backgroundRepeat?: string;
  backgroundPosition?: string;
};

interface PreviewViewportConfig {
  width?: number;
  height?: number;
  autoFit?: boolean;
  lockAspectRatio?: boolean;
  minWidth?: number;
  minHeight?: number;
  overflowMode?: string;
}

interface PreviewBackgroundConfig {
  kind?: string;
  value?: string;
  size?: string;
  position?: string;
  repeat?: string;
}

interface PreviewResolvedPageViewport {
  width: number;
  height: number;
  autoFit: boolean;
  lockAspectRatio: boolean;
  minWidth: number;
  minHeight: number;
  overflowMode: "auto" | "hidden" | "scroll";
}

interface PreviewResolvedCanvasBackground {
  backgroundColor: string;
  backgroundImage?: string;
  backgroundSize?: string;
  backgroundRepeat?: string;
  backgroundPosition?: string;
}

interface PreviewContainerSize {
  width: number;
  height: number;
}

interface PreviewResolvedCanvasLayout {
  width: number;
  height: number;
  modeLabel: string;
}

/** 统一把 grouped viewport 与旧平铺字段归一化，避免预览和编辑器侧出现双重语义。 */
function resolvePreviewViewport(config: PreviewCanvasPageConfig): PreviewResolvedPageViewport {
  const viewport = (config.viewport || null) as PreviewViewportConfig | null;
  const width = Number(viewport?.width ?? config.width ?? 1366);
  const height = Number(viewport?.height ?? config.height ?? 768);
  return {
    width: Number.isFinite(width) && width > 0 ? width : 1366,
    height: Number.isFinite(height) && height > 0 ? height : 768,
    autoFit: Boolean(viewport?.autoFit ?? config.autoFit),
    lockAspectRatio: Boolean(viewport?.lockAspectRatio ?? config.lockAspectRatio),
    minWidth: Number.isFinite(Number(viewport?.minWidth)) && Number(viewport?.minWidth) > 0
      ? Number(viewport?.minWidth)
      : 0,
    minHeight: Number.isFinite(Number(viewport?.minHeight)) && Number(viewport?.minHeight) > 0
      ? Number(viewport?.minHeight)
      : 0,
    overflowMode:
      viewport?.overflowMode === "hidden" || viewport?.overflowMode === "scroll"
        ? viewport.overflowMode
        : "auto",
  };
}

/**
 * 预览容器按当前预设决定大小：
 * - 页面实际尺寸：容器等于设计尺寸，便于查看原始画面
 * - 设备预设：容器等于预设尺寸，模拟运行容器
 */
function resolvePreviewContainerSize(
  viewKey: string,
  viewport: PreviewResolvedPageViewport,
  presets: readonly ViewPreset[],
): PreviewContainerSize {
  if (viewKey === "page") {
    return {
      width: viewport.width,
      height: viewport.height,
    };
  }
  const preset = presets.find((item) => item.key === viewKey);
  return {
    width: preset?.width || viewport.width,
    height: preset?.height || viewport.height,
  };
}

/**
 * 按运行容器规则计算页面根容器尺寸：
 * - 固定尺寸：保持设计稿宽高
 * - 自适应 + 锁比：先按容器完整显示，再受最小尺寸下限约束
 * - 自适应 + 非锁比：宽高分别贴合容器，再受最小尺寸约束
 */
function resolvePreviewCanvasLayout(
  viewport: PreviewResolvedPageViewport,
  container: PreviewContainerSize,
): PreviewResolvedCanvasLayout {
  if (!viewport.autoFit) {
    return {
      width: viewport.width,
      height: viewport.height,
      modeLabel: "固定尺寸",
    };
  }

  if (viewport.lockAspectRatio) {
    const fitScale = Math.min(
      container.width / Math.max(1, viewport.width),
      container.height / Math.max(1, viewport.height),
    );
    const minScale = Math.max(
      viewport.minWidth > 0 ? viewport.minWidth / Math.max(1, viewport.width) : 0,
      viewport.minHeight > 0 ? viewport.minHeight / Math.max(1, viewport.height) : 0,
    );
    const scale = Math.max(fitScale, minScale, 0);
    return {
      width: Math.max(1, Math.round(viewport.width * scale)),
      height: Math.max(1, Math.round(viewport.height * scale)),
      modeLabel: "自适应-等比",
    };
  }

  return {
    width: Math.max(container.width, viewport.minWidth || 0),
    height: Math.max(container.height, viewport.minHeight || 0),
    modeLabel: "自适应-拉伸",
  };
}

/** 归一化预览背景样式：grouped background 优先，旧 backgroundColor/backgroundImage 作为回退。 */
function resolvePreviewBackground(config: PreviewCanvasPageConfig): PreviewResolvedCanvasBackground {
  const background = (config.background || null) as PreviewBackgroundConfig | null;
  const fallbackColor = config.backgroundColor || "#ffffff";
  if (background?.kind === "color") {
    return {
      backgroundColor: background.value || fallbackColor,
    };
  }
  if (background?.kind === "image") {
    return {
      backgroundColor: fallbackColor,
      backgroundImage: `url(${background.value || ""})`,
      backgroundSize: background.size || "cover",
      backgroundRepeat: background.repeat || "no-repeat",
      backgroundPosition: background.position || "center",
    };
  }
  if (background?.kind === "gradient") {
    return {
      backgroundColor: fallbackColor,
      backgroundImage: background.value || "",
      backgroundSize: background.size || "cover",
      backgroundRepeat: background.repeat || "no-repeat",
      backgroundPosition: background.position || "center",
    };
  }
  if (config.backgroundImage) {
    return {
      backgroundColor: fallbackColor,
      backgroundImage: `url(${config.backgroundImage})`,
      backgroundSize: config.backgroundSize || "cover",
      backgroundRepeat: config.backgroundRepeat || "no-repeat",
      backgroundPosition: config.backgroundPosition || "center",
    };
  }
  return {
    backgroundColor: fallbackColor,
  };
}

const router = useRouter();
const route = useRoute();
const editorStore = useEditorStore();
const {
  currentPage,
  currentPageId,
  doc,
  docVersion,
  projectVariables,
  globalScripts,
  projectId,
  runtimeUsers,
  selectedPreviewRuntimeUserId,
} = storeToRefs(editorStore);
provide(canvasZoomKey, ref(1));

const viewKey = ref<string>("page");
const viewPresets: readonly ViewPreset[] = VIEW_PRESETS;
const previewOptions = computed(() => [{ key: "page", label: "页面实际尺寸" }, ...viewPresets]);

const rootNodeId = computed(() => currentPage.value?.rootNodeId || "");
const routePreviewUserId = computed(() => {
  const raw = route.query.previewUserId;
  return String(Array.isArray(raw) ? raw[0] || "" : raw || "").trim();
});
const previewRuntimeUser = computed(() => {
  const selectedId = routePreviewUserId.value || selectedPreviewRuntimeUserId?.value || "";
  if (!selectedId) return null;
  return (runtimeUsers?.value || []).find((user) => user.id === selectedId) || null;
});
const runtimeAccessSnapshot = computed(() =>
  createRuntimeAccessContext({
    config: currentPage.value?.config?.runtimeAccess || null,
    user: previewRuntimeUser.value,
  }),
);
provide(runtimeAccessContextKey, {
  get enabled() {
    return runtimeAccessSnapshot.value.enabled;
  },
  get schemes() {
    return runtimeAccessSnapshot.value.schemes;
  },
  get user() {
    return runtimeAccessSnapshot.value.user;
  },
  get canViewPage() {
    return runtimeAccessSnapshot.value.canViewPage;
  },
  evaluateScheme: (schemeId?: string) => runtimeAccessSnapshot.value.evaluateScheme(schemeId),
  isNodeVisible: (node: Record<string, any>) => runtimeAccessSnapshot.value.isNodeVisible(node),
  isNodeOperable: (node: Record<string, any>) => runtimeAccessSnapshot.value.isNodeOperable(node),
});
const previewPageConfig = computed(() =>
  resolvePreviewViewport(currentPage.value?.config || ({} as PreviewCanvasPageConfig)),
);
const previewContainerSize = computed(() =>
  resolvePreviewContainerSize(viewKey.value, previewPageConfig.value, viewPresets),
);
const previewCanvasLayout = computed(() =>
  resolvePreviewCanvasLayout(previewPageConfig.value, previewContainerSize.value),
);
const previewOverflowLabel = computed(() => {
  switch (previewPageConfig.value.overflowMode) {
    case "hidden":
      return "隐藏";
    case "scroll":
      return "始终显示";
    default:
      return "自动";
  }
});
const previewMinSizeLabel = computed(() => {
  const { minWidth, minHeight } = previewPageConfig.value;
  if (minWidth <= 0 && minHeight <= 0) {
    return "未限制";
  }
  return `${minWidth || 0} × ${minHeight || 0}`;
});
const previewSummaryItems = computed(() => [
  `设计尺寸：${previewPageConfig.value.width} × ${previewPageConfig.value.height}`,
  `容器：${previewContainerSize.value.width} × ${previewContainerSize.value.height}`,
  `适配：${previewCanvasLayout.value.modeLabel}`,
  `最小尺寸：${previewMinSizeLabel.value}`,
  `滚动：${previewOverflowLabel.value}`,
]);

// 预览框样式
const frameStyle = computed((): Record<string, string> => {
  return {
    width: `${previewContainerSize.value.width}px`,
    height: `${previewContainerSize.value.height}px`,
    maxWidth: "100%",
    maxHeight: "100%",
    overflow: previewPageConfig.value.overflowMode,
  };
});

const canvasStyle = computed((): Record<string, string> => {
  void docVersion.value;
  const config = currentPage.value?.config || ({} as PreviewCanvasPageConfig);
  const viewport = resolvePreviewViewport(config);
  const background = resolvePreviewBackground(config);
  const style: Record<string, string> = {
    width: `${previewCanvasLayout.value.width}px`,
    height: `${previewCanvasLayout.value.height}px`,
    backgroundColor: background.backgroundColor,
    position: "relative",
  };
  if (background.backgroundImage) {
    style.backgroundImage = background.backgroundImage;
  }
  if (background.backgroundSize) {
    style.backgroundSize = background.backgroundSize;
  }
  if (background.backgroundRepeat) {
    style.backgroundRepeat = background.backgroundRepeat;
  }
  if (background.backgroundPosition) {
    style.backgroundPosition = background.backgroundPosition;
  }
  return style;
});

/**
 * 返回编辑
 */
function handleBack() {
  const qPid = route.query.pid;
  const pid = Array.isArray(qPid) ? qPid[0] : qPid;
  const qPage = route.query.pageId;
  const pageId = (Array.isArray(qPage) ? qPage[0] : qPage) || "";
  router.replace({ path: "/", query: { pid: pid ?? "", pageId } });
}

/**
 * 刷新预览（重建 runtime，与依赖变更时一致）
 */
function handleRefresh() {
  void restartPreviewRuntime("manual");
}

const routeProjectMeta = computed(
  () => (route.meta as { project?: EditorRouteProjectMeta }).project,
);

async function loadProject() {
  const pid = routeProjectMeta.value?.id;
  if (!pid) return;
  if (editorStore.projectId === pid && currentPage.value?.rootNodeId) {
    return;
  }
  await editorStore.loadProject(pid);
  if (routePreviewUserId.value) {
    editorStore.setSelectedPreviewRuntimeUserId(routePreviewUserId.value);
  }
}

function queryPageIdFromRoute(): string | undefined {
  const qPage = route.query.pageId;
  const raw = Array.isArray(qPage) ? qPage[0] : qPage;
  return raw ? String(raw) : undefined;
}

function buildPreviewInitOptions(): PreviewRuntimeInitOptions {
  return {
    projectId: projectId.value || editorStore.projectId,
    projectVariables: projectVariables.value || {},
    globalScripts: globalScripts.value || {},
    pageLifecycle: (currentPage.value?.lifecycle || {}) as unknown as PreviewPageLifecycleShape,
    pageVariables: doc.value?.vars?.pages?.[currentPageId.value] || {},
    pageId: currentPage.value?.name || currentPage.value?.id || queryPageIdFromRoute() || null,
  };
}

let previewRuntime: PreviewRuntimeHandle | null = null;

/** 停止并清空单例后按当前 store 快照重建预览运行时 */
async function restartPreviewRuntime(_reason?: string) {
  previewRuntime?.stop?.();
  clearPreviewRuntime();
  previewRuntime = null;
  const runtime = initPreviewRuntime(buildPreviewInitOptions());
  previewRuntime = runtime ?? null;
  await Promise.resolve(runtime?.start?.());
}

/** 避免首屏 loadProject 过程中 watch 与 onMounted 双重重启 */
const previewRuntimeSetupReady = ref(false);

onMounted(async () => {
  await loadProject();
  previewRuntimeSetupReady.value = true;
  await restartPreviewRuntime("mount");
});

/**
 * 文档 / 页面切换：依赖 projectId、currentPageId、docVersion 的引用或标量变化（浅依赖，无 deep）。
 * 画布节点树由 Vue 随 doc 重渲；runtime 仍全量重启以保证生命周期脚本与 MQTT/映射一致。
 */
watch(
  () => [projectId.value, currentPageId.value, docVersion.value],
  () => {
    if (!previewRuntimeSetupReady.value) return;
    void restartPreviewRuntime("watch-doc-page");
  },
  { flush: "post" },
);

/**
 * 工程变量与全局脚本：仅当 store 替换整个 ref 时重启（避免 deep 监听导致频繁断开 MQTT）。
 * 就地改深层字段时请用工具栏「刷新」。
 */
watch(
  [projectVariables, globalScripts],
  () => {
    if (!previewRuntimeSetupReady.value) return;
    void restartPreviewRuntime("watch-scripts");
  },
  { flush: "post" },
);

onBeforeUnmount(() => {
  previewRuntime?.stop?.();
  clearPreviewRuntime();
  previewRuntime = null;
});
</script>

<template>
  <div class="flex flex-col h-screen bg-gray-100 dark:bg-gray-900">
    <!-- 工具栏 -->
    <header
      class="flex items-center justify-between h-12 px-3 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700"
    >
      <div class="flex items-center gap-2">
        <el-button size="small" @click="handleBack">
          <span aria-hidden="true">←</span>
          返回编辑
        </el-button>
        <el-divider direction="vertical" />
        <span class="text-sm font-medium">预览模式</span>
      </div>

      <div class="flex items-center gap-2">
        <el-radio-group v-model="viewKey" size="small">
          <el-radio-button v-for="preset in previewOptions" :key="preset.key" :value="preset.key">
            {{ preset.label }}
          </el-radio-button>
        </el-radio-group>
        <div class="preview-summary" data-testid="preview-summary">
          <span
            v-for="item in previewSummaryItems"
            :key="item"
            class="preview-summary__item"
          >
            {{ item }}
          </span>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <el-button size="small" type="primary" @click="handleRefresh">
          <span aria-hidden="true">↻</span>
          刷新
        </el-button>
      </div>
    </header>

    <!-- 预览内容 -->
    <main class="flex-1 flex items-center justify-center p-6 overflow-auto">
      <div
        class="preview-frame bg-white dark:bg-gray-800 shadow-lg rounded-lg overflow-hidden transition-all duration-300"
        :style="frameStyle"
      >
        <div v-if="!runtimeAccessSnapshot.canViewPage" class="preview-denied">
          <div class="preview-denied__title">无页面访问权限</div>
          <div class="preview-denied__desc">当前预览身份不在页面允许访问角色内。</div>
        </div>
        <div v-else class="preview-canvas" :style="canvasStyle">
          <NodeRenderer v-if="rootNodeId" :node-id="rootNodeId" :is-root="true" :readonly="true" />
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.preview-frame {
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
  overflow: auto;
}

.preview-canvas {
  flex-shrink: 0;
}

.preview-summary {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  margin-left: 8px;
  color: var(--designer-text-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.preview-summary__item {
  flex: 0 0 auto;
}

.preview-denied {
  display: flex;
  width: 100%;
  height: 100%;
  min-width: 320px;
  min-height: 200px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: #fff;
  color: var(--designer-border-strong);
}

.preview-denied__title {
  font-size: 18px;
  font-weight: 600;
}

.preview-denied__desc {
  color: var(--designer-text-secondary);
  font-size: 13px;
}

.preview-frame::-webkit-scrollbar {
  width: 10px;
  height: 10px;
}

.preview-frame::-webkit-scrollbar-track {
  background: transparent;
}

.preview-frame::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.6);
  border-radius: 8px;
  border: 2px solid transparent;
  background-clip: padding-box;
}

.preview-frame::-webkit-scrollbar-thumb:hover {
  background: rgba(100, 116, 139, 0.8);
  border: 2px solid transparent;
  background-clip: padding-box;
}

.preview-frame {
  scrollbar-width: thin;
  scrollbar-color: rgba(148, 163, 184, 0.6) transparent;
}
</style>
