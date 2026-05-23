import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { createPageForStore } from './page-create-actions'

describe('page-create-actions', () => {
  it('createPageForStore 会透传 path 并在有 schema 时执行页面保存', async () => {
    const createPage = vi.fn().mockResolvedValue({
      code: 0,
      msg: 'ok',
      data: { id: 'page-1' },
    })
    const updatePage = vi.fn().mockResolvedValue(undefined)
    const refreshPages = vi.fn().mockResolvedValue({
      pages: [],
      entryConfig: {},
    })

    const result = await createPageForStore(
      {
        projectId: ref('project-1'),
        projectApi: {
          createPage,
          updatePage,
        },
        refreshPages,
      },
      {
        name: '登录页',
        type: 'page',
        parentId: null,
        path: '/login',
        schemaContent: { page: { path: '/login' } },
      },
    )

    expect(createPage).toHaveBeenCalledWith('project-1', {
      name: '登录页',
      type: 'page',
      parentId: null,
      path: '/login',
      schemaContent: { page: { path: '/login' } },
    })
    expect(updatePage).toHaveBeenCalledWith('project-1', 'page-1', {
      page: { path: '/login' },
    })
    expect(refreshPages).toHaveBeenCalledTimes(1)
    expect(result).toEqual({ id: 'page-1' })
  })
})
