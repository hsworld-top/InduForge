<!--
  @deprecated 此组件已废弃
  请使用 ToolRail + DockPanel 架构替代
  新架构参见 DesignerView.vue 中的 leftRailItems 配置
  各功能面板已拆分为独立组件：PageTree、OutlineTree、MaterialPanel 等
-->
<template>
  <aside class="panel panel-left">
    <div class="panel-header">
      <span class="text-sm font-medium">资源面板</span>
    </div>
    <div class="panel-body">
      <el-collapse v-model="activeSections">
        <el-collapse-item name="drawing">
          <template #title>
            <span class="text-sm font-medium">绘图工具</span>
          </template>
          <div class="flex flex-wrap gap-2">
            <el-button
              v-for="tool in drawingTools"
              :key="tool.key"
              size="small"
              :type="drawingTool === tool.key ? 'primary' : ''"
              @click="handleDrawingToolChange(tool.key)"
            >
              {{ tool.label }}
            </el-button>
          </div>
        </el-collapse-item>

        <el-collapse-item name="components">
          <template #title>
            <span class="text-sm font-medium">组件面板</span>
          </template>
          <ComponentPanel />
        </el-collapse-item>

        <el-collapse-item name="symbols">
          <template #title>
            <span class="text-sm font-medium">符号库</span>
          </template>
          <SymbolLibraryPanel />
        </el-collapse-item>

        <el-collapse-item name="pages">
          <template #title>
            <span class="text-sm font-medium">页面树</span>
          </template>
          <PageTree />
        </el-collapse-item>

        <el-collapse-item name="outline">
          <template #title>
            <span class="text-sm font-medium">大纲树</span>
          </template>
          <OutlineTree />
        </el-collapse-item>

        <el-collapse-item name="datapoints">
          <template #title>
            <span class="text-sm font-medium">数据点</span>
          </template>
          <DatapointPanel />
        </el-collapse-item>
      </el-collapse>
    </div>
  </aside>
</template>

<script setup>
import { ref, toRefs } from "vue";
import ComponentPanel from "./ComponentPanel.vue";
import PageTree from "./PageTree.vue";
import OutlineTree from "./OutlineTree.vue";
import DatapointPanel from "./DatapointPanel.vue";
import SymbolLibraryPanel from "./SymbolLibraryPanel.vue";

const props = defineProps({
  drawingTool: {
    type: String,
    default: "",
  },
});

const emit = defineEmits(["update:drawingTool"]);

/**
 * 面板展开状态
 */
const activeSections = ref([
  "drawing",
  "components",
  "pages",
  "outline",
]);

/**
 * 绘图工具列表
 */
const drawingTools = [
  { key: "line", label: "线" },
  { key: "rect", label: "矩形" },
  { key: "ellipse", label: "圆形" },
  { key: "polygon", label: "多边形" },
  { key: "pipe", label: "管道" },
  { key: "text", label: "文字" },
];

const { drawingTool } = toRefs(props);

/**
 * 切换绘图工具
 * @param {string} tool - 工具类型
 */
const handleDrawingToolChange = (tool) => {
  emit("update:drawingTool", tool);
};
</script>
