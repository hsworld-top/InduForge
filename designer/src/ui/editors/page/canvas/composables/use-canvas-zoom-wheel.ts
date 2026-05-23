import type { Ref } from 'vue'

/**
 * Ctrl + 滚轮缩放画布（由容器 @wheel 调用）
 */
export function useCanvasZoomWheel(opts: {
  zoom: Ref<number>
  onZoomChange: (next: number, event: WheelEvent) => void
}) {
  function handleZoomWheel(event: WheelEvent) {
    if (!event.ctrlKey) return
    event.preventDefault()

    const step = 0.1
    const direction = event.deltaY > 0 ? -1 : 1
    const nextZoom = Math.min(5, Math.max(0.1, opts.zoom.value + step * direction))
    if (nextZoom === opts.zoom.value) return

    opts.onZoomChange(Number(nextZoom.toFixed(2)), event)
  }

  return { handleZoomWheel }
}
