import type { Ref } from "vue";
import { watch } from "vue";

export interface UseCanvasViewportPlacementOptions {
  containerSize: Ref<{ width: number; height: number }>;
  rulerInset: Ref<number>;
  width: Ref<number>;
  height: Ref<number>;
  zoom: Ref<number>;
  translateX: Ref<number>;
  translateY: Ref<number>;
  defaultPageMarginX: number;
  defaultPageMarginY: number;
  rootNodeId: Ref<string>;
  viewResetToken: Ref<number>;
}

/**
 * 默认页面居中落点、标尺内边距变化平移补偿、缩放时保持视口中心稳定
 */
export function useCanvasViewportPlacement(opts: UseCanvasViewportPlacementOptions): void {
  function applyDefaultPagePlacement() {
    const viewportWidth = Math.max(
      0,
      (opts.containerSize.value.width || 0) - opts.rulerInset.value,
    );
    const viewportHeight = Math.max(
      0,
      (opts.containerSize.value.height || 0) - opts.rulerInset.value,
    );
    if (!viewportWidth || !viewportHeight) return;

    const scaledWidth = opts.width.value * opts.zoom.value;
    const scaledHeight = opts.height.value * opts.zoom.value;
    const nextTranslateX =
      scaledWidth + opts.defaultPageMarginX * 2 <= viewportWidth
        ? Math.round(
            opts.defaultPageMarginX +
              (viewportWidth - scaledWidth - opts.defaultPageMarginX * 2) / 2,
          )
        : Math.max(0, Math.round((viewportWidth - scaledWidth) / 2));
    const nextTranslateY =
      scaledHeight + opts.defaultPageMarginY * 2 <= viewportHeight
        ? Math.round(
            opts.defaultPageMarginY +
              (viewportHeight - scaledHeight - opts.defaultPageMarginY * 2) / 2,
          )
        : Math.max(0, Math.round((viewportHeight - scaledHeight) / 2));

    opts.translateX.value = nextTranslateX;
    opts.translateY.value = nextTranslateY;
  }

  function keepViewportCenterStableOnZoom(prevZoom: number, nextZoom: number) {
    const viewportWidth = Math.max(
      0,
      (opts.containerSize.value.width || 0) - opts.rulerInset.value,
    );
    const viewportHeight = Math.max(
      0,
      (opts.containerSize.value.height || 0) - opts.rulerInset.value,
    );
    if (!viewportWidth || !viewportHeight) return;
    if (!prevZoom || !nextZoom || prevZoom === nextZoom) return;

    const centerX = opts.rulerInset.value + viewportWidth / 2;
    const centerY = opts.rulerInset.value + viewportHeight / 2;
    const canvasX = (centerX - opts.rulerInset.value - opts.translateX.value) / prevZoom;
    const canvasY = (centerY - opts.rulerInset.value - opts.translateY.value) / prevZoom;

    opts.translateX.value = Math.round(centerX - opts.rulerInset.value - canvasX * nextZoom);
    opts.translateY.value = Math.round(centerY - opts.rulerInset.value - canvasY * nextZoom);
  }

  watch(
    () => opts.zoom.value,
    (nextZoom, prevZoom) => {
      if (!Number.isFinite(nextZoom) || !Number.isFinite(prevZoom)) return;
      keepViewportCenterStableOnZoom(prevZoom, nextZoom);
    },
  );

  watch(
    () => opts.rulerInset.value,
    (nextInset, prevInset) => {
      if (!Number.isFinite(nextInset) || !Number.isFinite(prevInset)) return;
      const delta = prevInset - nextInset;
      opts.translateX.value += delta;
      opts.translateY.value += delta;
    },
  );

  watch(
    [
      () => opts.rootNodeId.value,
      () => opts.width.value,
      () => opts.height.value,
      () => opts.viewResetToken.value,
    ],
    () => {
      applyDefaultPagePlacement();
    },
    { immediate: true },
  );

  watch(
    () => [opts.containerSize.value.width, opts.containerSize.value.height],
    () => {
      applyDefaultPagePlacement();
    },
  );
}
