import type { DeepReadonly } from 'vue'
/**
 * 拖拽状态管理
 * 管理组件拖拽时的放置指示器状态
 */
import { reactive, readonly } from 'vue'

export interface DragIndicatorPosition {
  x: number
  y: number
  width: number
  height: number
}

export interface DragStateShape {
  isDragging: boolean
  dragType: string
  targetContainerId: string
  insertIndex: number
  indicatorPosition: DragIndicatorPosition
  layoutType: string
  direction: string
}

const state = reactive<DragStateShape>({
  isDragging: false,
  dragType: '',
  targetContainerId: '',
  insertIndex: -1,
  indicatorPosition: {
    x: 0,
    y: 0,
    width: 0,
    height: 0,
  },
  layoutType: 'flex',
  direction: 'column',
})

export function startDrag(componentType: string) {
  state.isDragging = true
  state.dragType = componentType
}

export function endDrag() {
  state.isDragging = false
  state.dragType = ''
  state.targetContainerId = ''
  state.insertIndex = -1
  state.indicatorPosition = { x: 0, y: 0, width: 0, height: 0 }
}

export interface UpdateDropTargetPayload {
  containerId?: string
  insertIndex?: number
  position?: DragIndicatorPosition
  layoutType?: string
  direction?: string
}

export function updateDropTarget(payload: UpdateDropTargetPayload) {
  state.targetContainerId = payload.containerId || ''
  state.insertIndex = payload.insertIndex ?? -1
  state.indicatorPosition = payload.position || {
    x: 0,
    y: 0,
    width: 0,
    height: 0,
  }
  state.layoutType = payload.layoutType || 'flex'
  state.direction = payload.direction || 'column'
}

export function clearDropTarget() {
  state.targetContainerId = ''
  state.insertIndex = -1
  state.indicatorPosition = { x: 0, y: 0, width: 0, height: 0 }
}

export function useDragState(): DeepReadonly<DragStateShape> {
  return readonly(state)
}

export function getDropInfo(): {
  containerId: string
  insertIndex: number
} {
  return {
    containerId: state.targetContainerId,
    insertIndex: state.insertIndex,
  }
}

export default {
  state,
  startDrag,
  endDrag,
  updateDropTarget,
  clearDropTarget,
  useDragState,
  getDropInfo,
}
