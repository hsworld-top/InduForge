/**
 * useDragDrop Composable - 拖拽逻辑
 * 处理组件拖拽位置和调整大小计算
 * Requirements: 3.4, 3.5
 */
import { ref, reactive } from 'vue'
import { snapToGrid } from './useCanvas'

/**
 * 最小尺寸常量
 */
const MIN_WIDTH = 20
const MIN_HEIGHT = 20

/**
 * 计算拖拽后的位置
 * Property 4: Drag Position Update
 * For initial position (x, y) and drag delta (dx, dy), result = (x + dx, y + dy)
 * 
 * @param {Object} initialPosition - 初始位置 { x, y } 或 { left, top }
 * @param {Object} delta - 拖拽增量 { dx, dy }
 * @returns {Object} 新位置 { left, top }
 */
export function calculateDragPosition(initialPosition, delta) {
  // 支持 { x, y } 或 { left, top } 格式
  const x = initialPosition.x ?? initialPosition.left ?? 0
  const y = initialPosition.y ?? initialPosition.top ?? 0
  const dx = delta.dx ?? delta.x ?? 0
  const dy = delta.dy ?? delta.y ?? 0
  
  return {
    left: x + dx,
    top: y + dy,
  }
}

/**
 * 计算调整大小后的尺寸
 * Property 5: Resize Dimension Update
 * For initial dimensions (w, h) and resize delta (dw, dh), 
 * result = (max(minWidth, w + dw), max(minHeight, h + dh))
 * 
 * @param {Object} initialDimensions - 初始尺寸 { width, height }
 * @param {Object} delta - 调整增量 { dw, dh }
 * @param {Object} options - 选项 { minWidth, minHeight }
 * @returns {Object} 新尺寸 { width, height }
 */
export function calculateResizeDimensions(initialDimensions, delta, options = {}) {
  const { minWidth = MIN_WIDTH, minHeight = MIN_HEIGHT } = options
  const w = initialDimensions.width ?? 0
  const h = initialDimensions.height ?? 0
  const dw = delta.dw ?? delta.width ?? 0
  const dh = delta.dh ?? delta.height ?? 0
  
  return {
    width: Math.max(minWidth, w + dw),
    height: Math.max(minHeight, h + dh),
  }
}

/**
 * useDragDrop Composable
 * 提供拖拽和调整大小的状态和操作方法
 */
export function useDragDrop(options = {}) {
  const {
    gridSize = 10,
    snapEnabled = true,
    minWidth = MIN_WIDTH,
    minHeight = MIN_HEIGHT,
  } = options
  
  // 拖拽状态
  const isDragging = ref(false)
  const isResizing = ref(false)
  
  // 拖拽起始信息
  const dragStart = reactive({
    x: 0,
    y: 0,
    componentLeft: 0,
    componentTop: 0,
    componentWidth: 0,
    componentHeight: 0,
    handle: null, // 调整大小的手柄位置
  })
  
  // 当前拖拽增量
  const dragDelta = reactive({
    dx: 0,
    dy: 0,
  })
  
  /**
   * 开始拖拽
   * @param {Object} event - 鼠标事件或触摸事件
   * @param {Object} componentStyle - 组件当前样式 { left, top, width, height }
   */
  function startDrag(event, componentStyle) {
    isDragging.value = true
    isResizing.value = false
    
    const clientX = event.clientX ?? event.touches?.[0]?.clientX ?? 0
    const clientY = event.clientY ?? event.touches?.[0]?.clientY ?? 0
    
    dragStart.x = clientX
    dragStart.y = clientY
    dragStart.componentLeft = componentStyle.left ?? 0
    dragStart.componentTop = componentStyle.top ?? 0
    dragStart.componentWidth = componentStyle.width ?? 0
    dragStart.componentHeight = componentStyle.height ?? 0
    dragStart.handle = null
    
    dragDelta.dx = 0
    dragDelta.dy = 0
  }
  
  /**
   * 开始调整大小
   * @param {Object} event - 鼠标事件或触摸事件
   * @param {Object} componentStyle - 组件当前样式
   * @param {string} handle - 手柄位置 ('n', 's', 'e', 'w', 'ne', 'nw', 'se', 'sw')
   */
  function startResize(event, componentStyle, handle) {
    isDragging.value = false
    isResizing.value = true
    
    const clientX = event.clientX ?? event.touches?.[0]?.clientX ?? 0
    const clientY = event.clientY ?? event.touches?.[0]?.clientY ?? 0
    
    dragStart.x = clientX
    dragStart.y = clientY
    dragStart.componentLeft = componentStyle.left ?? 0
    dragStart.componentTop = componentStyle.top ?? 0
    dragStart.componentWidth = componentStyle.width ?? 0
    dragStart.componentHeight = componentStyle.height ?? 0
    dragStart.handle = handle
    
    dragDelta.dx = 0
    dragDelta.dy = 0
  }
  
  /**
   * 更新拖拽位置
   * @param {Object} event - 鼠标事件或触摸事件
   */
  function updateDrag(event) {
    if (!isDragging.value && !isResizing.value) return
    
    const clientX = event.clientX ?? event.touches?.[0]?.clientX ?? 0
    const clientY = event.clientY ?? event.touches?.[0]?.clientY ?? 0
    
    dragDelta.dx = clientX - dragStart.x
    dragDelta.dy = clientY - dragStart.y
  }
  
  /**
   * 结束拖拽/调整大小
   */
  function endDrag() {
    isDragging.value = false
    isResizing.value = false
  }
  
  /**
   * 获取拖拽后的新位置
   * @param {boolean} applySnap - 是否应用网格吸附
   * @returns {Object} 新位置 { left, top }
   */
  function getDragResult(applySnap = snapEnabled) {
    const newPosition = calculateDragPosition(
      { left: dragStart.componentLeft, top: dragStart.componentTop },
      dragDelta
    )
    
    if (applySnap && gridSize > 0) {
      const snapped = snapToGrid({ x: newPosition.left, y: newPosition.top }, gridSize)
      return { left: snapped.x, top: snapped.y }
    }
    
    return newPosition
  }
  
  /**
   * 获取调整大小后的新样式
   * @param {boolean} applySnap - 是否应用网格吸附
   * @returns {Object} 新样式 { left, top, width, height }
   */
  function getResizeResult(applySnap = snapEnabled) {
    const handle = dragStart.handle
    let newLeft = dragStart.componentLeft
    let newTop = dragStart.componentTop
    let newWidth = dragStart.componentWidth
    let newHeight = dragStart.componentHeight
    
    // 根据手柄位置计算新尺寸
    if (handle?.includes('e')) {
      newWidth = Math.max(minWidth, dragStart.componentWidth + dragDelta.dx)
    }
    if (handle?.includes('w')) {
      const widthDelta = -dragDelta.dx
      newWidth = Math.max(minWidth, dragStart.componentWidth + widthDelta)
      // 只有当宽度实际改变时才移动位置
      if (newWidth > minWidth || dragStart.componentWidth + widthDelta >= minWidth) {
        newLeft = dragStart.componentLeft + dragDelta.dx
      }
    }
    if (handle?.includes('s')) {
      newHeight = Math.max(minHeight, dragStart.componentHeight + dragDelta.dy)
    }
    if (handle?.includes('n')) {
      const heightDelta = -dragDelta.dy
      newHeight = Math.max(minHeight, dragStart.componentHeight + heightDelta)
      // 只有当高度实际改变时才移动位置
      if (newHeight > minHeight || dragStart.componentHeight + heightDelta >= minHeight) {
        newTop = dragStart.componentTop + dragDelta.dy
      }
    }
    
    // 应用网格吸附
    if (applySnap && gridSize > 0) {
      const snappedPos = snapToGrid({ x: newLeft, y: newTop }, gridSize)
      const snappedSize = snapToGrid({ x: newWidth, y: newHeight }, gridSize)
      return {
        left: snappedPos.x,
        top: snappedPos.y,
        width: Math.max(minWidth, snappedSize.x),
        height: Math.max(minHeight, snappedSize.y),
      }
    }
    
    return {
      left: newLeft,
      top: newTop,
      width: newWidth,
      height: newHeight,
    }
  }
  
  return {
    // 状态
    isDragging,
    isResizing,
    dragStart,
    dragDelta,
    
    // 方法
    startDrag,
    startResize,
    updateDrag,
    endDrag,
    getDragResult,
    getResizeResult,
    
    // 工具函数
    calculateDragPosition,
    calculateResizeDimensions,
  }
}

export default useDragDrop
