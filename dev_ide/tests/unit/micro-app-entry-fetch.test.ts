import { describe, expect, it, vi } from 'vitest'
import { createMicroAppEntryFetch } from '@/utils/micro-app-entry-fetch'

describe('微前端入口请求', () => {
  it('重新加载时忽略旧的设计器入口缓存，获取当前构建入口', async () => {
    const fetcher = vi.fn<typeof window.fetch>().mockResolvedValue(new Response('new-entry'))
    const fetch = createMicroAppEntryFetch('designer', fetcher)

    await fetch('/designer/', { cache: 'force-cache', credentials: 'include' })

    expect(fetcher).toHaveBeenCalledWith(
      '/designer/',
      expect.objectContaining({ cache: 'no-store', credentials: 'include' }),
    )
  })

  it('仅入口 HTML 强制刷新，带 hash 的 CSS/JS 保持服务端缓存策略', async () => {
    const fetcher = vi.fn<typeof window.fetch>().mockResolvedValue(new Response('asset'))
    const fetch = createMicroAppEntryFetch('designer', fetcher)

    await fetch('/designer/assets/index-CvEIMyRj.css')
    await fetch('/designer/index.html')

    expect(fetcher).toHaveBeenNthCalledWith(1, '/designer/assets/index-CvEIMyRj.css', undefined)
    expect(fetcher).toHaveBeenNthCalledWith(
      2,
      '/designer/index.html',
      expect.objectContaining({ cache: 'no-store' }),
    )
  })

  it('不会影响其它微前端的入口请求', async () => {
    const fetcher = vi.fn<typeof window.fetch>().mockResolvedValue(new Response('datacenter'))
    const fetch = createMicroAppEntryFetch('designer', fetcher)

    await fetch('/datacenter/')

    expect(fetcher).toHaveBeenCalledWith('/datacenter/', undefined)
  })
})
