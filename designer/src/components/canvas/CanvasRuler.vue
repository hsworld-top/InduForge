<template>
    <div class="canvas-ruler-wrapper">
        <SketchRule :thick="thick" :scale="scale" :width="width" :height="height" :start-x="scrollLeft" :start-y="scrollTop" :palette="palette" :is-show-refer-line="true" />
    </div>
</template>

<script setup>
/**
 * CanvasRuler - 画布标尺组件
 * 使用 vue3-sketch-ruler 开源库实现
 */
import { ref, computed, onMounted, onUnmounted } from 'vue';
import SketchRule from 'vue3-sketch-ruler';
import 'vue3-sketch-ruler/lib/style.css';

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
});

// State
const width = ref(2000);
const height = ref(2000);
const thick = ref(20); // 标尺厚度

// 标尺配色方案 - 匹配 OpenTiny 风格
const palette = {
    bgColor: '#ffffff', // 背景色（标尺条的背景）
    longfgColor: '#909399', // 长刻度颜色
    shortfgColor: '#dcdfe6', // 短刻度颜色
    fontColor: '#575d6c', // 文字颜色
    shadowColor: 'rgba(0,0,0,0.1)', // 阴影颜色
    lineColor: '#5e7ce0', // 辅助线颜色
    borderColor: '#dcdfe6', // 边框颜色
    cornerActiveColor: '#5e7ce0', // 角落激活颜色
};

/**
 * 更新尺寸
 */
function updateSize() {
    width.value = window.innerWidth;
    height.value = window.innerHeight;
}

// Lifecycle
onMounted(() => {
    updateSize();
    window.addEventListener('resize', updateSize);
});

onUnmounted(() => {
    window.removeEventListener('resize', updateSize);
});
</script>

<style scoped>
.canvas-ruler-wrapper {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    pointer-events: none;
    z-index: 100;
}

/* 覆盖默认样式以匹配 OpenTiny 风格 */
:deep(.sketch-ruler) {
    font-family: 'Inter', system-ui, sans-serif;
    background: transparent !important;
}

/* 标尺的刻度容器可以接收事件 */
:deep(.h-container),
:deep(.v-container),
:deep(.corner) {
    pointer-events: auto;
    background-color: #ffffff;
}

/* 关键修复：标尺的主容器（canvasedit-parent）必须透明 */
:deep(.canvasedit-parent) {
    background: transparent !important;
    pointer-events: none !important;
}

/* 标尺的画布编辑区域也要透明 */
:deep(.canvasedit) {
    background: transparent !important;
    pointer-events: none !important;
}

/* 标尺的主体区域（画布区域）完全透明，不遮挡内容 */
:deep(.ruler-wrapper) {
    pointer-events: none;
    background: transparent !important;
}

/* 确保标尺的画布区域透明 */
:deep(.sketch-ruler > canvas) {
    background: transparent !important;
}

/* 标尺容器的背景透明 */
:deep(.sketch-ruler-container) {
    background: transparent !important;
}
</style>
