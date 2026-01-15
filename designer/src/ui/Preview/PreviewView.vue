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
          <el-radio-button
            v-for="preset in viewPresets"
            :key="preset.key"
            :value="preset.key"
          >
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
          <NodeRenderer
            v-if="rootNodeId"
            :node-id="rootNodeId"
            :is-root="true"
            :readonly="true"
          />
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, provide } from "vue";
import { useRouter, useRoute } from "vue-router";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import NodeRenderer from "@/ui/Canvas/NodeRenderer.vue";
import { initPreviewRuntime, clearPreviewRuntime } from "./previewRuntime";

// 图标导入
import IconEpArrowLeft from "~icons/ep/arrow-left";
import IconEpRefresh from "~icons/ep/refresh";
import { VIEW_PRESETS } from "@/constants";

const router = useRouter();
const route = useRoute();
const editorStore = useEditorStore();
const { currentPage, docVersion, projectVariables, globalScripts, projectId } =
  storeToRefs(editorStore);
provide("canvasZoom", ref(1));

const viewKey = ref("pc");
const viewPresets = VIEW_PRESETS;

const rootNodeId = computed(() => currentPage.value?.rootNodeId || "");

// 预览框样式
const frameStyle = computed(() => {
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

const canvasStyle = computed(() => {
  docVersion.value;
  const config = currentPage.value?.config || {};
  const preset = viewPresets.find((item) => item.key === viewKey.value);
  const width = config.width || preset?.width || 1200;
  const height = config.height || preset?.height || 800;
  const style = {
    width: `${width}px`,
    height: `${height}px`,
    backgroundColor: config.backgroundColor || "#ffffff",
    position: "relative",
  };
  if (config.backgroundImage) {
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
const handleBack = () => {
  const projectId = route.query.pid;
  router.push({ path: "/", query: { pid: projectId } });
};

/**
 * 刷新预览
 */
const handleRefresh = () => {
  // TODO: 重新加载数据
  console.log("刷新预览");
};

const loadProject = async () => {
  const projectId = route.meta.project?.id;
  if (!projectId) return;
  if (editorStore.projectId === projectId && currentPage.value?.rootNodeId) {
    return;
  }
  await editorStore.loadProject(projectId);
};

let previewRuntime = null;

onMounted(() => {
  void loadProject();
  const runtime = initPreviewRuntime({
    projectId: projectId.value || editorStore.projectId,
    projectVariables: projectVariables.value || {},
    globalScripts: globalScripts.value || {},
  });
  previewRuntime = runtime || null;
  runtime?.start?.();
});

onBeforeUnmount(() => {
  if (previewRuntime?.stop) {
    previewRuntime.stop();
  }
  clearPreviewRuntime();
});
</script>

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

