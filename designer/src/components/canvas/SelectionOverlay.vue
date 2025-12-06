<template>
  <div
    ref="targetRef"
    class="selection-overlay"
    :style="overlayStyle"
  >
    <!-- Moveable 会自动添加控制手柄 -->
  </div>
</template>

<script setup>
/**
 * SelectionOverlay - 选择覆盖层
 * 使用 Moveable 库实现拖拽、缩放、旋转等功能
 */
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import Moveable from 'moveable'

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
const emit = defineEmits(['drag', 'resize', 'rotate'])

// Refs
const targetRef = ref(null)
let moveableInstance = null

// Computed
const overlayStyle = computed(() => {
  const style = props.component.style || {}
  return {
    position: 'absolute',
    left: `${style.left || 0}px`,
    top: `${style.top || 0}px`,
    width: `${style.width || 100}px`,
    height: `${style.height || 100}px`,
    pointerEvents: 'none',
  }
})

// Methods
function initMoveable() {
  if (!targetRef.value) return

  // 销毁旧实例
  if (moveableInstance) {
    moveableInstance.destroy()
  }

  // 创建 Moveable 实例
  moveableInstance = new Moveable(document.body, {
    target: targetRef.value,
    draggable: !props.component.locked,
    resizable: !props.component.locked,
    rotatable: false, // 暂时禁用旋转
    snappable: true,
    snapThreshold: 5,
    isDisplaySnapDigit: true,
    snapGap: true,
    snapElement: true,
    snapVertical: true,
    snapHorizontal: true,
    snapCenter: true,
    bounds: { left: 0, top: 0, right: 0, bottom: 0, position: 'css' },
    origin: false,
    keepRatio: false,
    edge: false,
    throttleDrag: 0,
    throttleResize: 0,
    renderDirections: ['nw', 'n', 'ne', 'w', 'e', 'sw', 's', 'se'],
  })

  // 拖拽事件
  moveableInstance.on('drag', ({ target, left, top, transform }) => {
    target.style.left = `${left}px`
    target.style.top = `${top}px`
  })

  moveableInstance.on('dragEnd', ({ target }) => {
    const left = parseFloat(target.style.left)
    const top = parseFloat(target.style.top)
    emit('drag', {
      componentId: props.component.id,
      left: Math.round(left),
      top: Math.round(top),
    })
  })

  // 缩放事件
  moveableInstance.on('resize', ({ target, width, height, drag }) => {
    target.style.width = `${width}px`
    target.style.height = `${height}px`
    target.style.left = `${drag.left}px`
    target.style.top = `${drag.top}px`
  })

  moveableInstance.on('resizeEnd', ({ target }) => {
    const left = parseFloat(target.style.left)
    const top = parseFloat(target.style.top)
    const width = parseFloat(target.style.width)
    const height = parseFloat(target.style.height)
    
    emit('resize', {
      componentId: props.component.id,
      left: Math.round(left),
      top: Math.round(top),
      width: Math.round(width),
      height: Math.round(height),
    })
  })
}

// Watchers
watch(() => props.component, () => {
  nextTick(() => {
    initMoveable()
  })
}, { deep: true })

watch(() => props.component.locked, (locked) => {
  if (moveableInstance) {
    moveableInstance.draggable = !locked
    moveableInstance.resizable = !locked
  }
})

// Lifecycle
onMounted(() => {
  nextTick(() => {
    initMoveable()
  })
})

onUnmounted(() => {
  if (moveableInstance) {
    moveableInstance.destroy()
    moveableInstance = null
  }
})
</script>

<style scoped>
.selection-overlay {
  box-sizing: border-box;
  border: 2px solid #5e7ce0;
  background-color: rgba(94, 124, 224, 0.05);
  z-index: 1000;
}

/* Moveable 样式覆盖 */
:deep(.moveable-control-box) {
  --moveable-color: #5e7ce0;
}

:deep(.moveable-line) {
  background-color: #5e7ce0 !important;
}

:deep(.moveable-control) {
  width: 8px !important;
  height: 8px !important;
  margin-top: -4px !important;
  margin-left: -4px !important;
  background-color: #fff !important;
  border: 2px solid #5e7ce0 !important;
  border-radius: 50% !important;
}

:deep(.moveable-direction) {
  background-color: #5e7ce0 !important;
}
</style>
