import { describe, expect, it, vi } from 'vitest'
import { CollectorBrowseCache } from '@/components/collector-workbench/collector-browse-cache'

describe('collector browse cache', () => {
  it('reuses cached children until the context is refreshed', async () => {
    const cache = new CollectorBrowseCache<string>()
    let calls = 0
    const loader = async () => {
      calls += 1
      return [`result-${calls}`]
    }

    expect(await cache.get('connection-1', 'root', 'user', loader)).toEqual(['result-1'])
    expect(await cache.get('connection-1', 'root', 'user', loader)).toEqual(['result-1'])
    expect(calls).toBe(1)

    cache.refresh('connection-1')
    expect(await cache.get('connection-1', 'root', 'user', loader)).toEqual(['result-2'])
  })

  it('accepts batch-prefetched children in the current cache epoch', () => {
    const cache = new CollectorBrowseCache<string>()

    cache.set('connection-1', 'parent-1', ['child-1'])

    expect(cache.peek('connection-1', 'parent-1')).toEqual(['child-1'])
    cache.refresh('connection-1')
    expect(cache.peek('connection-1', 'parent-1')).toBeUndefined()
  })

  it('ignores stale batch results after refresh', () => {
    const cache = new CollectorBrowseCache<string>()
    const epoch = cache.epoch('connection-1')

    cache.refresh('connection-1')
    cache.set('connection-1', 'parent-1', ['stale'], epoch)

    expect(cache.peek('connection-1', 'parent-1')).toBeUndefined()
  })

  it('limits background prefetching to three active requests', async () => {
    const cache = new CollectorBrowseCache<string>(3)
    const releases = new Map<string, () => void>()
    let active = 0
    let maxActive = 0
    const loader = (nodeId: string) =>
      new Promise<string[]>((resolve) => {
        active += 1
        maxActive = Math.max(maxActive, active)
        releases.set(nodeId, () => {
          active -= 1
          resolve([nodeId])
        })
      })

    const requests = cache.prefetch('connection-1', ['a', 'b', 'c', 'd'], loader)
    await vi.waitFor(() => expect(releases.size).toBe(3))
    expect(maxActive).toBe(3)

    releases.get('a')?.()
    await vi.waitFor(() => expect(releases.has('d')).toBe(true))
    releases.get('b')?.()
    releases.get('c')?.()
    releases.get('d')?.()
    await Promise.all(requests)
    expect(maxActive).toBe(3)
  })

  it('starts a queued node immediately when the user expands it', async () => {
    const cache = new CollectorBrowseCache<string>(1)
    const releases = new Map<string, () => void>()
    const loader = (nodeId: string) =>
      new Promise<string[]>((resolve) => {
        releases.set(nodeId, () => resolve([nodeId]))
      })

    const [firstPrefetch, secondPrefetch] = cache.prefetch(
      'connection-1',
      ['first', 'second'],
      loader,
    )
    await vi.waitFor(() => expect(releases.has('first')).toBe(true))
    expect(releases.has('second')).toBe(false)

    const userRequest = cache.get('connection-1', 'second', 'user', () => loader('second'))
    await vi.waitFor(() => expect(releases.has('second')).toBe(true))

    releases.get('second')?.()
    releases.get('first')?.()
    await expect(userRequest).resolves.toEqual(['second'])
    await Promise.all([firstPrefetch, secondPrefetch])
  })
})
