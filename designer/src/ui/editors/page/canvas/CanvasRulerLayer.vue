<!--
  画布顶部与左侧标尺层（指针十字线与刻度标签）
-->
<script setup lang="ts">
defineProps<{
  rulerInset: number
  rulerXStyle: Record<string, string>
  rulerYStyle: Record<string, string>
  rulerMarksX: number[]
  rulerMarksY: number[]
  pointerXOnRuler: number
  pointerYOnRuler: number
  zoom: number
  translateX: number
  translateY: number
}>()
</script>

<template>
  <div class="ruler-layer" :style="{ '--ruler-size': `${rulerInset}px` }">
    <div class="ruler-zero" />
    <div class="ruler ruler-x" :style="rulerXStyle">
      <div class="ruler-crosshair-x" :style="{ left: `${pointerXOnRuler}px` }" />
      <span
        v-for="mark in rulerMarksX"
        :key="`x-${mark}`"
        class="ruler-label"
        :style="{ left: `${mark * zoom + translateX}px` }"
      >
        {{ mark }}
      </span>
    </div>
    <div class="ruler ruler-y" :style="rulerYStyle">
      <div class="ruler-crosshair-y" :style="{ top: `${pointerYOnRuler}px` }" />
      <span
        v-for="mark in rulerMarksY"
        :key="`y-${mark}`"
        class="ruler-label"
        :style="{ top: `${mark * zoom + translateY}px` }"
      >
        {{ mark }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.ruler-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 5;
  --ruler-size: 18px;
}

.ruler-crosshair-x {
  position: absolute;
  top: 0;
  height: 18px;
  width: 2px;
  background: #409eff;
  opacity: 0.9;
  z-index: 3;
}

.ruler-crosshair-y {
  position: absolute;
  left: 0;
  width: 18px;
  height: 2px;
  background: #409eff;
  opacity: 0.9;
  z-index: 3;
}

.ruler-zero {
  position: absolute;
  left: 0;
  top: 0;
  width: var(--ruler-size);
  height: var(--ruler-size);
  box-sizing: border-box;
  background: #f5f7fa;
  border-right: 1px solid #dcdfe6;
  border-bottom: 1px solid #dcdfe6;
  z-index: 3;
}

.ruler {
  position: absolute;
  color: #606266;
  font-size: 10px;
  background-color: #f5f7fa;
  border-color: #dcdfe6;
  overflow: hidden;
}

.ruler-x {
  left: var(--ruler-size);
  top: 0;
  height: 18px;
  right: 0;
  border-bottom: 1px solid #dcdfe6;
  z-index: 2;
}

.ruler-y {
  left: 0;
  top: var(--ruler-size);
  width: 18px;
  bottom: 0;
  border-right: 1px solid #dcdfe6;
  z-index: 1;
}

.ruler-x::before,
.ruler-x::after,
.ruler-y::before,
.ruler-y::after {
  content: '';
  position: absolute;
  pointer-events: none;
}

.ruler-x::before {
  left: 0;
  right: 0;
  bottom: 0;
  height: 6px;
  background-image: linear-gradient(to right, #c0c4cc 1px, transparent 1px);
  background-size: var(--ruler-minor, 10px) 100%;
  background-position: var(--ruler-offset, 0) 0;
}

.ruler-x::after {
  left: 0;
  right: 0;
  bottom: 0;
  height: 12px;
  background-image: linear-gradient(to right, #909399 1px, transparent 1px);
  background-size: var(--ruler-major, 100px) 100%;
  background-position: var(--ruler-offset, 0) 0;
}

.ruler-y::before {
  top: 0;
  bottom: 0;
  right: 0;
  width: 6px;
  background-image: linear-gradient(to bottom, #c0c4cc 1px, transparent 1px);
  background-size: 100% var(--ruler-minor, 10px);
  background-position: 0 var(--ruler-offset, 0);
}

.ruler-y::after {
  top: 0;
  bottom: 0;
  right: 0;
  width: 12px;
  background-image: linear-gradient(to bottom, #909399 1px, transparent 1px);
  background-size: 100% var(--ruler-major, 100px);
  background-position: 0 var(--ruler-offset, 0);
}

.ruler-label {
  position: absolute;
  padding: 2px 2px 0 2px;
  line-height: 1;
  white-space: nowrap;
}

.ruler-y .ruler-label {
  transform: rotate(-90deg);
  transform-origin: left top;
  left: 2px;
}
</style>
