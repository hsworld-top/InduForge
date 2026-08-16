import { describe, expect, it } from 'vitest'
import {
  clampExpandedAiPaneWidth,
  getCollapsedAiPaneWidth,
  MIN_AI_PANE_WIDTH,
  resolveAiPaneLayout,
  shouldUseCompactLayout,
} from './workspace-layout'

describe('workspace layout', () => {
  it('允许 AI 分栏扩大到接近工作台菜单', () => {
    expect(clampExpandedAiPaneWidth(100, 1400)).toBe(MIN_AI_PANE_WIDTH)
    expect(clampExpandedAiPaneWidth(900, 1400)).toBe(900)
    expect(getCollapsedAiPaneWidth(1400)).toBe(1348)
  })

  it('工作台内容接近最小宽度时吸附为仅菜单', () => {
    expect(resolveAiPaneLayout(1100, 1400)).toEqual({
      width: 1100,
      workbenchCollapsed: false,
    })
    expect(resolveAiPaneLayout(1200, 1400)).toEqual({
      width: 1348,
      workbenchCollapsed: true,
    })
  })

  it('空间不足时切换为单面板页签', () => {
    expect(shouldUseCompactLayout(905)).toBe(true)
    expect(shouldUseCompactLayout(906)).toBe(false)
  })
})
