<template>
  <div class="canvas-tools-panel">
    <div class="tool-list">
      <!-- 选择工具 -->
      <div
        class="tool-item"
        :class="{ active: modelValue === 'select' }"
        @click="selectTool('select')"
      >
        <IconEpPointer class="tool-icon" />
        <span class="tool-name">选择</span>
      </div>

      <el-divider />

      <!-- 基础图形 -->
      <div class="tool-group-title">基础图形</div>
      
      <div
        class="tool-item"
        :class="{ active: modelValue === 'line' }"
        @click="selectTool('line')"
      >
        <IconEpMinus class="tool-icon" />
        <span class="tool-name">直线</span>
      </div>

      <div
        class="tool-item"
        :class="{ active: modelValue === 'rect' }"
        @click="selectTool('rect')"
      >
        <IconEpCropSquare class="tool-icon" />
        <span class="tool-name">矩形</span>
      </div>

      <div
        class="tool-item"
        :class="{ active: modelValue === 'circle' }"
        @click="selectTool('circle')"
      >
        <IconEpCircle class="tool-icon" />
        <span class="tool-name">圆形</span>
      </div>

      <el-divider />

      <!-- 文本和图片 -->
      <div class="tool-group-title">文本与媒体</div>

      <div
        class="tool-item"
        :class="{ active: modelValue === 'text' }"
        @click="selectTool('text')"
      >
        <IconEpEditPen class="tool-icon" />
        <span class="tool-name">文本</span>
      </div>

      <div
        class="tool-item"
        :class="{ active: modelValue === 'image' }"
        @click="selectTool('image')"
      >
        <IconEpPicture class="tool-icon" />
        <span class="tool-name">图片</span>
      </div>

      <el-divider />

      <!-- 管道工具 -->
      <div class="tool-group-title">流程工具</div>

      <div
        class="tool-item"
        :class="{ active: modelValue === 'pipe' }"
        @click="selectTool('pipe')"
      >
        <IconEpConnection class="tool-icon" />
        <span class="tool-name">管道</span>
      </div>

      <div
        class="tool-item"
        :class="{ active: modelValue === 'path' }"
        @click="selectTool('path')"
      >
        <IconEpEditPen class="tool-icon" />
        <span class="tool-name">路径</span>
      </div>
    </div>

    <!-- 工具提示 -->
    <div v-if="modelValue" class="tool-hint">
      <div class="text-xs text-gray-500">
        {{ toolHints[modelValue] }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from "vue";
import IconEpPointer from "~icons/ep/pointer";
import IconEpMinus from "~icons/ep/minus";
import IconEpCropSquare from "~icons/ep/crop";
import IconEpCircle from "~icons/ep/circle-check";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpPicture from "~icons/ep/picture";
import IconEpConnection from "~icons/ep/connection";

const props = defineProps({
  modelValue: {
    type: String,
    default: "select",
  },
});

const emit = defineEmits(["update:modelValue"]);

/**
 * 工具提示信息
 */
const toolHints = {
  select: "选择和移动图元（V）",
  line: "绘制直线（L）",
  rect: "绘制矩形（R）",
  circle: "绘制圆形（C）",
  text: "添加文本（T）",
  image: "插入图片（I）",
  pipe: "绘制管道（P）",
  path: "绘制路径（Shift+P）",
};

/**
 * 选择工具
 */
const selectTool = (tool) => {
  emit("update:modelValue", tool);
};
</script>

<style scoped>
.canvas-tools-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  overflow-y: auto;
}

.tool-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
}

.tool-group-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  padding: 8px 8px 4px;
}

.tool-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.tool-item:hover {
  background-color: var(--el-fill-color-light);
}

.tool-item.active {
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.tool-icon {
  font-size: 18px;
}

.tool-name {
  flex: 1;
  font-size: 14px;
}

.tool-hint {
  padding: 12px;
  background-color: var(--el-fill-color-light);
  border-radius: 6px;
  margin: 0 8px;
}
</style>
