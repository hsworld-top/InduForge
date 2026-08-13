import { describe, expect, it } from 'vitest'
import {
  clampAiPaneWidth,
  MAX_AI_PANE_WIDTH,
  MIN_AI_PANE_WIDTH,
  shouldUseCompactLayout,
} from './workspace-layout'

describe('workspace layout', () => {
  it('将 AI 分栏宽度限制在可用范围内', () => {
    expect(clampAiPaneWidth(100, 1400)).toBe(MIN_AI_PANE_WIDTH)
    expect(clampAiPaneWidth(900, 1400)).toBe(MAX_AI_PANE_WIDTH)
    expect(clampAiPaneWidth(540, 950)).toBe(463)
  })

  it('空间不足时切换为单面板页签', () => {
    expect(shouldUseCompactLayout(906)).toBe(true)
    expect(shouldUseCompactLayout(907)).toBe(false)
  })
})
