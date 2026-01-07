<template>
  <div class="designer-layout">
    <!-- 顶部工具栏 -->
    <header class="designer-toolbar">
      <div class="flex items-center gap-2">
        <span class="font-semibold text-gray-800 dark:text-gray-200">
          InduForge Designer
        </span>
        <span class="text-gray-400">|</span>
        <span class="text-sm text-gray-600 dark:text-gray-400">
          {{ projectTitle }}
        </span>
      </div>

      <div class="flex items-center gap-2">
        <!-- 工具按钮组 -->
        <el-button-group>
          <el-button size="small" :type="activeTool === 'select' ? 'primary' : ''" @click="activeTool = 'select'">
            <IconEpPointer />
          </el-button>
          <el-button size="small" :type="activeTool === 'hand' ? 'primary' : ''" @click="activeTool = 'hand'">
            <IconEpRank />
          </el-button>
        </el-button-group>

        <el-divider direction="vertical" />

        <!-- 撤销重做 -->
        <el-button-group>
          <el-button size="small" :disabled="!canUndo" @click="handleUndo">
            <IconEpBack />
          </el-button>
          <el-button size="small" :disabled="!canRedo" @click="handleRedo">
            <IconEpRight />
          </el-button>
        </el-button-group>

        <el-divider direction="vertical" />

        <!-- 缩放 -->
        <span class="text-sm text-gray-600 dark:text-gray-400">
          {{ Math.round(zoom * 100) }}%
        </span>
      </div>

      <div class="flex items-center gap-2">
        <el-button size="small" @click="handlePreview">
          <IconEpView />
          预览
        </el-button>
        <el-button size="small" type="primary" @click="handleSave" :loading="saving">
          <IconEpUpload />
          保存
        </el-button>
      </div>
    </header>

    <!-- 主体区域 -->
    <div class="designer-main">
      <!-- 左侧面板 -->
      <aside class="panel panel-left">
        <div class="panel-header">
          <span class="text-sm font-medium">组件库</span>
        </div>
        <div class="panel-body">
          <p class="text-sm text-gray-400 text-center py-8">
            组件面板（待实现）
          </p>
        </div>
      </aside>

      <!-- 画布区域 -->
      <main class="canvas-container">
        <div class="canvas-wrapper">
          <div
            class="canvas"
            :style="{
              width: `${canvasWidth}px`,
              height: `${canvasHeight}px`,
              transform: `scale(${zoom})`,
            }"
          >
            <!-- Canvas 图形层 -->
            <div class="absolute inset-0 pointer-events-none">
              <!-- Konva 将在这里渲染 -->
            </div>

            <!-- DOM 组件层 -->
            <div class="absolute inset-0">
              <div class="flex flex-col items-center justify-center h-full text-gray-400">
                <IconEpPlus class="text-5xl mb-4" />
                <p>从左侧拖拽组件到画布</p>
              </div>
            </div>
          </div>
        </div>

        <!-- 状态栏 -->
        <div class="canvas-statusbar">
          <span>画布: {{ canvasWidth }} × {{ canvasHeight }}</span>
          <el-divider direction="vertical" />
          <span>缩放: {{ Math.round(zoom * 100) }}%</span>
        </div>
      </main>

      <!-- 右侧面板 -->
      <aside class="panel panel-right">
        <div class="panel-header">
          <el-tabs v-model="rightTab" class="flex-1">
            <el-tab-pane label="属性" name="props" />
            <el-tab-pane label="样式" name="style" />
            <el-tab-pane label="事件" name="events" />
          </el-tabs>
        </div>
        <div class="panel-body">
          <p class="text-sm text-gray-400 text-center py-8">
            请选择一个组件
          </p>
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";

// 图标导入
import IconEpPointer from "~icons/ep/pointer";
import IconEpRank from "~icons/ep/rank";
import IconEpBack from "~icons/ep/back";
import IconEpRight from "~icons/ep/right";
import IconEpView from "~icons/ep/view";
import IconEpUpload from "~icons/ep/upload";
import IconEpPlus from "~icons/ep/plus";

const route = useRoute();
const router = useRouter();

const editorStore = useEditorStore();
const { projectName, canUndo, canRedo, saving } = storeToRefs(editorStore);

// 工具状态
const activeTool = ref("select");
const zoom = ref(1);

// 画布尺寸
const canvasWidth = ref(1920);
const canvasHeight = ref(1080);

// 右侧面板标签
const rightTab = ref("props");

/**
 * 项目标题展示
 */
const projectTitle = computed(() => {
  const fallbackProjectId = route.meta.project?.id;
  return projectName.value || (fallbackProjectId ? `工程 ${fallbackProjectId}` : "未选择工程");
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
  router.push({ path: "/preview", query: { pid: projectId } });
};

/**
 * 保存
 */
const handleSave = async () => {
  try {
    await editorStore.saveCurrentPage();
    ElMessage.success("保存成功");
  } catch (error) {
    const message = error instanceof Error ? error.message : "未知错误";
    ElMessage.error("保存失败: " + message);
  }
};

/**
 * 加载工程数据
 */
const loadProject = async () => {
  const project = route.meta.project;
  if (!project?.id) return;

  const result = await editorStore.loadProject(project.id);
  if (!result.ok) {
    ElMessage.error(result.error?.message || "加载工程失败");
  }
};

onMounted(() => {
  void loadProject();
});
</script>

<style scoped>
/* 网格背景 */
.canvas {
  background-image: linear-gradient(
      rgba(0, 0, 0, 0.05) 1px,
      transparent 1px
    ),
    linear-gradient(90deg, rgba(0, 0, 0, 0.05) 1px, transparent 1px);
  background-size: 10px 10px;
}

.dark .canvas {
  background-image: linear-gradient(
      rgba(255, 255, 255, 0.05) 1px,
      transparent 1px
    ),
    linear-gradient(90deg, rgba(255, 255, 255, 0.05) 1px, transparent 1px);
}

/* 面板标签覆盖 */
.panel-header :deep(.el-tabs__header) {
  margin: 0;
}

.panel-header :deep(.el-tabs__nav-wrap::after) {
  display: none;
}
</style>

