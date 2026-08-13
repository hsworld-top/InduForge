import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const readWorkspaceFile = (path: string) => readFileSync(resolve(process.cwd(), '..', path), 'utf8')

describe('Wujie 子应用高度约束', () => {
  it('宿主标签内容和 Wujie 容器必须完整继承可用高度', () => {
    const microApp = readWorkspaceFile('dev_ide/src/components/WujieMicroApp.vue')
    const tabsArea = readWorkspaceFile('dev_ide/src/views/layout/DashboardTabsArea.vue')

    expect(microApp).toContain('width="100%"')
    expect(microApp).toContain('height="100%"')
    expect(microApp).toMatch(/\.wujie-micro-app\s*\{[\s\S]*height:\s*100%/)
    expect(tabsArea).toMatch(/\.dashboard-tabs \.el-tabs__content\s*\{[\s\S]*height:\s*0/)
    expect(tabsArea).toMatch(/\.dashboard-tab-panel-shell\s*\{[\s\S]*height:\s*100%/)
  })

  it('子应用主框架必须使用父容器高度，不得使用浏览器视口高度', () => {
    const constrainedRoots: Array<[string, RegExp]> = [
      ['datacenter/src/App.vue', /#app\s*\{([^}]*)\}/],
      [
        'datacenter/src/components/datapoint/DataPointWorkspace.vue',
        /\.datapoint-workspace\s*\{([^}]*)\}/,
      ],
      [
        'datacenter/src/views/storage-policy/StoragePolicyWorkspace.vue',
        /\.storage-workspace\s*\{([^}]*)\}/,
      ],
      ['designer/src/App.vue', /#app\s*\{([^}]*)\}/],
      [
        'designer/src/ui/workspaces/DesignerWorkspaceView.vue',
        /\.(?:workspace-shell|ai-workbench)\s*\{([^}]*)\}/,
      ],
    ]

    for (const [file, rootPattern] of constrainedRoots) {
      const source = readWorkspaceFile(file)
      const rootBlock = source.match(rootPattern)?.[1]
      expect(rootBlock, file).toBeDefined()
      expect(rootBlock, file).toMatch(/height:\s*100%/)
      expect(rootBlock, file).not.toMatch(/height:\s*(?:calc\()?100vh/)
    }
  })
})
