<!--
  画布插入位置指示线（teleport 到 body，避免被画布 overflow 裁剪）
-->
<script setup lang="ts">
import type { CanvasInsertLineBox, CanvasInsertLineStyle } from "./canvas-internal.types";

defineProps<{
  show: boolean;
  lineStyle: CanvasInsertLineStyle | null;
  lineBox: CanvasInsertLineBox | null;
}>();
</script>

<template>
  <teleport v-if="show && lineStyle && lineBox" to="body">
    <div
      class="canvas-insert-line"
      :class="lineStyle.orientation"
      :style="{
        left:
          lineStyle.orientation === 'vertical'
            ? `${lineBox.left + lineStyle.offset}px`
            : `${lineBox.left}px`,
        top:
          lineStyle.orientation === 'horizontal'
            ? `${lineBox.top + lineStyle.offset}px`
            : `${lineBox.top}px`,
        width: lineStyle.orientation === 'vertical' ? '2px' : `${lineBox.width}px`,
        height: lineStyle.orientation === 'horizontal' ? '2px' : `${lineBox.height}px`,
      }"
    />
  </teleport>
</template>

<style scoped>
.canvas-insert-line {
  position: absolute;
  background: var(--designer-primary);
  pointer-events: none;
  z-index: 9999;
  transition: all 0.1s ease;
  box-shadow: 0 0 4px rgba(64, 158, 255, 0.4);
}

.canvas-insert-line.horizontal {
  height: 2px;
  left: 0;
  right: 0;
}

.canvas-insert-line.vertical {
  width: 2px;
  top: 0;
  bottom: 0;
}
</style>
