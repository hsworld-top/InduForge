export const MIN_AI_PANE_WIDTH = 420
export const MIN_WORKBENCH_PANE_WIDTH = 480
export const WORKBENCH_MENU_WIDTH = 46
export const SPLIT_HANDLE_WIDTH = 6
export const WORKBENCH_COLLAPSE_SNAP_WIDTH = 160

export interface AiPaneLayout {
  width: number
  workbenchCollapsed: boolean
}

export function getCollapsedAiPaneWidth(containerWidth: number): number {
  return Math.max(MIN_AI_PANE_WIDTH, containerWidth - WORKBENCH_MENU_WIDTH - SPLIT_HANDLE_WIDTH)
}

export function clampExpandedAiPaneWidth(requested: number, containerWidth: number): number {
  const availableMaximum = Math.max(
    MIN_AI_PANE_WIDTH,
    getCollapsedAiPaneWidth(containerWidth) - WORKBENCH_COLLAPSE_SNAP_WIDTH,
  )
  return Math.min(Math.max(requested, MIN_AI_PANE_WIDTH), availableMaximum)
}

export function resolveAiPaneLayout(requested: number, containerWidth: number): AiPaneLayout {
  const collapsedWidth = getCollapsedAiPaneWidth(containerWidth)
  const collapseThreshold = collapsedWidth - WORKBENCH_COLLAPSE_SNAP_WIDTH
  if (requested >= collapseThreshold) {
    return { width: collapsedWidth, workbenchCollapsed: true }
  }
  return {
    width: clampExpandedAiPaneWidth(requested, containerWidth),
    workbenchCollapsed: false,
  }
}

export function shouldUseCompactLayout(containerWidth: number): boolean {
  return containerWidth < MIN_AI_PANE_WIDTH + MIN_WORKBENCH_PANE_WIDTH + SPLIT_HANDLE_WIDTH
}
