<!--
  Diagram2D - 2D 流程图组件
  Canvas 矢量绘图，双击进入编辑模式，支持线、矩形、圆等图元
-->
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import IconEpGrid from '~icons/ep/grid'

interface Diagram2DNodeLike {
  id: string
  props?: {
    diagramId?: string
    background?: string
    showGrid?: boolean
    gridSize?: number
  }
  absolutePos?: {
    w?: number
    h?: number
  }
}

interface Diagram2DEnterCanvasModePayload {
  nodeId: string
  diagramId: string
}

interface Diagram2DShapeLike {
  hidden?: boolean
  type?: string
  data?: {
    x?: number
    y?: number
    width?: number
    height?: number
    cx?: number
    cy?: number
    radius?: number
  }
  style?: {
    fill?: string
    stroke?: string
    strokeWidth?: number
  }
}

interface Diagram2DDataLike {
  shapes?: Diagram2DShapeLike[]
}

const props = defineProps<{
  /**
   * 节点数据
   */
  node: Diagram2DNodeLike
}>()

const emit = defineEmits<{
  (event: 'enterCanvasMode', payload: Diagram2DEnterCanvasModePayload): void
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)

/**
 * 绘图数据
 */
const diagramId = computed(() => props.node.props?.diagramId || '')
const diagramData = computed<Diagram2DDataLike | null>(() => {
  // TODO: 从 store 获取绘图数据
  return null
})

const isEmpty = computed(() => {
  return !diagramData.value || !diagramData.value.shapes?.length
})

const shapeCount = computed(() => {
  return diagramData.value?.shapes?.length || 0
})

/**
 * 组件样式
 */
const componentStyle = computed<Record<string, string>>(() => {
  const { background, showGrid, gridSize } = props.node.props || {}

  const style: Record<string, string> = {
    background: background || '#ffffff',
  }

  if (showGrid) {
    const size = gridSize || 10
    style.backgroundImage = `
      linear-gradient(rgba(0, 0, 0, 0.05) 1px, transparent 1px),
      linear-gradient(90deg, rgba(0, 0, 0, 0.05) 1px, transparent 1px)
    `
    style.backgroundSize = `${size}px ${size}px`
  }

  return style
})

/**
 * Canvas 尺寸
 */
const canvasWidth = computed(() => {
  return props.node.absolutePos?.w || 400
})

const canvasHeight = computed(() => {
  return props.node.absolutePos?.h || 300
})

/**
 * 双击进入Canvas编辑模式
 */
function handleDoubleClick(): void {
  emit('enterCanvasMode', {
    nodeId: props.node.id,
    diagramId: diagramId.value,
  })
}

/**
 * 渲染图元到 Canvas（预览）
 */
function renderShapes(): void {
  if (!canvasRef.value || !diagramData.value) return

  const ctx = canvasRef.value.getContext('2d')
  if (!ctx) return

  // 清空画布
  ctx.clearRect(0, 0, canvasWidth.value, canvasHeight.value)

  // 渲染图元（简化版预览）
  const shapes = diagramData.value.shapes || []
  for (const shape of shapes) {
    if (shape.hidden) continue
    const data = shape.data
    if (!data) continue

    // TODO: 根据图元类型渲染
    ctx.save()
    ctx.fillStyle = shape.style?.fill || '#cccccc'
    ctx.strokeStyle = shape.style?.stroke || '#000000'
    ctx.lineWidth = shape.style?.strokeWidth || 1

    switch (shape.type) {
      case 'rect':
        ctx.fillRect(data.x ?? 0, data.y ?? 0, data.width ?? 0, data.height ?? 0)
        ctx.strokeRect(data.x ?? 0, data.y ?? 0, data.width ?? 0, data.height ?? 0)
        break
      case 'circle':
        ctx.beginPath()
        ctx.arc(data.cx ?? 0, data.cy ?? 0, data.radius ?? 0, 0, 2 * Math.PI)
        ctx.fill()
        ctx.stroke()
        break
      // ... 其他图元类型
    }

    ctx.restore()
  }
}

watch(diagramData, renderShapes, { deep: true })

onMounted(() => {
  renderShapes()
})
</script>

<template>
  <div class="diagram-2d-component" :style="componentStyle" @dblclick.stop="handleDoubleClick">
    <!-- 预览渲染区域 -->
    <canvas ref="canvasRef" class="diagram-canvas" :width="canvasWidth" :height="canvasHeight" />

    <!-- 空状态提示 -->
    <div v-if="isEmpty" class="empty-hint">
      <IconEpGrid class="empty-icon" />
      <div class="empty-text">双击进入Canvas编辑模式</div>
      <div class="empty-subtext">在此绘制流程图、图表等</div>
    </div>

    <!-- 图元数量提示 -->
    <div v-else class="shape-count-badge">{{ shapeCount }} 个图元</div>
  </div>
</template>

<style scoped>
.diagram-2d-component {
  position: relative;
  width: 100%;
  height: 100%;
  min-width: 200px;
  min-height: 150px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s;
}

.diagram-2d-component:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.diagram-canvas {
  width: 100%;
  height: 100%;
  display: block;
}

.empty-hint {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--el-text-color-placeholder);
  pointer-events: none;
}

.empty-icon {
  font-size: 48px;
  opacity: 0.5;
}

.empty-text {
  font-size: 14px;
  font-weight: 500;
}

.empty-subtext {
  font-size: 12px;
  opacity: 0.7;
}

.shape-count-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 4px 8px;
  background-color: rgba(0, 0, 0, 0.6);
  color: #ffffff;
  font-size: 12px;
  border-radius: 4px;
  pointer-events: none;
}
</style>
