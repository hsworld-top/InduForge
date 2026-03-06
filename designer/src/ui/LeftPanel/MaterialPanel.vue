<template>
  <div class="flex flex-col gap-3 material-panel">
    <!-- 页面编辑模式：组件 + 绘图区 + 资源 -->
    <template v-if="editMode === 'page'">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="组件" name="components">
          <ComponentPanel />
        </el-tab-pane>
        <el-tab-pane label="绘图区(暂缓)" name="diagram" disabled>
          <DiagramAreaPanel />
        </el-tab-pane>
        <el-tab-pane label="资源" name="resources">
          <ResourcePanel />
        </el-tab-pane>
      </el-tabs>
    </template>

    <!-- Canvas 绘图模式：绘图工具 -->
    <template v-else-if="editMode === 'canvas'">
      <div class="canvas-tools-panel">
        <div class="panel-header">
          <h3 class="text-sm font-medium">Canvas 绘图工具</h3>
          <el-button size="small" text @click="exitCanvasMode">
            <IconEpBack />
            返回页面编辑
          </el-button>
        </div>
        <CanvasToolsPanel v-model="activeToolModel" />
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, ref } from "vue";
import ComponentPanel from "./ComponentPanel.vue";
import DiagramAreaPanel from "./DiagramAreaPanel.vue";
import CanvasToolsPanel from "./CanvasToolsPanel.vue";
import ResourcePanel from "./ResourcePanel.vue";
import IconEpBack from "~icons/ep/back";

const props = defineProps({
  /**
   * 编辑模式：page - 页面编辑，canvas - Canvas 绘图
   */
  editMode: {
    type: String,
    default: "page",
    validator: (v) => ["page", "canvas"].includes(v),
  },
  /**
   * 当前激活的工具
   */
  activeTool: {
    type: String,
    default: "",
  },
});

const emit = defineEmits(["update:editMode", "update:activeTool"]);

const activeTab = ref("components");

/**
 * Canvas 工具栏 v-model 代理，避免直接修改 props
 */
const activeToolModel = computed({
  get: () => props.activeTool,
  set: (value) => emit("update:activeTool", value),
});

/**
 * 退出 Canvas 模式
 */
const exitCanvasMode = () => {
  emit("update:editMode", "page");
};
</script>

<style scoped>
.material-panel {
  height: 100%;
  overflow: hidden;
}

.material-panel :deep(.el-tabs) {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.material-panel :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
}

.material-panel :deep(.el-tab-pane) {
  height: 100%;
}

.canvas-tools-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--el-border-color);
}
</style>
