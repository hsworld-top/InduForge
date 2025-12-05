<template>
  <div 
    ref="viewportRef"
    class="design-canvas-viewport"
    @click.self="handleCanvasClick"
    @mousedown.self="handleCanvasMouseDown"
  >
    <!-- 画布容器 -->
    <div 
      ref="canvasRef"
      class="design-canvas"
      :style="canvasStyle"
    >
      <!-- 网格背景 -->
      <div 
        v-if="showGrid && pageConfig?.snapToGrid"
        class="canvas-grid"
        :style="gridStyle"
      />
      
      <!-- 组件渲染区域 -->
      <div class="canvas-content">
        <CanvasComponent
          v-for="component in components"
          :key="component.id"
          :component="component"
          :scale="canvasState.scale"
          @select="handleComponentSelect"
        />
      </div>
      
      <!-- 选择覆盖层 -->
      <SelectionOverlay
        v-if="selectedComponent"
        :component="selectedComponent"
        :scale="canvasState.scale"
        @drag="handleDrag"
        @resize="handleResize"
      />
    </div>
  </div>
</template>

<script setup>
/**
 * DesignCanvas - 设计画布组件
 * 实现画布容器，支持缩放和平移
 * Requirements: 2.3, 2.4
 */
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useDesignStore } from '@/store/design'
import { useCanvas } from '@/composables/useCanvas'
import { useSelection } from '@/composables/useSelection'
import CanvasComponent from './CanvasComponent.vue'
import SelectionOverlay from './SelectionOverlay.vue'

// Props
const props = defineProps({
  showGrid: {
    type: Boolean,
    default: true,
  },
})

// Store
const designStore = useDesignStore()

// Composables
const { 
  canvasState, 
  updateViewportSize, 
  updateCanvasSize, 
  updateScaleMode,
  setGridSize,
  setSnapToGrid,
} = useCanvas()

const { deselect } = useSelection()

// Refs
const viewportRef = ref(null)
const canvasRef = ref(null)

// Computed
const pageConfig = computed(() => designStore.pageConfig)
const components = computed(() => designStore.components)
const selectedComponent = computed(() => designStore.selectedComponent)

/**
 * 画布样式
 * Requirements: 2.3 - 渲染 page config 指定的宽高和背景
 * Requirements: 2.4 - 支持缩放
 */
const canvasStyle = computed(() => {
  const config = pageConfig.value
  if (!config) {
    return {
      width: '1920px',
      height: '1080px',
      backgroundColor: '#ffffff',
      transform: `scale(${canvasState.scale})`,
      transformOrigin: 'center center',
    }
  }
  
  return {
    width: `${config.width || 1920}px`,
    height: `${config.height || 1080}px`,
    backgroundColor: config.backgroundColor || '#ffffff',
    transform: `scale(${canvasState.scale})`,
    transformOrigin: 'center center',
  }
})

/**
 * 网格样式
 */
const gridStyle = computed(() => {
  const gridSize = pageConfig.value?.gridSize || 10
  const scaledGridSize = gridSize * canvasState.scale
  
  return {
    backgroundSize: `${scaledGridSize}px ${scaledGridSize}px`,
    backgroundImage: `
      linear-gradient(to right, rgba(0, 0, 0, 0.05) 1px, transparent 1px),
      linear-gradient(to bottom, rgba(0, 0, 0, 0.05) 1px, transparent 1px)
    `,
  }
})

// Methods

/**
 * 处理画布空白区域点击
 * Requirements: 3.2 - 点击空白区域取消选择
 */
function handleCanvasClick() {
  deselect()
}

/**
 * 处理画布鼠标按下（用于平移等）
 */
function handleCanvasMouseDown(event) {
  // 预留平移功能
}

/**
 * 处理组件选择
 */
function handleComponentSelect(componentId) {
  designStore.selectComponent(componentId)
}

/**
 * 处理组件拖拽
 */
function handleDrag({ componentId, left, top }) {
  designStore.updateComponent(componentId, {
    style: { left, top },
  })
}

/**
 * 处理组件调整大小
 */
function handleResize({ componentId, left, top, width, height }) {
  designStore.updateComponent(componentId, {
    style: { left, top, width, height },
  })
}

/**
 * 更新视口尺寸
 */
function updateViewport() {
  if (viewportRef.value) {
    const rect = viewportRef.value.getBoundingClientRect()
    updateViewportSize({ width: rect.width, height: rect.height })
  }
}

/**
 * 同步页面配置到画布状态
 */
function syncPageConfig() {
  const config = pageConfig.value
  if (config) {
    updateCanvasSize({ width: config.width || 1920, height: config.height || 1080 })
    updateScaleMode(config.scaleMode || 'fit')
    setGridSize(config.gridSize || 10)
    setSnapToGrid(config.snapToGrid !== false)
  }
}

// Watchers
watch(pageConfig, syncPageConfig, { immediate: true, deep: true })

// Lifecycle
onMounted(() => {
  updateViewport()
  window.addEventListener('resize', updateViewport)
})

onUnmounted(() => {
  window.removeEventListener('resize', updateViewport)
})
</script>

<style scoped>
.design-canvas-viewport {
  width: 100%;
  height: 100%;
  overflow: auto;
  background-color: #f0f2f5;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.design-canvas {
  position: relative;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  transition: transform 0.1s ease-out;
}

.canvas-grid {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
}

.canvas-content {
  position: relative;
  width: 100%;
  height: 100%;
}
</style>
