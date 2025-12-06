<template>
  <div class="canvas-ruler-wrapper">
    <SketchRule
      :thick="20"
      :scale="scale"
      :width="width"
      :height="height"
      :start-x="scrollLeft"
      :start-y="scrollTop"
      :palette="palette"
    />
  </div>
</template>

<script setup>
/**
 * CanvasRuler - 画布标尺组件
 * 使用 vue3-sketch-ruler 开源库实现
 */
import { ref, computed, onMounted, onUnmounted } from 'vue'
import SketchRule from 'vue3-sketch-ruler'
import 'vue3-sketch-ruler/lib/style.css'

// Props
const props = defineProps({
  scale: {
    type: Number,
    default: 1,
  },
  scrollLeft: {
    type: Number,
    default: 0,
  },
  scrollTop: {
    type: Number,
    default: 0,
  },
})

// State
const width = ref(2000)
const height = ref(2000)

// 标尺配色方案 - 匹配 OpenTiny 风格
const palette = {
  bgColor: '#e4e7ed',           // 背景色
  longfgColor: '#909399',       // 长刻度颜色
  shortfgColor: '#dcdfe6',      // 短刻度颜色
  fontColor: '#575d6c',         // 文字颜色
  shadowColor: 'transparent',   // 阴影颜色
  lineColor: '#5e7ce0',         // 辅助线颜色
  borderColor: '#dcdfe6',       // 边框颜色
  cornerActiveColor: '#5e7ce0', // 角落激活颜色
}

/**
 * 更新尺寸
 */
function updateSize() {
  width.value = window.innerWidth
  height.value = window.innerHeight
}

// Lifecycle
onMounted(() => {
  updateSize()
  window.addEventListener('resize', updateSize)
})

onUnmounted(() => {
  window.removeEventListener('resize', updateSize)
})
</script>

<style scoped>
.canvas-ruler-wrapper {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  z-index: 10;
}

/* 覆盖默认样式以匹配 OpenTiny 风格 */
:deep(.sketch-ruler) {
  font-family: 'Inter', system-ui, sans-serif;
}
</style>
