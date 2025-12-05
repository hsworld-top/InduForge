<template>
  <div
    class="selection-overlay"
    :style="overlayStyle"
  >
    <!-- 选择边框 - 可拖拽区域 -->
    <div 
      class="selection-border" 
      @mousedown.stop="handleMouseDown"
    />
    
    <!-- 调整大小手柄 -->
    <!-- Requirements: 2.5 - 显示选中组件的调整手柄 -->
    <div
      v-for="handle in resizeHandles"
      :key="handle.position"
      class="resize-handle"
      :class="`handle-${handle.position}`"
      :style="handle.style"
      @mousedown.stop="(e) => handleResizeStart(e, handle.position)"
    />
    
    <!-- 组件信息提示 -->
    <div class="component-info">
      <span class="component-type">{{ component.type }}</span>
      <span class="component-size">{{ Math.round(component.style?.width || 0) }} × {{ Math.round(component.style?.height || 0) }}</span>
    </div>
  </div>
</template>

<script setup>
/**
 * SelectionOverlay - 选择覆盖层组件
 * 显示选中组件的边框和调整手柄
 * Requirements: 2.5, 3.4, 3.5
 */
import { computed, onUnmounted } from 'vue'
import { useDragDrop } from '@/composables/useDragDrop'
import { useCanvas } from '@/composables/useCanvas'
import { useDesignStore } from '@/store/design'

// Props
const props = defineProps({
  component: {
    type: Object,
    required: true,
  },
  scale: {
    type: Number,
    default: 1,
  },
})

// Emits
const emit = defineEmits(['drag', 'resize'])

// Store
const designStore = useDesignStore()

// Composables
const { canvasState, applySnapToGrid } = useCanvas()
const {
  isDragging,
  isResizing,
  startDrag,
  startResize,
  updateDrag,
  endDrag,
  getDragResult,
  getResizeResult,
} = useDragDrop({
  gridSize: canvasState.gridSize,
  snapEnabled: canvasState.snapToGrid,
})

// Computed

/**
 * 覆盖层样式
 * Requirements: 2.5 - 显示选中组件的边框
 */
const overlayStyle = computed(() => {
  const style = props.component.style || {}
  
  return {
    position: 'absolute',
    left: typeof style.left === 'number' ? `${style.left}px` : style.left || '0px',
    top: typeof style.top === 'number' ? `${style.top}px` : style.top || '0px',
    width: typeof style.width === 'number' ? `${style.width}px` : style.width || '100px',
    height: typeof style.height === 'number' ? `${style.height}px` : style.height || '100px',
    zIndex: 9999,
    pointerEvents: 'auto',
  }
})

/**
 * 调整大小手柄配置
 */
const resizeHandles = computed(() => {
  const handleSize = 8
  const halfSize = handleSize / 2
  
  return [
    // 四角
    { position: 'nw', style: { top: `-${halfSize}px`, left: `-${halfSize}px`, cursor: 'nw-resize' } },
    { position: 'ne', style: { top: `-${halfSize}px`, right: `-${halfSize}px`, cursor: 'ne-resize' } },
    { position: 'sw', style: { bottom: `-${halfSize}px`, left: `-${halfSize}px`, cursor: 'sw-resize' } },
    { position: 'se', style: { bottom: `-${halfSize}px`, right: `-${halfSize}px`, cursor: 'se-resize' } },
    // 四边中点
    { position: 'n', style: { top: `-${halfSize}px`, left: '50%', transform: 'translateX(-50%)', cursor: 'n-resize' } },
    { position: 's', style: { bottom: `-${halfSize}px`, left: '50%', transform: 'translateX(-50%)', cursor: 's-resize' } },
    { position: 'w', style: { top: '50%', left: `-${halfSize}px`, transform: 'translateY(-50%)', cursor: 'w-resize' } },
    { position: 'e', style: { top: '50%', right: `-${halfSize}px`, transform: 'translateY(-50%)', cursor: 'e-resize' } },
  ]
})

// Methods

/**
 * 处理鼠标按下（开始拖拽）
 * Requirements: 3.4 - 拖拽选中组件更新位置
 */
function handleMouseDown(event) {
  // 锁定的组件不能拖拽
  if (props.component.locked) {
    return
  }
  
  const style = props.component.style || {}
  startDrag(event, {
    left: style.left || 0,
    top: style.top || 0,
    width: style.width || 100,
    height: style.height || 100,
  })
  
  // 添加全局事件监听
  document.addEventListener('mousemove', handleMouseMove)
  document.addEventListener('mouseup', handleMouseUp)
}

/**
 * 处理调整大小开始
 * Requirements: 3.5 - 调整大小更新尺寸
 */
function handleResizeStart(event, handle) {
  // 锁定的组件不能调整大小
  if (props.component.locked) {
    return
  }
  
  const style = props.component.style || {}
  startResize(event, {
    left: style.left || 0,
    top: style.top || 0,
    width: style.width || 100,
    height: style.height || 100,
  }, handle)
  
  // 添加全局事件监听
  document.addEventListener('mousemove', handleMouseMove)
  document.addEventListener('mouseup', handleMouseUp)
}

/**
 * 处理鼠标移动
 */
function handleMouseMove(event) {
  updateDrag(event)
  
  if (isDragging.value) {
    // 实时更新位置（可选：用于显示预览）
    const result = getDragResult(canvasState.snapToGrid)
    emit('drag', {
      componentId: props.component.id,
      left: result.left,
      top: result.top,
    })
  } else if (isResizing.value) {
    // 实时更新尺寸
    const result = getResizeResult(canvasState.snapToGrid)
    emit('resize', {
      componentId: props.component.id,
      left: result.left,
      top: result.top,
      width: result.width,
      height: result.height,
    })
  }
}

/**
 * 处理鼠标释放
 */
function handleMouseUp() {
  if (isDragging.value) {
    const result = getDragResult(canvasState.snapToGrid)
    emit('drag', {
      componentId: props.component.id,
      left: result.left,
      top: result.top,
    })
  } else if (isResizing.value) {
    const result = getResizeResult(canvasState.snapToGrid)
    emit('resize', {
      componentId: props.component.id,
      left: result.left,
      top: result.top,
      width: result.width,
      height: result.height,
    })
  }
  
  endDrag()
  
  // 移除全局事件监听
  document.removeEventListener('mousemove', handleMouseMove)
  document.removeEventListener('mouseup', handleMouseUp)
}

// Lifecycle
onUnmounted(() => {
  // 确保清理事件监听
  document.removeEventListener('mousemove', handleMouseMove)
  document.removeEventListener('mouseup', handleMouseUp)
})
</script>

<style scoped>
.selection-overlay {
  pointer-events: none;
}

.selection-border {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  border: 2px solid #409eff;
  pointer-events: auto;
  cursor: move;
}

.resize-handle {
  position: absolute;
  width: 8px;
  height: 8px;
  background-color: #409eff;
  border: 1px solid #ffffff;
  border-radius: 1px;
  pointer-events: auto;
}

.resize-handle:hover {
  background-color: #66b1ff;
}

/* 手柄位置样式 */
.handle-nw { cursor: nw-resize; }
.handle-ne { cursor: ne-resize; }
.handle-sw { cursor: sw-resize; }
.handle-se { cursor: se-resize; }
.handle-n { cursor: n-resize; }
.handle-s { cursor: s-resize; }
.handle-w { cursor: w-resize; }
.handle-e { cursor: e-resize; }

.component-info {
  position: absolute;
  top: -24px;
  left: 0;
  display: flex;
  gap: 8px;
  font-size: 11px;
  color: #ffffff;
  background-color: #409eff;
  padding: 2px 6px;
  border-radius: 2px;
  white-space: nowrap;
  pointer-events: none;
}

.component-type {
  font-weight: 500;
}

.component-size {
  opacity: 0.8;
}
</style>
