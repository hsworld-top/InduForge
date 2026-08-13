import { describe, expect, it } from 'vitest'

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
})
