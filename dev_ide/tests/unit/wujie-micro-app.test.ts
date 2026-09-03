import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const readWorkspaceFile = (path: string) => readFileSync(resolve(process.cwd(), '..', path), 'utf8')

describe('Wujie 子应用路由约定', () => {
  it('工作区路由只携带工程和子应用类型，不传认证信息', () => {
    const route = {
      name: 'workspace-micro-app',
      params: { projectId: 'project-1', appType: 'designer' },
    }

    expect(route).toEqual({
      name: 'workspace-micro-app',
      params: { projectId: 'project-1', appType: 'designer' },
    })
    expect(JSON.stringify(route)).not.toContain('token')
  })

  it('设计器必须通过可见的 Wujie 容器加载固定入口', () => {
    const source = readWorkspaceFile('dev_ide/src/components/WujieMicroApp.vue')

    expect(source).toContain(':url="appUrl"')
    expect(source).toContain('const appUrl = computed(() => `/${props.appType}/`)')
    expect(source).toMatch(/\.wujie-micro-app\s*\{[\s\S]*display:\s*block/)
    expect(source).toMatch(/\.wujie-micro-app\s*\{[\s\S]*height:\s*100%/)
  })
})
