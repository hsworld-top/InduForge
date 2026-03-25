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
// 图标导入
import IconEpArrowLeft from "~icons/ep/arrow-left";
import IconEpRefresh from "~icons/ep/refresh";
import { VIEW_PRESETS } from "@/constants";

import { useEditorStore } from "@/stores/editor-store";
import { canvasZoomKey } from "@/ui/editors/page/canvas/injection-keys";
import NodeRenderer from "@/ui/editors/page/canvas/NodeRenderer.vue";
import { clearPreviewRuntime, initPreviewRuntime } from "./previewRuntime";

/** 预览画布实际读取的页面配置（含壳层/历史遗留的扁平背景字段） */
type PreviewCanvasPageConfig = PageConfig & {
  backgroundColor?: string;
  backgroundImage?: string;
  backgroundSize?: string;
};

const router = useRouter();
const route = useRoute();
const editorStore = useEditorStore();
const { currentPage, currentPageId, doc, docVersion, projectVariables, globalScripts, projectId } =
  storeToRefs(editorStore);
provide(canvasZoomKey, ref(1));

const viewKey = ref<string>("page");
const viewPresets: readonly ViewPreset[] = VIEW_PRESETS;
const previewOptions = computed(() => [{ key: "page", label: "页面实际尺寸" }, ...viewPresets]);

const rootNodeId = computed(() => currentPage.value?.rootNodeId || "");

// 预览框样式
const frameStyle = computed((): Record<string, string> => {
  const config = {
    width: 1366,
    height: 768,
    ...(currentPage.value?.config || {}),
  } as PageConfig;
  if (viewKey.value === "page") {
    return {
      width: `${config.width || 1366}px`,
      height: `${config.height || 768}px`,
      maxWidth: "100%",
      maxHeight: "100%",
    };
  }
  const preset = viewPresets.find((item) => item.key === viewKey.value);
  const size = preset
    ? { width: `${preset.width}px`, height: `${preset.height}px` }
    : { width: "100%", height: "100%" };
  return {
    width: size.width,
    height: size.height,
    maxWidth: "100%",
    maxHeight: "100%",
  };
});

const canvasStyle = computed((): Record<string, string> => {
  docVersion.value;
  const config = {
    width: 1366,
    height: 768,
    ...(currentPage.value?.config || {}),
  } as PreviewCanvasPageConfig;
  const preset = viewPresets.find((item) => item.key === viewKey.value);
  const autoFit = Boolean(config.autoFit);
  const width = autoFit ? "100%" : config.width || preset?.width || 1200;
  const height = autoFit ? "100%" : config.height || preset?.height || 800;
  const style: Record<string, string> = {
    width: typeof width === "number" ? `${width}px` : width,
    height: typeof height === "number" ? `${height}px` : height,
    backgroundColor: config.backgroundColor || "#ffffff",
    position: "relative",
  };
  const background = (config.background || null) as {
    kind?: string;
    value?: string;
  } | null;
  if (background?.kind === "color") {
    style.backgroundColor = background.value || "#ffffff";
  } else if (background?.kind === "image") {
    style.backgroundImage = `url(${background.value || ""})`;
    style.backgroundSize = "cover";
    style.backgroundRepeat = "no-repeat";
    style.backgroundPosition = "center";
  } else if (background?.kind === "gradient") {
    style.backgroundImage = background.value || "";
    style.backgroundSize = "cover";
    style.backgroundRepeat = "no-repeat";
    style.backgroundPosition = "center";
  } else if (config.backgroundImage) {
    style.backgroundImage = `url(${config.backgroundImage})`;
    style.backgroundSize = config.backgroundSize || "cover";
    style.backgroundRepeat = "no-repeat";
    style.backgroundPosition = "center";
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
  router.push({ path: "/", query: { pid: pid ?? "", pageId } });
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
          <IconEpArrowLeft />
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
      </div>

      <div class="flex items-center gap-2">
        <el-button size="small" type="primary" @click="handleRefresh">
          <IconEpRefresh />
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
        <div class="preview-canvas" :style="canvasStyle">
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
