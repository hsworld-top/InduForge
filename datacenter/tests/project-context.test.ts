import { describe, expect, it } from 'vitest'
import { resolveRouteProjectContext } from '@/router/project-context'

describe('project-context', () => {
  it('优先使用当前 iframe 的 bootstrap 工程上下文', () => {
    const storage = {
      getProjectId: () => 'project-storage',
      getTenantId: () => 'tenant-storage',
    }

    expect(
      resolveRouteProjectContext(
        {
          projectId: 'project-host',
          tenantId: 'tenant-host',
        },
        storage,
      ),
    ).toEqual({
      id: 'project-host',
      tenantId: 'tenant-host',
    })
  })

  it('没有 iframe 上下文时回退到本地存储，兼容独立打开', () => {
    const storage = {
      getProjectId: () => 'project-storage',
      getTenantId: () => 'tenant-storage',
    }

    expect(resolveRouteProjectContext({}, storage)).toEqual({
      id: 'project-storage',
      tenantId: 'tenant-storage',
    })
  })
})
