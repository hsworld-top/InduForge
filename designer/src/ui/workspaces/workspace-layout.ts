export const MIN_AI_PANE_WIDTH = 420
export const MAX_AI_PANE_WIDTH = 560
export const MIN_PREVIEW_PANE_WIDTH = 480
export const SPLIT_HANDLE_WIDTH = 7

export function clampAiPaneWidth(requested: number, containerWidth: number): number {
  const availableMaximum = Math.max(
    MIN_AI_PANE_WIDTH,
    containerWidth - MIN_PREVIEW_PANE_WIDTH - SPLIT_HANDLE_WIDTH,
  )
  return Math.min(Math.max(requested, MIN_AI_PANE_WIDTH), MAX_AI_PANE_WIDTH, availableMaximum)
}

export function shouldUseCompactLayout(containerWidth: number): boolean {
  return containerWidth < MIN_AI_PANE_WIDTH + MIN_PREVIEW_PANE_WIDTH + SPLIT_HANDLE_WIDTH
}
