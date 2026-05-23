<!--
  DropIndicator - 拖拽放置指示器
  根据 hint 显示：Flex 插入线、Grid 单元格高亮、Free 十字线
-->
<script setup lang="ts">
import { computed } from 'vue'

/**
 * Ghost 拖影位置接口
 */
interface GhostPosition {
  x: number
  y: number
  width: number
  height: number
}

interface DropIndicatorVisualHint {
  orientation?: 'horizontal' | 'vertical'
  offset?: number
  highlightRect?: {
    left: number
    top: number
    width: number
    height: number
  }
  x?: number
  y?: number
}

interface DropIndicatorHint {
  insertRule?: 'before_after' | 'grid_cell' | 'absolute_position'
  visualHint?: DropIndicatorVisualHint | null
}

const props = defineProps<{
  /**
   * 拖拽决策对象
   */
  hint?: DropIndicatorHint | null
  /**
   * Ghost 拖影位置与尺寸（跟随鼠标的半透明组件轮廓）
   */
  ghost?: GhostPosition | null
}>()

/**
 * Ghost 拖影样式（D-07: 透明度 0.5，跟随鼠标）
 */
const ghostStyle = computed(() => {
  if (!props.ghost) return {}
  return {
    left: `${props.ghost.x}px`,
    top: `${props.ghost.y}px`,
    width: `${props.ghost.width}px`,
    height: `${props.ghost.height}px`,
  }
})

/**
 * 插入线样式（Flex 容器）
 */
const insertLineStyle = computed(() => {
  if (!props.hint?.visualHint) return {}

  const { orientation, offset } = props.hint.visualHint

  if (orientation === 'horizontal') {
    return {
      top: `${offset}px`,
    }
  } else {
    return {
      left: `${offset}px`,
    }
  }
})

/**
 * 单元格高亮样式（Grid 容器）
 */
const gridCellStyle = computed(() => {
  if (!props.hint?.visualHint?.highlightRect) return {}

  const { left, top, width, height } = props.hint.visualHint.highlightRect

  return {
    left: `${left}px`,
    top: `${top}px`,
    width: `${width}px`,
    height: `${height}px`,
  }
})

/**
 * 十字线样式（Free 容器）
 */
const crosshairStyle = computed(() => {
  if (!props.hint?.visualHint) return {}

  const { x, y } = props.hint.visualHint

  return {
    left: `${x}px`,
    top: `${y}px`,
  }
})
</script>

<template>
  <!-- Ghost 拖影（D-07: 跟随鼠标的半透明轮廓） -->
  <div v-if="ghost" class="ghost-drag-shadow" :style="ghostStyle" />

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
/* Ghost 拖影（D-07: 半透明轮廓跟随鼠标） */
.ghost-drag-shadow {
  position: absolute;
  background: rgba(0, 0, 0, 0.5);
  border: 1px dashed #666;
  border-radius: 4px;
  pointer-events: none;
  z-index: 9997;
  opacity: 0.5;
  transition: all 0.1s ease;
}

/* Flex 容器：插入线 */
.insert-line {
  position: absolute;
  background: var(--designer-primary);
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
  border: 2px solid var(--designer-primary);
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
  background: var(--designer-primary);
  opacity: 0.6;
}

.crosshair-v {
  position: absolute;
  top: -50vh;
  bottom: -50vh;
  left: 0;
  width: 1px;
  background: var(--designer-primary);
  opacity: 0.6;
}

.crosshair-center {
  position: absolute;
  left: -3px;
  top: -3px;
  width: 6px;
  height: 6px;
  background: var(--designer-primary);
  border: 2px solid var(--designer-shell-surface);
  border-radius: 50%;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}
</style>
