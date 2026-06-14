import { computed, onBeforeUnmount, ref } from 'vue'

type WorkbenchBottomPanelResizeOptions = {
  defaultHeight?: number
  minHeight?: number
  maxHeight?: number
  bodyClass?: string
}

export function useWorkbenchBottomPanelResize(options: WorkbenchBottomPanelResizeOptions = {}) {
  const defaultHeight = options.defaultHeight ?? 320
  const minHeight = options.minHeight ?? 160
  const maxHeight = options.maxHeight ?? 720
  const bodyClass = options.bodyClass ?? 'workbench-panel--resizing'

  const panelHeight = ref(defaultHeight)
  let resizeState: { startY: number; startHeight: number } | null = null

  const clampHeight = (height: number) =>
    Math.min(maxHeight, Math.max(minHeight, Math.round(height)))

  const stopResize = () => {
    if (!resizeState) return
    resizeState = null
    document.body.classList.remove(bodyClass)
    window.removeEventListener('mousemove', handleResize)
    window.removeEventListener('mouseup', stopResize)
  }

  const handleResize = (event: MouseEvent) => {
    if (!resizeState) return
    const delta = resizeState.startY - event.clientY
    panelHeight.value = clampHeight(resizeState.startHeight + delta)
  }

  const startResize = (event: MouseEvent) => {
    resizeState = {
      startY: event.clientY,
      startHeight: panelHeight.value,
    }
    document.body.classList.add(bodyClass)
    window.addEventListener('mousemove', handleResize)
    window.addEventListener('mouseup', stopResize)
  }

  const resetHeight = () => {
    panelHeight.value = defaultHeight
  }

  onBeforeUnmount(stopResize)

  const panelStyle = computed(() => ({
    height: `${panelHeight.value}px`,
  }))

  return {
    panelHeight,
    panelStyle,
    startResize,
    resetHeight,
  }
}
