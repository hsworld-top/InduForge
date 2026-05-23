import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

function read(relativePath: string) {
  return readFileSync(resolve(__dirname, relativePath), 'utf-8')
}

describe('designer sidebar panel theme tokens', () => {
  it('物料区组件面板使用设计器主题变量', () => {
    const content = read('../../ui/editors/page/sidebar-panels/left/ComponentPanel.vue')
    expect(content).toContain('var(--designer-border-color)')
    expect(content).toContain('var(--designer-shell-surface)')
  })

  it('资源区使用设计器主题变量', () => {
    const content = read('../../ui/editors/page/sidebar-panels/left/ResourcePanel.vue')
    expect(content).toContain('var(--designer-border-color)')
    expect(content).toContain('var(--designer-shadow-popover)')
  })

  it('数据点区使用设计器主题变量', () => {
    const content = read('../../ui/editors/page/sidebar-panels/left/DatapointPanel.vue')
    expect(content).toContain('var(--designer-border-color)')
    expect(content).toContain('var(--designer-shell-surface)')
  })

  it('脚本区使用设计器主题变量', () => {
    const content = read('../../ui/shared/tool-panels/ScriptVarsPanel.vue')
    expect(content).toContain('var(--designer-border-color)')
    expect(content).toContain('var(--designer-shadow-popover)')
  })
})
