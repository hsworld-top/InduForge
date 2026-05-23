import type { Ref } from 'vue'
import { nextTick, watch } from 'vue'

export interface CanvasViewportZoomAnchor {
  viewportX: number
  viewportY: number
  canvasX: number
  canvasY: number
}

export interface UseCanvasViewportPlacementOptions {
  containerSize: Ref<{ width: number; height: number }>
  rulerInset: Ref<number>
  width: Ref<number>
  height: Ref<number>
  zoom: Ref<number>
  translateX: Ref<number>
  translateY: Ref<number>
  canvasOverflowOffset?: Ref<{ left: number; top: number }>
  wrapperRef?: Ref<HTMLElement | null>
  zoomAnchor?: Ref<CanvasViewportZoomAnchor | null>
  defaultPageMarginX: number
  defaultPageMarginY: number
  rootNodeId: Ref<string>
  viewResetToken: Ref<number>
}

/**
 * 默认页面居中落点、标尺内边距变化平移补偿、缩放时保持视口中心稳定
 */
export function useCanvasViewportPlacement(opts: UseCanvasViewportPlacementOptions): void {
  function clampPagePlacementToScrollableOrigin() {
    // 页面左上角不能进入滚动容器的负坐标，否则 scrollLeft/scrollTop 为 0 时也无法看到最左/最上内容。
    opts.translateX.value = Math.max(0, Math.round(opts.translateX.value))
    opts.translateY.value = Math.max(0, Math.round(opts.translateY.value))
  }

  function resolveCurrentOverflowOffsetForZoom(zoom: number): { left: number; top: number } {
    const current = opts.canvasOverflowOffset?.value || { left: 0, top: 0 }
    const activeZoom = opts.zoom.value || zoom
    if (!activeZoom) return { left: 0, top: 0 }
    const ratio = zoom / activeZoom
    return {
      left: current.left * ratio,
      top: current.top * ratio,
    }
  }

  function keepViewportPointStableOnZoom(anchor: CanvasViewportZoomAnchor, nextZoom: number): void {
    const overflowOffset = opts.canvasOverflowOffset?.value || { left: 0, top: 0 }
    const nextTranslateX = Math.max(
      0,
      Math.round(
        anchor.viewportX - opts.rulerInset.value - overflowOffset.left - anchor.canvasX * nextZoom,
      ),
    )
    const nextTranslateY = Math.max(
      0,
      Math.round(
        anchor.viewportY - opts.rulerInset.value - overflowOffset.top - anchor.canvasY * nextZoom,
      ),
    )
    const nextScrollLeft = Math.max(
      0,
      Math.round(
        opts.rulerInset.value +
          nextTranslateX +
          overflowOffset.left +
          anchor.canvasX * nextZoom -
          anchor.viewportX,
      ),
    )
    const nextScrollTop = Math.max(
      0,
      Math.round(
        opts.rulerInset.value +
          nextTranslateY +
          overflowOffset.top +
          anchor.canvasY * nextZoom -
          anchor.viewportY,
      ),
    )

    opts.translateX.value = nextTranslateX
    opts.translateY.value = nextTranslateY
    clampPagePlacementToScrollableOrigin()
    void nextTick(() => {
      const wrapper = opts.wrapperRef?.value || null
      if (!wrapper) return
      wrapper.scrollLeft = nextScrollLeft
      wrapper.scrollTop = nextScrollTop
      const scrollDeltaX = wrapper.scrollLeft - nextScrollLeft
      const scrollDeltaY = wrapper.scrollTop - nextScrollTop
      if (scrollDeltaX) {
        opts.translateX.value = Math.max(0, Math.round(opts.translateX.value + scrollDeltaX))
      }
      if (scrollDeltaY) {
        opts.translateY.value = Math.max(0, Math.round(opts.translateY.value + scrollDeltaY))
      }
    })
  }

  function applyDefaultPagePlacement() {
    const viewportWidth = Math.max(0, (opts.containerSize.value.width || 0) - opts.rulerInset.value)
    const viewportHeight = Math.max(
      0,
      (opts.containerSize.value.height || 0) - opts.rulerInset.value,
    )
    if (!viewportWidth || !viewportHeight) return

    const scaledWidth = opts.width.value * opts.zoom.value
    const scaledHeight = opts.height.value * opts.zoom.value
    const desiredTranslateX =
      scaledWidth + opts.defaultPageMarginX * 2 <= viewportWidth
        ? Math.round(
            opts.defaultPageMarginX +
              (viewportWidth - scaledWidth - opts.defaultPageMarginX * 2) / 2,
          )
        : Math.max(0, Math.round((viewportWidth - scaledWidth) / 2))
    const desiredTranslateY =
      scaledHeight + opts.defaultPageMarginY * 2 <= viewportHeight
        ? Math.round(
            opts.defaultPageMarginY +
              (viewportHeight - scaledHeight - opts.defaultPageMarginY * 2) / 2,
          )
        : Math.max(0, Math.round((viewportHeight - scaledHeight) / 2))
    const overflowOffset = opts.canvasOverflowOffset?.value || { left: 0, top: 0 }
    const nextTranslateX = Math.max(0, Math.round(desiredTranslateX - overflowOffset.left))
    const nextTranslateY = Math.max(0, Math.round(desiredTranslateY - overflowOffset.top))
    const nextScrollLeft = Math.max(
      0,
      Math.round(overflowOffset.left + nextTranslateX - desiredTranslateX),
    )
    const nextScrollTop = Math.max(
      0,
      Math.round(overflowOffset.top + nextTranslateY - desiredTranslateY),
    )

    opts.translateX.value = nextTranslateX
    opts.translateY.value = nextTranslateY
    clampPagePlacementToScrollableOrigin()
    void nextTick(() => {
      const wrapper = opts.wrapperRef?.value || null
      if (!wrapper) return
      wrapper.scrollLeft = nextScrollLeft
      wrapper.scrollTop = nextScrollTop
    })
  }

  function keepViewportCenterStableOnZoom(prevZoom: number, nextZoom: number) {
    const viewportWidth = Math.max(0, (opts.containerSize.value.width || 0) - opts.rulerInset.value)
    const viewportHeight = Math.max(
      0,
      (opts.containerSize.value.height || 0) - opts.rulerInset.value,
    )
    if (!viewportWidth || !viewportHeight) return
    if (!prevZoom || !nextZoom || prevZoom === nextZoom) return

    const anchor = opts.zoomAnchor?.value || null
    if (anchor) {
      opts.zoomAnchor!.value = null
      keepViewportPointStableOnZoom(anchor, nextZoom)
      return
    }

    const wrapper = opts.wrapperRef?.value || null
    if (wrapper) {
      const viewportX = viewportWidth / 2
      const viewportY = viewportHeight / 2
      const prevOverflowOffset = resolveCurrentOverflowOffsetForZoom(prevZoom)
      keepViewportPointStableOnZoom(
        {
          viewportX,
          viewportY,
          canvasX:
            (wrapper.scrollLeft +
              viewportX -
              opts.rulerInset.value -
              opts.translateX.value -
              prevOverflowOffset.left) /
            prevZoom,
          canvasY:
            (wrapper.scrollTop +
              viewportY -
              opts.rulerInset.value -
              opts.translateY.value -
              prevOverflowOffset.top) /
            prevZoom,
        },
        nextZoom,
      )
      return
    }

    const centerX = viewportWidth / 2
    const centerY = viewportHeight / 2
    const prevOverflowOffset = resolveCurrentOverflowOffsetForZoom(prevZoom)
    const nextOverflowOffset = opts.canvasOverflowOffset?.value || { left: 0, top: 0 }
    const canvasX =
      (centerX - opts.rulerInset.value - opts.translateX.value - prevOverflowOffset.left) / prevZoom
    const canvasY =
      (centerY - opts.rulerInset.value - opts.translateY.value - prevOverflowOffset.top) / prevZoom

    opts.translateX.value = Math.round(
      centerX - opts.rulerInset.value - nextOverflowOffset.left - canvasX * nextZoom,
    )
    opts.translateY.value = Math.round(
      centerY - opts.rulerInset.value - nextOverflowOffset.top - canvasY * nextZoom,
    )
    clampPagePlacementToScrollableOrigin()
  }

  watch(
    () => opts.zoom.value,
    (nextZoom, prevZoom) => {
      if (!Number.isFinite(nextZoom) || !Number.isFinite(prevZoom)) return
      keepViewportCenterStableOnZoom(prevZoom, nextZoom)
    },
  )

  watch(
    () => opts.rulerInset.value,
    (nextInset, prevInset) => {
      if (!Number.isFinite(nextInset) || !Number.isFinite(prevInset)) return
      const delta = prevInset - nextInset
      opts.translateX.value += delta
      opts.translateY.value += delta
      clampPagePlacementToScrollableOrigin()
    },
  )

  watch(
    [
      () => opts.rootNodeId.value,
      () => opts.width.value,
      () => opts.height.value,
      () => opts.viewResetToken.value,
    ],
    () => {
      applyDefaultPagePlacement()
    },
    { immediate: true },
  )

  watch(
    () => [opts.containerSize.value.width, opts.containerSize.value.height],
    () => {
      applyDefaultPagePlacement()
    },
  )
}
