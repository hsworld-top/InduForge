<template>
  <div 
    ref="viewportRef"
    class="design-canvas-viewport"
  >
    <!-- Konva Canvas 容器 -->
    <div 
      ref="canvasContainerRef"
      class="design-canvas-container"
    />
  </div>
</template>

<script setup>
/**
 * DesignCanvas - 设计画布组件（Konva 版本）
 * 使用 Konva.js 实现高性能 Canvas 渲染
 * Requirements: 2.3, 2.4, 8.2
 */
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useDesignStore } from '@/store/design'
import { useCanvas } from '@/composables/useCanvas'
import { CanvasEngine } from '@/engine/canvas'

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
const { canvasState } = useCanvas()

// Refs
const viewportRef = ref(null)
const canvasContainerRef = ref(null)

// Canvas Engine
let canvasEngine = null

// Computed
const pageConfig = computed(() => designStore.pageConfig)
const components = computed(() => designStore.components)
const selectedComponentId = computed(() => designStore.selectedComponentId)

/**
 * 初始化 Canvas 引擎
 */
async function initCanvasEngine() {
  if (!canvasContainerRef.value) return
  
  await nextTick()
  
  const config = pageConfig.value || {}
  
  // 创建 Canvas 引擎
  canvasEngine = new CanvasEngine(canvasContainerRef.value, {
    width: config.width || 1920,
    height: config.height || 1080,
  })
  
  // 监听选择变化事件
  canvasEngine.on('selection:change', ({ ids }) => {
    if (ids.length === 1) {
      designStore.selectComponent(ids[0])
    } else if (ids.length === 0) {
      designStore.selectComponent(null)
    }
  })
  
  // 监听组件更新事件
  canvasEngine.on('component:update', ({ id, updates }) => {
    designStore.updateComponent(id, updates)
    // 保存历史记录
    designStore.saveHistory(`更新组件 ${id}`)
  })
  
  // 监听组件点击事件
  canvasEngine.on('component:click', ({ id }) => {
    designStore.selectComponent(id)
  })
  
  // 渲染所有组件
  renderAllComponents()
  
  // 设置吸附
  canvasEngine.setSnapEnabled(config.snapToGrid !== false)
  canvasEngine.setSnapThreshold(config.gridSize || 10)
  
  console.log('✅ Canvas Engine initialized')
}

/**
 * 渲染所有组件
 */
function renderAllComponents() {
  if (!canvasEngine) return
  
  // 清空画布
  canvasEngine.clear()
  
  // 渲染所有组件
  const comps = components.value || []
  comps.forEach(component => {
    canvasEngine.renderComponent(component)
  })
  
  // 恢复选择
  if (selectedComponentId.value) {
    canvasEngine.selectComponents(selectedComponentId.value)
  }
}

/**
 * 更新画布配置
 */
function updateCanvasConfig() {
  if (!canvasEngine) return
  
  const config = pageConfig.value || {}
  
  // 更新画布大小
  canvasEngine.resize(config.width || 1920, config.height || 1080)
  
  // 更新吸附设置
  canvasEngine.setSnapEnabled(config.snapToGrid !== false)
  canvasEngine.setSnapThreshold(config.gridSize || 10)
}

/**
 * 更新缩放
 */
function updateScale() {
  if (!canvasEngine) return
  canvasEngine.setScale(canvasState.scale)
}

// Watchers

// 监听页面配置变化
watch(pageConfig, () => {
  updateCanvasConfig()
}, { deep: true })

// 监听组件列表变化
watch(components, (newComponents, oldComponents) => {
  if (!canvasEngine) return
  
  // 简单策略：重新渲染所有组件
  // TODO: 优化为增量更新
  renderAllComponents()
}, { deep: true })

// 监听选中组件变化
watch(selectedComponentId, (newId, oldId) => {
  if (!canvasEngine) return
  
  if (newId) {
    canvasEngine.selectComponents(newId)
  } else {
    canvasEngine.clearSelection()
  }
})

// 监听缩放变化
watch(() => canvasState.scale, () => {
  updateScale()
})

// Lifecycle
onMounted(async () => {
  await initCanvasEngine()
})

onUnmounted(() => {
  if (canvasEngine) {
    canvasEngine.destroy()
    canvasEngine = null
  }
})

// 暴露方法给父组件
defineExpose({
  canvasEngine,
  renderAllComponents,
  updateCanvasConfig,
})
</script>

<style scoped>
.design-canvas-viewport {
  width: 100%;
  height: 100%;
  overflow: auto;
  background-color: #f5f5f5;
  background-image: 
    linear-gradient(to right, #e0e0e0 1px, transparent 1px),
    linear-gradient(to bottom, #e0e0e0 1px, transparent 1px);
  background-size: 20px 20px;
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
  padding: 40px;
}

.design-canvas-container {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.12);
  background-color: #ffffff;
}
</style>
