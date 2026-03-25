<!--
  DrawingPanel - 绘图工具面板
  提供线、矩形、圆形、多边形、管道、文字等绘图工具选择，含符号库
-->
<script setup>
import SymbolLibraryPanel from "./SymbolLibraryPanel.vue";

const props = defineProps({
  modelValue: {
    type: String,
    default: "",
  },
});

const emit = defineEmits(["update:modelValue"]);

const drawingTools = [
  { key: "line", label: "线" },
  { key: "rect", label: "矩形" },
  { key: "ellipse", label: "圆形" },
  { key: "polygon", label: "多边形" },
  { key: "pipe", label: "管道" },
  { key: "text", label: "文字" },
];

/**
 * 切换绘图工具
 * @param {string} tool - 工具类型
 */
function handleSelect(tool) {
  emit("update:modelValue", tool);
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex flex-wrap gap-2">
      <el-button
        v-for="tool in drawingTools"
        :key="tool.key"
        size="small"
        :type="modelValue === tool.key ? 'primary' : ''"
        @click="handleSelect(tool.key)"
      >
        {{ tool.label }}
      </el-button>
    </div>
    <SymbolLibraryPanel />
  </div>
</template>
