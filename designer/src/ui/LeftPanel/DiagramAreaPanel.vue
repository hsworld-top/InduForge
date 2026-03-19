<!--
  DiagramAreaPanel - 绘图区面板
  2D 流程图等可拖拽到画布的图表组件入口
-->
<template>
  <div class="flex flex-col gap-3 diagram-area-panel">
    <div class="text-xs text-gray-500 px-2">拖拽到画布，双击进入编辑模式</div>

    <!-- 2D 流程图 -->
    <div class="diagram-category">
      <h4 class="category-title">2D 流程图</h4>
      <div class="diagram-items">
        <div
          class="diagram-item"
          draggable="true"
          @mousedown="handlePointerStart('Diagram2D', $event)"
          @dragstart="handleDragStart('Diagram2D', $event)"
          @dragend="handleDragEnd"
        >
          <div class="diagram-icon">
            <IconEpGrid />
          </div>
          <div class="diagram-info">
            <div class="diagram-name">2D 流程图</div>
            <div class="diagram-desc">Canvas 矢量绘图</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 3D 编辑器（占位） -->
    <div class="diagram-category">
      <h4 class="category-title">3D 编辑</h4>
      <div class="diagram-items">
        <div class="diagram-item disabled">
          <div class="diagram-icon">
            <IconEpBox />
          </div>
          <div class="diagram-info">
            <div class="diagram-name">3D 编辑器</div>
            <div class="diagram-desc">敬请期待</div>
          </div>
          <el-tag size="small" type="info">开发中</el-tag>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import IconEpGrid from "~icons/ep/grid";
import IconEpBox from "~icons/ep/box";
import { startDrag, endDrag } from "@/ui/Canvas/use-drag-state";

/**
 * 处理拖拽开始
 * @param {string} type - 组件类型
 * @param {DragEvent} event - 拖拽事件
 */
const handleDragStart = (type, event) => {
  startDrag(type);
  if (!event.dataTransfer) return;
  const payload = JSON.stringify({ type });
  event.dataTransfer.effectAllowed = "copy";
  event.dataTransfer.setData("application/x-designer-component", payload);
  event.dataTransfer.setData("text/plain", type);
};

/**
 * 处理鼠标拖拽开始（HTML5 drag 失效时兜底）
 * @param {string} type - 组件类型
 * @param {MouseEvent} event - 鼠标事件
 */
const handlePointerStart = (type, event) => {
  if (event.button !== 0) return;
  startDrag(type);
};

/**
 * 处理拖拽结束
 */
const handleDragEnd = () => {
  endDrag();
};
</script>

<style scoped>
.diagram-area-panel {
  padding: 8px;
}

.diagram-category {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.category-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  padding: 0 4px;
}

.diagram-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.diagram-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  cursor: grab;
  transition: all 0.2s;
}

.diagram-item:hover {
  border-color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-9);
}

.diagram-item.disabled {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none;
}

.diagram-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background-color: var(--el-color-primary-light-9);
  border-radius: 6px;
  font-size: 20px;
  color: var(--el-color-primary);
}

.diagram-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.diagram-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.diagram-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
