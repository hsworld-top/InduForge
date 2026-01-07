<template>
  <main class="canvas-container">
    <div class="canvas-wrapper">
      <div
        class="canvas"
        :style="{
          width: `${width}px`,
          height: `${height}px`,
          transform: `scale(${zoom})`,
        }"
      >
        <div class="absolute inset-0 pointer-events-none">
          <slot name="canvas-layer" />
        </div>
        <div class="absolute inset-0">
          <slot>
            <div
              class="flex flex-col items-center justify-center h-full text-gray-400"
            >
              <IconEpPlus class="text-5xl mb-4" />
              <p>从左侧拖拽组件到画布</p>
            </div>
          </slot>
        </div>
      </div>
    </div>
    <div class="canvas-statusbar">
      <span>画布: {{ width }} × {{ height }}</span>
      <el-divider direction="vertical" />
      <span>缩放: {{ Math.round(zoom * 100) }}%</span>
    </div>
  </main>
</template>

<script setup>
import { toRefs } from "vue";
import IconEpPlus from "~icons/ep/plus";

const props = defineProps({
  width: {
    type: Number,
    default: 1920,
  },
  height: {
    type: Number,
    default: 1080,
  },
  zoom: {
    type: Number,
    default: 1,
  },
});

const { width, height, zoom } = toRefs(props);
</script>

<style scoped>
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
</style>
