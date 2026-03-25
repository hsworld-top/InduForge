<!--
  DropIndicator - 拖拽放置指示器
  根据 hint 显示：Flex 插入线、Grid 单元格高亮、Free 十字线
-->
<script setup>
import { computed } from "vue";

const props = defineProps({
  /**
   * 拖拽决策对象
   * @type {Object}
   */
  hint: {
    type: Object,
    default: null,
  },
});

/**
 * 插入线样式（Flex 容器）
 */
const insertLineStyle = computed(() => {
  if (!props.hint?.visualHint) return {};

  const { orientation, offset } = props.hint.visualHint;

  if (orientation === "horizontal") {
    return {
      top: `${offset}px`,
    };
  } else {
    return {
      left: `${offset}px`,
    };
  }
});

/**
 * 单元格高亮样式（Grid 容器）
 */
const gridCellStyle = computed(() => {
  if (!props.hint?.visualHint?.highlightRect) return {};

  const { left, top, width, height } = props.hint.visualHint.highlightRect;

  return {
    left: `${left}px`,
    top: `${top}px`,
    width: `${width}px`,
    height: `${height}px`,
  };
});

/**
 * 十字线样式（Free 容器）
 */
const crosshairStyle = computed(() => {
  if (!props.hint?.visualHint) return {};

  const { x, y } = props.hint.visualHint;

  return {
    left: `${x}px`,
    top: `${y}px`,
  };
});
</script>

<template>
  <!-- Flex 容器：插入线 -->
  <div
    v-if="hint?.insertRule === 'before_after' && hint.visualHint"
    class="insert-line"
    :class="hint.visualHint.orientation"
    :style="insertLineStyle"
  />

  <!-- Grid 容器：单元格高亮 -->
  <div
    v-if="hint?.insertRule === 'grid_cell' && hint.visualHint?.highlightRect"
    class="grid-cell-highlight"
    :style="gridCellStyle"
  />

  <!-- Free 容器：十字线 -->
  <div
    v-if="hint?.insertRule === 'absolute_position' && hint.visualHint"
    class="crosshair"
    :style="crosshairStyle"
  >
    <div class="crosshair-h" />
    <div class="crosshair-v" />
    <div class="crosshair-center" />
  </div>
</template>

<style scoped>
/* Flex 容器：插入线 */
.insert-line {
  position: absolute;
  background: #3b82f6;
  pointer-events: none;
  z-index: 9999;
  transition: all 0.1s ease;
}

.insert-line.horizontal {
  height: 2px;
  left: 0;
  right: 0;
}

.insert-line.vertical {
  width: 2px;
  top: 0;
  bottom: 0;
}

/* Grid 容器：单元格高亮 */
.grid-cell-highlight {
  position: absolute;
  background-color: rgba(59, 130, 246, 0.2);
  border: 2px solid #3b82f6;
  pointer-events: none;
  z-index: 9998;
  transition: all 0.1s ease;
  box-sizing: border-box;
}

/* Free 容器：十字线 */
.crosshair {
  position: absolute;
  pointer-events: none;
  z-index: 9999;
  transform: translate(-50%, -50%);
}

.crosshair-h {
  position: absolute;
  left: -50vw;
  right: -50vw;
  top: 0;
  height: 1px;
  background: #3b82f6;
  opacity: 0.6;
}

.crosshair-v {
  position: absolute;
  top: -50vh;
  bottom: -50vh;
  left: 0;
  width: 1px;
  background: #3b82f6;
  opacity: 0.6;
}

.crosshair-center {
  position: absolute;
  left: -3px;
  top: -3px;
  width: 6px;
  height: 6px;
  background: #3b82f6;
  border: 2px solid #ffffff;
  border-radius: 50%;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}
</style>
