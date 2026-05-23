import type { ComputedRef, Ref } from 'vue'
import { ref } from 'vue'

export interface UseCanvasRulerPointerOptions {
  showRuler: Ref<boolean>
  containerRef: Ref<HTMLElement | null>
  zoom: Ref<number>
  translateX: Ref<number>
  translateY: Ref<number>
  rulerInset: Ref<number> | ComputedRef<number>
}

/**
 * 画布标尺：指针位置与节点变换时的十字线同步
 */
export function useCanvasRulerPointer(options: UseCanvasRulerPointerOptions) {
  const { showRuler, containerRef, zoom, translateX, translateY, rulerInset } = options

  const pointerX = ref(-9999)
  const pointerY = ref(-9999)
  const isNodeTransforming = ref(false)

  function handleRulerMouseMove(event: MouseEvent) {
    if (!showRuler.value) return
    if (isNodeTransforming.value) return
    if (!containerRef.value) return
    const rect = containerRef.value.getBoundingClientRect()
    pointerX.value = Math.min(Math.max(0, event.clientX - rect.left), rect.width)
    pointerY.value = Math.min(Math.max(0, event.clientY - rect.top), rect.height)
  }

  function handleRulerMouseLeave() {
    if (!showRuler.value) return
    if (isNodeTransforming.value) return
    pointerX.value = -9999
    pointerY.value = -9999
  }

  function handleNodeTransform(event: Event) {
    if (!showRuler.value) return
    const ce = event as CustomEvent<{ x?: number; y?: number }>
    if (!containerRef.value || !ce.detail) return
    const rect = containerRef.value.getBoundingClientRect()
    const x = Number(ce.detail.x) || 0
    const y = Number(ce.detail.y) || 0
    const zi = rulerInset.value
    pointerX.value = Math.min(Math.max(0, x * zoom.value + translateX.value + zi), rect.width)
    pointerY.value = Math.min(Math.max(0, y * zoom.value + translateY.value + zi), rect.height)
    isNodeTransforming.value = true
  }

  function handleNodeTransformEnd() {
    isNodeTransforming.value = false
  }

  return {
    pointerX,
    pointerY,
    isNodeTransforming,
    handleRulerMouseMove,
    handleRulerMouseLeave,
    handleNodeTransform,
    handleNodeTransformEnd,
  }
}
